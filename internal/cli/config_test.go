package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

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
			formatEffectiveConfig(&buf, tt.config)

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
	formatEffectiveConfig(&buf, config)

	output := buf.String()

	// Should still have section headers
	sections := []string{"[agents]", "[complete]", "[diff]", "[log]", "[tasks]", "[tui]", "[worktree]"}
	for _, section := range sections {
		if !strings.Contains(output, section) {
			t.Errorf("output should contain section %q, got:\n%s", section, output)
		}
	}
}
