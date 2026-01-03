// Package config provides configuration loading functionality.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

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

	// Resolve inheritance after merging
	if err := base.ResolveInheritance(); err != nil {
		return nil, err
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

	// Resolve inheritance after merging
	if err := base.ResolveInheritance(); err != nil {
		return nil, err
	}

	return base, nil
}

// loadFile loads a configuration from a file.
func (l *Loader) loadFile(path string) (*domain.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	return convertRawToDomainConfig(raw), nil
}

// convertRawToDomainConfig converts the raw map to domain config and collects warnings.
func convertRawToDomainConfig(raw map[string]any) *domain.Config {
	res := &domain.Config{
		Workers: make(map[string]domain.WorkerAgent),
	}
	var warnings []string

	for section, value := range raw {
		switch section {
		case "workers":
			if m, ok := value.(map[string]any); ok {
				wc := parseWorkersSection(m)
				res.WorkersConfig.Default = wc.Default
				res.WorkersConfig.Prompt = wc.Prompt
				for name, def := range wc.Defs {
					res.Workers[name] = domain.WorkerAgent{
						Inherit:         def.Inherit,
						CommandTemplate: def.CommandTemplate,
						Command:         def.Command,
						SystemArgs:      def.SystemArgs,
						Args:            def.Args,
						Prompt:          def.Prompt,
						Model:           def.Model,
					}
					for k := range def.Extra {
						warnings = append(warnings, fmt.Sprintf("unknown key in [workers.%s]: %s", name, k))
					}
				}
				for _, k := range wc.Unknowns {
					warnings = append(warnings, fmt.Sprintf("unknown key in [workers]: %s", k))
				}
			}
		case "complete":
			if m, ok := value.(map[string]any); ok {
				for k, v := range m {
					switch k {
					case "command":
						if s, ok := v.(string); ok {
							res.Complete.Command = s
						}
					default:
						warnings = append(warnings, fmt.Sprintf("unknown key in [complete]: %s", k))
					}
				}
			}
		case "diff":
			if m, ok := value.(map[string]any); ok {
				for k, v := range m {
					switch k {
					case "command":
						if s, ok := v.(string); ok {
							res.Diff.Command = s
						}
					case "tui_command":
						if s, ok := v.(string); ok {
							res.Diff.TUICommand = s
						}
					default:
						warnings = append(warnings, fmt.Sprintf("unknown key in [diff]: %s", k))
					}
				}
			}
		case "log":
			if m, ok := value.(map[string]any); ok {
				for k, v := range m {
					switch k {
					case "level":
						if s, ok := v.(string); ok {
							res.Log.Level = s
						}
					default:
						warnings = append(warnings, fmt.Sprintf("unknown key in [log]: %s", k))
					}
				}
			}
		case "tasks":
			if m, ok := value.(map[string]any); ok {
				for k, v := range m {
					switch k {
					case "store":
						if s, ok := v.(string); ok {
							res.Tasks.Store = s
						}
					case "namespace":
						if s, ok := v.(string); ok {
							res.Tasks.Namespace = s
						}
					case "encrypt":
						if b, ok := v.(bool); ok {
							res.Tasks.Encrypt = b
						}
					default:
						warnings = append(warnings, fmt.Sprintf("unknown key in [tasks]: %s", k))
					}
				}
			}
		default:
			warnings = append(warnings, fmt.Sprintf("unknown section: %s", section))
		}
	}

	sort.Strings(warnings)
	res.Warnings = warnings
	return res
}

// workersConfig holds the parsed [workers] section.
type workersConfig struct {
	Defs     map[string]workerDef // Per-worker definitions from [workers.<name>]
	Default  string               // Default worker name from [workers].default
	Prompt   string               // Common prompt from [workers].prompt
	Unknowns []string             // Unknown keys in [workers]
}

type workerDef struct {
	Inherit         string
	CommandTemplate string
	Command         string
	SystemArgs      string
	Args            string
	Prompt          string
	Model           string
	Extra           map[string]any
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
				def := workerDef{
					Extra: make(map[string]any),
				}
				for k, v := range subMap {
					switch k {
					case "inherit":
						if s, ok := v.(string); ok {
							def.Inherit = s
						}
					case "command_template":
						if s, ok := v.(string); ok {
							def.CommandTemplate = s
						}
					case "command":
						if s, ok := v.(string); ok {
							def.Command = s
						}
					case "system_args":
						if s, ok := v.(string); ok {
							def.SystemArgs = s
						}
					case "args":
						if s, ok := v.(string); ok {
							def.Args = s
						}
					case "prompt":
						if s, ok := v.(string); ok {
							def.Prompt = s
						}
					case "model":
						if s, ok := v.(string); ok {
							def.Model = s
						}
					default:
						def.Extra[k] = v
					}
				}
				result.Defs[key] = def
			} else {
				result.Unknowns = append(result.Unknowns, key)
			}
		}
	}

	return result
}

// mergeConfigs merges two configs, with override taking precedence.
func mergeConfigs(base, override *domain.Config) *domain.Config {
	result := &domain.Config{
		WorkersConfig: base.WorkersConfig,
		Complete:      base.Complete,
		Diff:          base.Diff,
		Log:           base.Log,
		Tasks:         base.Tasks,
		Workers:       make(map[string]domain.WorkerAgent),
		Warnings:      append([]string{}, base.Warnings...),
	}

	// Add override warnings
	result.Warnings = append(result.Warnings, override.Warnings...)

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
	if override.Tasks.Store != "" {
		result.Tasks.Store = override.Tasks.Store
	}
	if override.Tasks.Namespace != "" {
		result.Tasks.Namespace = override.Tasks.Namespace
	}
	if override.Tasks.Encrypt {
		result.Tasks.Encrypt = override.Tasks.Encrypt
	}

	// Merge workers: override individual fields, not entire worker
	for name, overrideWorker := range override.Workers {
		baseWorker := result.Workers[name]
		if overrideWorker.Inherit != "" {
			baseWorker.Inherit = overrideWorker.Inherit
		}
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
		if overrideWorker.Model != "" {
			baseWorker.Model = overrideWorker.Model
		}
		result.Workers[name] = baseWorker
	}

	return result
}
