// Package builtin provides built-in agent configurations for known CLI tools.
// This package is responsible for CLI-specific details that domain should not know about.
package builtin

import "github.com/runoshun/git-crew/v2/internal/domain"

// agentConfig defines a built-in agent configuration (internal use only).
// SystemArgs is role-specific: Workers and Managers have different system arguments.
type agentConfig struct {
	CommandTemplate     string
	Command             string
	WorkerSystemArgs    string
	ManagerSystemArgs   string
	DefaultArgs         string
	DefaultModel        string
	Description         string
	WorktreeSetupScript string
	ExcludePatterns     []string
}

// builtinAgents contains preset configurations for known agents.
var builtinAgents = map[string]agentConfig{
	"claude":   claudeAgent,
	"opencode": opencodeAgent,
}

// Register adds all built-in agents, workers, and managers to the given config.
// This should be called after NewDefaultConfig() and before merging user config.
func Register(cfg *domain.Config) {
	for name, builtin := range builtinAgents {
		cfg.Agents[name] = domain.Agent{
			Command:             builtin.Command,
			CommandTemplate:     builtin.CommandTemplate,
			DefaultModel:        builtin.DefaultModel,
			Description:         builtin.Description,
			WorktreeSetupScript: builtin.WorktreeSetupScript,
			ExcludePatterns:     builtin.ExcludePatterns,
		}
		// Create builtin workers with role-specific SystemArgs
		cfg.Workers[name] = domain.Worker{
			Agent:           name,
			CommandTemplate: builtin.CommandTemplate,
			Command:         builtin.Command,
			SystemArgs:      builtin.WorkerSystemArgs,
			Args:            builtin.DefaultArgs,
			Description:     builtin.Description,
		}
		// Create builtin managers with role-specific SystemArgs
		cfg.Managers[name] = domain.Manager{
			Agent:       name,
			SystemArgs:  builtin.ManagerSystemArgs,
			Description: builtin.Description,
		}
	}

}
