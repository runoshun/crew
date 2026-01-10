package config

import "github.com/runoshun/git-crew/v2/internal/domain"

// codexAgents contains the built-in configuration for the Codex CLI.
// Note: Codex currently only supports notify hook on agent-turn-complete,
// so automatic status transitions (needs_input, in_progress) are not available.
// Only in_review transition can be configured via global ~/.codex/config.toml.
var codexAgents = builtinAgentSet{
	Worker: domain.Agent{
		CommandTemplate: "codex --model {{.Model}} {{.Args}} {{.Prompt}}",
		DefaultModel:    "gpt-5.2-codex",
		Description:     "General purpose coding agent via Codex CLI",
	},
	Manager: domain.Agent{
		Description: "Codex manager agent for task orchestration",
	},
	Reviewer: domain.Agent{
		// Non-interactive mode: codex exec for synchronous execution
		CommandTemplate: "codex exec --model {{.Model}} {{.Args}} {{.Prompt}}",
		DefaultModel:    "gpt-5.2-codex",
		Description:     "Code review agent via Codex CLI",
	},
}
