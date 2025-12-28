package domain

import (
	"bytes"
	_ "embed"
	"path/filepath"
	"text/template"
)

// Config represents the application configuration.
type Config struct {
	WorkersConfig WorkersConfig          // Common [workers] settings (including default)
	Workers       map[string]WorkerAgent // Per-worker settings [workers.<name>]
	Complete      CompleteConfig         // [complete] settings
	Log           LogConfig              // [log] settings
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

// CommandData holds data for rendering agent commands.
type CommandData struct {
	// Environment
	GitDir   string // Path to .git directory
	RepoRoot string // Repository root path
	Worktree string // Worktree path (if using worktrees)

	// Task information
	Title       string
	Description string
	Prompt      string // Prompt to pass to the agent

	// Task ID
	TaskID int
}

// RenderCommand renders the full command string for this agent.
// It performs two-phase template expansion:
// 1. Expand SystemArgs, Args, and Prompt with CommandData (for GitDir, RepoRoot, TaskID, etc.)
// 2. Expand CommandTemplate with Command, expanded SystemArgs/Args, and expanded Prompt
func (a *WorkerAgent) RenderCommand(data CommandData) (string, error) {
	// Phase 1: Expand SystemArgs, Args, and Prompt
	systemArgs, err := expandString(a.SystemArgs, data)
	if err != nil {
		return "", err
	}
	args, err := expandString(a.Args, data)
	if err != nil {
		return "", err
	}
	prompt, err := expandString(data.Prompt, data)
	if err != nil {
		return "", err
	}

	// Phase 2: Expand CommandTemplate
	cmdData := map[string]string{
		"Command":    a.Command,
		"SystemArgs": systemArgs,
		"Args":       args,
		"Prompt":     prompt,
	}

	tmpl, err := template.New("cmd").Parse(a.CommandTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cmdData); err != nil {
		return "", err
	}

	return buf.String(), nil
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

// LogConfig holds logging settings from [log] section.
type LogConfig struct {
	Level string // Log level: debug, info, warn, error
}

// Default configuration values.
const (
	DefaultLogLevel     = "info"
	DefaultWorkerName   = "claude"
	DefaultWorkerPrompt = "When the task is complete, commit your changes and run 'crew complete' to mark it for review."
)

// BuiltinWorker defines a built-in worker configuration.
type BuiltinWorker struct {
	CommandTemplate string // Template: {{.Command}}, {{.SystemArgs}}, {{.Args}}, {{.Prompt}}
	Command         string // Base command (e.g., "claude")
	SystemArgs      string // System arguments for crew operation
	DefaultArgs     string // Default user-customizable arguments
}

const claudeAllowedTools = `--allowedTools='Bash(git add:*) Bash(git commit:*) Bash(crew complete) Bash(crew show)'`

// BuiltinWorkers contains preset configurations for known workers.
// Note: Use --option=value format for options that take variadic arguments (like --allowedTools)
// to prevent them from consuming the prompt argument.
var BuiltinWorkers = map[string]BuiltinWorker{
	"claude": {
		CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
		Command:         "claude",
		SystemArgs:      "--permission-mode acceptEdits " + claudeAllowedTools,
		DefaultArgs:     "--model opus",
	},
	"opencode": {
		CommandTemplate: "{{.Command}} {{.Args}} -p {{.Prompt}}",
		Command:         "opencode",
		SystemArgs:      "",
		DefaultArgs:     "-m anthropic/claude-opus-4-5",
	},
	"codex": {
		CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
		Command:         "codex",
		SystemArgs:      "--full-auto",
		DefaultArgs:     "--model gpt-5.2-codex",
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
		"WorkersPrompt": cfg.WorkersConfig.Prompt,
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
