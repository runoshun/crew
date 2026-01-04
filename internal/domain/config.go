package domain

import (
	"bytes"
	_ "embed"
	"path/filepath"
	"strings"
	"text/template"
)

// Config represents the application configuration.
type Config struct {
	Agents        map[string]Agent   // Agent definitions from [agents.<name>]
	WorkersConfig WorkersConfig      // Common [workers] settings (including default)
	Workers       map[string]Worker  // Per-worker settings [workers.<name>]
	Managers      map[string]Manager // Manager definitions from [managers.<name>]
	Complete      CompleteConfig     // [complete] settings
	Diff          DiffConfig         // [diff] settings
	Log           LogConfig          // [log] settings
	Tasks         TasksConfig        // [tasks] settings
	Worktree      WorktreeConfig     // [worktree] settings
	Warnings      []string           // [warning] Unknown keys or other issues
}

// TasksConfig holds settings for task storage from [tasks] section.
type TasksConfig struct {
	Store     string // Storage backend: "git" (default) or "json"
	Namespace string // Git namespace for refs (default: "crew")
	Encrypt   bool   // Enable encryption for task data (default: false)
}

// WorkersConfig holds common settings for all workers from [workers] section.
type WorkersConfig struct {
	Default      string // Default worker name (e.g., "claude")
	SystemPrompt string // Default system prompt for all workers
	Prompt       string // Default prompt for all workers (can be overridden per worker)
}

// Agent defines a base agent configuration that Workers and Managers can reference.
// Agents define the core command execution pattern without being tied to a specific role.
type Agent struct {
	Command             string   // Base command (e.g., "claude", "opencode")
	CommandTemplate     string   // Template for assembling the command (e.g., "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}")
	SystemArgs          string   // System arguments required for crew operation (auto-applied)
	DefaultModel        string   // Default model for this agent
	Description         string   // Description of the agent's purpose
	WorktreeSetupScript string   // Script to run after worktree creation (template-expanded)
	ExcludePatterns     []string // Patterns to add to .git/info/exclude for this agent
}

// Worker holds per-worker configuration from [workers.<name>] sections.
// Workers are task execution agents that can reference an Agent for base settings.
type Worker struct {
	Agent           string // Name of the Agent to inherit from (optional)
	Inherit         string // Name of worker to inherit from (optional, for worker-to-worker inheritance)
	CommandTemplate string // Template for assembling the command (e.g., "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}")
	Command         string // Base command (e.g., "claude", "opencode")
	SystemArgs      string // System arguments required for crew operation (auto-applied)
	Args            string // User-customizable arguments (e.g., model selection)
	Model           string // Default model for this worker (overrides builtin default)
	SystemPrompt    string // System prompt template for this worker
	Prompt          string // Prompt template for this worker
	Description     string // Description of the worker's purpose
}

// Manager holds configuration for manager agents from [managers.<name>] sections.
// Managers are read-only orchestration agents that can create and monitor tasks.
type Manager struct {
	Agent       string // Name of the Agent to inherit from (optional)
	Model       string // Model override for this manager
	Args        string // Additional arguments for this manager
	Description string // Description of the manager's purpose
}

// CommandData holds data for rendering agent commands and prompts.
// Fields are ordered to minimize memory padding.
type CommandData struct {
	// Environment
	GitDir   string // Path to .git directory
	RepoRoot string // Repository root path
	Worktree string // Worktree path (if using worktrees)

	// Task information
	Title       string
	Description string
	Branch      string // Branch name (e.g., "crew-1")

	// Runtime options
	Model string // Model name override (e.g., "sonnet", "gpt-4o")

	// Integer fields grouped together for alignment
	Issue  int // GitHub issue number (0 if not linked)
	TaskID int
}

// RenderCommandResult holds the results of RenderCommand.
type RenderCommandResult struct {
	Command string // The full command to execute (e.g., `claude --model opus "$PROMPT"`)
	Prompt  string // The prompt content to be stored in PROMPT shell variable
}

// RenderCommand renders the full command string and prompt content for this worker.
// It performs three-phase template expansion:
// 1. Expand SystemArgs and Args with CommandData (for GitDir, RepoRoot, TaskID, etc.)
// 2. Expand SystemPrompt and Prompt templates with CommandData to generate prompt content
// 3. Expand CommandTemplate with Command, expanded SystemArgs/Args, and shell variable reference
//
// The promptOverride parameter is the shell variable reference (e.g., `"$PROMPT"`) that will be
// embedded in the command and expanded at runtime to the actual prompt content.
// The defaultSystemPrompt is used when Worker.SystemPrompt is empty.
// The defaultPrompt is used when Worker.Prompt is empty.
func (w *Worker) RenderCommand(data CommandData, promptOverride, defaultSystemPrompt, defaultPrompt string) (RenderCommandResult, error) {
	// Phase 1: Expand SystemArgs and Args
	systemArgs, err := expandString(w.SystemArgs, data)
	if err != nil {
		return RenderCommandResult{}, err
	}
	args, err := expandString(w.Args, data)
	if err != nil {
		return RenderCommandResult{}, err
	}

	// Phase 2: Expand Prompt templates to generate prompt content
	sysPromptTemplate := w.SystemPrompt
	if sysPromptTemplate == "" {
		sysPromptTemplate = defaultSystemPrompt
	}
	sysPromptContent, err := expandString(sysPromptTemplate, data)
	if err != nil {
		return RenderCommandResult{}, err
	}

	userPromptTemplate := w.Prompt
	if userPromptTemplate == "" {
		userPromptTemplate = defaultPrompt
	}
	userPromptContent, err := expandString(userPromptTemplate, data)
	if err != nil {
		return RenderCommandResult{}, err
	}

	// Combine SystemPrompt and Prompt
	var promptContent string
	if sysPromptContent != "" && userPromptContent != "" {
		promptContent = sysPromptContent + "\n\n" + userPromptContent
	} else if sysPromptContent != "" {
		promptContent = sysPromptContent
	} else {
		promptContent = userPromptContent
	}

	// Phase 3: Expand CommandTemplate
	cmdData := map[string]string{
		"Command":    w.Command,
		"SystemArgs": systemArgs,
		"Args":       args,
		"Prompt":     promptOverride,
	}

	tmpl, err := template.New("cmd").Parse(w.CommandTemplate)
	if err != nil {
		return RenderCommandResult{}, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cmdData); err != nil {
		return RenderCommandResult{}, err
	}

	return RenderCommandResult{
		Command: buf.String(),
		Prompt:  promptContent,
	}, nil
}

// expandString expands template variables in a string.
func expandString(s string, data CommandData) (string, error) {
	if s == "" {
		return "", nil
	}

	tmpl, err := template.New("expand").Parse(s)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// CompleteConfig holds completion gate settings from [complete] section.
type CompleteConfig struct {
	Command string // Command to run as CI gate on complete
}

// DiffConfig holds diff display settings from [diff] section.
type DiffConfig struct {
	Command    string // Command to display diff (with {{.Args}} template support)
	TUICommand string // Command for TUI diff display
}

// LogConfig holds logging settings from [log] section.
type LogConfig struct {
	Level string // Log level: debug, info, warn, error
}

// WorktreeConfig holds worktree customization settings from [worktree] section.
type WorktreeConfig struct {
	SetupCommand string   // Command to run after worktree creation
	Copy         []string // Files/directories to copy (with CoW if available)
}

// Default configuration values.
const (
	DefaultLogLevel   = "info"
	DefaultWorkerName = "claude"
)

// DefaultSystemPrompt is the default system prompt template for workers.
// It uses Go template syntax with CommandData fields.
const DefaultSystemPrompt = `You are working on Task #{{.TaskID}}.

IMPORTANT: First run 'crew --help-worker' and follow the workflow instructions exactly.`

// BuiltinAgent defines a built-in agent configuration.
type BuiltinAgent struct {
	CommandTemplate     string   // Template: {{.Command}}, {{.SystemArgs}}, {{.Args}}, {{.Prompt}}
	Command             string   // Base command (e.g., "claude")
	SystemArgs          string   // System arguments (model, permissions) - NOT overridable by user config
	DefaultArgs         string   // Default user-customizable arguments (overridable in config.toml)
	DefaultModel        string   // Default model name for this agent
	Description         string   // Description of the agent's purpose
	WorktreeSetupScript string   // Script to run after worktree creation (template-expanded)
	ExcludePatterns     []string // Patterns to add to .git/info/exclude for this agent
}

// BuiltinAgents contains preset configurations for known agents.
var BuiltinAgents = map[string]BuiltinAgent{
	"claude":   claudeAgentConfig,
	"opencode": opencodeAgentConfig,
}

// Directory and file names for git-crew.
const (
	CrewDirName    = "crew"        // Directory name for crew data
	ConfigFileName = "config.toml" // Config file name
)

// RepoCrewDir returns the crew directory path for a repository.
func RepoCrewDir(repoRoot string) string {
	return filepath.Join(repoRoot, ".git", CrewDirName)
}

// RepoConfigPath returns the repo config path.
func RepoConfigPath(repoRoot string) string {
	return filepath.Join(RepoCrewDir(repoRoot), ConfigFileName)
}

// GlobalCrewDir returns the global crew directory path.
// configHome is typically XDG_CONFIG_HOME or ~/.config (resolved by caller).
func GlobalCrewDir(configHome string) string {
	return filepath.Join(configHome, CrewDirName)
}

// GlobalConfigPath returns the global config path.
// configHome is typically XDG_CONFIG_HOME or ~/.config (resolved by caller).
func GlobalConfigPath(configHome string) string {
	return filepath.Join(GlobalCrewDir(configHome), ConfigFileName)
}

// NewDefaultConfig returns a Config with default values.
func NewDefaultConfig() *Config {
	agents := make(map[string]Agent)
	workers := make(map[string]Worker)
	for name, builtin := range BuiltinAgents {
		agents[name] = Agent{
			Command:             builtin.Command,
			CommandTemplate:     builtin.CommandTemplate,
			SystemArgs:          builtin.SystemArgs,
			DefaultModel:        builtin.DefaultModel,
			Description:         builtin.Description,
			WorktreeSetupScript: builtin.WorktreeSetupScript,
			ExcludePatterns:     builtin.ExcludePatterns,
		}
		workers[name] = Worker{
			Agent:           name,
			CommandTemplate: builtin.CommandTemplate,
			Command:         builtin.Command,
			SystemArgs:      builtin.SystemArgs,
			Args:            builtin.DefaultArgs,
			Description:     builtin.Description,
			// SystemPrompt/Prompt are empty; falls back to WorkersConfig
		}
	}
	return &Config{
		Agents: agents,
		WorkersConfig: WorkersConfig{
			Default:      DefaultWorkerName,
			SystemPrompt: DefaultSystemPrompt,
			Prompt:       "",
		},
		Workers: workers,
		Managers: map[string]Manager{
			"default": {
				Agent:       "opencode",
				Description: "Default manager agent",
			},
		},
		Log: LogConfig{
			Level: DefaultLogLevel,
		},
	}
}

// configTemplate is the raw configuration file template with << >> delimiters.
//
//go:embed config_template.toml
var configTemplate string

// RenderConfigTemplate renders the config template with default values.
func RenderConfigTemplate() (string, error) {
	cfg := NewDefaultConfig()

	data := map[string]any{
		"DefaultWorker":       cfg.WorkersConfig.Default,
		"WorkersSystemPrompt": formatPromptForTemplate(cfg.WorkersConfig.SystemPrompt),
		"WorkersPrompt":       formatPromptForTemplate(cfg.WorkersConfig.Prompt),
		"LogLevel":            cfg.Log.Level,
		"ClaudeArgs":          cfg.Workers["claude"].Args,
		"OpencodeArgs":        cfg.Workers["opencode"].Args,
	}

	tmpl, err := template.New("config").Delims("<<", ">>").Parse(configTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func formatPromptForTemplate(prompt string) string {
	if prompt == "" {
		return ""
	}

	trimmed := strings.TrimRight(prompt, "\n")
	lines := strings.Split(trimmed, "\n")

	var buf strings.Builder
	buf.WriteString(`"""`)
	for _, line := range lines {
		buf.WriteString("\n# ")
		buf.WriteString(line)
	}
	buf.WriteString("\n# \"\"\"")

	return buf.String()
}

// ResolveInheritance resolves worker inheritance by applying parent worker settings
// to child workers. It detects circular dependencies and returns an error if found.
// Workers without Inherit field are left unchanged.
// Only non-empty fields from the child override the parent fields.
func (c *Config) ResolveInheritance() error {
	// Track visited workers during traversal to detect circular dependencies
	visited := make(map[string]bool)
	resolving := make(map[string]bool)

	// Resolve each worker
	for name := range c.Workers {
		if err := c.resolveWorker(name, visited, resolving); err != nil {
			return err
		}
	}

	return nil
}

// resolveWorker recursively resolves inheritance for a single worker.
func (c *Config) resolveWorker(name string, visited, resolving map[string]bool) error {
	// Already resolved
	if visited[name] {
		return nil
	}

	// Circular dependency detected
	if resolving[name] {
		return ErrCircularInheritance
	}

	worker := c.Workers[name]

	// No inheritance, mark as resolved
	if worker.Inherit == "" {
		visited[name] = true
		return nil
	}

	// Mark as currently resolving
	resolving[name] = true

	// Check if parent exists
	_, exists := c.Workers[worker.Inherit]
	if !exists {
		delete(resolving, name)
		return ErrInheritParentNotFound
	}

	// Resolve parent first (recursive)
	if err := c.resolveWorker(worker.Inherit, visited, resolving); err != nil {
		delete(resolving, name)
		return err
	}

	// Get the resolved parent
	parent := c.Workers[worker.Inherit]

	// Apply inheritance: parent fields are used as defaults, child overrides if non-empty
	resolved := parent

	// Child overrides
	if worker.Agent != "" {
		resolved.Agent = worker.Agent
	}
	if worker.CommandTemplate != "" {
		resolved.CommandTemplate = worker.CommandTemplate
	}
	if worker.Command != "" {
		resolved.Command = worker.Command
	}
	if worker.SystemArgs != "" {
		resolved.SystemArgs = worker.SystemArgs
	}
	if worker.Args != "" {
		resolved.Args = worker.Args
	}
	if worker.SystemPrompt != "" {
		resolved.SystemPrompt = worker.SystemPrompt
	}
	if worker.Prompt != "" {
		resolved.Prompt = worker.Prompt
	}
	if worker.Model != "" {
		resolved.Model = worker.Model
	}
	if worker.Description != "" {
		resolved.Description = worker.Description
	}

	// Clear Inherit field after resolution
	resolved.Inherit = ""

	// Save resolved worker
	c.Workers[name] = resolved

	// Mark as resolved
	delete(resolving, name)
	visited[name] = true

	return nil
}
