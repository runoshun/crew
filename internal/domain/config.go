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
type Config struct {
	Agents       map[string]Agent // Agent definitions from [agents.<name>]
	AgentsConfig AgentsConfig     // Common [agents] settings
	Complete     CompleteConfig   // [complete] settings
	Diff         DiffConfig       // [diff] settings
	Log          LogConfig        // [log] settings
	Tasks        TasksConfig      // [tasks] settings
	Worktree     WorktreeConfig   // [worktree] settings
	Warnings     []string         // [warning] Unknown keys or other issues
}

// TasksConfig holds settings for task storage from [tasks] section.
type TasksConfig struct {
	Store     string // Storage backend: "git" (default) or "json"
	Namespace string // Git namespace for refs (default: "crew")
	Encrypt   bool   // Enable encryption for task data (default: false)
}

// AgentsConfig holds common settings for all agents from [agents] section.
type AgentsConfig struct {
	DefaultWorker  string // Default worker agent name
	DefaultManager string // Default manager agent name
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
	Inherit string // Name of agent to inherit from (optional)

	// Command execution
	CommandTemplate string // Full command template (e.g., "opencode -m {{.Model}} {{.Args}} --prompt {{.Prompt}}")

	// Role configuration
	Role         Role   // Role: worker, reviewer, manager
	SystemPrompt string // System prompt template
	Prompt       string // User prompt template
	Args         string // Additional arguments

	// Agent metadata
	DefaultModel string // Default model for this agent
	Description  string // Description of the agent's purpose

	// Worktree setup (for workers/reviewers)
	SetupScript string // Setup script (replaces worktree_setup_script and exclude_patterns)

	// Visibility
	Hidden bool // Hide from TUI agent list
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

	userPromptTemplate := a.Prompt
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
	DefaultLogLevel = "info"
)

// DefaultSystemPrompt is the default system prompt template for workers.
// It uses Go template syntax with CommandData fields.
const DefaultSystemPrompt = `You are working on Task #{{.TaskID}}.

IMPORTANT: First run 'crew --help-worker' and follow the workflow instructions exactly.`

// DefaultManagerSystemPrompt is the default system prompt template for managers.
const DefaultManagerSystemPrompt = `You are a Manager agent for git-crew.

IMPORTANT: Run 'crew --help-manager' for detailed usage instructions.

Your role is to:
- Support users with task management
- Create, monitor, and manage tasks using crew commands
- Delegate code implementation to worker agents (do not edit files directly)`

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
// This returns an empty Agents map.
// Builtin agents should be registered by calling builtin.Register(cfg)
// in the infra layer before merging user config.
func NewDefaultConfig() *Config {
	return &Config{
		Agents:       make(map[string]Agent),
		AgentsConfig: AgentsConfig{},
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
	for _, name := range agentNames {
		agent := cfg.Agents[name]
		if agent.Role == RoleWorker && firstWorkerName == "" {
			firstWorkerName = name
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
