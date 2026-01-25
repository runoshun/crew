// Package config provides configuration loading and built-in agent definitions.
package config

import "github.com/runoshun/git-crew/v2/internal/domain"

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

// Register adds all built-in agents to the given config.
// This should be called after NewDefaultConfig() and before merging user config.
// Creates worker agents (e.g., "claude", "opencode"), manager agents (e.g., "claude-manager", "opencode-manager"),
// and reviewer agents (e.g., "claude-reviewer", "opencode-reviewer").
func Register(cfg *domain.Config) {
	for name, agentSet := range builtinAgents {
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
		cfg.Agents[managerName] = manager

		// Register reviewer agent (use complete definition from agentSet)
		reviewerName := name + "-reviewer"
		reviewer := agentSet.Reviewer
		reviewer.Role = domain.RoleReviewer
		reviewer.SystemPrompt = domain.DefaultReviewerSystemPrompt
		cfg.Agents[reviewerName] = reviewer
	}

	// Set default worker, manager, and reviewer
	cfg.AgentsConfig.DefaultWorker = "opencode"
	cfg.AgentsConfig.DefaultManager = "opencode-manager"
	cfg.AgentsConfig.DefaultReviewer = "opencode-reviewer"
}
