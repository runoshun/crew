package usecase

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ReviewTaskInput contains the parameters for reviewing a task.
// Fields are ordered to minimize memory padding.
type ReviewTaskInput struct {
	Agent   string // Agent name (optional, uses default reviewer if empty)
	Model   string // Model name override (optional, uses agent's default if empty)
	Message string // Additional instructions for the reviewer (optional)
	TaskID  int    // Task ID to review
	Wait    bool   // Wait for review to complete (synchronous mode)
}

// ReviewTaskOutput contains the result of reviewing a task.
// Fields are ordered to minimize memory padding.
type ReviewTaskOutput struct {
	Task        *domain.Task // The reviewed task
	Review      string       // Review result from the agent (only in sync mode)
	SessionName string       // Session name (only in background mode)
}

// ReviewTask is the use case for reviewing a task with an AI agent.
// Fields are ordered to minimize memory padding.
type ReviewTask struct {
	tasks        domain.TaskRepository
	sessions     domain.SessionManager
	worktrees    domain.WorktreeManager
	configLoader domain.ConfigLoader
	executor     domain.CommandExecutor
	clock        domain.Clock
	logger       domain.Logger
	stdout       io.Writer
	stderr       io.Writer
	crewDir      string
	repoRoot     string
}

// NewReviewTask creates a new ReviewTask use case.
func NewReviewTask(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	worktrees domain.WorktreeManager,
	configLoader domain.ConfigLoader,
	executor domain.CommandExecutor,
	clock domain.Clock,
	logger domain.Logger,
	crewDir string,
	repoRoot string,
	stdout, stderr io.Writer,
) *ReviewTask {
	return &ReviewTask{
		tasks:        tasks,
		sessions:     sessions,
		worktrees:    worktrees,
		configLoader: configLoader,
		executor:     executor,
		clock:        clock,
		logger:       logger,
		crewDir:      crewDir,
		repoRoot:     repoRoot,
		stdout:       stdout,
		stderr:       stderr,
	}
}

// Execute reviews a task using the specified AI agent.
// By default, runs in background. Use Wait=true for synchronous execution.
func (uc *ReviewTask) Execute(ctx context.Context, in ReviewTaskInput) (*ReviewTaskOutput, error) {
	// Get task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	// Validate status - must be in_progress or done
	// done is allowed because CompleteTask transitions to it before starting review
	if task.Status != domain.StatusInProgress && task.Status != domain.StatusDone {
		return nil, fmt.Errorf("cannot review task in %s status (must be in_progress or done): %w", task.Status, domain.ErrInvalidTransition)
	}

	// Check if review session is already running
	sessionName := domain.ReviewSessionName(task.ID)
	if runningErr := shared.EnsureNoRunningReviewSession(uc.sessions, task.ID); runningErr != nil {
		return nil, runningErr
	}

	reviewCmd, err := shared.PrepareReviewCommand(shared.ReviewCommandDeps{
		ConfigLoader: uc.configLoader,
		Worktrees:    uc.worktrees,
		RepoRoot:     uc.repoRoot,
	}, shared.ReviewCommandInput{
		Task:    task,
		Agent:   in.Agent,
		Model:   in.Model,
		Message: in.Message,
	})
	if err != nil {
		return nil, err
	}

	if in.Wait {
		// Synchronous execution
		return uc.executeSync(ctx, task, reviewCmd)
	}

	// Background execution (default)
	return uc.executeBackground(ctx, task, reviewCmd.WorktreePath, reviewCmd.Result, reviewCmd.AgentName, sessionName)
}

// executeSync runs the review synchronously and returns the result.
func (uc *ReviewTask) executeSync(ctx context.Context, task *domain.Task, reviewCmd *shared.ReviewCommandOutput) (*ReviewTaskOutput, error) {
	// Keep status as in_progress during review
	originalStatus := task.Status
	task.Status = domain.StatusInProgress
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	reviewOut, err := shared.ExecuteReview(ctx, shared.ReviewDeps{
		Tasks:    uc.tasks,
		Executor: uc.executor,
		Clock:    uc.clock,
		Stderr:   uc.stderr,
	}, shared.ReviewInput{
		Task:            task,
		WorktreePath:    reviewCmd.WorktreePath,
		Result:          reviewCmd.Result,
		SkipStatusCheck: true,
	})
	if err != nil {
		// Revert status to original on error
		task.Status = originalStatus
		_ = uc.tasks.Save(task)
		return nil, err
	}

	// Update task status to done (review complete)
	task.Status = domain.StatusDone
	if err := uc.tasks.Save(task); err != nil {
		// Log but don't fail - the review was successful
		_, _ = fmt.Fprintf(uc.stderr, "warning: failed to update task status: %v\n", err)
	}

	return &ReviewTaskOutput{
		Review: reviewOut.Review,
		Task:   task,
	}, nil
}

// executeBackground starts the review in a background tmux session.
func (uc *ReviewTask) executeBackground(ctx context.Context, task *domain.Task, wtPath string, result domain.RenderCommandResult, agentName, sessionName string) (*ReviewTaskOutput, error) {
	// Generate the review script
	scriptPath, err := uc.generateReviewScript(task, wtPath, result)
	if err != nil {
		return nil, fmt.Errorf("generate review script: %w", err)
	}

	// Keep status as in_progress during background review
	originalStatus := task.Status
	task.Status = domain.StatusInProgress
	if err := uc.tasks.Save(task); err != nil {
		uc.cleanupScript(task.ID)
		return nil, fmt.Errorf("save task: %w", err)
	}

	// Start session with the generated script
	if err := uc.sessions.Start(ctx, domain.StartSessionOptions{
		Name:      sessionName,
		Dir:       wtPath,
		Command:   scriptPath,
		TaskID:    task.ID,
		TaskTitle: task.Title,
		TaskAgent: agentName,
		Type:      domain.SessionTypeReviewer,
	}); err != nil {
		// Rollback status
		task.Status = originalStatus
		_ = uc.tasks.Save(task)
		uc.cleanupScript(task.ID)
		return nil, fmt.Errorf("start session: %w", err)
	}

	// Log review start
	if uc.logger != nil {
		uc.logger.Info(task.ID, "review", fmt.Sprintf("started with agent %q", agentName))
	}

	return &ReviewTaskOutput{
		Task:        task,
		SessionName: sessionName,
	}, nil
}

// generateReviewScript creates the review script with embedded prompt.
func (uc *ReviewTask) generateReviewScript(task *domain.Task, worktreePath string, result domain.RenderCommandResult) (string, error) {
	scriptsDir := filepath.Join(uc.crewDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0750); err != nil {
		return "", fmt.Errorf("create scripts directory: %w", err)
	}

	// Find crew binary path (for _review-session-ended callback)
	crewBin, err := os.Executable()
	if err != nil {
		// Fallback to "crew" and hope it's in PATH
		crewBin = "crew"
	}

	sessionName := domain.ReviewSessionName(task.ID)
	logPath := domain.SessionLogPath(uc.crewDir, sessionName)

	// Write session log header before script execution
	if err := writeReviewSessionLogHeader(logPath, sessionName, worktreePath, result.Command); err != nil {
		return "", fmt.Errorf("write session log header: %w", err)
	}

	tmpl := template.Must(template.New("review-script").Parse(reviewScriptTemplate))

	data := reviewScriptData{
		AgentCommand: result.Command,
		Prompt:       result.Prompt,
		CrewBin:      crewBin,
		LogPath:      logPath,
		TaskID:       task.ID,
	}

	var script strings.Builder
	if err := tmpl.Execute(&script, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	// Write script file (executable)
	scriptPath := domain.ReviewScriptPath(uc.crewDir, task.ID)
	// G306: We intentionally use 0700 because this is an executable script
	if err := os.WriteFile(scriptPath, []byte(script.String()), 0700); err != nil { //nolint:gosec // executable script requires execute permission
		return "", fmt.Errorf("write script file: %w", err)
	}

	return scriptPath, nil
}

// writeReviewSessionLogHeader writes the session metadata header to the log file.
func writeReviewSessionLogHeader(logPath, sessionName, workDir, command string) error {
	logsDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logsDir, 0750); err != nil {
		return fmt.Errorf("create logs directory: %w", err)
	}

	header := fmt.Sprintf(`================================================================================
Session: %s
Started: %s
Directory: %s
Command: %s
================================================================================

`, sessionName, time.Now().UTC().Format(time.RFC3339), workDir, command)

	if err := os.WriteFile(logPath, []byte(header), 0600); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	return nil
}

// cleanupScript removes the generated review script file.
func (uc *ReviewTask) cleanupScript(taskID int) {
	scriptPath := domain.ReviewScriptPath(uc.crewDir, taskID)
	_ = os.Remove(scriptPath)
}

// reviewScriptData holds the data for review script template execution.
type reviewScriptData struct {
	AgentCommand string
	Prompt       string
	CrewBin      string
	LogPath      string
	TaskID       int
}

// reviewScriptTemplate is the template for the review script.
// The prompt is embedded using a heredoc to avoid escaping issues.
// Output is captured and sent to _review-session-ended on completion.
const reviewScriptTemplate = `#!/bin/bash
set -o pipefail

# Redirect stderr to session log
exec 2>>"{{.LogPath}}"

# Output capture file
OUTPUT_FILE=$(mktemp)

# Embedded prompt
read -r -d '' PROMPT << 'END_OF_PROMPT'
{{.Prompt}}
END_OF_PROMPT

# Callback on session termination (also cleans up temp file)
REVIEW_SESSION_ENDED() {
  local code=$?
  "{{.CrewBin}}" _review-session-ended {{.TaskID}} "$code" "$OUTPUT_FILE" || true
  rm -f "$OUTPUT_FILE"
}

# Signal handling
trap REVIEW_SESSION_ENDED EXIT    # Both normal and abnormal exit
trap 'exit 130' INT               # Ctrl+C -> exit code 130
trap 'exit 143' TERM              # kill -> exit code 143
trap 'exit 129' HUP               # hangup -> exit code 129

# Run agent and capture output
{{.AgentCommand}} | tee "$OUTPUT_FILE"
`
