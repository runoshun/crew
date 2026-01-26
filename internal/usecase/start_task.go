// Package usecase contains application use cases.
package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// StartTaskInput contains the parameters for starting a task.
// Fields are ordered to minimize memory padding.
type StartTaskInput struct {
	SkipReview        *bool    // Set task's SkipReview flag on start (nil=no change, true=skip, false=require review)
	Agent             string   // Agent name (required in MVP)
	Model             string   // Model name override (optional, uses agent's default if empty)
	AdditionalPrompts []string // Additional prompts to append (optional, multiple allowed)
	TaskID            int      // Task ID to start
	Continue          bool     // Continue from previous session (adds agent-specific continue args)
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
	runner       domain.ScriptRunner
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
	runner domain.ScriptRunner,
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
		runner:       runner,
		crewDir:      crewDir,
		repoRoot:     repoRoot,
	}
}

// Execute starts a task with the given input.
func (uc *StartTask) Execute(ctx context.Context, in StartTaskInput) (*StartTaskOutput, error) {
	// Get task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	// Check if task is blocked
	if task.IsBlocked() {
		return nil, fmt.Errorf("%w: %q", domain.ErrTaskBlocked, task.BlockReason)
	}

	// Check if session is already running
	sessionName := domain.SessionName(task.ID)
	if runningErr := shared.EnsureNoRunningSession(uc.sessions, task.ID); runningErr != nil {
		return nil, runningErr
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

	// Create or resolve worktree
	branch := domain.BranchName(task.ID, task.Issue)
	baseBranch, err := resolveBaseBranch(task, uc.git)
	if err != nil {
		return nil, err
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
	scriptPath, err := uc.generateScript(task, wtPath, agent, model, in.Continue, in.AdditionalPrompts, cfg)
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
	if in.SkipReview != nil {
		task.SkipReview = in.SkipReview
	}
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
func (uc *StartTask) generateScript(task *domain.Task, worktreePath string, agent domain.Agent, model string, continueFlag bool, additionalPrompts []string, cfg *domain.Config) (string, error) {
	scriptsDir := filepath.Join(uc.crewDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0750); err != nil {
		return "", fmt.Errorf("create scripts directory: %w", err)
	}

	// Build command and prompt using RenderCommand
	script, err := uc.buildScript(task, worktreePath, agent, model, continueFlag, additionalPrompts, cfg)
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
	EnvExports   string
	LogPath      string
	TaskID       int
}

// buildScript constructs the task script with embedded prompt and session-ended callback.
func (uc *StartTask) buildScript(task *domain.Task, worktreePath string, agent domain.Agent, model string, continueFlag bool, additionalPrompts []string, cfg *domain.Config) (string, error) {
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
	// Priority: Agent.Prompt > AgentsConfig.WorkerPrompt > empty
	defaultSystemPrompt := domain.DefaultSystemPrompt
	defaultPrompt := cfg.AgentsConfig.WorkerPrompt

	switch agent.Role {
	case domain.RoleManager:
		defaultSystemPrompt = domain.DefaultManagerSystemPrompt
		defaultPrompt = cfg.AgentsConfig.ManagerPrompt
	case domain.RoleReviewer:
		defaultSystemPrompt = domain.DefaultReviewerSystemPrompt
		defaultPrompt = cfg.AgentsConfig.ReviewerPrompt
	case domain.RoleWorker:
		// Already set as defaults
	}

	// Render command and prompt using Agent.RenderCommand
	// Pass shell variable reference as promptOverride - will be expanded at runtime
	result, err := agent.RenderCommand(cmdData, `"$PROMPT"`, defaultSystemPrompt, defaultPrompt)
	if err != nil {
		return "", fmt.Errorf("render agent command: %w", err)
	}

	// Append additional prompts if provided
	finalPrompt := result.Prompt
	for _, p := range additionalPrompts {
		if p != "" {
			if finalPrompt != "" {
				finalPrompt = finalPrompt + "\n\n" + p
			} else {
				finalPrompt = p
			}
		}
	}

	tmpl := template.Must(template.New("script").Parse(scriptTemplate))

	envExports, err := buildEnvExports(agent.Env)
	if err != nil {
		return "", err
	}

	sessionName := domain.SessionName(task.ID)
	logPath := domain.SessionLogPath(uc.crewDir, sessionName)
	data := scriptTemplateData{
		AgentCommand: result.Command,
		Prompt:       finalPrompt,
		CrewBin:      crewBin,
		TaskID:       task.ID,
		EnvExports:   envExports,
		LogPath:      logPath,
	}

	// Write session log header before script execution
	if err := writeSessionLogHeader(logPath, sessionName, worktreePath, result.Command); err != nil {
		return "", fmt.Errorf("write session log header: %w", err)
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

func buildEnvExports(env map[string]string) (string, error) {
	if len(env) == 0 {
		return "", nil
	}

	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		if !envNamePattern.MatchString(key) {
			return "", fmt.Errorf("%w: %q", domain.ErrInvalidEnvVarName, key)
		}
		builder.WriteString("export ")
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(shellQuote(env[key]))
		builder.WriteString("\n")
	}

	return strings.TrimRight(builder.String(), "\n"), nil
}

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

// writeSessionLogHeader writes the session metadata header to the log file.
// This is done in Go to avoid shell escaping issues with command strings.
func writeSessionLogHeader(logPath, sessionName, workDir, command string) error {
	// Ensure logs directory exists
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

	// Write header (truncate existing file)
	if err := os.WriteFile(logPath, []byte(header), 0600); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	return nil
}

// scriptTemplate is the template for the task script.
// The prompt is embedded using a heredoc to avoid escaping issues.
const scriptTemplate = `#!/bin/bash
set -o pipefail

# Redirect stderr to session log
exec 2>>"{{.LogPath}}"

# Embedded prompt
read -r -d '' PROMPT << 'END_OF_PROMPT'
{{.Prompt}}
END_OF_PROMPT

# Agent environment variables
{{.EnvExports}}

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

	// Execute the script using ScriptRunner
	if err := uc.runner.Run(wtPath, script.String()); err != nil {
		return fmt.Errorf("run setup script: %w", err)
	}

	return nil
}
