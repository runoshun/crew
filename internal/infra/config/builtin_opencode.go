package config

import (
	_ "embed"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

//go:embed builtin_opencode_plugin.ts
var opencodePluginTS string

// opencodeAgents contains the built-in configuration for the OpenCode CLI.
var opencodeAgents = builtinAgentSet{
	Worker: domain.Agent{
		CommandTemplate: "opencode -m {{.Model}} {{.Args}}{{if .Continue}} -c{{end}} --prompt {{.Prompt}}",
		DefaultModel:    "anthropic/claude-opus-4-5",
		Description:     "General purpose coding agent via opencode CLI",
		SetupScript:     opencodeSetupScript,
	},
	Manager: domain.Agent{
		Description: "OpenCode manager agent for task orchestration",
	},
	Reviewer: domain.Agent{
		// Non-interactive mode: opencode run for synchronous execution
		CommandTemplate: "opencode run -m {{.Model}} {{.Args}} {{.Prompt}}",
		DefaultModel:    "anthropic/claude-sonnet-4-5",
		Description:     "Code review agent via opencode CLI",
	},
}

var opencodeSetupScript = `#!/bin/bash
cd {{.Worktree}}

PLUGIN_DIR=.opencode/plugin
PLUGIN_FILE=${PLUGIN_DIR}/crew-hooks.ts

mkdir -p ${PLUGIN_DIR}/

cat > ${PLUGIN_FILE} << 'EOF'
` + opencodePluginTS + `EOF

# Add exclude pattern to git (use git rev-parse for worktree support)
GIT_COMMON_DIR=$(git rev-parse --git-common-dir 2>/dev/null) && \
  (grep -qxF ".opencode/plugin/crew-hooks.ts" "${GIT_COMMON_DIR}/info/exclude" || echo ".opencode/plugin/crew-hooks.ts") >> "${GIT_COMMON_DIR}/info/exclude" || true
`
