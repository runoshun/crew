package builtin

// opencodeAgent contains the built-in configuration for the OpenCode CLI.
var opencodeAgent = agentConfig{
	CommandTemplate:   "{{.Command}} {{.SystemArgs}} {{.Args}} --prompt {{.Prompt}}",
	Command:           "opencode",
	WorkerSystemArgs:  "-m {{.Model}}",
	ManagerSystemArgs: "-m {{.Model}}",
	DefaultArgs:       "",
	DefaultModel:      "anthropic/claude-opus-4-5",
	Description:       "General purpose coding agent via opencode CLI",
	// WorktreeSetupScript and ExcludePatterns are Worker-only (Managers don't use worktrees)
	WorktreeSetupScript: opencodeSetupScript,
	ExcludePatterns:     []string{".opencode/plugin/crew-hooks.ts"},
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
				if (result.exitCode !== 0) {
					await ` + "$`crew show {{.TaskID}} | grep -q \"^Status: in_complete\" || crew edit {{.TaskID}} --status needs_input`" + `;
				}
			}
		}
  }
}
EOF
`
