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
	WorkersConfig WorkersConfig          // Common [workers] settings (including default)
	Workers       map[string]WorkerAgent // Per-worker settings [workers.<name>]
	Complete      CompleteConfig         // [complete] settings
	Diff          DiffConfig             // [diff] settings
	Log           LogConfig              // [log] settings
	Tasks         TasksConfig            // [tasks] settings
}

// TasksConfig holds settings for task storage from [tasks] section.
type TasksConfig struct {
	Store     string // Storage backend: "json" (default) or "git"
	Namespace string // Git namespace for refs (default: "crew")
}

// WorkersConfig holds common settings for all workers from [workers] section.
type WorkersConfig struct {
	Default string // Default worker name (e.g., "claude")
	Prompt  string // Default prompt for all workers (can be overridden per worker)
}

// WorkerAgent holds per-worker configuration from [workers.<name>] sections.
type WorkerAgent struct {
	CommandTemplate string // Template for assembling the command (e.g., "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}")
	Command         string // Base command (e.g., "claude", "opencode")
	SystemArgs      string // System arguments required for crew operation (auto-applied)
	Args            string // User-customizable arguments (e.g., model selection)
	Prompt          string // Prompt template for this worker
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

// RenderCommand renders the full command string and prompt content for this agent.
// It performs three-phase template expansion:
// 1. Expand SystemArgs and Args with CommandData (for GitDir, RepoRoot, TaskID, etc.)
// 2. Expand Prompt template with CommandData to generate prompt content
// 3. Expand CommandTemplate with Command, expanded SystemArgs/Args, and shell variable reference
//
// The promptOverride parameter is the shell variable reference (e.g., `"$PROMPT"`) that will be
// embedded in the command and expanded at runtime to the actual prompt content.
// The defaultPrompt is used when WorkerAgent.Prompt is empty.
func (a *WorkerAgent) RenderCommand(data CommandData, promptOverride, defaultPrompt string) (RenderCommandResult, error) {
	// Phase 1: Expand SystemArgs and Args
	systemArgs, err := expandString(a.SystemArgs, data)
	if err != nil {
		return RenderCommandResult{}, err
	}
	args, err := expandString(a.Args, data)
	if err != nil {
		return RenderCommandResult{}, err
	}

	// Phase 2: Expand Prompt template to generate prompt content
	promptTemplate := a.Prompt
	if promptTemplate == "" {
		promptTemplate = defaultPrompt
	}
	promptContent, err := expandString(promptTemplate, data)
	if err != nil {
		return RenderCommandResult{}, err
	}

	// Phase 3: Expand CommandTemplate
	cmdData := map[string]string{
		"Command":    a.Command,
		"SystemArgs": systemArgs,
		"Args":       args,
		"Prompt":     promptOverride,
	}

	tmpl, err := template.New("cmd").Parse(a.CommandTemplate)
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

// Default configuration values.
const (
	DefaultLogLevel   = "info"
	DefaultWorkerName = "claude"
)

// DefaultWorkerPrompt is the default prompt template for workers.
// It uses Go template syntax with CommandData fields.
// Task details should be viewed with 'crew show <id>' to see comments and full description.
const DefaultWorkerPrompt = `You are working on Task #{{.TaskID}}.

Run 'crew show' to see full task details including description and comments.

When the task is complete, commit your changes and run 'crew complete'.
`

// BuiltinWorker defines a built-in worker configuration.
type BuiltinWorker struct {
	CommandTemplate string // Template: {{.Command}}, {{.SystemArgs}}, {{.Args}}, {{.Prompt}}
	Command         string // Base command (e.g., "claude")
	SystemArgs      string // System arguments (model, permissions) - NOT overridable by user config
	DefaultArgs     string // Default user-customizable arguments (overridable in config.toml)
	DefaultModel    string // Default model name for this worker
}

const claudeAllowedTools = `--allowedTools='Bash(git add:*) Bash(git commit:*) Bash(crew complete) Bash(crew show)'`

// BuiltinWorkers contains preset configurations for known workers.
// Note: Use --option=value format for options that take variadic arguments (like --allowedTools)
// to prevent them from consuming the prompt argument.
// The {{.Model}} template variable in SystemArgs is replaced with the runtime model selection.
// SystemArgs cannot be overridden by user config; Args can be customized in config.toml.
var BuiltinWorkers = map[string]BuiltinWorker{
	"claude": {
		CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
		Command:         "claude",
		SystemArgs:      "--model {{.Model}} --permission-mode acceptEdits " + claudeAllowedTools,
		DefaultArgs:     "",
		DefaultModel:    "opus",
	},
	"opencode": {
		CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} --prompt {{.Prompt}}",
		Command:         "opencode",
		SystemArgs:      "-m {{.Model}}",
		DefaultArgs:     "",
		DefaultModel:    "anthropic/claude-opus-4-5",
	},
	"codex": {
		CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
		Command:         "codex",
		SystemArgs:      "--model {{.Model}} --full-auto",
		DefaultArgs:     "",
		DefaultModel:    "gpt-5.2-codex",
	},
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
	workers := make(map[string]WorkerAgent)
	for name, builtin := range BuiltinWorkers {
		workers[name] = WorkerAgent{
			CommandTemplate: builtin.CommandTemplate,
			Command:         builtin.Command,
			SystemArgs:      builtin.SystemArgs,
			Args:            builtin.DefaultArgs,
			// Prompt is empty; falls back to WorkersConfig.Prompt
		}
	}
	return &Config{
		WorkersConfig: WorkersConfig{
			Default: DefaultWorkerName,
			Prompt:  DefaultWorkerPrompt,
		},
		Workers: workers,
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
		"DefaultWorker": cfg.WorkersConfig.Default,
		"WorkersPrompt": formatPromptForTemplate(cfg.WorkersConfig.Prompt),
		"LogLevel":      cfg.Log.Level,
		"ClaudeArgs":    cfg.Workers["claude"].Args,
		"OpencodeArgs":  cfg.Workers["opencode"].Args,
		"CodexArgs":     cfg.Workers["codex"].Args,
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
