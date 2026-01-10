// Package config provides configuration loading and built-in agent definitions.
package config

import (
	"os/exec"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// builtinAgentSet contains all agent variants for a CLI tool.
// Each field is a complete domain.Agent that can be registered directly.
type builtinAgentSet struct {
	Worker   domain.Agent
	Manager  domain.Agent
	Reviewer domain.Agent
}

// builtinAgents contains preset configurations for known agents.
var builtinAgents = map[string]builtinAgentSet{
	"claude":   claudeAgents,
	"codex":    codexAgents,
	"opencode": opencodeAgents,
}

// RegisterWithLookPath adds built-in agents to the given config, checking if the required CLI commands exist.
// This should be called after NewDefaultConfig() and before merging user config.
// Creates worker agents (e.g., "claude", "opencode"), manager agents (e.g., "claude-manager", "opencode-manager"),
// and reviewer agents (e.g., "claude-reviewer", "opencode-reviewer") only for commands that exist in PATH.
// The lookPath function is used to check if a command exists (typically os/exec.LookPath).
func RegisterWithLookPath(cfg *domain.Config, lookPath func(string) (string, error)) {
	availableCommands := make(map[string]bool)

	// Check which commands are available
	for name := range builtinAgents {
		_, err := lookPath(name)
		availableCommands[name] = err == nil
	}

	// Register agents for available commands
	for name, agentSet := range builtinAgents {
		if !availableCommands[name] {
			continue // Skip if command is not available
		}

		// Register worker agent (use complete definition from agentSet)
		worker := agentSet.Worker
		worker.Role = domain.RoleWorker
		worker.SystemPrompt = domain.DefaultSystemPrompt
		cfg.Agents[name] = worker

		// Register manager agent (use complete definition, override role-specific fields)
		managerName := name + "-manager"
		manager := agentSet.Manager
		manager.Inherit = name // Inherit from worker agent
		manager.Role = domain.RoleManager
		manager.SystemPrompt = domain.DefaultManagerSystemPrompt
		manager.Hidden = true
		cfg.Agents[managerName] = manager

		// Register reviewer agent (use complete definition from agentSet)
		reviewerName := name + "-reviewer"
		reviewer := agentSet.Reviewer
		reviewer.Role = domain.RoleReviewer
		reviewer.SystemPrompt = domain.DefaultReviewerSystemPrompt
		reviewer.Hidden = true
		cfg.Agents[reviewerName] = reviewer
	}

	// Set default agents based on available commands
	// Prefer opencode if available, otherwise use claude if available
	if availableCommands["opencode"] {
		cfg.AgentsConfig.DefaultWorker = "opencode"
		cfg.AgentsConfig.DefaultManager = "opencode-manager"
		cfg.AgentsConfig.DefaultReviewer = "opencode-reviewer"
	} else if availableCommands["claude"] {
		cfg.AgentsConfig.DefaultWorker = "claude"
		cfg.AgentsConfig.DefaultManager = "claude-manager"
		cfg.AgentsConfig.DefaultReviewer = "claude-reviewer"
	}
}

// Register adds all built-in agents to the given config using exec.LookPath to check for command existence.
// This is a wrapper around RegisterWithLookPath that uses os/exec.LookPath by default.
func Register(cfg *domain.Config) {
	RegisterWithLookPath(cfg, exec.LookPath)
}
