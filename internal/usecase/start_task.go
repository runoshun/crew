// Package usecase contains application use cases.
package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// StartTaskInput contains the parameters for starting a task.
// Fields are ordered to minimize memory padding.
type StartTaskInput struct {
	Agent    string // Agent name (required in MVP)
	Model    string // Model name override (optional, uses agent's default if empty)
	TaskID   int    // Task ID to start
	Continue bool   // Continue from previous session (adds agent-specific continue args)
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
	git          domain.Git
	clock        domain.Clock
	logger       domain.Logger
	crewDir      string // Path to .git/crew directory
	repoRoot     string // Repository root path
}

// NewStartTask creates a new StartTask use case.
func NewStartTask(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	worktrees domain.WorktreeManager,
	configLoader domain.ConfigLoader,
	git domain.Git,
	clock domain.Clock,
	logger domain.Logger,
	crewDir string,
	repoRoot string,
) *StartTask {
	return &StartTask{
		tasks:        tasks,
		sessions:     sessions,
		worktrees:    worktrees,
		configLoader: configLoader,
		git:          git,
		clock:        clock,
		logger:       logger,
		crewDir:      crewDir,
		repoRoot:     repoRoot,
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

	// Check if session is already running
	sessionName := domain.SessionName(task.ID)
	running, err := uc.sessions.IsRunning(sessionName)
	if err != nil {
		return nil, fmt.Errorf("check session: %w", err)
	}
	if running {
		return nil, fmt.Errorf("task #%d session is already running: %w", task.ID, domain.ErrSessionRunning)
	}

	// Load config for agent resolution and command building
	cfg, loadErr := uc.configLoader.Load()
	if loadErr != nil {
		return nil, fmt.Errorf("load config: %w", loadErr)
	}

	// Resolve agent from input or default
	agentName := in.Agent
	if agentName == "" {
		agentName = cfg.AgentsConfig.DefaultWorker
	}

	// Get agent configuration
	agent, ok := cfg.Agents[agentName]
	if !ok {
		return nil, fmt.Errorf("agent %q: %w", agentName, domain.ErrAgentNotFound)
	}

	// Check if agent is disabled
	if domain.IsAgentDisabled(agentName, cfg.AgentsConfig.DisabledAgents) {
		return nil, fmt.Errorf("agent %q is disabled: %w", agentName, domain.ErrAgentDisabled)
	}

	// Resolve model priority: CLI flag > agent config > builtin default
	model := in.Model
	if model == "" {
		model = agent.DefaultModel
	}

	// Create or resolve worktree
	branch := domain.BranchName(task.ID, task.Issue)
	baseBranch := task.BaseBranch
	if baseBranch == "" {
		// Use GetDefaultBranch for backward compatibility
		defaultBranch, defaultErr := uc.git.GetDefaultBranch()
		if defaultErr != nil {
			return nil, fmt.Errorf("get default branch: %w", defaultErr)
		}
		baseBranch = defaultBranch
	}

	wtPath, err := uc.worktrees.Create(branch, baseBranch)
	if err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	// Setup worktree (copy files and run setup command)
	if setupErr := uc.worktrees.SetupWorktree(wtPath, &cfg.Worktree); setupErr != nil {
		_ = uc.worktrees.Remove(branch)
		return nil, fmt.Errorf("setup worktree: %w", setupErr)
	}

	// Setup agent-specific configurations (SetupScript)
	if setupErr := uc.setupAgent(task, wtPath, agent); setupErr != nil {
		_ = uc.worktrees.Remove(branch)
		return nil, fmt.Errorf("setup agent: %w", setupErr)
	}

	// Generate prompt and script files
	scriptPath, err := uc.generateScript(task, wtPath, agent, model, in.Continue, cfg)
	if err != nil {
		_ = uc.worktrees.Remove(branch)
		return nil, fmt.Errorf("generate script: %w", err)
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
		// Rollback: cleanup script and worktree
		uc.cleanupScript(task.ID)
		_ = uc.worktrees.Remove(branch)
		return nil, fmt.Errorf("start session: %w", err)
	}

	// Update task status
	task.Status = domain.StatusInProgress
	task.Agent = agentName
	task.Session = sessionName
	task.Started = uc.clock.Now()
	if err := uc.tasks.Save(task); err != nil {
		// Rollback: stop session, cleanup script and worktree
		_ = uc.sessions.Stop(sessionName)
		uc.cleanupScript(task.ID)
		_ = uc.worktrees.Remove(branch)
		return nil, fmt.Errorf("save task: %w", err)
	}

	// Log task start
	if uc.logger != nil {
		uc.logger.Info(task.ID, "task", fmt.Sprintf("started with agent %q", agentName))
	}

	return &StartTaskOutput{
		SessionName:  sessionName,
		WorktreePath: wtPath,
	}, nil
}

// generateScript creates the task script with embedded prompt.
// Returns the path to the generated script.
func (uc *StartTask) generateScript(task *domain.Task, worktreePath string, agent domain.Agent, model string, continueFlag bool, cfg *domain.Config) (string, error) {
	scriptsDir := filepath.Join(uc.crewDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0750); err != nil {
		return "", fmt.Errorf("create scripts directory: %w", err)
	}

	// Build command and prompt using RenderCommand
	script, err := uc.buildScript(task, worktreePath, agent, model, continueFlag, cfg)
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

// scriptTemplateData holds the data for script template execution.
// Fields are ordered to minimize memory padding.
type scriptTemplateData struct {
	AgentCommand string
	Prompt       string
	CrewBin      string
	TaskID       int
}

// buildScript constructs the task script with embedded prompt and session-ended callback.
func (uc *StartTask) buildScript(task *domain.Task, worktreePath string, agent domain.Agent, model string, continueFlag bool, cfg *domain.Config) (string, error) {
	// Find crew binary path (for _session-ended callback)
	crewBin, err := os.Executable()
	if err != nil {
		// Fallback to "crew" and hope it's in PATH
		crewBin = "crew"
	}

	// Build command data for template expansion
	gitDir := filepath.Join(uc.repoRoot, ".git")
	cmdData := domain.CommandData{
		GitDir:      gitDir,
		RepoRoot:    uc.repoRoot,
		Worktree:    worktreePath,
		Title:       task.Title,
		Description: task.Description,
		Branch:      domain.BranchName(task.ID, task.Issue),
		Issue:       task.Issue,
		TaskID:      task.ID,
		Model:       model,
		Continue:    continueFlag,
	}

	// Determine default prompts: agent has its own SystemPrompt, or use default
	defaultSystemPrompt := domain.DefaultSystemPrompt
	defaultPrompt := cfg.AgentsConfig.WorkerPrompt

	// Render command and prompt using Agent.RenderCommand
	// Pass shell variable reference as promptOverride - will be expanded at runtime
	result, err := agent.RenderCommand(cmdData, `"$PROMPT"`, defaultSystemPrompt, defaultPrompt)
	if err != nil {
		return "", fmt.Errorf("render agent command: %w", err)
	}

	tmpl := template.Must(template.New("script").Parse(scriptTemplate))

	data := scriptTemplateData{
		TaskID:       task.ID,
		AgentCommand: result.Command,
		Prompt:       result.Prompt,
		CrewBin:      crewBin,
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
{{.AgentCommand}}
`

// agentSetupData holds the data for agent setup script template expansion.
type agentSetupData struct {
	GitDir   string // Path to .git directory
	RepoRoot string // Repository root path
	Worktree string // Worktree path
	TaskID   int    // Task ID
}

// setupAgent runs agent-specific setup: SetupScript execution.
// The setup script now handles exclude patterns directly.
func (uc *StartTask) setupAgent(task *domain.Task, wtPath string, agent domain.Agent) error {
	// Run setup script (which now includes exclude pattern setup)
	if agent.SetupScript != "" {
		if err := uc.runSetupScript(task, wtPath, agent.SetupScript); err != nil {
			return fmt.Errorf("run setup script: %w", err)
		}
	}

	return nil
}

// runSetupScript expands and runs the agent's setup script.
func (uc *StartTask) runSetupScript(task *domain.Task, wtPath, scriptTemplate string) error {
	gitDir := filepath.Join(uc.repoRoot, ".git")

	// Expand template
	data := agentSetupData{
		GitDir:   gitDir,
		RepoRoot: uc.repoRoot,
		Worktree: wtPath,
		TaskID:   task.ID,
	}

	tmpl, err := template.New("setup").Parse(scriptTemplate)
	if err != nil {
		return fmt.Errorf("parse setup script template: %w", err)
	}

	var script strings.Builder
	if err := tmpl.Execute(&script, data); err != nil {
		return fmt.Errorf("expand setup script template: %w", err)
	}

	// Execute the script
	// G204: Script content is from built-in agent config or user config (trusted source)
	cmd := exec.Command("sh", "-c", script.String()) //nolint:gosec // Script from trusted config
	cmd.Dir = wtPath

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("execute setup script: %w: %s", err, string(out))
	}

	return nil
}
