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
[workers.claude]
args = "--model claude-sonnet-4-20250514"
model = "anthropic/claude-sonnet-4-20250514"
prompt = "When done, run 'crew complete'."
description = "Custom Claude description"

[complete]
command = "mise run ci"

[log]
level = "debug"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "--model claude-sonnet-4-20250514", cfg.Workers["claude"].Args)
	assert.Equal(t, "anthropic/claude-sonnet-4-20250514", cfg.Workers["claude"].Model)
	assert.Equal(t, "When done, run 'crew complete'.", cfg.Workers["claude"].Prompt)
	assert.Equal(t, "Custom Claude description", cfg.Workers["claude"].Description)
	assert.Equal(t, "mise run ci", cfg.Complete.Command)
	assert.Equal(t, "debug", cfg.Log.Level)
}

func TestLoader_Load_GlobalConfigOnly(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config only
	globalConfig := `
[workers.opencode]
args = "-m gpt-4"
model = "github-copilot/gpt-4"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "-m gpt-4", cfg.Workers["opencode"].Args)
	assert.Equal(t, "github-copilot/gpt-4", cfg.Workers["opencode"].Model)
}

func TestLoader_Load_MergeRepoOverridesGlobal(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
[workers.opencode]
args = "-m gpt-4"
model = "global-opencode-model"
prompt = "Global prompt"

[workers.claude]
args = "--model global-model"
model = "global-claude-model"

[complete]
command = "go test ./..."

[log]
level = "info"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0644)
	require.NoError(t, err)

	// Write repo config (overrides some values)
	repoConfig := `
[workers.claude]
args = "--model repo-model"
model = "repo-claude-model"

[complete]
command = "mise run ci"
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify: repo overrides global
	assert.Equal(t, "Global prompt", cfg.Workers["opencode"].Prompt)        // From global (not overridden)
	assert.Equal(t, "--model repo-model", cfg.Workers["claude"].Args)       // Overridden by repo
	assert.Equal(t, "repo-claude-model", cfg.Workers["claude"].Model)       // Overridden by repo
	assert.Equal(t, "-m gpt-4", cfg.Workers["opencode"].Args)               // From global
	assert.Equal(t, "global-opencode-model", cfg.Workers["opencode"].Model) // From global
	assert.Equal(t, "mise run ci", cfg.Complete.Command)                    // Overridden by repo
	assert.Equal(t, "info", cfg.Log.Level)                                  // From global (not overridden)
}

func TestLoader_Load_NoConfigFiles(t *testing.T) {
	// Setup: empty directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify: default config is returned (values from domain constants)
	assert.Equal(t, domain.DefaultLogLevel, cfg.Log.Level)
	// Default system prompt comes from WorkersConfig
	assert.Equal(t, domain.DefaultSystemPrompt, cfg.WorkersConfig.SystemPrompt)
	// Default prompt comes from WorkersConfig, which should be empty by default
	assert.Empty(t, cfg.WorkersConfig.Prompt)
	// Default worker should exist
	defaultWorker, ok := cfg.Workers[domain.DefaultWorkerName]
	assert.True(t, ok, "default worker should exist")
	assert.Equal(t, domain.DefaultWorkerAgent, defaultWorker.Agent)
	// Builtin workers should have values from BuiltinWorkers
	for name, builtin := range domain.BuiltinAgents {
		worker := cfg.Workers[name]
		assert.Equal(t, builtin.CommandTemplate, worker.CommandTemplate)
		assert.Equal(t, builtin.Command, worker.Command)
		assert.Equal(t, builtin.SystemArgs, worker.SystemArgs)
		assert.Equal(t, builtin.DefaultArgs, worker.Args)
		// Worker.SystemPrompt and Worker.Prompt are empty; falls back to WorkersConfig
		assert.Empty(t, worker.SystemPrompt)
		assert.Empty(t, worker.Prompt)
	}
}

func TestLoader_LoadGlobal(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
[workers.opencode]
args = "-m custom"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0644)
	require.NoError(t, err)

	// Load global config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.LoadGlobal()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "-m custom", cfg.Workers["opencode"].Args)
}

func TestLoader_LoadGlobal_NotFound(t *testing.T) {
	// Setup: empty directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Load global config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
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
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(invalidConfig), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()

	// Verify: returns error
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoader_Load_CustomWorkerCommand(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write config with custom worker
	config := `
[workers.my-worker]
command = 'my-custom-agent --task "{{.Title}}"'
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, `my-custom-agent --task "{{.Title}}"`, cfg.Workers["my-worker"].Command)
}

func TestLoader_LoadRepo(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config
	repoConfig := `
[workers.claude]
args = "--model claude-sonnet"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0644)
	require.NoError(t, err)

	// Load repo config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.LoadRepo()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "--model claude-sonnet", cfg.Workers["claude"].Args)
}

func TestLoader_LoadRepo_NotFound(t *testing.T) {
	// Setup: empty directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Load repo config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
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
[workers.opencode]
args = "-m gpt-4"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0644)
	require.NoError(t, err)

	// Write repo config
	repoConfig := `
[workers.claude]
args = "--model repo"
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0644)
	require.NoError(t, err)

	// Load with IgnoreGlobal
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.LoadWithOptions(domain.LoadConfigOptions{IgnoreGlobal: true})
	require.NoError(t, err)

	// Verify: only repo config is used
	assert.Equal(t, "--model repo", cfg.Workers["claude"].Args)
	// opencode worker should only have builtin defaults, not the global config args
	assert.Equal(t, domain.BuiltinAgents["opencode"].DefaultArgs, cfg.Workers["opencode"].Args)
}

func TestLoader_LoadWithOptions_IgnoreRepo(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
[workers.opencode]
args = "-m gpt-4"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0644)
	require.NoError(t, err)

	// Write repo config
	repoConfig := `
[workers.claude]
args = "--model repo-model"
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0644)
	require.NoError(t, err)

	// Load with IgnoreRepo
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.LoadWithOptions(domain.LoadConfigOptions{IgnoreRepo: true})
	require.NoError(t, err)

	// Verify: only global config is used
	assert.Equal(t, "-m gpt-4", cfg.Workers["opencode"].Args)
	// claude worker should only have builtin defaults
	assert.Equal(t, domain.BuiltinAgents["claude"].DefaultArgs, cfg.Workers["claude"].Args)
}

func TestLoader_LoadWithOptions_IgnoreBoth(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write both configs
	globalConfig := `
[workers.opencode]
args = "-m global"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0644)
	require.NoError(t, err)

	repoConfig := `
[workers.claude]
args = "--model repo"
`
	err = os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0644)
	require.NoError(t, err)

	// Load with both ignored
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.LoadWithOptions(domain.LoadConfigOptions{IgnoreGlobal: true, IgnoreRepo: true})
	require.NoError(t, err)

	// Verify: only defaults are used - default worker should exist
	_, ok := cfg.Workers[domain.DefaultWorkerName]
	assert.True(t, ok, "default worker should exist")
}

func TestLoader_Load_UnknownKeys(t *testing.T) {
	// Setup: create temp directories
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write config with unknown keys
	config := `
[unknown_section]
key = "value"

[workers]
unknown_workers_key = "value"

[workers.claude]
args = "--model claude-sonnet"
unknown_worker_key = "value"

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
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify warnings
	expected := []string{
		"unknown key in [complete]: unknown_complete_key",
		"unknown key in [diff]: unknown_diff_key",
		"unknown key in [log]: unknown_log_key",
		"unknown key in [tasks]: unknown_tasks_key",
		"unknown key in [workers.claude]: unknown_worker_key",
		"unknown key in [workers]: unknown_workers_key",
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
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
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
[workers.claude]
args = "--model test"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify empty worktree config
	assert.Equal(t, "", cfg.Worktree.SetupCommand)
	assert.Empty(t, cfg.Worktree.Copy)
}

func TestLoader_Load_ManagersConfig(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write repo config with managers section
	repoConfig := `
[managers]
system_prompt = "Manager system prompt"
prompt = "Manager user prompt"

[managers.custom]
agent = "claude"
model = "opus"
args = "--verbose"
system_prompt = "Custom system prompt"
prompt = "Custom user prompt"
description = "Custom manager"
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(repoConfig), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify managers config
	assert.Equal(t, "Manager system prompt", cfg.ManagersConfig.SystemPrompt)
	assert.Equal(t, "Manager user prompt", cfg.ManagersConfig.Prompt)

	// Verify custom manager
	manager, ok := cfg.Managers["custom"]
	require.True(t, ok)
	assert.Equal(t, "claude", manager.Agent)
	assert.Equal(t, "opus", manager.Model)
	assert.Equal(t, "--verbose", manager.Args)
	assert.Equal(t, "Custom system prompt", manager.SystemPrompt)
	assert.Equal(t, "Custom user prompt", manager.Prompt)
	assert.Equal(t, "Custom manager", manager.Description)
}
