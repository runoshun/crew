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
	Agent  string // Agent name (required in MVP)
	Model  string // Model name override (optional, uses agent's default if empty)
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
	repoRoot     string // Repository root path
}

// NewStartTask creates a new StartTask use case.
func NewStartTask(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	worktrees domain.WorktreeManager,
	configLoader domain.ConfigLoader,
	clock domain.Clock,
	crewDir string,
	repoRoot string,
) *StartTask {
	return &StartTask{
		tasks:        tasks,
		sessions:     sessions,
		worktrees:    worktrees,
		configLoader: configLoader,
		clock:        clock,
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

	// Load config for agent resolution and command building
	cfg, loadErr := uc.configLoader.Load()
	if loadErr != nil {
		return nil, fmt.Errorf("load config: %w", loadErr)
	}

	// Resolve agent from input or default
	agent := in.Agent
	if agent == "" {
		agent = domain.DefaultWorkerName
	}

	// Get agent configuration
	resolved := uc.resolveWorkerAgent(agent, cfg)

	// Resolve model priority: CLI flag > worker config > builtin default
	model := in.Model
	if model == "" && resolved.Worker.Model != "" {
		model = resolved.Worker.Model
	}
	if model == "" {
		model = resolved.DefaultModel
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

	// Setup worktree (copy files and run setup command)
	if setupErr := uc.worktrees.SetupWorktree(wtPath, &cfg.Worktree); setupErr != nil {
		_ = uc.worktrees.Remove(branch)
		return nil, fmt.Errorf("setup worktree: %w", setupErr)
	}

	// Setup agent-specific configurations (SetupScript and ExcludePatterns)
	if setupErr := uc.setupAgent(task, wtPath, resolved); setupErr != nil {
		_ = uc.worktrees.Remove(branch)
		return nil, fmt.Errorf("setup agent: %w", setupErr)
	}

	// Generate prompt and script files
	scriptPath, err := uc.generateScript(task, wtPath, resolved.Worker, cfg, model)
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
func (uc *StartTask) generateScript(task *domain.Task, worktreePath string, worker domain.Worker, cfg *domain.Config, model string) (string, error) {
	scriptsDir := filepath.Join(uc.crewDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0750); err != nil {
		return "", fmt.Errorf("create scripts directory: %w", err)
	}

	// Build command and prompt using RenderCommand
	script, err := uc.buildScript(task, worktreePath, worker, cfg, model)
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

// resolvedWorkerResult holds the result of resolveWorkerAgent.
type resolvedWorkerResult struct {
	Worker              domain.Worker
	DefaultModel        string
	WorktreeSetupScript string   // Agent's setup script (template to expand)
	ExcludePatterns     []string // Agent's exclude patterns to add to .git/info/exclude
}

// resolveWorkerAgent resolves the Worker configuration for the given agent name.
// Returns the Worker, default model, and agent-specific settings (SetupScript, ExcludePatterns).
func (uc *StartTask) resolveWorkerAgent(agent string, cfg *domain.Config) resolvedWorkerResult {
	// Check if agent is configured in config
	if worker, ok := cfg.Workers[agent]; ok {
		result := resolvedWorkerResult{
			Worker:       worker,
			DefaultModel: "",
		}

		// Check for agent reference in worker - inherit fields from Agent
		agentRef := worker.Agent
		if agentRef == "" {
			agentRef = agent // Use worker name as agent reference if not specified
		}

		// Try to inherit from config Agent first
		if agentDef, ok := cfg.Agents[agentRef]; ok {
			result.WorktreeSetupScript = agentDef.WorktreeSetupScript
			result.ExcludePatterns = agentDef.ExcludePatterns
			result.DefaultModel = agentDef.DefaultModel
			// Inherit command fields if not set in worker
			// Note: SystemArgs is NOT inherited from Agent; it's Worker-specific
			if result.Worker.CommandTemplate == "" {
				result.Worker.CommandTemplate = agentDef.CommandTemplate
			}
			if result.Worker.Command == "" {
				result.Worker.Command = agentDef.Command
			}
		}

		// Check for built-in agent defaults
		if builtin, ok := domain.BuiltinAgents[agentRef]; ok {
			if result.DefaultModel == "" {
				result.DefaultModel = builtin.DefaultModel
			}
			// Use built-in setup script/patterns if not set from config Agent
			if result.WorktreeSetupScript == "" {
				result.WorktreeSetupScript = builtin.WorktreeSetupScript
			}
			if len(result.ExcludePatterns) == 0 {
				result.ExcludePatterns = builtin.ExcludePatterns
			}
			// Inherit command fields from builtin if not set
			// Use WorkerSystemArgs for Workers
			if result.Worker.CommandTemplate == "" {
				result.Worker.CommandTemplate = builtin.CommandTemplate
			}
			if result.Worker.Command == "" {
				result.Worker.Command = builtin.Command
			}
			if result.Worker.SystemArgs == "" {
				result.Worker.SystemArgs = builtin.WorkerSystemArgs
			}
		}

		return result
	}

	// Check if it's a built-in agent
	if builtin, ok := domain.BuiltinAgents[agent]; ok {
		return resolvedWorkerResult{
			Worker: domain.Worker{
				Agent:           agent,
				CommandTemplate: builtin.CommandTemplate,
				Command:         builtin.Command,
				SystemArgs:      builtin.WorkerSystemArgs,
				Args:            builtin.DefaultArgs,
			},
			DefaultModel:        builtin.DefaultModel,
			WorktreeSetupScript: builtin.WorktreeSetupScript,
			ExcludePatterns:     builtin.ExcludePatterns,
		}
	}

	// Unknown agent - use the agent name as-is (custom agent)
	return resolvedWorkerResult{
		Worker: domain.Worker{
			CommandTemplate: "{{.Command}} {{.Prompt}}",
			Command:         agent,
		},
	}
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
func (uc *StartTask) buildScript(task *domain.Task, worktreePath string, worker domain.Worker, cfg *domain.Config, model string) (string, error) {
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
	}

	// Determine default prompts: use config's WorkersConfig settings
	defaultSystemPrompt := cfg.WorkersConfig.SystemPrompt
	if defaultSystemPrompt == "" {
		defaultSystemPrompt = domain.DefaultSystemPrompt
	}
	defaultPrompt := cfg.WorkersConfig.Prompt

	// Render command and prompt using Worker.RenderCommand
	// Pass shell variable reference as promptOverride - will be expanded at runtime
	result, err := worker.RenderCommand(cmdData, `"$PROMPT"`, defaultSystemPrompt, defaultPrompt)
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

// setupAgent runs agent-specific setup: SetupScript execution and ExcludePatterns application.
func (uc *StartTask) setupAgent(task *domain.Task, wtPath string, resolved resolvedWorkerResult) error {
	gitDir := filepath.Join(uc.repoRoot, ".git")

	// Apply exclude patterns
	if len(resolved.ExcludePatterns) > 0 {
		if err := uc.applyExcludePatterns(gitDir, resolved.ExcludePatterns); err != nil {
			return fmt.Errorf("apply exclude patterns: %w", err)
		}
	}

	// Run setup script
	if resolved.WorktreeSetupScript != "" {
		if err := uc.runSetupScript(task, wtPath, resolved.WorktreeSetupScript); err != nil {
			return fmt.Errorf("run setup script: %w", err)
		}
	}

	return nil
}

// applyExcludePatterns adds patterns to .git/info/exclude if not already present.
func (uc *StartTask) applyExcludePatterns(gitDir string, patterns []string) error {
	excludePath := filepath.Join(gitDir, "info", "exclude")

	// Ensure info directory exists
	infoDir := filepath.Dir(excludePath)
	if err := os.MkdirAll(infoDir, 0750); err != nil {
		return fmt.Errorf("create info directory: %w", err)
	}

	// Read existing patterns
	existingPatterns := make(map[string]bool)
	if content, err := os.ReadFile(excludePath); err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				existingPatterns[line] = true
			}
		}
	}

	// Collect patterns to add
	var toAdd []string
	for _, pattern := range patterns {
		if !existingPatterns[pattern] {
			toAdd = append(toAdd, pattern)
		}
	}

	// Append new patterns
	if len(toAdd) > 0 {
		// Use 0644 for exclude file as it's not sensitive and needs to be readable
		f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //nolint:gosec // exclude file is not sensitive
		if err != nil {
			return fmt.Errorf("open exclude file: %w", err)
		}

		var writeErr error
		for _, pattern := range toAdd {
			if _, err := f.WriteString(pattern + "\n"); err != nil {
				writeErr = fmt.Errorf("write pattern: %w", err)
				break
			}
		}

		if err := f.Close(); err != nil && writeErr == nil {
			return fmt.Errorf("close exclude file: %w", err)
		}
		if writeErr != nil {
			return writeErr
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
