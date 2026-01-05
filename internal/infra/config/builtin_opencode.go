package config

// opencodeAgents contains the built-in configuration for the OpenCode CLI.
var opencodeAgents = builtinAgentSet{
	Worker: builtinAgentDef{
		CommandTemplate:   "opencode -m {{.Model}} {{.Args}}{{if .Continue}} -c{{end}} --prompt {{.Prompt}}",
		DefaultModel:      "anthropic/claude-opus-4-5",
		Description:       "General purpose coding agent via opencode CLI",
		WorkerSetupScript: opencodeSetupScript,
	},
	Manager: builtinAgentDef{
		Description: "OpenCode manager agent for task orchestration",
	},
}

const opencodeSetupScript = `#!/bin/bash
cd {{.Worktree}}

PLUGIN_DIR=.opencode/plugin
PLUGIN_FILE=${PLUGIN_DIR}/crew-hooks.ts

mkdir -p ${PLUGIN_DIR}/

cat > ${PLUGIN_FILE} << 'EOF'
import type { Plugin } from "@opencode-ai/plugin"

export const CrewHooksPlugin: Plugin = async ({ $ }) => {
  return {
		event: async ({ event }) => {
			if (event.type === "session.idle") {
				await ` + "$`crew show {{.TaskID}} | grep -q \"^Status: in_review\" || crew edit {{.TaskID}} --status needs_input`" + `;
			}
		}
  }
}
EOF

# Add exclude pattern to git (use git rev-parse for worktree support)
GIT_DIR=$(git rev-parse --git-dir 2>/dev/null) && \
  echo ".opencode/plugin/crew-hooks.ts" >> "${GIT_DIR}/info/exclude" || true
`
