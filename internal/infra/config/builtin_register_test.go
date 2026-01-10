package config

import (
	"errors"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

func TestRegisterWithLookPath_BothCommandsAvailable(t *testing.T) {
	cfg := domain.NewDefaultConfig()
	lookPath := mockLookPath(map[string]bool{
		"claude":   true,
		"opencode": true,
	})
	RegisterWithLookPath(cfg, lookPath)

	// Resolve inheritance (required for agents using Inherit field)
	if err := cfg.ResolveInheritance(); err != nil {
		t.Fatalf("failed to resolve inheritance: %v", err)
	}

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
		// Manager should have CommandTemplate set (from builtin definition)
		if agent.CommandTemplate == "" {
			t.Errorf("manager agent %q should have a CommandTemplate", name)
		}
	}

	// Check default agents are set (should prefer opencode)
	if cfg.AgentsConfig.DefaultWorker != "opencode" {
		t.Errorf("DefaultWorker = %q, want %q", cfg.AgentsConfig.DefaultWorker, "opencode")
	}
	if cfg.AgentsConfig.DefaultManager != "opencode-manager" {
		t.Errorf("DefaultManager = %q, want %q", cfg.AgentsConfig.DefaultManager, "opencode-manager")
	}
	if cfg.AgentsConfig.DefaultReviewer != "opencode-reviewer" {
		t.Errorf("DefaultReviewer = %q, want %q", cfg.AgentsConfig.DefaultReviewer, "opencode-reviewer")
	}
}

func TestRegisterWithLookPath_OnlyClaudeAvailable(t *testing.T) {
	cfg := domain.NewDefaultConfig()
	lookPath := mockLookPath(map[string]bool{
		"claude":   true,
		"opencode": false,
	})
	RegisterWithLookPath(cfg, lookPath)

	// Check that only claude worker agent is registered
	if _, ok := cfg.Agents["claude"]; !ok {
		t.Errorf("expected worker agent %q to be registered", "claude")
	}
	if _, ok := cfg.Agents["opencode"]; ok {
		t.Errorf("expected worker agent %q NOT to be registered", "opencode")
	}

	// Check that claude manager and reviewer are registered
	if _, ok := cfg.Agents["claude-manager"]; !ok {
		t.Errorf("expected manager agent %q to be registered", "claude-manager")
	}
	if _, ok := cfg.Agents["opencode-manager"]; ok {
		t.Errorf("expected manager agent %q NOT to be registered", "opencode-manager")
	}

	// Check default agents are set to claude
	if cfg.AgentsConfig.DefaultWorker != "claude" {
		t.Errorf("DefaultWorker = %q, want %q", cfg.AgentsConfig.DefaultWorker, "claude")
	}
	if cfg.AgentsConfig.DefaultManager != "claude-manager" {
		t.Errorf("DefaultManager = %q, want %q", cfg.AgentsConfig.DefaultManager, "claude-manager")
	}
	if cfg.AgentsConfig.DefaultReviewer != "claude-reviewer" {
		t.Errorf("DefaultReviewer = %q, want %q", cfg.AgentsConfig.DefaultReviewer, "claude-reviewer")
	}
}

func TestRegisterWithLookPath_OnlyOpencodeAvailable(t *testing.T) {
	cfg := domain.NewDefaultConfig()
	lookPath := mockLookPath(map[string]bool{
		"claude":   false,
		"opencode": true,
	})
	RegisterWithLookPath(cfg, lookPath)

	// Check that only opencode worker agent is registered
	if _, ok := cfg.Agents["opencode"]; !ok {
		t.Errorf("expected worker agent %q to be registered", "opencode")
	}
	if _, ok := cfg.Agents["claude"]; ok {
		t.Errorf("expected worker agent %q NOT to be registered", "claude")
	}

	// Check default agents are set to opencode
	if cfg.AgentsConfig.DefaultWorker != "opencode" {
		t.Errorf("DefaultWorker = %q, want %q", cfg.AgentsConfig.DefaultWorker, "opencode")
	}
}

func TestRegisterWithLookPath_NoCommandsAvailable(t *testing.T) {
	cfg := domain.NewDefaultConfig()
	lookPath := mockLookPath(map[string]bool{
		"claude":   false,
		"opencode": false,
	})
	RegisterWithLookPath(cfg, lookPath)

	// Check that no worker agents are registered
	if _, ok := cfg.Agents["claude"]; ok {
		t.Errorf("expected worker agent %q NOT to be registered", "claude")
	}
	if _, ok := cfg.Agents["opencode"]; ok {
		t.Errorf("expected worker agent %q NOT to be registered", "opencode")
	}

	// Check that no default agents are set
	if cfg.AgentsConfig.DefaultWorker != "" {
		t.Errorf("DefaultWorker = %q, want empty string", cfg.AgentsConfig.DefaultWorker)
	}
	if cfg.AgentsConfig.DefaultManager != "" {
		t.Errorf("DefaultManager = %q, want empty string", cfg.AgentsConfig.DefaultManager)
	}
	if cfg.AgentsConfig.DefaultReviewer != "" {
		t.Errorf("DefaultReviewer = %q, want empty string", cfg.AgentsConfig.DefaultReviewer)
	}
}

// mockLookPath creates a mock LookPath function that returns success or error based on the provided map
func mockLookPath(available map[string]bool) func(string) (string, error) {
	return func(name string) (string, error) {
		if available[name] {
			return name, nil
		}
		return "", errors.New("not found")
	}
}

func TestBuiltinAgentConfigs(t *testing.T) {
	// Verify claude agent config
	claudeSet := claudeAgents
	if claudeSet.Worker.DefaultModel != "opus" {
		t.Errorf("claude agent default model = %q, want %q", claudeSet.Worker.DefaultModel, "opus")
	}
	if claudeSet.Worker.SetupScript == "" {
		t.Error("claude agent should have worker setup script")
	}

	// Verify opencode agent config
	opencodeSet := opencodeAgents
	if opencodeSet.Worker.DefaultModel != "anthropic/claude-opus-4-5" {
		t.Errorf("opencode agent default model = %q, want %q", opencodeSet.Worker.DefaultModel, "anthropic/claude-opus-4-5")
	}
	if opencodeSet.Worker.SetupScript == "" {
		t.Error("opencode agent should have worker setup script")
	}
}
