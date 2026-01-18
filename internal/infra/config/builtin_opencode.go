package config

import "github.com/runoshun/git-crew/v2/internal/domain"

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
		// Transition to needs_input: session idle
		if (event.type === "session.idle") {
			await ` + "$`crew edit {{.TaskID}} --status needs_input --if-status in_progress`" + `;
		}

		// Transition to needs_input or in_progress: session status change
		if (event.type === "session.status") {
			if (event.status.type === "idle") {
				await ` + "$`crew edit {{.TaskID}} --status needs_input --if-status in_progress`" + `;
			} else if (event.status.type === "busy") {
				await ` + "$`crew edit {{.TaskID}} --status in_progress --if-status needs_input`" + `;
			}
		}

		// Transition to needs_input: permission asked
		if (event.type === "permission.asked") {
			const { id, metadata } = event.properties;
			
			// Auto-approve safe git operations in worktree
			// metadata.command contains the shell command being executed
			if (metadata && typeof metadata.command === 'string') {
				const command = metadata.command.trim();
				// Allow: git status, diff, log, add, commit
				// Deny: push, reset --hard, clean -fd (by exclusion/not matching)
				if (/^git\s+(status|diff|log|add|commit)(\s+|$)/.test(command)) {
					await $.client.permission.reply({ requestID: id, reply: "once" });
					return;
				}
			}

			await ` + "$`crew edit {{.TaskID}} --status needs_input --if-status in_progress`" + `;
		}

		// Transition to in_progress: permission replied
		if (event.type === "permission.replied") {
			await ` + "$`crew edit {{.TaskID}} --status in_progress --if-status needs_input`" + `;
		}
	}
  }
}
EOF

# Add exclude pattern to git (use git rev-parse for worktree support)
GIT_COMMON_DIR=$(git rev-parse --git-common-dir 2>/dev/null) && \
  echo ".opencode/plugin/crew-hooks.ts" >> "${GIT_COMMON_DIR}/info/exclude" || true
`
