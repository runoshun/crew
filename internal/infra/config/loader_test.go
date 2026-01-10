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
