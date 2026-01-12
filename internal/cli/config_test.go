package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newConfigTestContainer creates an app.Container with proper config infrastructure for testing.
// This uses real config implementations since we're testing config commands.
func newConfigTestContainer(t *testing.T) *app.Container {
	t.Helper()

	// Create temporary directory
	repoRoot := t.TempDir()

	// Initialize a real git repository using git command
	cmd := exec.Command("git", "init")
	cmd.Dir = repoRoot
	require.NoError(t, cmd.Run())

	// Configure git user for the test repo
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoRoot
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoRoot
	require.NoError(t, cmd.Run())

	// Create .git/crew directory (needed for config file creation)
	crewDir := filepath.Join(repoRoot, ".git", "crew")
	require.NoError(t, os.MkdirAll(crewDir, 0755))

	// Set HOME to a temporary directory to isolate global config
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Create container using New (which sets up real config infrastructure)
	container, err := app.New(repoRoot)
	require.NoError(t, err)

	return container
}

// =============================================================================
// Config Command Tests
// =============================================================================

func TestConfigCommand_NoSubcommand_ShowsHelp(t *testing.T) {
	// Setup
	container := newConfigTestContainer(t)

	// Create command
	cmd := newConfigCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	// Execute
	err := cmd.Execute()

	// Assert - should show help with subcommand list
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Available Commands:")
	assert.Contains(t, output, "show")
	assert.Contains(t, output, "template")
	assert.Contains(t, output, "init")
}

// =============================================================================
// Config Show Subcommand Tests
// =============================================================================

func TestConfigShowCommand_DisplaysEffectiveConfig(t *testing.T) {
	// Setup
	container := newConfigTestContainer(t)

	// Create command
	cmd := newConfigCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"show"})

	// Execute
	err := cmd.Execute()

	// Assert
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "[Loaded from]")
	assert.Contains(t, buf.String(), "[Effective Config]")
}

// =============================================================================
// Config Template Subcommand Tests
// =============================================================================

func TestConfigTemplateCommand_OutputsTemplate(t *testing.T) {
	// Setup
	container := newConfigTestContainer(t)

	// Create command
	cmd := newConfigCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"template"})

	// Execute
	err := cmd.Execute()

	// Assert
	require.NoError(t, err)
	output := buf.String()

	// Template should contain TOML config structure
	assert.Contains(t, output, "[agents]")
	assert.Contains(t, output, "worker_default")
	assert.Contains(t, output, "manager_default")

	// Should not contain metadata headers (just template content)
	assert.NotContains(t, output, "[Loaded from]")
	assert.NotContains(t, output, "[Effective Config]")
}

// =============================================================================
// Config Init Subcommand Tests
// =============================================================================

func TestConfigInitCommand_CreatesRepoConfig(t *testing.T) {
	// Setup
	container := newConfigTestContainer(t)

	// Create command
	cmd := newConfigCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"init"})

	// Execute
	err := cmd.Execute()

	// Assert
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Created config file:")

	// Verify file was created
	info := container.ConfigManager.GetRepoConfigInfo()
	assert.True(t, info.Exists)
	assert.Contains(t, info.Content, "[agents]")
}

func TestConfigInitCommand_WithGlobalFlag(t *testing.T) {
	// Setup
	container := newConfigTestContainer(t)

	// Create command
	cmd := newConfigCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"init", "--global"})

	// Execute
	err := cmd.Execute()

	// Assert
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Created config file:")

	// Verify file was created
	info := container.ConfigManager.GetGlobalConfigInfo()
	assert.True(t, info.Exists)
	assert.Contains(t, info.Content, "[agents]")
}

func TestConfigInitCommand_ErrorIfFileExists(t *testing.T) {
	// Setup
	container := newConfigTestContainer(t)

	// Create config file first
	cfg := domain.NewDefaultConfig()
	err := container.ConfigManager.InitRepoConfig(cfg)
	require.NoError(t, err)

	// Create command
	cmd := newConfigCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"init"})

	// Execute
	err = cmd.Execute()

	// Assert - should fail because file already exists
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrConfigExists)
}
