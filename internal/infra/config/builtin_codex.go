package config

import "github.com/runoshun/git-crew/v2/internal/domain"

// codexAgents contains the built-in configuration for the Codex CLI.
var codexAgents = builtinAgentSet{
	Worker: domain.Agent{
		CommandTemplate: "codex -c 'notify=[\"crew\", \"edit\", \"{{.TaskID}}\", \"--status\", \"in_progress\", \"--if-status\", \"in_progress\"]' --model {{.Model}} --full-auto {{.Args}} {{.Prompt}}",
		DefaultModel:    "gpt-5.2-codex",
		Description:     "General purpose coding agent via Codex CLI",
	},
	Manager: domain.Agent{
		Description: "Codex manager agent for task orchestration",
	},
	Reviewer: domain.Agent{
		// Non-interactive mode: codex exec for synchronous execution
		CommandTemplate: "codex exec -s read-only --model {{.Model}} {{.Args}} {{.Prompt}}",
		DefaultModel:    "gpt-5.2-codex",
		Description:     "Code review agent via Codex CLI",
	},
}
