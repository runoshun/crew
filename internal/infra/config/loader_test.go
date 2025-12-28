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
default_agent = "claude"

[agent]
prompt = "When done, run 'git crew complete'."

[agents.claude]
args = "--model claude-sonnet-4-20250514"

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
	assert.Equal(t, "claude", cfg.DefaultAgent)
	assert.Equal(t, "When done, run 'git crew complete'.", cfg.Agent.Prompt)
	assert.Equal(t, "--model claude-sonnet-4-20250514", cfg.Agents["claude"].Args)
	assert.Equal(t, "mise run ci", cfg.Complete.Command)
	assert.Equal(t, "debug", cfg.Log.Level)
}

func TestLoader_Load_GlobalConfigOnly(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config only
	globalConfig := `
default_agent = "opencode"

[agents.opencode]
args = "-m gpt-4"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "opencode", cfg.DefaultAgent)
	assert.Equal(t, "-m gpt-4", cfg.Agents["opencode"].Args)
}

func TestLoader_Load_MergeRepoOverridesGlobal(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
default_agent = "opencode"

[agent]
prompt = "Global prompt"

[agents.opencode]
args = "-m gpt-4"

[agents.claude]
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
default_agent = "claude"

[agents.claude]
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
	assert.Equal(t, "claude", cfg.DefaultAgent)                      // Overridden by repo
	assert.Equal(t, "Global prompt", cfg.Agent.Prompt)               // From global (not overridden)
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
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify: default config is returned
	assert.Equal(t, "", cfg.DefaultAgent)
	assert.Empty(t, cfg.Agents)
	assert.Equal(t, "info", cfg.Log.Level) // Default log level
}

func TestLoader_LoadGlobal(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write global config
	globalConfig := `
default_agent = "opencode"
`
	err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(globalConfig), 0644)
	require.NoError(t, err)

	// Load global config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.LoadGlobal()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "opencode", cfg.DefaultAgent)
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

func TestLoader_Load_CustomAgentCommand(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	globalDir := t.TempDir()

	// Write config with custom agent
	config := `
[agents.my-agent]
command = 'my-custom-agent --task "{{.Title}}"'
`
	err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(config), 0644)
	require.NoError(t, err)

	// Load config
	loader := NewLoaderWithGlobalDir(crewDir, globalDir)
	cfg, err := loader.Load()
	require.NoError(t, err)

	// Verify
	assert.Equal(t, `my-custom-agent --task "{{.Title}}"`, cfg.Agents["my-agent"].Command)
}
