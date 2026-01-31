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
	crewDir       string // Path to .crew directory
	repoRoot      string // Path to repository root
	globalConfDir string // Path to global config directory (e.g., ~/.config/crew)
}

// NewLoader creates a new Loader.
func NewLoader(crewDir, repoRoot string) *Loader {
	return &Loader{
		crewDir:       crewDir,
		repoRoot:      repoRoot,
		globalConfDir: defaultGlobalConfigDir(),
	}
}

// NewLoaderWithGlobalDir creates a new Loader with a custom global config directory.
// This is useful for testing.
func NewLoaderWithGlobalDir(crewDir, repoRoot, globalConfDir string) *Loader {
	return &Loader{
		crewDir:       crewDir,
		repoRoot:      repoRoot,
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

// Load returns the merged configuration.
// Priority (later takes precedence): global < override < .crew.toml < config.toml < config.runtime.toml
func (l *Loader) Load() (*domain.Config, error) {
	return l.LoadWithOptions(domain.LoadConfigOptions{})
}

// LoadGlobal returns only the global configuration.
func (l *Loader) LoadGlobal() (*domain.Config, error) {
	if l.globalConfDir == "" {
		return nil, os.ErrNotExist
	}
	globalPath := filepath.Join(l.globalConfDir, domain.ConfigFileName)
	return l.loadFile(globalPath)
}

// LoadRootRepo returns only the repository root configuration (.crew.toml).
func (l *Loader) LoadRootRepo() (*domain.Config, error) {
	if l.repoRoot == "" {
		return nil, os.ErrNotExist
	}
	rootRepoPath := domain.RepoRootConfigPath(l.repoRoot)
	return l.loadFile(rootRepoPath)
}

// LoadRepo returns only the repository configuration (.crew/config.toml).
func (l *Loader) LoadRepo() (*domain.Config, error) {
	repoPath := filepath.Join(l.crewDir, domain.ConfigFileName)
	return l.loadFile(repoPath)
}

// LoadRuntime returns only the runtime configuration (.crew/config.runtime.toml).
func (l *Loader) LoadRuntime() (*domain.Config, error) {
	runtimePath := filepath.Join(l.crewDir, domain.ConfigRuntimeFileName)
	return l.loadFile(runtimePath)
}

// LoadOverride returns only the global override configuration (config.override.toml).
func (l *Loader) LoadOverride() (*domain.Config, error) {
	if l.globalConfDir == "" {
		return nil, os.ErrNotExist
	}
	overridePath := filepath.Join(l.globalConfDir, domain.ConfigOverrideFileName)
	return l.loadFile(overridePath)
}

// LoadWithOptions returns the merged configuration with options to ignore sources.
func (l *Loader) LoadWithOptions(opts domain.LoadConfigOptions) (*domain.Config, error) {
	var global, override, rootRepo, repo, runtime *domain.Config
	var err error

	// Load global config unless ignored
	if !opts.IgnoreGlobal {
		global, err = l.LoadGlobal()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	// Load override config unless ignored
	if !opts.IgnoreOverride {
		override, err = l.LoadOverride()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	// Load root repo config unless ignored
	if !opts.IgnoreRootRepo {
		rootRepo, err = l.LoadRootRepo()
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

	// Load runtime config unless ignored
	if !opts.IgnoreRuntime {
		runtime, err = l.LoadRuntime()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	// Start with default config and register builtin agents
	base := domain.NewDefaultConfig()
	Register(base)

	// If all don't exist or are ignored, return default config
	if global == nil && override == nil && rootRepo == nil && repo == nil && runtime == nil {
		return base, nil
	}

	// Merge: default <- global <- override <- rootRepo <- repo <- runtime (later takes precedence)
	if global != nil {
		base = mergeConfigs(base, global)
	}
	if override != nil {
		base = mergeConfigs(base, override)
	}
	if rootRepo != nil {
		base = mergeConfigs(base, rootRepo)
	}
	if repo != nil {
		base = mergeConfigs(base, repo)
	}
	if runtime != nil {
		base = mergeConfigs(base, runtime)
	}

	// Resolve inheritance after merging
	if err := base.ResolveInheritance(); err != nil {
		return nil, err
	}

	appendHelpWarnings(base)
	if len(base.Warnings) > 1 {
		sort.Strings(base.Warnings)
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

	return convertRawToDomainConfig(raw, path), nil
}

// convertRawToDomainConfig converts the raw map to domain config and collects warnings.
func convertRawToDomainConfig(raw map[string]any, sourcePath string) *domain.Config {
	res := &domain.Config{
		Agents: make(map[string]domain.Agent),
	}
	var warnings []string
	sourceDir := ""
	if sourcePath != "" {
		sourceDir = filepath.Dir(sourcePath)
	}

	for section, value := range raw {
		switch section {
		case "agents":
			if m, ok := value.(map[string]any); ok {
				ac := parseAgentsSection(m)
				// Handle top-level agents config
				if ac.DefaultWorker != "" {
					res.AgentsConfig.DefaultWorker = ac.DefaultWorker
				}
				if ac.DefaultManager != "" {
					res.AgentsConfig.DefaultManager = ac.DefaultManager
				}
				if ac.DefaultReviewer != "" {
					res.AgentsConfig.DefaultReviewer = ac.DefaultReviewer
				}
				if ac.WorkerPrompt != "" {
					res.AgentsConfig.WorkerPrompt = ac.WorkerPrompt
				}
				if ac.ManagerPrompt != "" {
					res.AgentsConfig.ManagerPrompt = ac.ManagerPrompt
				}
				if ac.ReviewerPrompt != "" {
					res.AgentsConfig.ReviewerPrompt = ac.ReviewerPrompt
				}
				if len(ac.DisabledAgents) > 0 {
					res.AgentsConfig.DisabledAgents = ac.DisabledAgents
				}
				for name, def := range ac.Defs {
					res.Agents[name] = domain.Agent{
						Inherit:         def.Inherit,
						CommandTemplate: def.CommandTemplate,
						Role:            domain.Role(def.Role),
						SystemPrompt:    def.SystemPrompt,
						Prompt:          def.Prompt,
						Args:            def.Args,
						DefaultModel:    def.DefaultModel,
						Description:     def.Description,
						SetupScript:     def.SetupScript,
						Hidden:          def.Hidden,
						Env:             def.Env,
					}
					for k := range def.Extra {
						warnings = append(warnings, fmt.Sprintf("unknown key in [agents.%s]: %s", name, k))
					}
				}
				for _, k := range ac.Unknowns {
					warnings = append(warnings, fmt.Sprintf("unknown key in [agents]: %s", k))
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
					case "min_reviews":
						if i, ok := v.(int64); ok {
							if i <= 0 {
								warnings = append(warnings, fmt.Sprintf("invalid value for complete.min_reviews: %d (expected >= 1)", i))
							} else {
								res.Complete.MinReviews = int(i)
							}
						}
					case "review_mode":
						if s, ok := v.(string); ok {
							mode := domain.ReviewMode(s)
							if mode.IsValid() {
								res.Complete.ReviewMode = mode
								res.Complete.ReviewModeSet = true
							} else {
								warnings = append(warnings, fmt.Sprintf("invalid value for complete.review_mode: %q (expected \"auto\", \"manual\", or \"auto_fix\")", s))
							}
						}
					case "auto_fix":
						// Legacy: preserve auto_fix for use case compatibility
						if b, ok := v.(bool); ok {
							res.Complete.AutoFix = b //nolint:staticcheck // Legacy compatibility
							res.Complete.AutoFixSet = true
						}
					case "auto_fix_max_retries":
						if i, ok := v.(int64); ok {
							res.Complete.AutoFixMaxRetries = int(i)
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
		case "help":
			if m, ok := value.(map[string]any); ok {
				for k, v := range m {
					switch k {
					case "worker":
						if s, ok := v.(string); ok {
							res.Help.Worker = s
						}
					case "worker_file":
						if s, ok := v.(string); ok {
							res.Help.WorkerFile = resolveHelpFilePath(s, sourceDir)
						}
					case "manager":
						if s, ok := v.(string); ok {
							res.Help.Manager = s
						}
					case "manager_file":
						if s, ok := v.(string); ok {
							res.Help.ManagerFile = resolveHelpFilePath(s, sourceDir)
						}
					case "manager_onboarding":
						if s, ok := v.(string); ok {
							res.Help.ManagerOnboarding = s
						}
					case "manager_onboarding_file":
						if s, ok := v.(string); ok {
							res.Help.ManagerOnboardingFile = resolveHelpFilePath(s, sourceDir)
						}
					case "manager_auto":
						if s, ok := v.(string); ok {
							res.Help.ManagerAuto = s
						}
					case "manager_auto_file":
						if s, ok := v.(string); ok {
							res.Help.ManagerAutoFile = resolveHelpFilePath(s, sourceDir)
						}
					default:
						warnings = append(warnings, fmt.Sprintf("unknown key in [help]: %s", k))
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
					case "new_task_base":
						if s, ok := v.(string); ok {
							res.Tasks.NewTaskBase = s
							// Validate: only "", "current", or "default" are valid
							if s != "" && s != "current" && s != "default" {
								warnings = append(warnings, fmt.Sprintf("invalid value for tasks.new_task_base: %q (expected \"current\" or \"default\")", s))
							}
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
		case "tui":
			if m, ok := value.(map[string]any); ok {
				for k, v := range m {
					switch k {
					case "keybindings":
						if kbMap, ok := v.(map[string]any); ok {
							if res.TUI.Keybindings == nil {
								res.TUI.Keybindings = make(map[string]domain.TUIKeybinding)
							}
							for key, binding := range kbMap {
								if bMap, ok := binding.(map[string]any); ok {
									kb := domain.TUIKeybinding{}
									for bk, bv := range bMap {
										switch bk {
										case "command":
											if s, ok := bv.(string); ok {
												kb.Command = s
											}
										case "description":
											if s, ok := bv.(string); ok {
												kb.Description = s
											}
										case "override":
											if b, ok := bv.(bool); ok {
												kb.Override = b
											}
										case "worktree":
											if b, ok := bv.(bool); ok {
												kb.Worktree = b
											}
										default:
											warnings = append(warnings, fmt.Sprintf("unknown key in [tui.keybindings.%s]: %s", key, bk))
										}
									}
									res.TUI.Keybindings[key] = kb
								}
							}
						}
					default:
						warnings = append(warnings, fmt.Sprintf("unknown key in [tui]: %s", k))
					}
				}
			}
		case "onboarding_done":
			if b, ok := value.(bool); ok {
				res.OnboardingDone = b
			}
		default:
			warnings = append(warnings, fmt.Sprintf("unknown section: %s", section))
		}
	}

	sort.Strings(warnings)
	res.Warnings = warnings
	return res
}

// agentsConfig holds the parsed [agents] section.
type agentsConfig struct {
	Defs            map[string]agentDef // Per-agent definitions from [agents.<name>]
	DefaultWorker   string              // Default worker agent name
	DefaultManager  string              // Default manager agent name
	DefaultReviewer string              // Default reviewer agent name
	WorkerPrompt    string              // Default prompt for all worker agents
	ManagerPrompt   string              // Default prompt for all manager agents
	ReviewerPrompt  string              // Default prompt for all reviewer agents
	DisabledAgents  []string            // List of agent names to disable
	Unknowns        []string            // Unknown keys in [agents]
}

type agentDef struct {
	Extra           map[string]any
	Env             map[string]string
	Inherit         string
	CommandTemplate string
	Role            string
	SystemPrompt    string
	Prompt          string
	Args            string
	DefaultModel    string
	Description     string
	SetupScript     string
	Hidden          bool
}

// parseAgentsSection parses the raw agents map into structured agentsConfig.
func parseAgentsSection(raw map[string]any) agentsConfig {
	result := agentsConfig{
		Defs: make(map[string]agentDef),
	}

	for key, value := range raw {
		switch key {
		case "worker_default":
			if s, ok := value.(string); ok {
				result.DefaultWorker = s
			}
		case "manager_default":
			if s, ok := value.(string); ok {
				result.DefaultManager = s
			}
		case "reviewer_default":
			if s, ok := value.(string); ok {
				result.DefaultReviewer = s
			}
		case "worker_prompt":
			if s, ok := value.(string); ok {
				result.WorkerPrompt = s
			}
		case "manager_prompt":
			if s, ok := value.(string); ok {
				result.ManagerPrompt = s
			}
		case "reviewer_prompt":
			if s, ok := value.(string); ok {
				result.ReviewerPrompt = s
			}
		case "disabled_agents":
			if arr, ok := value.([]any); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						result.DisabledAgents = append(result.DisabledAgents, s)
					}
				}
			}
		default:
			if subMap, ok := value.(map[string]any); ok {
				def := agentDef{
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
					case "role":
						if s, ok := v.(string); ok {
							def.Role = s
						}
					case "system_prompt":
						if s, ok := v.(string); ok {
							def.SystemPrompt = s
						}
					case "prompt":
						if s, ok := v.(string); ok {
							def.Prompt = s
						}
					case "args":
						if s, ok := v.(string); ok {
							def.Args = s
						}
					case "default_model":
						if s, ok := v.(string); ok {
							def.DefaultModel = s
						}
					case "description":
						if s, ok := v.(string); ok {
							def.Description = s
						}
					case "setup_script":
						if s, ok := v.(string); ok {
							def.SetupScript = s
						}
					case "hidden":
						if b, ok := v.(bool); ok {
							def.Hidden = b
						}
					case "env":
						if envMap, ok := v.(map[string]any); ok {
							def.Env = make(map[string]string)
							for envKey, envVal := range envMap {
								if s, ok := envVal.(string); ok {
									def.Env[envKey] = s
								} else {
									// Non-string env values are treated as unknown keys
									def.Extra["env."+envKey] = envVal
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
	}

	return result
}

func resolveHelpFilePath(value, sourceDir string) string {
	if value == "" {
		return ""
	}
	if sourceDir == "" || filepath.IsAbs(value) {
		return value
	}
	joined := filepath.Join(sourceDir, value)
	resolved, err := filepath.Abs(joined)
	if err != nil {
		return filepath.Clean(joined)
	}
	return resolved
}

func appendHelpWarnings(cfg *domain.Config) {
	warnBoth := func(inline, file, key string) {
		if inline == "" || file == "" {
			return
		}
		cfg.Warnings = append(cfg.Warnings, fmt.Sprintf("both help.%s and help.%s_file are set; using help.%s_file", key, key, key))
	}

	warnBoth(cfg.Help.Worker, cfg.Help.WorkerFile, "worker")
	warnBoth(cfg.Help.Manager, cfg.Help.ManagerFile, "manager")
	warnBoth(cfg.Help.ManagerOnboarding, cfg.Help.ManagerOnboardingFile, "manager_onboarding")
	warnBoth(cfg.Help.ManagerAuto, cfg.Help.ManagerAutoFile, "manager_auto")
}

// mergeConfigs merges two configs, with override taking precedence.
func mergeConfigs(base, override *domain.Config) *domain.Config {
	result := &domain.Config{
		Agents:       make(map[string]domain.Agent),
		AgentsConfig: base.AgentsConfig,
		Complete:     base.Complete,
		Diff:         base.Diff,
		Log:          base.Log,
		Help:         base.Help,
		Tasks:        base.Tasks,
		TUI:          base.TUI,
		Worktree:     base.Worktree,
		Warnings:     append([]string{}, base.Warnings...),
	}

	// Add override warnings
	result.Warnings = append(result.Warnings, override.Warnings...)

	// Copy base agents
	for name, agent := range base.Agents {
		result.Agents[name] = agent
	}

	// Override with override config (AgentsConfig)
	if override.AgentsConfig.DefaultWorker != "" {
		result.AgentsConfig.DefaultWorker = override.AgentsConfig.DefaultWorker
	}
	if override.AgentsConfig.DefaultManager != "" {
		result.AgentsConfig.DefaultManager = override.AgentsConfig.DefaultManager
	}
	if override.AgentsConfig.DefaultReviewer != "" {
		result.AgentsConfig.DefaultReviewer = override.AgentsConfig.DefaultReviewer
	}
	if override.AgentsConfig.WorkerPrompt != "" {
		result.AgentsConfig.WorkerPrompt = override.AgentsConfig.WorkerPrompt
	}
	if override.AgentsConfig.ManagerPrompt != "" {
		result.AgentsConfig.ManagerPrompt = override.AgentsConfig.ManagerPrompt
	}
	if override.AgentsConfig.ReviewerPrompt != "" {
		result.AgentsConfig.ReviewerPrompt = override.AgentsConfig.ReviewerPrompt
	}
	if len(override.AgentsConfig.DisabledAgents) > 0 {
		result.AgentsConfig.DisabledAgents = override.AgentsConfig.DisabledAgents
	}

	// Override other sections
	if override.Complete.Command != "" {
		result.Complete.Command = override.Complete.Command
	}
	if override.Complete.ReviewModeSet {
		result.Complete.ReviewMode = override.Complete.ReviewMode
		result.Complete.ReviewModeSet = true
	}
	if override.Complete.AutoFixSet {
		result.Complete.AutoFix = override.Complete.AutoFix //nolint:staticcheck // Legacy compatibility
		result.Complete.AutoFixSet = true
	}
	if override.Complete.MinReviews > 0 {
		result.Complete.MinReviews = override.Complete.MinReviews
	}
	if override.Complete.AutoFixMaxRetries > 0 {
		result.Complete.AutoFixMaxRetries = override.Complete.AutoFixMaxRetries
	}
	if override.Diff.Command != "" {
		result.Diff.Command = override.Diff.Command
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
	if override.Tasks.NewTaskBase != "" {
		result.Tasks.NewTaskBase = override.Tasks.NewTaskBase
	}
	if override.Tasks.Encrypt {
		result.Tasks.Encrypt = override.Tasks.Encrypt
	}
	if override.Help.Worker != "" {
		result.Help.Worker = override.Help.Worker
	}
	if override.Help.WorkerFile != "" {
		result.Help.WorkerFile = override.Help.WorkerFile
	}
	if override.Help.Manager != "" {
		result.Help.Manager = override.Help.Manager
	}
	if override.Help.ManagerFile != "" {
		result.Help.ManagerFile = override.Help.ManagerFile
	}
	if override.Help.ManagerOnboarding != "" {
		result.Help.ManagerOnboarding = override.Help.ManagerOnboarding
	}
	if override.Help.ManagerOnboardingFile != "" {
		result.Help.ManagerOnboardingFile = override.Help.ManagerOnboardingFile
	}
	if override.Help.ManagerAuto != "" {
		result.Help.ManagerAuto = override.Help.ManagerAuto
	}
	if override.Help.ManagerAutoFile != "" {
		result.Help.ManagerAutoFile = override.Help.ManagerAutoFile
	}
	if override.Worktree.SetupCommand != "" {
		result.Worktree.SetupCommand = override.Worktree.SetupCommand
	}
	if len(override.Worktree.Copy) > 0 {
		result.Worktree.Copy = override.Worktree.Copy
	}
	if len(override.TUI.Keybindings) > 0 {
		if result.TUI.Keybindings == nil {
			result.TUI.Keybindings = make(map[string]domain.TUIKeybinding)
		}
		for key, binding := range override.TUI.Keybindings {
			result.TUI.Keybindings[key] = binding
		}
	}
	if override.OnboardingDone {
		result.OnboardingDone = override.OnboardingDone
	}

	// Merge agents: override individual fields, not entire agent
	for name, overrideAgent := range override.Agents {
		baseAgent := result.Agents[name]
		if overrideAgent.Inherit != "" {
			baseAgent.Inherit = overrideAgent.Inherit
		}
		if overrideAgent.CommandTemplate != "" {
			baseAgent.CommandTemplate = overrideAgent.CommandTemplate
		}
		if overrideAgent.Role != "" {
			baseAgent.Role = overrideAgent.Role
		}
		if overrideAgent.SystemPrompt != "" {
			baseAgent.SystemPrompt = overrideAgent.SystemPrompt
		}
		if overrideAgent.Prompt != "" {
			baseAgent.Prompt = overrideAgent.Prompt
		}
		if overrideAgent.Args != "" {
			baseAgent.Args = overrideAgent.Args
		}
		if overrideAgent.DefaultModel != "" {
			baseAgent.DefaultModel = overrideAgent.DefaultModel
		}
		if overrideAgent.Description != "" {
			baseAgent.Description = overrideAgent.Description
		}
		if overrideAgent.SetupScript != "" {
			baseAgent.SetupScript = overrideAgent.SetupScript
		}
		if overrideAgent.Hidden {
			baseAgent.Hidden = overrideAgent.Hidden
		}
		if len(overrideAgent.Env) > 0 {
			baseAgent.Env = mergeEnv(baseAgent.Env, overrideAgent.Env)
		}
		result.Agents[name] = baseAgent
	}

	return result
}

// mergeEnv merges two environment maps.
// Child values override parent values for the same key.
func mergeEnv(parent, child map[string]string) map[string]string {
	if parent == nil && child == nil {
		return nil
	}
	merged := make(map[string]string)
	for key, value := range parent {
		merged[key] = value
	}
	for key, value := range child {
		merged[key] = value
	}
	return merged
}
