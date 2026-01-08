// Package config provides configuration loading and built-in agent definitions.
package config

import "github.com/runoshun/git-crew/v2/internal/domain"

// builtinAgentDef defines a built-in agent configuration (internal use only).
// Each CLI tool (claude, opencode) has a base configuration and role-specific variants.
type builtinAgentDef struct {
	// Base configuration
	CommandTemplate string // Full command template
	DefaultModel    string // Default model for this agent
	Description     string // Description of the agent

	// Worker-specific
	WorkerSetupScript string // Setup script for workers (includes exclude patterns)

	// Manager-specific (hidden by default)
}

// builtinAgentSet contains all agent variants for a CLI tool.
type builtinAgentSet struct {
	Worker   builtinAgentDef
	Manager  builtinAgentDef
	Reviewer builtinAgentDef
}

// builtinAgents contains preset configurations for known agents.
var builtinAgents = map[string]builtinAgentSet{
	"claude":   claudeAgents,
	"opencode": opencodeAgents,
}

// Register adds all built-in agents to the given config.
// This should be called after NewDefaultConfig() and before merging user config.
// Creates worker agents (e.g., "claude", "opencode"), manager agents (e.g., "claude-manager", "opencode-manager"),
// and reviewer agents (e.g., "claude-reviewer", "opencode-reviewer").
func Register(cfg *domain.Config) {
	for name, agentSet := range builtinAgents {
		// Register worker agent
		cfg.Agents[name] = domain.Agent{
			CommandTemplate: agentSet.Worker.CommandTemplate,
			Role:            domain.RoleWorker,
			SystemPrompt:    domain.DefaultSystemPrompt,
			DefaultModel:    agentSet.Worker.DefaultModel,
			Description:     agentSet.Worker.Description,
			SetupScript:     agentSet.Worker.WorkerSetupScript,
		}

		// Register manager agent (hidden by default)
		managerName := name + "-manager"
		cfg.Agents[managerName] = domain.Agent{
			Inherit:      name,
			Role:         domain.RoleManager,
			SystemPrompt: domain.DefaultManagerSystemPrompt,
			Description:  agentSet.Manager.Description,
			Hidden:       true,
		}

		// Register reviewer agent (hidden by default)
		reviewerName := name + "-reviewer"
		cfg.Agents[reviewerName] = domain.Agent{
			CommandTemplate: agentSet.Reviewer.CommandTemplate,
			Role:            domain.RoleReviewer,
			SystemPrompt:    domain.DefaultReviewerSystemPrompt,
			DefaultModel:    agentSet.Reviewer.DefaultModel,
			Description:     agentSet.Reviewer.Description,
			Hidden:          true,
		}
	}

	// Set default worker, manager, and reviewer
	cfg.AgentsConfig.DefaultWorker = "opencode"
	cfg.AgentsConfig.DefaultManager = "opencode-manager"
	cfg.AgentsConfig.DefaultReviewer = "opencode-reviewer"
}
