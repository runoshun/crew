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
[workers]
default = "claude"

[workers.claude]
args = "--model claude-sonnet-4-20250514"
prompt = "When done, run 'crew complete'."

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
	assert.Equal(t, "claude", cfg.WorkersConfig.Default)
	assert.Equal(t, "--model claude-sonnet-4-20250514", cfg.Workers["claude"].Args)
	assert.Equal(t, "When done, run 'crew complete'.", cfg.Workers["claude"].Prompt)
	assert.Equal(t, "mise run ci", cfg.Complete.Command)
	assert.Equal(t, "debug", cfg.Log.Level)
}

func TestLoader_Load_GlobalConfigOnly(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config only
	globalConfig := `
[workers]
default = "opencode"

[workers.opencode]
args = "-m gpt-4"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "opencode", cfg.WorkersConfig.Default)
	assert.Equal(t, "-m gpt-4", cfg.Workers["opencode"].Args)
}

func TestLoader_Load_MergeRepoOverridesGlobal(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
[workers]
default = "opencode"

[workers.opencode]
args = "-m gpt-4"
prompt = "Global prompt"

[workers.claude]
args = "--model global-model"

[complete]
command = "go test ./..."

[log]
level = "info"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0644)
	require.NoError(t, err)

	// Write repo config (overrides some values)
	repoConfig := `
[workers]
default = "claude"

[workers.claude]
args = "--model repo-model"

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
	assert.Equal(t, "claude", cfg.WorkersConfig.Default)              // Overridden by repo
	assert.Equal(t, "Global prompt", cfg.Workers["opencode"].Prompt)  // From global (not overridden)
	assert.Equal(t, "--model repo-model", cfg.Workers["claude"].Args) // Overridden by repo
	assert.Equal(t, "-m gpt-4", cfg.Workers["opencode"].Args)         // From global
	assert.Equal(t, "mise run ci", cfg.Complete.Command)              // Overridden by repo
	assert.Equal(t, "info", cfg.Log.Level)                            // From global (not overridden)
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
	assert.Equal(t, domain.DefaultWorkerName, cfg.WorkersConfig.Default)
	assert.Equal(t, domain.DefaultLogLevel, cfg.Log.Level)
	// Default prompt comes from WorkersConfig, not individual workers
	assert.Equal(t, domain.DefaultWorkerPrompt, cfg.WorkersConfig.Prompt)
	// Builtin workers should have values from BuiltinWorkers
	for name, builtin := range domain.BuiltinWorkers {
		worker := cfg.Workers[name]
		assert.Equal(t, builtin.CommandTemplate, worker.CommandTemplate)
		assert.Equal(t, builtin.Command, worker.Command)
		assert.Equal(t, builtin.SystemArgs, worker.SystemArgs)
		assert.Equal(t, builtin.DefaultArgs, worker.Args)
		// Worker.Prompt is empty; falls back to WorkersConfig.Prompt
		assert.Empty(t, worker.Prompt)
	}
}

func TestLoader_LoadGlobal(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
[workers]
default = "opencode"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0644)
	require.NoError(t, err)

	// Load global config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.LoadGlobal()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "opencode", cfg.WorkersConfig.Default)
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
