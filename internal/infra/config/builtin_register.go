// Package config provides configuration loading and built-in agent definitions.
package config

import (
	"os/exec"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

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
