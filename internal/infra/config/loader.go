// Package config provides configuration loading functionality.
package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Ensure Loader implements domain.ConfigLoader.
var _ domain.ConfigLoader = (*Loader)(nil)

// Loader loads configuration from TOML files.
type Loader struct {
	crewDir       string // Path to .git/crew directory
	globalConfDir string // Path to global config directory (e.g., ~/.config/git-crew)
}

// NewLoader creates a new Loader.
func NewLoader(crewDir string) *Loader {
	return &Loader{
		crewDir:       crewDir,
		globalConfDir: defaultGlobalConfigDir(),
	}
}

// NewLoaderWithGlobalDir creates a new Loader with a custom global config directory.
// This is useful for testing.
func NewLoaderWithGlobalDir(crewDir, globalConfDir string) *Loader {
	return &Loader{
		crewDir:       crewDir,
		globalConfDir: globalConfDir,
	}
}

// defaultGlobalConfigDir returns the default global config directory.
func defaultGlobalConfigDir() string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, domain.GlobalConfigDirName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", domain.GlobalConfigDirName)
}

// configFile represents the structure of config.toml.
// Using separate struct to handle TOML tag mapping.
type configFile struct {
	DefaultAgent string              `toml:"default_agent"`
	Agent        agentSection        `toml:"agent"`
	Agents       map[string]agentDef `toml:"agents"`
	Complete     completeSection     `toml:"complete"`
	Log          logSection          `toml:"log"`
}

type agentSection struct {
	Prompt string `toml:"prompt"`
}

type agentDef struct {
	Args    string `toml:"args"`
	Command string `toml:"command"`
}

type completeSection struct {
	Command string `toml:"command"`
}

type logSection struct {
	Level string `toml:"level"`
}

// Load returns the merged configuration (repo + global).
// Repository config takes precedence over global config.
func (l *Loader) Load() (*domain.Config, error) {
	// Load global config first
	global, err := l.LoadGlobal()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	// Load repo config
	repoPath := filepath.Join(l.crewDir, domain.ConfigFileName)
	repo, err := l.loadFile(repoPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	// Start with default config
	base := domain.NewDefaultConfig()

	// If both don't exist, return default config
	if global == nil && repo == nil {
		return base, nil
	}

	// Merge: default <- global <- repo (later takes precedence)
	if global != nil {
		base = mergeConfigs(base, global)
	}
	if repo != nil {
		base = mergeConfigs(base, repo)
	}

	return base, nil
}

// LoadGlobal returns only the global configuration.
func (l *Loader) LoadGlobal() (*domain.Config, error) {
	if l.globalConfDir == "" {
		return nil, os.ErrNotExist
	}
	globalPath := filepath.Join(l.globalConfDir, domain.ConfigFileName)
	return l.loadFile(globalPath)
}

// loadFile loads a configuration from a file.
func (l *Loader) loadFile(path string) (*domain.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cf configFile
	if err := toml.Unmarshal(data, &cf); err != nil {
		return nil, err
	}

	return convertToDomainConfig(&cf), nil
}

// convertToDomainConfig converts the file config to domain config.
func convertToDomainConfig(cf *configFile) *domain.Config {
	agents := make(map[string]domain.Agent)
	for name, def := range cf.Agents {
		agents[name] = domain.Agent{
			Args:    def.Args,
			Command: def.Command,
		}
	}

	return &domain.Config{
		DefaultAgent: cf.DefaultAgent,
		Agent: domain.AgentConfig{
			Prompt: cf.Agent.Prompt,
		},
		Agents: agents,
		Complete: domain.CompleteConfig{
			Command: cf.Complete.Command,
		},
		Log: domain.LogConfig{
			Level: cf.Log.Level,
		},
	}
}

// mergeConfigs merges two configs, with override taking precedence.
func mergeConfigs(base, override *domain.Config) *domain.Config {
	result := &domain.Config{
		DefaultAgent: base.DefaultAgent,
		Agent:        base.Agent,
		Complete:     base.Complete,
		Log:          base.Log,
		Agents:       make(map[string]domain.Agent),
	}

	// Copy base agents
	for name, agent := range base.Agents {
		result.Agents[name] = agent
	}

	// Override with override config
	if override.DefaultAgent != "" {
		result.DefaultAgent = override.DefaultAgent
	}
	if override.Agent.Prompt != "" {
		result.Agent.Prompt = override.Agent.Prompt
	}
	if override.Complete.Command != "" {
		result.Complete.Command = override.Complete.Command
	}
	if override.Log.Level != "" {
		result.Log.Level = override.Log.Level
	}

	// Merge agents: override takes precedence
	for name, agent := range override.Agents {
		result.Agents[name] = agent
	}

	return result
}
