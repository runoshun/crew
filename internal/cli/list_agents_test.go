package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

func TestListAgentsCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		agents      map[string]domain.Agent
		disabled    []string
		wantAgents  []string
		wantMissing []string
	}{
		{
			name: "list enabled agents only",
			args: []string{},
			agents: map[string]domain.Agent{
				"agent1":   {Role: domain.RoleWorker, Description: "First agent"},
				"agent2":   {Role: domain.RoleWorker, Description: "Second agent"},
				"disabled": {Role: domain.RoleWorker, Description: "Disabled agent"},
			},
			disabled:    []string{"disabled"},
			wantAgents:  []string{"agent1", "agent2"},
			wantMissing: []string{"disabled"},
		},
		{
			name: "list all agents with --all",
			args: []string{"--all"},
			agents: map[string]domain.Agent{
				"agent1":   {Role: domain.RoleWorker, Description: "First agent"},
				"disabled": {Role: domain.RoleWorker, Description: "Disabled agent"},
			},
			disabled:    []string{"disabled"},
			wantAgents:  []string{"agent1", "disabled"},
			wantMissing: []string{},
		},
		{
			name: "list disabled agents only with --disabled",
			args: []string{"--disabled"},
			agents: map[string]domain.Agent{
				"agent1":   {Role: domain.RoleWorker, Description: "First agent"},
				"disabled": {Role: domain.RoleWorker, Description: "Disabled agent"},
			},
			disabled:    []string{"disabled"},
			wantAgents:  []string{"disabled"},
			wantMissing: []string{"agent1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &domain.Config{
				Agents: tt.agents,
				AgentsConfig: domain.AgentsConfig{
					DisabledAgents: tt.disabled,
				},
			}

			container := &app.Container{
				ConfigLoader: &mockConfigLoader{cfg: cfg},
			}

			cmd := newListAgentsCommand(container)
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			output := out.String()

			for _, want := range tt.wantAgents {
				if !strings.Contains(output, want) {
					t.Errorf("output should contain %q, got:\n%s", want, output)
				}
			}

			for _, missing := range tt.wantMissing {
				// Check that the agent name is not in any data line (skip header)
				lines := strings.Split(output, "\n")
				for _, line := range lines[1:] { // Skip header line
					if strings.HasPrefix(strings.TrimSpace(line), missing) {
						t.Errorf("output should not contain %q in data lines, got:\n%s", missing, output)
					}
				}
			}
		})
	}
}

// mockConfigLoader implements domain.ConfigLoader for testing.
type mockConfigLoader struct {
	cfg *domain.Config
	err error
}

func (m *mockConfigLoader) Load() (*domain.Config, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.cfg, nil
}

func (m *mockConfigLoader) LoadGlobal() (*domain.Config, error) {
	return m.Load()
}

func (m *mockConfigLoader) LoadRepo() (*domain.Config, error) {
	return m.Load()
}

func (m *mockConfigLoader) LoadWithOptions(_ domain.LoadConfigOptions) (*domain.Config, error) {
	return m.Load()
}
