package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_Load_RepoConfigOnly(t *testing.T) {
	// Setup: create temp directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config
	repoConfig := `
[agents.claude]
args = "--model claude-sonnet-4-20250514"
prompt = "When done, run 'crew complete'."
description = "Custom Claude description"

[complete]
command = "mise run ci"

[log]
level = "debug"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "--model claude-sonnet-4-20250514", cfg.Agents["claude"].Args)
	assert.Equal(t, "When done, run 'crew complete'.", cfg.Agents["claude"].Prompt)
	assert.Equal(t, "Custom Claude description", cfg.Agents["claude"].Description)
	assert.Equal(t, "mise run ci", cfg.Complete.Command)
	assert.Equal(t, "debug", cfg.Log.Level)
}

func TestLoader_Load_GlobalConfigOnly(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config only
	globalConfig := `
[agents.opencode]
args = "-m gpt-4"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "-m gpt-4", cfg.Agents["opencode"].Args)
}

func TestLoader_Load_MergeRepoOverridesGlobal(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
[agents.opencode]
args = "-m gpt-4"
prompt = "Global prompt"

[agents.claude]
args = "--model global-model"

[complete]
command = "go test ./..."

[log]
level = "info"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// Write repo config (overrides some values)
	repoConfig := `
[agents.claude]
args = "--model repo-model"

[complete]
command = "mise run ci"
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify: repo overrides global
	assert.Equal(t, "Global prompt", cfg.Agents["opencode"].Prompt)  // From global (not overridden)
	assert.Equal(t, "--model repo-model", cfg.Agents["claude"].Args) // Overridden by repo
	assert.Equal(t, "-m gpt-4", cfg.Agents["opencode"].Args)         // From global
	assert.Equal(t, "mise run ci", cfg.Complete.Command)             // Overridden by repo
	assert.Equal(t, "info", cfg.Log.Level)                           // From global (not overridden)
}

func TestLoader_Load_NoConfigFiles(t *testing.T) {
	// Setup: empty directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Get expected config by creating default and registering builtins
	expectedCfg := domain.NewDefaultConfig()
	Register(expectedCfg)

	// Verify: default config is returned (values from domain constants)
	assert.Equal(t, domain.DefaultLogLevel, cfg.Log.Level)

	// Builtin agents should be registered (derived from expectedCfg)
	for name, expectedAgent := range expectedCfg.Agents {
		actualAgent, exists := cfg.Agents[name]
		assert.True(t, exists, "builtin agent %s should exist", name)
		assert.Equal(t, expectedAgent.CommandTemplate, actualAgent.CommandTemplate, "agent %s CommandTemplate mismatch", name)
	}
}

func TestLoader_LoadGlobal(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
[agents.opencode]
args = "-m custom"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// Load global config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.LoadGlobal()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "-m custom", cfg.Agents["opencode"].Args)
}

func TestLoader_LoadGlobal_NotFound(t *testing.T) {
	// Setup: empty directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Load global config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.LoadGlobal()

	// Verify: returns error for non-existent file
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Nil(t, cfg)
}

func TestLoader_Load_InvalidTOML(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write invalid TOML
	invalidConfig := `
this is not valid toml [[[
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(invalidConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()

	// Verify: returns error
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoader_Load_CustomAgentCommand(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write config with custom agent
	config := `
[agents.my-worker]
command_template = 'my-custom-agent --task "{{.Title}}"'
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, `my-custom-agent --task "{{.Title}}"`, cfg.Agents["my-worker"].CommandTemplate)
}

func TestLoader_LoadRepo(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config
	repoConfig := `
[agents.claude]
args = "--model claude-sonnet"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load repo config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.LoadRepo()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "--model claude-sonnet", cfg.Agents["claude"].Args)
}

func TestLoader_LoadRepo_NotFound(t *testing.T) {
	// Setup: empty directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Load repo config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.LoadRepo()

	// Verify: returns error for non-existent file
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Nil(t, cfg)
}

func TestLoader_LoadWithOptions_IgnoreGlobal(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
[agents.opencode]
args = "-m gpt-4"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// Write repo config
	repoConfig := `
[agents.claude]
args = "--model repo"
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load with IgnoreGlobal
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.LoadWithOptions(domain.LoadConfigOptions{IgnoreGlobal: true})
	require.NoError(t, err)

	// Verify: only repo config is used
	assert.Equal(t, "--model repo", cfg.Agents["claude"].Args)
	// opencode agent should only have builtin defaults, not the global config args
	defaultCfg := domain.NewDefaultConfig()
	Register(defaultCfg)
	assert.Equal(t, defaultCfg.Agents["opencode"].Args, cfg.Agents["opencode"].Args)
}

func TestLoader_LoadWithOptions_IgnoreRepo(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
[agents.opencode]
args = "-m gpt-4"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// Write repo config
	repoConfig := `
[agents.claude]
args = "--model repo-model"
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load with IgnoreRepo
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.LoadWithOptions(domain.LoadConfigOptions{IgnoreRepo: true})
	require.NoError(t, err)

	// Verify: only global config is used
	assert.Equal(t, "-m gpt-4", cfg.Agents["opencode"].Args)
	// claude agent should only have builtin defaults
	defaultCfg := domain.NewDefaultConfig()
	Register(defaultCfg)
	assert.Equal(t, defaultCfg.Agents["claude"].Args, cfg.Agents["claude"].Args)
}

func TestLoader_LoadWithOptions_IgnoreBoth(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write both configs
	globalConfig := `
[agents.opencode]
args = "-m global"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	repoConfig := `
[agents.claude]
args = "--model repo"
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load with both ignored
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.LoadWithOptions(domain.LoadConfigOptions{IgnoreGlobal: true, IgnoreRepo: true})
	require.NoError(t, err)

	// Get expected config by creating default and registering builtins
	expectedCfg := domain.NewDefaultConfig()
	Register(expectedCfg)

	// Builtin agents should exist (derived from expectedCfg)
	for name := range expectedCfg.Agents {
		_, exists := cfg.Agents[name]
		assert.True(t, exists, "builtin agent %s should exist", name)
	}
}

func TestLoader_Load_UnknownKeys(t *testing.T) {
	// Setup: create temp directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write config with unknown keys
	config := `
[unknown_section]
key = "value"

[agents]
unknown_agents_key = "value"

[agents.claude]
args = "--model claude-sonnet"
unknown_agent_key = "value"

[complete]
command = "mise run ci"
unknown_complete_key = "value"

[tasks]
unknown_tasks_key = "value"

[diff]
unknown_diff_key = "value"

[log]
unknown_log_key = "value"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify warnings
	expected := []string{
		"unknown key in [agents.claude]: unknown_agent_key",
		"unknown key in [agents]: unknown_agents_key",
		"unknown key in [complete]: unknown_complete_key",
		"unknown key in [diff]: unknown_diff_key",
		"unknown key in [log]: unknown_log_key",
		"unknown key in [tasks]: unknown_tasks_key",
		"unknown section: unknown_section",
	}
	assert.Equal(t, expected, cfg.Warnings)
}

func TestLoader_Load_InvalidReviewMode(t *testing.T) {
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	config := `
[complete]
review_mode = "invalid"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0o644)
	require.NoError(t, err)

	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	assert.False(t, cfg.Complete.ReviewModeSet)
	assert.Contains(t, cfg.Warnings, "invalid value for complete.review_mode: \"invalid\" (expected \"auto\", \"manual\", or \"auto_fix\")")
}

func TestLoader_Load_WorktreeConfig(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config with worktree section
	repoConfig := `
[worktree]
setup_command = "mise install && npm install"
copy = ["node_modules", ".env.local"]
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify worktree config
	assert.Equal(t, "mise install && npm install", cfg.Worktree.SetupCommand)
	assert.Equal(t, []string{"node_modules", ".env.local"}, cfg.Worktree.Copy)
}

func TestLoader_Load_WorktreeConfig_Empty(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config without worktree section
	repoConfig := `
[agents.claude]
args = "--model test"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify empty worktree config
	assert.Equal(t, "", cfg.Worktree.SetupCommand)
	assert.Empty(t, cfg.Worktree.Copy)
}

func TestLoader_Load_Priority(t *testing.T) {
	// Setup
	repoRootDir := t.TempDir()
	crewDir := filepath.Join(repoRootDir, ".git", "crew")
	err := os.MkdirAll(crewDir, 0o755)
	require.NoError(t, err)
	globalDir := t.TempDir()

	// 1. Global config
	globalConfig := `
[log]
level = "debug"
[agents.opencode]
args = "global"
`
	err = os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// 2. Root repo config (.crew.toml)
	rootRepoConfig := `
[log]
level = "info"
[agents.opencode]
args = "root"
[agents.claude]
args = "root"
`
	err = os.WriteFile(filepath.Join(repoRootDir, domain.RootConfigFileName), []byte(rootRepoConfig), 0o644)
	require.NoError(t, err)

	// 3. Repo config (.git/crew/config.toml)
	repoConfig := `
[agents.claude]
args = "repo"
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, repoRootDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify priority: repo > rootRepo > global
	assert.Equal(t, "info", cfg.Log.Level)               // rootRepo overrides global
	assert.Equal(t, "root", cfg.Agents["opencode"].Args) // rootRepo overrides global
	assert.Equal(t, "repo", cfg.Agents["claude"].Args)   // repo overrides rootRepo
}

func TestLoader_LoadWithOptions_IgnoreRootRepo(t *testing.T) {
	// Setup
	repoRootDir := t.TempDir()
	crewDir := filepath.Join(repoRootDir, ".git", "crew")
	err := os.MkdirAll(crewDir, 0o755)
	require.NoError(t, err)
	globalDir := t.TempDir()

	// Write root repo config
	rootRepoConfig := `
[agents.claude]
args = "root"
`
	err = os.WriteFile(filepath.Join(repoRootDir, domain.RootConfigFileName), []byte(rootRepoConfig), 0o644)
	require.NoError(t, err)

	// Load with IgnoreRootRepo
	loader := NewLoaderWithGlobalDir(crewDir, repoRootDir, globalDir)
	cfg, err := loader.LoadWithOptions(domain.LoadConfigOptions{IgnoreRootRepo: true})
	require.NoError(t, err)

	// Verify: root repo config is ignored
	defaultCfg := domain.NewDefaultConfig()
	Register(defaultCfg)
	assert.Equal(t, defaultCfg.Agents["claude"].Args, cfg.Agents["claude"].Args)
}

func TestLoader_Load_AgentsConfig_ReviewerPromptAndDefault(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write config with reviewer_prompt and reviewer_default
	config := `
[agents]
reviewer_prompt = "Custom reviewer prompt"
reviewer_default = "custom-reviewer"

[agents.custom-reviewer]
command_template = "my-reviewer {{.Prompt}}"
role = "reviewer"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "Custom reviewer prompt", cfg.AgentsConfig.ReviewerPrompt)
	assert.Equal(t, "custom-reviewer", cfg.AgentsConfig.DefaultReviewer)
	assert.Equal(t, "my-reviewer {{.Prompt}}", cfg.Agents["custom-reviewer"].CommandTemplate)
	assert.Equal(t, domain.RoleReviewer, cfg.Agents["custom-reviewer"].Role)
}

func TestLoader_Load_ReviewerDefault(t *testing.T) {
	// Setup: create temp directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config with reviewer_default and reviewer_prompt
	repoConfig := `
[agents]
reviewer_default = "claude-reviewer"
reviewer_prompt = "Please review this code."

[agents.claude-reviewer]
role = "reviewer"
args = "--model claude-opus-4"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify reviewer config
	assert.Equal(t, "claude-reviewer", cfg.AgentsConfig.DefaultReviewer)
	assert.Equal(t, "Please review this code.", cfg.AgentsConfig.ReviewerPrompt)
	assert.Len(t, cfg.Warnings, 0, "should have no warnings for valid config")
}

func TestLoader_Load_AgentsConfig_Merge(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config with all prompts
	globalConfig := `
[agents]
worker_prompt = "Global worker prompt"
manager_prompt = "Global manager prompt"
reviewer_prompt = "Global reviewer prompt"
worker_default = "global-worker"
manager_default = "global-manager"
reviewer_default = "global-reviewer"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// Write repo config that overrides some values
	repoConfig := `
[agents]
worker_prompt = "Repo worker prompt"
reviewer_default = "repo-reviewer"
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify merging: repo overrides global
	assert.Equal(t, "Repo worker prompt", cfg.AgentsConfig.WorkerPrompt)       // Overridden
	assert.Equal(t, "Global manager prompt", cfg.AgentsConfig.ManagerPrompt)   // From global
	assert.Equal(t, "Global reviewer prompt", cfg.AgentsConfig.ReviewerPrompt) // From global
	assert.Equal(t, "global-worker", cfg.AgentsConfig.DefaultWorker)           // From global
	assert.Equal(t, "global-manager", cfg.AgentsConfig.DefaultManager)         // From global
	assert.Equal(t, "repo-reviewer", cfg.AgentsConfig.DefaultReviewer)         // Overridden
}

func TestLoader_Load_OverrideConfig(t *testing.T) {
	// Setup: create temp directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
[agents]
worker_default = "opencode"

[agents.opencode]
args = "--verbose"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// Write override config
	overrideConfig := `
[agents]
worker_default = "claude"
disabled_agents = ["opencode", "codex"]
`
	err = os.WriteFile(filepath.Join(globalDir, domain.ConfigOverrideFileName), []byte(overrideConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify override takes precedence
	assert.Equal(t, "claude", cfg.AgentsConfig.DefaultWorker)
	assert.Equal(t, []string{"opencode", "codex"}, cfg.AgentsConfig.DisabledAgents)
}

func TestLoader_Load_DisabledAgents(t *testing.T) {
	// Setup: create temp directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config with disabled_agents
	repoConfig := `
[agents]
disabled_agents = ["agent1", "agent2"]
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify disabled_agents
	assert.Equal(t, []string{"agent1", "agent2"}, cfg.AgentsConfig.DisabledAgents)
}

func TestLoader_Load_IgnoreOverride(t *testing.T) {
	// Setup: create temp directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
[agents]
worker_default = "opencode"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// Write override config
	overrideConfig := `
[agents]
worker_default = "claude"
`
	err = os.WriteFile(filepath.Join(globalDir, domain.ConfigOverrideFileName), []byte(overrideConfig), 0o644)
	require.NoError(t, err)

	// Load config with IgnoreOverride option
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.LoadWithOptions(domain.LoadConfigOptions{IgnoreOverride: true})
	require.NoError(t, err)

	// Verify override was ignored, global config is used
	assert.Equal(t, "opencode", cfg.AgentsConfig.DefaultWorker)
}

func TestLoader_Load_OnboardingDone(t *testing.T) {
	// Setup: create temp directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config with onboarding_done
	repoConfig := `
onboarding_done = true
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify onboarding_done is true
	assert.True(t, cfg.OnboardingDone)
}

func TestLoader_Load_OnboardingDone_DefaultFalse(t *testing.T) {
	// Setup: create temp directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config without onboarding_done
	repoConfig := `
[log]
level = "debug"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify onboarding_done defaults to false
	assert.False(t, cfg.OnboardingDone)
}

func TestLoader_Load_NewTaskBase_Valid(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"current", "current", "current"},
		{"default", "default", "default"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crewDir := t.TempDir()
			globalDir := t.TempDir()

			config := `[tasks]
`
			if tt.value != "" {
				config += `new_task_base = "` + tt.value + `"
`
			}
			err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0o644)
			require.NoError(t, err)

			loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
			cfg, err := loader.Load()
			require.NoError(t, err)

			assert.Equal(t, tt.expected, cfg.Tasks.NewTaskBase)
			assert.Empty(t, cfg.Warnings, "valid value should not produce warnings")
		})
	}
}

func TestLoader_Load_NewTaskBase_Invalid(t *testing.T) {
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	config := `[tasks]
new_task_base = "invalid_value"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0o644)
	require.NoError(t, err)

	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Value is still set (for debugging purposes)
	assert.Equal(t, "invalid_value", cfg.Tasks.NewTaskBase)

	// Warning should be generated
	require.Len(t, cfg.Warnings, 1)
	assert.Contains(t, cfg.Warnings[0], "invalid value for tasks.new_task_base")
	assert.Contains(t, cfg.Warnings[0], "invalid_value")
}

func TestLoader_Load_NewTaskBase_Merge(t *testing.T) {
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Global config sets "current"
	globalConfig := `[tasks]
new_task_base = "current"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// Repo config overrides to "default"
	repoConfig := `[tasks]
new_task_base = "default"
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Repo should override global
	assert.Equal(t, "default", cfg.Tasks.NewTaskBase)
	assert.Empty(t, cfg.Warnings)
}

func TestLoader_Load_CompleteAutoFix(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write config with [complete] section containing auto_fix
	config := `
[complete]
command = "mise run ci"
auto_fix = true
auto_fix_max_retries = 5
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify complete config
	assert.Equal(t, "mise run ci", cfg.Complete.Command)
	assert.True(t, cfg.Complete.AutoFix) //nolint:staticcheck // Testing legacy field
	assert.Equal(t, 5, cfg.Complete.AutoFixMaxRetries)
}

func TestLoader_Load_CompleteAutoFix_Merge(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config with auto_fix false
	globalConfig := `
[complete]
command = "global-cmd"
auto_fix = false
auto_fix_max_retries = 3
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// Write repo config that overrides auto_fix
	repoConfig := `
[complete]
auto_fix = true
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify merging: repo overrides global
	assert.Equal(t, "global-cmd", cfg.Complete.Command) // From global
	assert.True(t, cfg.Complete.AutoFix)                //nolint:staticcheck // Testing legacy field - Overridden by repo
	assert.Equal(t, 3, cfg.Complete.AutoFixMaxRetries)  // From global
}

func TestLoader_Load_CompleteAutoFix_ExplicitFalseOverridesTrue(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config with auto_fix = true
	globalConfig := `
[complete]
command = "global-cmd"
auto_fix = true
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// Write repo config that explicitly sets auto_fix = false
	repoConfig := `
[complete]
auto_fix = false
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify: explicit false in repo should override true from global
	assert.Equal(t, "global-cmd", cfg.Complete.Command) // From global
	assert.False(t, cfg.Complete.AutoFix)               //nolint:staticcheck // Testing legacy field - Overridden by repo (explicit false)
}

func TestLoader_Load_RuntimeConfig(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config
	repoConfig := `
[complete]
command = "mise run ci"
auto_fix = false
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Write runtime config (should override repo config)
	runtimeConfig := `
[complete]
auto_fix = true
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName), []byte(runtimeConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify: runtime config overrides repo config
	assert.Equal(t, "mise run ci", cfg.Complete.Command) // From repo (not overridden)
	assert.True(t, cfg.Complete.AutoFix)                 //nolint:staticcheck // Testing legacy field - Overridden by runtime
}

func TestLoader_Load_RuntimeConfig_Priority(t *testing.T) {
	// Setup
	repoRootDir := t.TempDir()
	crewDir := filepath.Join(repoRootDir, ".git", "crew")
	err := os.MkdirAll(crewDir, 0o755)
	require.NoError(t, err)
	globalDir := t.TempDir()

	// 1. Global config
	globalConfig := `
[complete]
auto_fix = false
auto_fix_max_retries = 1
`
	err = os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
	require.NoError(t, err)

	// 2. Repo config
	repoConfig := `
[complete]
auto_fix_max_retries = 2
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// 3. Runtime config (highest priority)
	runtimeConfig := `
[complete]
auto_fix = true
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName), []byte(runtimeConfig), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, repoRootDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify priority: runtime > repo > global
	assert.True(t, cfg.Complete.AutoFix)               //nolint:staticcheck // Testing legacy field - Overridden by runtime
	assert.Equal(t, 2, cfg.Complete.AutoFixMaxRetries) // From repo
}

func TestLoader_LoadRuntime(t *testing.T) {
	t.Run("returns runtime config when file exists", func(t *testing.T) {
		crewDir := t.TempDir()
		globalDir := t.TempDir()

		// Write runtime config
		runtimeConfig := `
[complete]
auto_fix = true
`
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName), []byte(runtimeConfig), 0o644)
		require.NoError(t, err)

		// Load runtime config
		loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
		cfg, err := loader.LoadRuntime()
		require.NoError(t, err)

		// Verify
		assert.True(t, cfg.Complete.AutoFix) //nolint:staticcheck // Testing legacy field
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		crewDir := t.TempDir()
		globalDir := t.TempDir()

		// Load runtime config (file doesn't exist)
		loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
		cfg, err := loader.LoadRuntime()

		// Verify: returns error for non-existent file
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Nil(t, cfg)
	})
}

func TestLoader_LoadWithOptions_IgnoreRuntime(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config
	repoConfig := `
[complete]
auto_fix = false
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
	require.NoError(t, err)

	// Write runtime config
	runtimeConfig := `
[complete]
auto_fix = true
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName), []byte(runtimeConfig), 0o644)
	require.NoError(t, err)

	// Load with IgnoreRuntime
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.LoadWithOptions(domain.LoadConfigOptions{IgnoreRuntime: true})
	require.NoError(t, err)

	// Verify: runtime config is ignored, repo config is used
	assert.False(t, cfg.Complete.AutoFix) //nolint:staticcheck // Testing legacy field - From repo, runtime ignored
}

func TestLoader_Load_TUIKeybindings(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write config with TUI keybindings including worktree flag
	config := `
[tui.keybindings]
"g" = { command = "git log --oneline -10", description = "repo git log" }
"s" = { command = "git status", description = "worktree status", worktree = true }
"o" = { command = "echo override", description = "override test", override = true }
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0o644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify keybindings
	require.Len(t, cfg.TUI.Keybindings, 3)

	// "g" - default worktree = false
	gBinding := cfg.TUI.Keybindings["g"]
	assert.Equal(t, "git log --oneline -10", gBinding.Command)
	assert.Equal(t, "repo git log", gBinding.Description)
	assert.False(t, gBinding.Worktree) // Default value

	// "s" - worktree = true
	sBinding := cfg.TUI.Keybindings["s"]
	assert.Equal(t, "git status", sBinding.Command)
	assert.Equal(t, "worktree status", sBinding.Description)
	assert.True(t, sBinding.Worktree)

	// "o" - override = true
	oBinding := cfg.TUI.Keybindings["o"]
	assert.Equal(t, "echo override", oBinding.Command)
	assert.True(t, oBinding.Override)
	assert.False(t, oBinding.Worktree) // Default value
}

func TestLoader_Load_AgentEnvMerge(t *testing.T) {
	t.Run("adds env to builtin agent", func(t *testing.T) {
		// Setup
		crewDir := t.TempDir()
		globalDir := t.TempDir()

		// Write config that adds env to builtin opencode agent
		config := `
[agents.opencode]
env = { ANTHROPIC_API_KEY = "sk-test-key" }
`
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0o644)
		require.NoError(t, err)

		// Load config
		loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
		cfg, err := loader.Load()
		require.NoError(t, err)

		// Verify env is merged
		agent := cfg.Agents["opencode"]
		assert.Equal(t, "sk-test-key", agent.Env["ANTHROPIC_API_KEY"])
		// Builtin fields should still be present
		assert.NotEmpty(t, agent.CommandTemplate)
	})

	t.Run("merges env from global and repo configs", func(t *testing.T) {
		// Setup
		crewDir := t.TempDir()
		globalDir := t.TempDir()

		// Global config sets some env vars
		globalConfig := `
[agents.claude]
env = { GLOBAL_VAR = "global-value", SHARED_VAR = "global-shared" }
`
		err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0o644)
		require.NoError(t, err)

		// Repo config overrides one and adds another
		repoConfig := `
[agents.claude]
env = { REPO_VAR = "repo-value", SHARED_VAR = "repo-shared" }
`
		err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0o644)
		require.NoError(t, err)

		// Load config
		loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
		cfg, err := loader.Load()
		require.NoError(t, err)

		// Verify env is merged correctly
		agent := cfg.Agents["claude"]
		assert.Equal(t, "global-value", agent.Env["GLOBAL_VAR"]) // From global
		assert.Equal(t, "repo-value", agent.Env["REPO_VAR"])     // From repo
		assert.Equal(t, "repo-shared", agent.Env["SHARED_VAR"])  // Repo overrides global
	})

	t.Run("preserves builtin env when adding user env", func(t *testing.T) {
		// Setup
		crewDir := t.TempDir()
		globalDir := t.TempDir()

		// Write config that adds env to builtin claude agent (which may have builtin env)
		config := `
[agents.claude]
env = { USER_VAR = "user-value" }
`
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0o644)
		require.NoError(t, err)

		// Load config
		loader := NewLoaderWithGlobalDir(crewDir, "", globalDir)
		cfg, err := loader.Load()
		require.NoError(t, err)

		// Verify user env is added
		agent := cfg.Agents["claude"]
		assert.Equal(t, "user-value", agent.Env["USER_VAR"])
		// Builtin fields should still be present
		assert.NotEmpty(t, agent.CommandTemplate)
	})
}
