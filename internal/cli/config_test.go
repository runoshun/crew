package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// Verify builtin agents are actually registered with their sections
	// Check for worker agents (claude, opencode, codex)
	assert.Contains(t, output, "[agents.claude]", "claude worker agent should be in template")
	assert.Contains(t, output, "[agents.opencode]", "opencode worker agent should be in template")
	assert.Contains(t, output, "[agents.codex]", "codex worker agent should be in template")

	// Check for manager agents
	assert.Contains(t, output, "[agents.claude-manager]", "claude manager agent should be in template")

	// Check for role field in agent sections
	assert.Contains(t, output, `role = "worker"`, "worker role should be present")
	assert.Contains(t, output, `role = "manager"`, "manager role should be present")

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

// =============================================================================
// formatEffectiveConfig Tests
// =============================================================================

// TestFormatEffectiveConfig tests that formatEffectiveConfig outputs all expected sections.
func TestFormatEffectiveConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *domain.Config
		wantContent []string // strings that should be present in output
	}{
		{
			name: "AgentsConfig fields are output",
			config: &domain.Config{
				Agents: map[string]domain.Agent{
					"test-agent": {
						Role:        domain.RoleWorker,
						Description: "Test agent",
					},
				},
				AgentsConfig: domain.AgentsConfig{
					DefaultWorker:   "test-worker",
					DefaultManager:  "test-manager",
					DefaultReviewer: "test-reviewer",
					DisabledAgents:  []string{"disabled1", "disabled2"},
				},
				Log: domain.LogConfig{
					Level: "info",
				},
			},
			wantContent: []string{
				"worker_default = 'test-worker'",
				"manager_default = 'test-manager'",
				"reviewer_default = 'test-reviewer'",
				"disabled_agents = ['disabled1', 'disabled2']",
			},
		},
		{
			name: "main sections are present",
			config: &domain.Config{
				Agents:       map[string]domain.Agent{},
				AgentsConfig: domain.AgentsConfig{},
				Complete: domain.CompleteConfig{
					Command: "mise run ci",
				},
				Diff: domain.DiffConfig{
					Command: "git diff",
				},
				Log: domain.LogConfig{
					Level: "debug",
				},
				Tasks: domain.TasksConfig{
					Store: "git",
				},
				TUI: domain.TUIConfig{
					Keybindings: map[string]domain.TUIKeybinding{},
				},
				Worktree: domain.WorktreeConfig{
					SetupCommand: "npm install",
				},
			},
			wantContent: []string{
				"[agents]",
				"[complete]",
				"[diff]",
				"[log]",
				"[tasks]",
				"[tui]",
				"[worktree]",
			},
		},
		{
			name: "section values are correct",
			config: &domain.Config{
				Agents:       map[string]domain.Agent{},
				AgentsConfig: domain.AgentsConfig{},
				Complete: domain.CompleteConfig{
					Command: "test-command",
				},
				Log: domain.LogConfig{
					Level: "warn",
				},
				Tasks: domain.TasksConfig{
					Store:     "json",
					Namespace: "custom",
				},
				Worktree: domain.WorktreeConfig{
					SetupCommand: "echo hello",
				},
			},
			wantContent: []string{
				"command = 'test-command'",
				"level = 'warn'",
				"store = 'json'",
				"namespace = 'custom'",
				"setup_command = 'echo hello'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := formatEffectiveConfig(&buf, tt.config)
			require.NoError(t, err)

			output := buf.String()

			for _, want := range tt.wantContent {
				if !strings.Contains(output, want) {
					t.Errorf("output should contain %q, got:\n%s", want, output)
				}
			}
		})
	}
}

// TestFormatEffectiveConfig_EmptyValues tests that empty values are handled correctly.
func TestFormatEffectiveConfig_EmptyValues(t *testing.T) {
	config := &domain.Config{
		Agents:       map[string]domain.Agent{},
		AgentsConfig: domain.AgentsConfig{},
		Complete:     domain.CompleteConfig{},
		Diff:         domain.DiffConfig{},
		Log: domain.LogConfig{
			Level: domain.DefaultLogLevel,
		},
		Tasks:    domain.TasksConfig{},
		TUI:      domain.TUIConfig{},
		Worktree: domain.WorktreeConfig{},
	}

	var buf bytes.Buffer
	err := formatEffectiveConfig(&buf, config)
	require.NoError(t, err)

	output := buf.String()

	// Should still have section headers
	sections := []string{"[agents]", "[complete]", "[diff]", "[log]", "[tasks]", "[tui]", "[worktree]"}
	for _, section := range sections {
		if !strings.Contains(output, section) {
			t.Errorf("output should contain section %q, got:\n%s", section, output)
		}
	}
}
