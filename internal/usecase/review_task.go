package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

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

	// Validate status - must be for_review or reviewing
	// reviewing is allowed because CompleteTask transitions to it before starting review
	if task.Status != domain.StatusForReview && task.Status != domain.StatusReviewing {
		return nil, fmt.Errorf("cannot review task in %s status (must be for_review or reviewing): %w", task.Status, domain.ErrInvalidTransition)
	}

	// Check if review session is already running
	sessionName := domain.ReviewSessionName(task.ID)
	if runningErr := shared.EnsureNoRunningReviewSession(uc.sessions, task.ID); runningErr != nil {
		return nil, runningErr
	}

	// Load config for agent resolution
	cfg, err := uc.configLoader.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Resolve agent from input or default
	agentName := in.Agent
	if agentName == "" {
		agentName = cfg.AgentsConfig.DefaultReviewer
	}

	// Get agent configuration from enabled agents only
	agent, ok := cfg.EnabledAgents()[agentName]
	if !ok {
		// Check if agent exists but is disabled
		if _, exists := cfg.Agents[agentName]; exists {
			return nil, fmt.Errorf("agent %q is disabled: %w", agentName, domain.ErrAgentDisabled)
		}
		return nil, fmt.Errorf("agent %q: %w", agentName, domain.ErrAgentNotFound)
	}

	// Resolve model priority: CLI flag > agent config > builtin default
	model := in.Model
	if model == "" {
		model = agent.DefaultModel
	}

	// Get worktree path
	branch := domain.BranchName(task.ID, task.Issue)
	wtPath, err := uc.worktrees.Resolve(branch)
	if err != nil {
		return nil, fmt.Errorf("resolve worktree: %w", err)
	}

	// Build command data for template expansion
	cmdData := domain.CommandData{
		GitDir:      uc.repoRoot + "/.git",
		RepoRoot:    uc.repoRoot,
		Worktree:    wtPath,
		Title:       task.Title,
		Description: task.Description,
		Branch:      branch,
		Issue:       task.Issue,
		TaskID:      task.ID,
		Model:       model,
	}

	// Build user prompt (fallback for when Agent.Prompt is empty)
	// Final priority: Agent.Prompt > in.Message > AgentsConfig.ReviewerPrompt > default
	// Note: Agent.Prompt is applied in agent.RenderCommand() and overrides this userPrompt
	userPrompt := in.Message
	if userPrompt == "" {
		userPrompt = cfg.AgentsConfig.ReviewerPrompt
	}
	if userPrompt == "" {
		userPrompt = "Please review this task."
	}

	// Render command and prompt
	defaultSystemPrompt := domain.DefaultReviewerSystemPrompt
	result, err := agent.RenderCommand(cmdData, `"$PROMPT"`, defaultSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("render agent command: %w", err)
	}

	if in.Wait {
		// Synchronous execution
		return uc.executeSync(ctx, task, wtPath, result)
	}

	// Background execution (default)
	return uc.executeBackground(ctx, task, wtPath, result, agentName, sessionName)
}

// executeSync runs the review synchronously and returns the result.
func (uc *ReviewTask) executeSync(ctx context.Context, task *domain.Task, wtPath string, result domain.RenderCommandResult) (*ReviewTaskOutput, error) {
	// Update status to reviewing
	task.Status = domain.StatusReviewing
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	// Build the script to run
	script := fmt.Sprintf(`#!/bin/bash
set -o pipefail

read -r -d '' PROMPT << 'END_OF_PROMPT'
%s
END_OF_PROMPT

%s
`, result.Prompt, result.Command)

	// Build command
	execCmd := domain.NewBashCommand(script, wtPath)

	// Capture output
	var stdoutBuf, stderrBuf bytes.Buffer

	// Run the command using CommandExecutor
	if err := uc.executor.ExecuteWithContext(ctx, execCmd, &stdoutBuf, &stderrBuf); err != nil {
		// Revert status to for_review on error
		task.Status = domain.StatusForReview
		_ = uc.tasks.Save(task)

		// Include stderr in error message for debugging
		errMsg := strings.TrimSpace(stderrBuf.String())
		if errMsg != "" {
			return nil, fmt.Errorf("run reviewer: %w: %s", err, errMsg)
		}
		return nil, fmt.Errorf("run reviewer: %w", err)
	}

	// Extract final review result
	review := strings.TrimSpace(extractReviewResult(stdoutBuf.String()))

	// Save review result as comment
	if review != "" {
		comment := domain.Comment{
			Text:   review,
			Time:   uc.clock.Now(),
			Author: "reviewer",
		}
		if err := uc.tasks.AddComment(task.ID, comment); err != nil {
			// Log but don't fail - the review was successful
			_, _ = fmt.Fprintf(uc.stderr, "warning: failed to add review comment: %v\n", err)
		}
	}

	// Update task status to reviewed
	task.Status = domain.StatusReviewed
	if err := uc.tasks.Save(task); err != nil {
		// Log but don't fail - the review was successful
		_, _ = fmt.Fprintf(uc.stderr, "warning: failed to update task status: %v\n", err)
	}

	return &ReviewTaskOutput{
		Review: strings.TrimSpace(review),
		Task:   task,
	}, nil
}

// executeBackground starts the review in a background tmux session.
func (uc *ReviewTask) executeBackground(ctx context.Context, task *domain.Task, wtPath string, result domain.RenderCommandResult, agentName, sessionName string) (*ReviewTaskOutput, error) {
	// Generate the review script
	scriptPath, err := uc.generateReviewScript(task, result)
	if err != nil {
		return nil, fmt.Errorf("generate review script: %w", err)
	}

	// Update status to reviewing
	task.Status = domain.StatusReviewing
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
	}); err != nil {
		// Rollback status
		task.Status = domain.StatusForReview
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
func (uc *ReviewTask) generateReviewScript(task *domain.Task, result domain.RenderCommandResult) (string, error) {
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

	tmpl := template.Must(template.New("review-script").Parse(reviewScriptTemplate))

	data := reviewScriptData{
		TaskID:       task.ID,
		AgentCommand: result.Command,
		Prompt:       result.Prompt,
		CrewBin:      crewBin,
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
	TaskID       int
}

// reviewScriptTemplate is the template for the review script.
// The prompt is embedded using a heredoc to avoid escaping issues.
// Output is captured and sent to _review-session-ended on completion.
const reviewScriptTemplate = `#!/bin/bash
set -o pipefail

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

// extractReviewResult extracts the final review result from the full output.
// It looks for domain.ReviewResultMarker and returns everything after it.
// If the marker is not found, it returns the full output as a fallback.
func extractReviewResult(output string) string {
	if idx := strings.Index(output, domain.ReviewResultMarker); idx != -1 {
		return output[idx+len(domain.ReviewResultMarker):]
	}
	return output // fallback: return full output
}
