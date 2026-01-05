package builtin

import (
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

func TestRegister(t *testing.T) {
	cfg := domain.NewDefaultConfig()
	Register(cfg)

	// Check that builtin agents are registered
	expectedAgents := []string{"claude", "opencode"}
	for _, name := range expectedAgents {
		agent, ok := cfg.Agents[name]
		if !ok {
			t.Errorf("expected agent %q to be registered", name)
			continue
		}
		if agent.Command == "" {
			t.Errorf("agent %q should have a Command", name)
		}
		if agent.CommandTemplate == "" {
			t.Errorf("agent %q should have a CommandTemplate", name)
		}
	}

	// Check that builtin workers are registered
	for _, name := range expectedAgents {
		worker, ok := cfg.Workers[name]
		if !ok {
			t.Errorf("expected worker %q to be registered", name)
			continue
		}
		if worker.Agent != name {
			t.Errorf("worker %q should reference agent %q, got %q", name, name, worker.Agent)
		}
		// Note: SystemArgs may be empty - model is now in CommandTemplate directly
	}

	// Check that builtin managers are registered
	for _, name := range expectedAgents {
		manager, ok := cfg.Managers[name]
		if !ok {
			t.Errorf("expected manager %q to be registered", name)
			continue
		}
		if manager.Agent != name {
			t.Errorf("manager %q should reference agent %q, got %q", name, name, manager.Agent)
		}
	}
}

func TestBuiltinAgentConfigs(t *testing.T) {
	// Verify claude agent config
	if claudeAgent.Command != "claude" {
		t.Errorf("claude agent command = %q, want %q", claudeAgent.Command, "claude")
	}
	if claudeAgent.DefaultModel != "opus" {
		t.Errorf("claude agent default model = %q, want %q", claudeAgent.DefaultModel, "opus")
	}
	if len(claudeAgent.ExcludePatterns) == 0 {
		t.Error("claude agent should have exclude patterns")
	}

	// Verify opencode agent config
	if opencodeAgent.Command != "opencode" {
		t.Errorf("opencode agent command = %q, want %q", opencodeAgent.Command, "opencode")
	}
	if opencodeAgent.DefaultModel != "anthropic/claude-opus-4-5" {
		t.Errorf("opencode agent default model = %q, want %q", opencodeAgent.DefaultModel, "anthropic/claude-opus-4-5")
	}
	if len(opencodeAgent.ExcludePatterns) == 0 {
		t.Error("opencode agent should have exclude patterns")
	}
}
