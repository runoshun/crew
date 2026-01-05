package builtin

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
			if (event.type === "session.idle" || event.type === "permission.updated") {
				await ` + "$`crew show {{.TaskID}} | grep -q \"^Status: in_complete\" || crew edit {{.TaskID}} --status needs_input`" + `;
			}
			else if (event.type === "message.updated" && event.properties.info.role === "user") {
				await ` + "$`crew show {{.TaskID}} | grep -q \"^Status: needs_input\" || crew edit {{.TaskID}} --status in_progress`" + `;
			}
		}
  }
}
EOF

# Add exclude pattern to git
echo ".opencode/plugin/crew-hooks.ts" >> .git/info/exclude
`
