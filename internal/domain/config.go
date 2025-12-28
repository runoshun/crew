package domain

import (
	_ "embed"
	"path/filepath"
)

// Config represents the application configuration.
type Config struct {
	DefaultAgent string           // Top-level default_agent
	Agent        AgentConfig      // Common [agent] settings
	Agents       map[string]Agent // Per-agent settings [agents.<name>]
	Complete     CompleteConfig   // [complete] settings
	Log          LogConfig        // [log] settings
}

// AgentConfig holds common agent settings from [agent] section.
type AgentConfig struct {
	Prompt string // Common prompt appended to all agents
}

// Agent holds per-agent configuration from [agents.<name>] sections.
type Agent struct {
	Args    string // Additional arguments to pass to the agent
	Command string // Custom command (overrides built-in agent command)
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
	DefaultLogLevel = "info"
)

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
	return &Config{
		Agents: make(map[string]Agent),
		Log: LogConfig{
			Level: DefaultLogLevel,
		},
	}
}

// DefaultConfigTemplate is the default configuration file template.
//
//go:embed config_template.toml
var DefaultConfigTemplate string
