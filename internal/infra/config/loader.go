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
		Agents:   make(map[string]domain.Agent),
		Workers:  make(map[string]domain.Worker),
		Managers: make(map[string]domain.Manager),
	}
	var warnings []string

	for section, value := range raw {
		switch section {
		case "agents":
			if m, ok := value.(map[string]any); ok {
				ac := parseAgentsSection(m)
				for name, def := range ac.Defs {
					res.Agents[name] = domain.Agent{
						Command:             def.Command,
						CommandTemplate:     def.CommandTemplate,
						SystemArgs:          def.SystemArgs,
						DefaultModel:        def.DefaultModel,
						Description:         def.Description,
						WorktreeSetupScript: def.WorktreeSetupScript,
						ExcludePatterns:     def.ExcludePatterns,
					}
					for k := range def.Extra {
						warnings = append(warnings, fmt.Sprintf("unknown key in [agents.%s]: %s", name, k))
					}
				}
				for _, k := range ac.Unknowns {
					warnings = append(warnings, fmt.Sprintf("unknown key in [agents]: %s", k))
				}
			}
		case "workers":
			if m, ok := value.(map[string]any); ok {
				wc := parseWorkersSection(m)
				res.WorkersConfig.SystemPrompt = wc.SystemPrompt
				res.WorkersConfig.Prompt = wc.Prompt
				for name, def := range wc.Defs {
					res.Workers[name] = domain.Worker{
						Agent:           def.Agent,
						Inherit:         def.Inherit,
						CommandTemplate: def.CommandTemplate,
						Command:         def.Command,
						SystemArgs:      def.SystemArgs,
						Args:            def.Args,
						SystemPrompt:    def.SystemPrompt,
						Prompt:          def.Prompt,
						Model:           def.Model,
						Description:     def.Description,
					}

					for k := range def.Extra {
						warnings = append(warnings, fmt.Sprintf("unknown key in [workers.%s]: %s", name, k))
					}
				}
				for _, k := range wc.Unknowns {
					warnings = append(warnings, fmt.Sprintf("unknown key in [workers]: %s", k))
				}
			}
		case "managers":
			if m, ok := value.(map[string]any); ok {
				mc := parseManagersSection(m)
				res.ManagersConfig.SystemPrompt = mc.SystemPrompt
				res.ManagersConfig.Prompt = mc.Prompt
				for name, def := range mc.Defs {
					res.Managers[name] = domain.Manager{
						Agent:        def.Agent,
						Model:        def.Model,
						Args:         def.Args,
						SystemPrompt: def.SystemPrompt,
						Prompt:       def.Prompt,
						Description:  def.Description,
					}
					for k := range def.Extra {
						warnings = append(warnings, fmt.Sprintf("unknown key in [managers.%s]: %s", name, k))
					}
				}
				for _, k := range mc.Unknowns {
					warnings = append(warnings, fmt.Sprintf("unknown key in [managers]: %s", k))
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
		case "worktree":
			if m, ok := value.(map[string]any); ok {
				for k, v := range m {
					switch k {
					case "setup_command":
						if s, ok := v.(string); ok {
							res.Worktree.SetupCommand = s
						}
					case "copy":
						if arr, ok := v.([]any); ok {
							for _, item := range arr {
								if s, ok := item.(string); ok {
									res.Worktree.Copy = append(res.Worktree.Copy, s)
								}
							}
						}
					default:
						warnings = append(warnings, fmt.Sprintf("unknown key in [worktree]: %s", k))
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
	Defs         map[string]workerDef // Per-worker definitions from [workers.<name>]
	SystemPrompt string               // Common system prompt from [workers].system_prompt
	Prompt       string               // Common prompt from [workers].prompt
	Unknowns     []string             // Unknown keys in [workers]
}

type workerDef struct {
	Extra           map[string]any
	Agent           string
	Inherit         string
	CommandTemplate string
	Command         string
	SystemArgs      string
	Args            string
	SystemPrompt    string
	Prompt          string
	Model           string
	Description     string
}

// agentsConfig holds the parsed [agents] section.
type agentsConfig struct {
	Defs     map[string]agentDef // Per-agent definitions from [agents.<name>]
	Unknowns []string            // Unknown keys in [agents]
}

type agentDef struct {
	Extra               map[string]any
	Command             string
	CommandTemplate     string
	SystemArgs          string
	DefaultModel        string
	Description         string
	WorktreeSetupScript string
	ExcludePatterns     []string
}

// parseAgentsSection parses the raw agents map into structured agentsConfig.
func parseAgentsSection(raw map[string]any) agentsConfig {
	result := agentsConfig{
		Defs: make(map[string]agentDef),
	}

	for key, value := range raw {
		if subMap, ok := value.(map[string]any); ok {
			def := agentDef{
				Extra: make(map[string]any),
			}
			for k, v := range subMap {
				switch k {
				case "command":
					if s, ok := v.(string); ok {
						def.Command = s
					}
				case "command_template":
					if s, ok := v.(string); ok {
						def.CommandTemplate = s
					}
				case "system_args":
					if s, ok := v.(string); ok {
						def.SystemArgs = s
					}
				case "default_model":
					if s, ok := v.(string); ok {
						def.DefaultModel = s
					}
				case "description":
					if s, ok := v.(string); ok {
						def.Description = s
					}
				case "worktree_setup_script":
					if s, ok := v.(string); ok {
						def.WorktreeSetupScript = s
					}
				case "exclude_patterns":
					if arr, ok := v.([]any); ok {
						for _, item := range arr {
							if s, ok := item.(string); ok {
								def.ExcludePatterns = append(def.ExcludePatterns, s)
							}
						}
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

	return result
}

// parseWorkersSection parses the raw workers map into structured workersConfig.
func parseWorkersSection(raw map[string]any) workersConfig {
	result := workersConfig{
		Defs: make(map[string]workerDef),
	}

	for key, value := range raw {
		switch key {
		case "default":
			// Deprecated: default field is ignored, use workers.default instead
		case "system_prompt":
			if s, ok := value.(string); ok {
				result.SystemPrompt = s
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
					case "agent":
						if s, ok := v.(string); ok {
							def.Agent = s
						}
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
					case "system_prompt":
						if s, ok := v.(string); ok {
							def.SystemPrompt = s
						}
					case "prompt":
						if s, ok := v.(string); ok {
							def.Prompt = s
						}
					case "model":
						if s, ok := v.(string); ok {
							def.Model = s
						}
					case "description":
						if s, ok := v.(string); ok {
							def.Description = s
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

// managersConfig holds the parsed [managers] section.
type managersConfig struct {
	Defs         map[string]managerDef // Per-manager definitions from [managers.<name>]
	SystemPrompt string                // Common system prompt from [managers].system_prompt
	Prompt       string                // Common prompt from [managers].prompt
	Unknowns     []string              // Unknown keys in [managers]
}

type managerDef struct {
	Extra        map[string]any
	Agent        string
	Model        string
	Args         string
	SystemPrompt string
	Prompt       string
	Description  string
}

// parseManagersSection parses the raw managers map into structured managersConfig.
func parseManagersSection(raw map[string]any) managersConfig {
	result := managersConfig{
		Defs: make(map[string]managerDef),
	}

	for key, value := range raw {
		switch key {
		case "system_prompt":
			if s, ok := value.(string); ok {
				result.SystemPrompt = s
			}
		case "prompt":
			if s, ok := value.(string); ok {
				result.Prompt = s
			}
		default:
			if subMap, ok := value.(map[string]any); ok {
				def := managerDef{
					Extra: make(map[string]any),
				}
				for k, v := range subMap {
					switch k {
					case "agent":
						if s, ok := v.(string); ok {
							def.Agent = s
						}
					case "model":
						if s, ok := v.(string); ok {
							def.Model = s
						}
					case "args":
						if s, ok := v.(string); ok {
							def.Args = s
						}
					case "system_prompt":
						if s, ok := v.(string); ok {
							def.SystemPrompt = s
						}
					case "prompt":
						if s, ok := v.(string); ok {
							def.Prompt = s
						}
					case "description":
						if s, ok := v.(string); ok {
							def.Description = s
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
		Agents:         make(map[string]domain.Agent),
		WorkersConfig:  base.WorkersConfig,
		Workers:        make(map[string]domain.Worker),
		ManagersConfig: base.ManagersConfig,
		Managers:       make(map[string]domain.Manager),
		Complete:       base.Complete,
		Diff:           base.Diff,
		Log:            base.Log,
		Tasks:          base.Tasks,
		Worktree:       base.Worktree,
		Warnings:       append([]string{}, base.Warnings...),
	}

	// Add override warnings
	result.Warnings = append(result.Warnings, override.Warnings...)

	// Copy base agents
	for name, agent := range base.Agents {
		result.Agents[name] = agent
	}

	// Copy base workers
	for name, worker := range base.Workers {
		result.Workers[name] = worker
	}

	// Copy base managers
	for name, manager := range base.Managers {
		result.Managers[name] = manager
	}

	// Override with override config (WorkersConfig)
	if override.WorkersConfig.SystemPrompt != "" {
		result.WorkersConfig.SystemPrompt = override.WorkersConfig.SystemPrompt
	}
	if override.WorkersConfig.Prompt != "" {
		result.WorkersConfig.Prompt = override.WorkersConfig.Prompt
	}

	// Override with override config (ManagersConfig)
	if override.ManagersConfig.SystemPrompt != "" {
		result.ManagersConfig.SystemPrompt = override.ManagersConfig.SystemPrompt
	}
	if override.ManagersConfig.Prompt != "" {
		result.ManagersConfig.Prompt = override.ManagersConfig.Prompt
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
	if override.Worktree.SetupCommand != "" {
		result.Worktree.SetupCommand = override.Worktree.SetupCommand
	}
	if len(override.Worktree.Copy) > 0 {
		result.Worktree.Copy = override.Worktree.Copy
	}

	// Merge agents: override individual fields, not entire agent
	for name, overrideAgent := range override.Agents {
		baseAgent := result.Agents[name]
		if overrideAgent.Command != "" {
			baseAgent.Command = overrideAgent.Command
		}
		if overrideAgent.CommandTemplate != "" {
			baseAgent.CommandTemplate = overrideAgent.CommandTemplate
		}
		if overrideAgent.SystemArgs != "" {
			baseAgent.SystemArgs = overrideAgent.SystemArgs
		}
		if overrideAgent.DefaultModel != "" {
			baseAgent.DefaultModel = overrideAgent.DefaultModel
		}
		if overrideAgent.Description != "" {
			baseAgent.Description = overrideAgent.Description
		}
		if overrideAgent.WorktreeSetupScript != "" {
			baseAgent.WorktreeSetupScript = overrideAgent.WorktreeSetupScript
		}
		if len(overrideAgent.ExcludePatterns) > 0 {
			baseAgent.ExcludePatterns = overrideAgent.ExcludePatterns
		}
		result.Agents[name] = baseAgent
	}

	// Merge workers: override individual fields, not entire worker
	for name, overrideWorker := range override.Workers {
		baseWorker := result.Workers[name]
		if overrideWorker.Agent != "" {
			baseWorker.Agent = overrideWorker.Agent
		}
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
		if overrideWorker.SystemPrompt != "" {
			baseWorker.SystemPrompt = overrideWorker.SystemPrompt
		}
		if overrideWorker.Prompt != "" {
			baseWorker.Prompt = overrideWorker.Prompt
		}
		if overrideWorker.Model != "" {
			baseWorker.Model = overrideWorker.Model
		}
		if overrideWorker.Description != "" {
			baseWorker.Description = overrideWorker.Description
		}
		result.Workers[name] = baseWorker
	}

	// Merge managers: override individual fields, not entire manager
	for name, overrideManager := range override.Managers {
		baseManager := result.Managers[name]
		if overrideManager.Agent != "" {
			baseManager.Agent = overrideManager.Agent
		}
		if overrideManager.Model != "" {
			baseManager.Model = overrideManager.Model
		}
		if overrideManager.Args != "" {
			baseManager.Args = overrideManager.Args
		}
		if overrideManager.SystemPrompt != "" {
			baseManager.SystemPrompt = overrideManager.SystemPrompt
		}
		if overrideManager.Prompt != "" {
			baseManager.Prompt = overrideManager.Prompt
		}
		if overrideManager.Description != "" {
			baseManager.Description = overrideManager.Description
		}
		result.Managers[name] = baseManager
	}

	return result
}
