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
		// Check if current status is in_review (if so, skip auto-switching)
		const isInReview = async () => {
			try {
				const { exitCode } = await ` + "$`crew show {{.TaskID}} | grep -q \"^Status: in_review\"`" + `;
				return exitCode === 0;
			} catch {
				return false;
			}
		};

		// Transition to needs_input: session idle
		if (event.type === "session.idle") {
			if (!(await isInReview())) {
				await ` + "$`crew edit {{.TaskID}} --status needs_input`" + `;
			}
		}

		// Transition to needs_input or in_progress: session status change
		if (event.type === "session.status") {
			if (!(await isInReview())) {
				if (event.status.type === "idle") {
					await ` + "$`crew edit {{.TaskID}} --status needs_input`" + `;
				} else if (event.status.type === "busy") {
					await ` + "$`crew edit {{.TaskID}} --status in_progress`" + `;
				}
			}
		}

		// Transition to needs_input: permission asked
		if (event.type === "permission.asked") {
			if (!(await isInReview())) {
				await ` + "$`crew edit {{.TaskID}} --status needs_input`" + `;
			}
		}
	}
  }
}
EOF

# Add exclude pattern to git (use git rev-parse for worktree support)
GIT_COMMON_DIR=$(git rev-parse --git-common-dir 2>/dev/null) && \
  echo ".opencode/plugin/crew-hooks.ts" >> "${GIT_COMMON_DIR}/info/exclude" || true
`
