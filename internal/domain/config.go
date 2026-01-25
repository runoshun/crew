package domain

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

//go:embed config_template.toml
var configTemplateContent string

// Config represents the application configuration.
// Fields are ordered to minimize memory padding.
type Config struct {
	Agents       map[string]Agent `toml:"agents"` // Agent definitions from [agents.<name>]
	Warnings     []string         `toml:"-"`
	AgentsConfig AgentsConfig     `toml:"agents"` // Common [agents] settings
	TUI          TUIConfig        `toml:"tui"`
	Worktree     WorktreeConfig   `toml:"worktree"`
	Tasks        TasksConfig      `toml:"tasks"`
	Diff         DiffConfig       `toml:"diff"`
	Log          LogConfig        `toml:"log"`
	Complete     CompleteConfig   `toml:"complete"`

	OnboardingDone bool `toml:"onboarding_done,omitempty"` // Whether onboarding has been completed
}

// TasksConfig holds settings for task storage from [tasks] section.
type TasksConfig struct {
	Store       string `toml:"store,omitempty"`         // Storage backend: "git" (default) or "json"
	Namespace   string `toml:"namespace,omitempty"`     // Git namespace for refs (default: "crew")
	NewTaskBase string `toml:"new_task_base,omitempty"` // Base branch for new tasks: "current" (default) or "default"
	Encrypt     bool   `toml:"encrypt,omitempty"`       // Enable encryption for task data (default: false)
	SkipReview  bool   `toml:"skip_review,omitempty"`   // Default skip_review setting for new tasks (default: false)
}

// AgentsConfig holds common settings for all agents from [agents] section.
type AgentsConfig struct {
	DefaultWorker   string   `toml:"worker_default,omitempty"`   // Default worker agent name
	DefaultManager  string   `toml:"manager_default,omitempty"`  // Default manager agent name
	DefaultReviewer string   `toml:"reviewer_default,omitempty"` // Default reviewer agent name
	WorkerPrompt    string   `toml:"worker_prompt,omitempty"`    // Default prompt for all worker agents
	ManagerPrompt   string   `toml:"manager_prompt,omitempty"`   // Default prompt for all manager agents
	ReviewerPrompt  string   `toml:"reviewer_prompt,omitempty"`  // Default prompt for all reviewer agents
	DisabledAgents  []string `toml:"disabled_agents,omitempty"`  // List of agent names to disable
}

// Role represents the role of an agent.
type Role string

// Valid roles for agents.
const (
	RoleWorker   Role = "worker"
	RoleReviewer Role = "reviewer"
	RoleManager  Role = "manager"
)

// Agent defines a unified agent configuration that can serve as worker, reviewer, or manager.
// This replaces the previous separate Worker and Manager types.
type Agent struct {
	// Inheritance
	Inherit string `toml:"inherit,omitempty"` // Name of agent to inherit from (optional)

	// Command execution
	CommandTemplate string `toml:"command_template,omitempty"` // Full command template (e.g., "opencode -m {{.Model}} {{.Args}} --prompt {{.Prompt}}")

	// Role configuration
	Role         Role   `toml:"role,omitempty"`          // Role: worker, reviewer, manager
	SystemPrompt string `toml:"system_prompt,omitempty"` // System prompt template
	Prompt       string `toml:"prompt,omitempty"`        // User prompt template
	Args         string `toml:"args,omitempty"`          // Additional arguments

	// Agent metadata
	DefaultModel string `toml:"default_model,omitempty"` // Default model for this agent
	Description  string `toml:"description,omitempty"`   // Description of the agent's purpose

	// Worktree setup (for workers/reviewers)
	SetupScript string `toml:"setup_script,omitempty"` // Setup script (replaces worktree_setup_script and exclude_patterns)

	// Visibility
	Hidden bool `toml:"hidden,omitempty"` // Hide from TUI agent list
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

	// Boolean fields
	Continue bool // --continue flag was specified
}

// RenderCommandResult holds the results of RenderCommand.
type RenderCommandResult struct {
	Command string // The full command to execute (e.g., `claude --model opus "$PROMPT"`)
	Prompt  string // The prompt content to be stored in PROMPT shell variable
}

// RenderCommand renders the full command string and prompt content for this agent.
// It performs two-phase template expansion:
// 1. Expand SystemPrompt and Prompt templates with CommandData to generate prompt content
// 2. Expand CommandTemplate with Args, Model, Continue, and shell variable reference
//
// The promptOverride parameter is the shell variable reference (e.g., `"$PROMPT"`) that will be
// embedded in the command and expanded at runtime to the actual prompt content.
// The defaultSystemPrompt is used when Agent.SystemPrompt is empty.
// The defaultPrompt is used when Agent.Prompt is empty.
func (a *Agent) RenderCommand(data CommandData, promptOverride, defaultSystemPrompt, defaultPrompt string) (RenderCommandResult, error) {
	// Phase 1: Expand Args
	args, err := expandString(a.Args, data)
	if err != nil {
		return RenderCommandResult{}, err
	}

	// Phase 2: Expand Prompt templates to generate prompt content
	sysPromptTemplate := a.SystemPrompt
	if sysPromptTemplate == "" {
		sysPromptTemplate = defaultSystemPrompt
	}
	sysPromptContent, err := expandString(sysPromptTemplate, data)
	if err != nil {
		return RenderCommandResult{}, err
	}

	// Concatenate defaultPrompt and a.Prompt (skip if empty to avoid extra newlines)
	var userPromptParts []string
	if defaultPrompt != "" {
		userPromptParts = append(userPromptParts, defaultPrompt)
	}
	if a.Prompt != "" {
		userPromptParts = append(userPromptParts, a.Prompt)
	}
	userPromptTemplate := strings.Join(userPromptParts, "\n\n")

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
	cmdData := map[string]any{
		"Args":     args,
		"Prompt":   promptOverride,
		"Continue": data.Continue,
		"Model":    data.Model,
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

// ReviewMode specifies how the review process should be handled on completion.
type ReviewMode string

const (
	// ReviewModeAuto starts review asynchronously in background after completion.
	// This is the default mode and matches the previous auto_fix=false behavior.
	ReviewModeAuto ReviewMode = "auto"

	// ReviewModeManual does not start review automatically.
	// Sets status to for_review and waits for manual review initiation.
	ReviewModeManual ReviewMode = "manual"

	// ReviewModeAutoFix starts review synchronously and attempts auto-correction.
	// Matches the previous auto_fix=true behavior.
	ReviewModeAutoFix ReviewMode = "auto_fix"
)

// AllReviewModes returns all valid review mode values.
func AllReviewModes() []ReviewMode {
	return []ReviewMode{ReviewModeAuto, ReviewModeManual, ReviewModeAutoFix}
}

// IsValid returns true if the review mode is a known valid value.
func (m ReviewMode) IsValid() bool {
	switch m {
	case ReviewModeAuto, ReviewModeManual, ReviewModeAutoFix:
		return true
	default:
		return false
	}
}

// Display returns a human-readable representation of the review mode.
func (m ReviewMode) Display() string {
	switch m {
	case ReviewModeAuto:
		return "Auto"
	case ReviewModeManual:
		return "Manual"
	case ReviewModeAutoFix:
		return "Auto-fix"
	default:
		return string(m)
	}
}

// NextMode returns the next review mode in the cycle: auto -> manual -> auto_fix -> auto.
func (m ReviewMode) NextMode() ReviewMode {
	switch m {
	case ReviewModeAuto:
		return ReviewModeManual
	case ReviewModeManual:
		return ReviewModeAutoFix
	case ReviewModeAutoFix:
		return ReviewModeAuto
	default:
		return ReviewModeAuto
	}
}

// CompleteConfig holds completion gate settings from [complete] section.
// Fields are ordered to minimize memory padding.
type CompleteConfig struct {
	Command           string     `toml:"command,omitempty"`              // Command to run as CI gate on complete
	ReviewMode        ReviewMode `toml:"review_mode,omitempty"`          // Review mode: auto (default), manual, auto_fix
	AutoFixMaxRetries int        `toml:"auto_fix_max_retries,omitempty"` // Maximum retry count for auto-fix mode (default: 3)
	ReviewModeSet     bool       `toml:"-"`                              // True if ReviewMode was explicitly set in config (not exported to TOML)

	// Deprecated: Use ReviewMode instead. Kept for backward compatibility with existing configs.
	AutoFix    bool `toml:"auto_fix,omitempty"` // Enable auto-fix mode (run review synchronously)
	AutoFixSet bool `toml:"-"`                  // True if AutoFix was explicitly set in config (not exported to TOML)
}

// DiffConfig holds diff display settings from [diff] section.
type DiffConfig struct {
	Command string `toml:"command,omitempty"` // Command to display diff (with {{.Args}} template support)
}

// LogConfig holds logging settings from [log] section.
type LogConfig struct {
	Level string `toml:"level,omitempty"` // Log level: debug, info, warn, error
}

// WorktreeConfig holds worktree customization settings from [worktree] section.
type WorktreeConfig struct {
	SetupCommand string   `toml:"setup_command,omitempty"` // Command to run after worktree creation
	Copy         []string `toml:"copy,omitempty"`          // Files/directories to copy (with CoW if available)
}

// TUIKeybinding represents a custom keybinding configuration for TUI.
type TUIKeybinding struct {
	Command     string `toml:"command"`     // Command to execute
	Description string `toml:"description"` // Description shown in help
	Override    bool   `toml:"override"`    // Allow overriding existing keybindings
	Worktree    bool   `toml:"worktree"`    // Execute in task worktree instead of repository root
}

// TUIConfig holds TUI customization settings from [tui] section.
type TUIConfig struct {
	Keybindings map[string]TUIKeybinding `toml:"keybindings"` // Custom keybindings
}

// Default configuration values.
const (
	DefaultLogLevel = "info"
)

// ReviewResultMarker is the marker line that separates verbose output from the final review result.
const ReviewResultMarker = "---REVIEW_RESULT---"

// ReviewLGTMPrefix is the prefix indicating the review passed (LGTM).
const ReviewLGTMPrefix = "✅ LGTM"

// DefaultSystemPrompt is the default system prompt template for workers.
// It uses Go template syntax with CommandData fields.
const DefaultSystemPrompt = `You are working on Task #{{.TaskID}}.

IMPORTANT: First run 'crew --help-worker' and follow the workflow instructions exactly.`

// DefaultManagerSystemPrompt is the default system prompt template for managers.
const DefaultManagerSystemPrompt = `You are a Manager agent for git-crew.

IMPORTANT: First run 'crew --help-manager' and follow the usage instructions.

Support users with task management as an assistant.
- Understand current status and suggest next actions
- Execute operations on behalf of users and report results concisely
- Proactively report problems
- Delegate code implementation to worker agents`

// DefaultReviewerSystemPrompt is the default system prompt template for reviewers.
const DefaultReviewerSystemPrompt = `You are a code reviewer for git-crew Task #{{.TaskID}}.

## Available Commands

- crew show {{.TaskID}} - View task details
- crew diff {{.TaskID}} - View changes

## Review Checklist

1. Correctness - Does the code work as intended?
2. Tests - Are edge cases covered?
3. Architecture - Does it follow project patterns?
4. Error handling - Are errors handled appropriately?
5. Readability - Will future developers understand this?

## Output Format

IMPORTANT: Output your final review after the marker line:
` + ReviewResultMarker + `

Start with: ` + "`✅ LGTM`" + `, ` + "`⚠️ Minor issues`" + `, or ` + "`❌ Needs changes`" + `
Then list specific issues with file:line references.`

// Directory and file names for git-crew.
const (
	CrewDirName            = "crew"                 // Directory name for crew data
	ConfigFileName         = "config.toml"          // Config file name
	ConfigOverrideFileName = "config.override.toml" // Override config file name
	ConfigRuntimeFileName  = "config.runtime.toml"  // Runtime config file name (TUI/system state)
	RootConfigFileName     = ".crew.toml"           // Config file name in repository root
)

// RepoCrewDir returns the crew directory path for a repository.
func RepoCrewDir(repoRoot string) string {
	return filepath.Join(repoRoot, ".git", CrewDirName)
}

// RepoConfigPath returns the repo config path.
func RepoConfigPath(repoRoot string) string {
	return filepath.Join(RepoCrewDir(repoRoot), ConfigFileName)
}

// RepoRootConfigPath returns the repo root config path.
func RepoRootConfigPath(repoRoot string) string {
	return filepath.Join(repoRoot, RootConfigFileName)
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

// GlobalOverrideConfigPath returns the global override config path.
// configHome is typically XDG_CONFIG_HOME or ~/.config (resolved by caller).
func GlobalOverrideConfigPath(configHome string) string {
	return filepath.Join(GlobalCrewDir(configHome), ConfigOverrideFileName)
}

// Default configuration values for CompleteConfig.
const (
	DefaultAutoFixMaxRetries = 3
)

// NewDefaultConfig returns a Config with default values.
// This returns an empty Agents map.
// Builtin agents should be registered by calling builtin.Register(cfg)
// in the infra layer before merging user config.
func NewDefaultConfig() *Config {
	return &Config{
		Agents:       make(map[string]Agent),
		AgentsConfig: AgentsConfig{},
		Complete: CompleteConfig{
			AutoFixMaxRetries: DefaultAutoFixMaxRetries,
		},
		Log: LogConfig{
			Level: DefaultLogLevel,
		},
	}
}

// agentTemplateData holds data for a single agent in the template.
type agentTemplateData struct {
	Name        string
	Role        Role
	Args        string
	Description string
}

// templateData holds all data for rendering the config template.
type templateData struct {
	DefaultWorkerName     string
	DefaultManagerName    string
	FormattedSystemPrompt string
	LogLevel              string
	DefaultSystemPrompt   string
	Agents                []agentTemplateData
}

// RenderConfigTemplate renders a config template dynamically from the given Config.
// The generated template includes all registered agents as
// commented examples that users can uncomment and customize.
func RenderConfigTemplate(cfg *Config) string {
	// Prepare agent data (sorted for deterministic output)
	agentNames := sortedMapKeys(cfg.Agents)
	agents := make([]agentTemplateData, 0, len(agentNames))
	firstWorkerName := ""
	firstManagerName := ""
	for _, name := range agentNames {
		agent := cfg.Agents[name]
		if agent.Role == RoleWorker && firstWorkerName == "" {
			firstWorkerName = name
		}
		if agent.Role == RoleManager && firstManagerName == "" {
			firstManagerName = name
		}
		agents = append(agents, agentTemplateData{
			Name:        name,
			Role:        agent.Role,
			Args:        agent.Args,
			Description: agent.Description,
		})
	}

	data := templateData{
		DefaultWorkerName:     firstWorkerName,
		DefaultManagerName:    firstManagerName,
		DefaultSystemPrompt:   DefaultSystemPrompt,
		FormattedSystemPrompt: formatPromptForTemplate(DefaultSystemPrompt),
		Agents:                agents,
		LogLevel:              cfg.Log.Level,
	}

	tmpl, err := template.New("config").Delims("<<", ">>").Parse(configTemplateContent)
	if err != nil {
		// Should never happen with embedded template
		panic(fmt.Sprintf("failed to parse config template: %v", err))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Should never happen with valid data
		panic(fmt.Sprintf("failed to execute config template: %v", err))
	}

	return buf.String()
}

// sortedMapKeys returns the keys of a map sorted alphabetically.
func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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

// EnabledAgents returns a map of agents that are not disabled.
// It filters out agents that match any pattern in DisabledAgents.
func (c *Config) EnabledAgents() map[string]Agent {
	result := make(map[string]Agent)
	for name, agent := range c.Agents {
		if !IsAgentDisabled(name, c.AgentsConfig.DisabledAgents) {
			result[name] = agent
		}
	}
	return result
}

// GetReviewerAgents returns a sorted list of enabled reviewer agent names.
// It filters for agents with RoleReviewer, not disabled, and not hidden.
func (c *Config) GetReviewerAgents() []string {
	var reviewers []string
	for name, agent := range c.EnabledAgents() {
		if agent.Role == RoleReviewer && !agent.Hidden {
			reviewers = append(reviewers, name)
		}
	}
	sort.Strings(reviewers)
	return reviewers
}

// ResolveInheritance resolves agent inheritance by applying parent agent settings
// to child agents. It detects circular dependencies and returns an error if found.
// Agents without Inherit field are left unchanged.
// Only non-empty fields from the child override the parent fields.
func (c *Config) ResolveInheritance() error {
	// Track visited agents during traversal to detect circular dependencies
	visited := make(map[string]bool)
	resolving := make(map[string]bool)

	// Resolve each agent
	for name := range c.Agents {
		if err := c.resolveAgent(name, visited, resolving); err != nil {
			return err
		}
	}

	return nil
}

// resolveAgent recursively resolves inheritance for a single agent.
func (c *Config) resolveAgent(name string, visited, resolving map[string]bool) error {
	// Already resolved
	if visited[name] {
		return nil
	}

	// Circular dependency detected
	if resolving[name] {
		return ErrCircularInheritance
	}

	agent := c.Agents[name]

	// No inheritance, mark as resolved
	if agent.Inherit == "" {
		visited[name] = true
		return nil
	}

	// Mark as currently resolving
	resolving[name] = true

	// Check if parent exists
	_, exists := c.Agents[agent.Inherit]
	if !exists {
		delete(resolving, name)
		return ErrInheritParentNotFound
	}

	// Resolve parent first (recursive)
	if err := c.resolveAgent(agent.Inherit, visited, resolving); err != nil {
		delete(resolving, name)
		return err
	}

	// Get the resolved parent
	parent := c.Agents[agent.Inherit]

	// Apply inheritance: parent fields are used as defaults, child overrides if non-empty
	resolved := parent

	// Child overrides
	if agent.CommandTemplate != "" {
		resolved.CommandTemplate = agent.CommandTemplate
	}
	if agent.Role != "" {
		resolved.Role = agent.Role
	}
	if agent.SystemPrompt != "" {
		resolved.SystemPrompt = agent.SystemPrompt
	}
	if agent.Prompt != "" {
		resolved.Prompt = agent.Prompt
	}
	if agent.Args != "" {
		resolved.Args = agent.Args
	}
	if agent.DefaultModel != "" {
		resolved.DefaultModel = agent.DefaultModel
	}
	if agent.Description != "" {
		resolved.Description = agent.Description
	}
	if agent.SetupScript != "" {
		resolved.SetupScript = agent.SetupScript
	}
	// Hidden is a boolean, only override if explicitly set to true
	if agent.Hidden {
		resolved.Hidden = agent.Hidden
	}

	// Clear Inherit field after resolution
	resolved.Inherit = ""

	// Save resolved agent
	c.Agents[name] = resolved

	// Mark as resolved
	delete(resolving, name)
	visited[name] = true

	return nil
}
