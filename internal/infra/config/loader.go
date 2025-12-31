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
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configHome = filepath.Join(home, ".config")
	}
	return domain.GlobalCrewDir(configHome)
}

// configFile represents the structure of config.toml.
// Using separate struct to handle TOML tag mapping.
// Note: Workers is map[string]any because it contains both top-level fields (default, prompt)
// and subtables (per-worker definitions). We parse it manually after unmarshaling.
type configFile struct {
	Workers  map[string]any  `toml:"workers"`
	Complete completeSection `toml:"complete"`
	Diff     diffSection     `toml:"diff"`
	Log      logSection      `toml:"log"`
}

// workersConfig holds the parsed [workers] section.
type workersConfig struct {
	Defs    map[string]workerDef // Per-worker definitions from [workers.<name>]
	Default string               // Default worker name from [workers].default
	Prompt  string               // Common prompt from [workers].prompt
}

// parseWorkersSection parses the raw workers map into structured workersConfig.
func parseWorkersSection(raw map[string]any) workersConfig {
	result := workersConfig{
		Defs: make(map[string]workerDef),
	}

	for key, value := range raw {
		switch key {
		case "default":
			if s, ok := value.(string); ok {
				result.Default = s
			}
		case "prompt":
			if s, ok := value.(string); ok {
				result.Prompt = s
			}
		default:
			// Treat as worker definition
			if subMap, ok := value.(map[string]any); ok {
				def := workerDef{}
				if v, ok := subMap["command_template"].(string); ok {
					def.CommandTemplate = v
				}
				if v, ok := subMap["command"].(string); ok {
					def.Command = v
				}
				if v, ok := subMap["system_args"].(string); ok {
					def.SystemArgs = v
				}
				if v, ok := subMap["args"].(string); ok {
					def.Args = v
				}
				if v, ok := subMap["prompt"].(string); ok {
					def.Prompt = v
				}
				result.Defs[key] = def
			}
		}
	}

	return result
}

type workerDef struct {
	CommandTemplate string
	Command         string
	SystemArgs      string
	Args            string
	Prompt          string
}

type completeSection struct {
	Command string `toml:"command"`
}

type diffSection struct {
	Command    string `toml:"command"`
	TUICommand string `toml:"tui_command"`
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

// LoadRepo returns only the repository configuration.
func (l *Loader) LoadRepo() (*domain.Config, error) {
	repoPath := filepath.Join(l.crewDir, domain.ConfigFileName)
	return l.loadFile(repoPath)
}

// LoadWithOptions returns the merged configuration with options to ignore sources.
func (l *Loader) LoadWithOptions(opts domain.LoadConfigOptions) (*domain.Config, error) {
	var global, repo *domain.Config
	var err error

	// Load global config unless ignored
	if !opts.IgnoreGlobal {
		global, err = l.LoadGlobal()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	// Load repo config unless ignored
	if !opts.IgnoreRepo {
		repo, err = l.LoadRepo()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	// Start with default config
	base := domain.NewDefaultConfig()

	// If both don't exist or are ignored, return default config
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
	// Parse the workers section
	wc := parseWorkersSection(cf.Workers)

	workers := make(map[string]domain.WorkerAgent)
	for name, def := range wc.Defs {
		workers[name] = domain.WorkerAgent{
			CommandTemplate: def.CommandTemplate,
			Command:         def.Command,
			SystemArgs:      def.SystemArgs,
			Args:            def.Args,
			Prompt:          def.Prompt,
		}
	}

	return &domain.Config{
		WorkersConfig: domain.WorkersConfig{
			Default: wc.Default,
			Prompt:  wc.Prompt,
		},
		Workers: workers,
		Complete: domain.CompleteConfig{
			Command: cf.Complete.Command,
		},
		Diff: domain.DiffConfig{
			Command:    cf.Diff.Command,
			TUICommand: cf.Diff.TUICommand,
		},
		Log: domain.LogConfig{
			Level: cf.Log.Level,
		},
	}
}

// mergeConfigs merges two configs, with override taking precedence.
func mergeConfigs(base, override *domain.Config) *domain.Config {
	result := &domain.Config{
		WorkersConfig: base.WorkersConfig,
		Complete:      base.Complete,
		Diff:          base.Diff,
		Log:           base.Log,
		Workers:       make(map[string]domain.WorkerAgent),
	}

	// Copy base workers
	for name, worker := range base.Workers {
		result.Workers[name] = worker
	}

	// Override with override config
	if override.WorkersConfig.Default != "" {
		result.WorkersConfig.Default = override.WorkersConfig.Default
	}
	if override.WorkersConfig.Prompt != "" {
		result.WorkersConfig.Prompt = override.WorkersConfig.Prompt
	}
	if override.Complete.Command != "" {
		result.Complete.Command = override.Complete.Command
	}
	if override.Diff.Command != "" {
		result.Diff.Command = override.Diff.Command
	}
	if override.Diff.TUICommand != "" {
		result.Diff.TUICommand = override.Diff.TUICommand
	}
	if override.Log.Level != "" {
		result.Log.Level = override.Log.Level
	}

	// Merge workers: override individual fields, not entire worker
	for name, overrideWorker := range override.Workers {
		baseWorker := result.Workers[name]
		if overrideWorker.CommandTemplate != "" {
			baseWorker.CommandTemplate = overrideWorker.CommandTemplate
		}
		if overrideWorker.Command != "" {
			baseWorker.Command = overrideWorker.Command
		}
		if overrideWorker.SystemArgs != "" {
			baseWorker.SystemArgs = overrideWorker.SystemArgs
		}
		if overrideWorker.Args != "" {
			baseWorker.Args = overrideWorker.Args
		}
		if overrideWorker.Prompt != "" {
			baseWorker.Prompt = overrideWorker.Prompt
		}
		result.Workers[name] = baseWorker
	}

	return result
}
