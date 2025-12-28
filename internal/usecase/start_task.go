// Package usecase contains application use cases.
package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// StartTaskInput contains the parameters for starting a task.
// Fields are ordered to minimize memory padding.
type StartTaskInput struct {
	Agent  string // Agent name (required in MVP)
	TaskID int    // Task ID to start
}

// StartTaskOutput contains the result of starting a task.
type StartTaskOutput struct {
	SessionName  string // Name of the tmux session
	WorktreePath string // Path to the worktree
}

// StartTask is the use case for starting a task.
type StartTask struct {
	tasks        domain.TaskRepository
	sessions     domain.SessionManager
	worktrees    domain.WorktreeManager
	configLoader domain.ConfigLoader
	clock        domain.Clock
	crewDir      string // Path to .git/crew directory
}

// NewStartTask creates a new StartTask use case.
func NewStartTask(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	worktrees domain.WorktreeManager,
	configLoader domain.ConfigLoader,
	clock domain.Clock,
	crewDir string,
) *StartTask {
	return &StartTask{
		tasks:        tasks,
		sessions:     sessions,
		worktrees:    worktrees,
		configLoader: configLoader,
		clock:        clock,
		crewDir:      crewDir,
	}
}

// Execute starts a task with the given input.
func (uc *StartTask) Execute(ctx context.Context, in StartTaskInput) (*StartTaskOutput, error) {
	// Get task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Validate status transition
	if !task.Status.CanTransitionTo(domain.StatusInProgress) {
		return nil, fmt.Errorf("cannot start task in %s status: %w", task.Status, domain.ErrInvalidTransition)
	}

	// Check if session is already running
	sessionName := domain.SessionName(task.ID)
	running, err := uc.sessions.IsRunning(sessionName)
	if err != nil {
		return nil, fmt.Errorf("check session: %w", err)
	}
	if running {
		return nil, domain.ErrSessionRunning
	}

	// Resolve agent from input or config
	agent := in.Agent
	if agent == "" {
		// Try to get default_agent from config
		cfg, loadErr := uc.configLoader.Load()
		if loadErr != nil {
			return nil, fmt.Errorf("load config: %w", loadErr)
		}
		agent = cfg.DefaultAgent
	}
	if agent == "" {
		return nil, domain.ErrNoAgent
	}

	// Create or resolve worktree
	branch := domain.BranchName(task.ID, task.Issue)
	baseBranch := task.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	wtPath, err := uc.worktrees.Create(branch, baseBranch)
	if err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	// Generate prompt and script files
	scriptPath, err := uc.generateScript(task, agent, wtPath)
	if err != nil {
		_ = uc.worktrees.Remove(branch)
		return nil, fmt.Errorf("generate script: %w", err)
	}

	// Start session with the generated script
	if err := uc.sessions.Start(ctx, domain.StartSessionOptions{
		Name:    sessionName,
		Dir:     wtPath,
		Command: scriptPath,
		TaskID:  task.ID,
	}); err != nil {
		// Rollback: cleanup script and worktree
		uc.cleanupScript(task.ID)
		_ = uc.worktrees.Remove(branch)
		return nil, fmt.Errorf("start session: %w", err)
	}

	// Update task status
	task.Status = domain.StatusInProgress
	task.Agent = agent
	task.Session = sessionName
	task.Started = uc.clock.Now()
	if err := uc.tasks.Save(task); err != nil {
		// Rollback: stop session, cleanup script and worktree
		_ = uc.sessions.Stop(sessionName)
		uc.cleanupScript(task.ID)
		_ = uc.worktrees.Remove(branch)
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &StartTaskOutput{
		SessionName:  sessionName,
		WorktreePath: wtPath,
	}, nil
}

// generateScript creates the task script with embedded prompt.
// Returns the path to the generated script.
func (uc *StartTask) generateScript(task *domain.Task, agent, worktreePath string) (string, error) {
	scriptsDir := filepath.Join(uc.crewDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0750); err != nil {
		return "", fmt.Errorf("create scripts directory: %w", err)
	}

	// Build prompt and script
	prompt := uc.buildPrompt(task, worktreePath)
	script, err := uc.buildScript(task, agent, prompt)
	if err != nil {
		return "", fmt.Errorf("build script: %w", err)
	}

	// Write script file (executable)
	scriptPath := domain.ScriptPath(uc.crewDir, task.ID)
	// G306: We intentionally use 0700 because this is an executable script
	if err := os.WriteFile(scriptPath, []byte(script), 0700); err != nil { //nolint:gosec // executable script requires execute permission
		return "", fmt.Errorf("write script file: %w", err)
	}

	return scriptPath, nil
}

// buildPrompt constructs the prompt text for the agent.
func (uc *StartTask) buildPrompt(task *domain.Task, worktreePath string) string {
	// Build a prompt that includes task information
	prompt := fmt.Sprintf(`# Task %d: %s

`, task.ID, task.Title)

	if task.Description != "" {
		prompt += task.Description + "\n\n"
	}

	prompt += fmt.Sprintf(`## Task Information
- Branch: %s
- Worktree: %s
`, domain.BranchName(task.ID, task.Issue), worktreePath)

	if task.Issue > 0 {
		prompt += fmt.Sprintf("- Issue: #%d\n", task.Issue)
	}

	// Add completion instruction
	prompt += `
## Instructions
When the task is complete, run 'git crew complete' to mark it as done.
`

	return prompt
}

// scriptTemplateData holds the data for script template execution.
// Fields are ordered to minimize memory padding.
type scriptTemplateData struct {
	Agent   string
	Prompt  string
	CrewBin string
	TaskID  int
}

// buildScript constructs the task script with embedded prompt and session-ended callback.
func (uc *StartTask) buildScript(task *domain.Task, agent string, prompt string) (string, error) {
	// Find crew binary path (for _session-ended callback)
	crewBin, err := os.Executable()
	if err != nil {
		// Fallback to "crew" and hope it's in PATH
		crewBin = "crew"
	}

	tmpl := template.Must(template.New("script").Parse(scriptTemplate))

	data := scriptTemplateData{
		TaskID:  task.ID,
		Agent:   agent,
		Prompt:  prompt,
		CrewBin: crewBin,
	}

	var script strings.Builder
	if err := tmpl.Execute(&script, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return script.String(), nil
}

// cleanupScript removes the generated script file.
func (uc *StartTask) cleanupScript(taskID int) {
	scriptPath := domain.ScriptPath(uc.crewDir, taskID)
	_ = os.Remove(scriptPath)
}

// scriptTemplate is the template for the task script.
// The prompt is embedded using a heredoc to avoid escaping issues.
const scriptTemplate = `#!/bin/bash
set -o pipefail

# Embedded prompt
read -r -d '' PROMPT << 'END_OF_PROMPT'
{{.Prompt}}
END_OF_PROMPT

# Callback on session termination
SESSION_ENDED() {
  local code=$?
  "{{.CrewBin}}" _session-ended {{.TaskID}} "$code" || true
}

# Signal handling
trap SESSION_ENDED EXIT    # Both normal and abnormal exit
trap 'exit 130' INT        # Ctrl+C -> exit code 130
trap 'exit 143' TERM       # kill -> exit code 143
trap 'exit 129' HUP        # hangup -> exit code 129

# Run agent
{{.Agent}} "$PROMPT"
`
