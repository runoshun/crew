package builtin

import (
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

func TestRegister(t *testing.T) {
	cfg := domain.NewDefaultConfig()
	Register(cfg)

	// Check that builtin worker agents are registered
	expectedWorkers := []string{"claude", "opencode"}
	for _, name := range expectedWorkers {
		agent, ok := cfg.Agents[name]
		if !ok {
			t.Errorf("expected worker agent %q to be registered", name)
			continue
		}
		if agent.CommandTemplate == "" {
			t.Errorf("agent %q should have a CommandTemplate", name)
		}
		if agent.Role != domain.RoleWorker {
			t.Errorf("agent %q should have Role=worker, got %q", name, agent.Role)
		}
	}

	// Check that builtin manager agents are registered
	expectedManagers := []string{"claude-manager", "opencode-manager"}
	for _, name := range expectedManagers {
		agent, ok := cfg.Agents[name]
		if !ok {
			t.Errorf("expected manager agent %q to be registered", name)
			continue
		}
		if agent.Role != domain.RoleManager {
			t.Errorf("agent %q should have Role=manager, got %q", name, agent.Role)
		}
		if !agent.Hidden {
			t.Errorf("manager agent %q should be hidden by default", name)
		}
	}

	// Check default agents are set
	if cfg.AgentsConfig.DefaultWorker != "opencode" {
		t.Errorf("DefaultWorker = %q, want %q", cfg.AgentsConfig.DefaultWorker, "opencode")
	}
	if cfg.AgentsConfig.DefaultManager != "opencode-manager" {
		t.Errorf("DefaultManager = %q, want %q", cfg.AgentsConfig.DefaultManager, "opencode-manager")
	}
}

func TestBuiltinAgentConfigs(t *testing.T) {
	// Verify claude agent config
	claudeSet := claudeAgents
	if claudeSet.Worker.DefaultModel != "opus" {
		t.Errorf("claude agent default model = %q, want %q", claudeSet.Worker.DefaultModel, "opus")
	}
	if claudeSet.Worker.WorkerSetupScript == "" {
		t.Error("claude agent should have worker setup script")
	}

	// Verify opencode agent config
	opencodeSet := opencodeAgents
	if opencodeSet.Worker.DefaultModel != "anthropic/claude-opus-4-5" {
		t.Errorf("opencode agent default model = %q, want %q", opencodeSet.Worker.DefaultModel, "anthropic/claude-opus-4-5")
	}
	if opencodeSet.Worker.WorkerSetupScript == "" {
		t.Error("opencode agent should have worker setup script")
	}
}
