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
		// Session idle: keep in_progress (input waiting is represented by in_progress)
		if (event.type === "session.idle") {
			await ` + "$`crew edit {{.TaskID}} --status in_progress --if-status in_progress`" + `;
		}

		// Session status change: keep in_progress
		if (event.type === "session.status") {
			if (event.status.type === "idle") {
				await ` + "$`crew edit {{.TaskID}} --status in_progress --if-status in_progress`" + `;
			} else if (event.status.type === "busy") {
				await ` + "$`crew edit {{.TaskID}} --status in_progress --if-status in_progress`" + `;
			}
		}

		// Permission asked: keep in_progress (input waiting is represented by in_progress)
		if (event.type === "permission.asked") {
			const { id, metadata } = event.properties;
			
			// Auto-approve safe git operations in worktree
			if (metadata && typeof metadata.command === 'string') {
				const command = metadata.command.trim();
				
				// Security: Deny chained commands (&, |, ;) to prevent injection
				const isSafeCommand = !/[\&\|;]/.test(command);
				
				// Security: Ensure command is running within the worktree
				// Note: {{.Worktree}} is injected by the Go template
				const cwd = metadata.cwd;
				const isSafeDir = typeof cwd === 'string' && cwd.startsWith("{{.Worktree}}");

				if (isSafeCommand && isSafeDir) {
					// Allow: git status, diff, log, add, commit
					if (/^git\s+(status|diff|log|add|commit)(\s+|$)/.test(command)) {
						await $.client.permission.reply({ requestID: id, reply: "once" });
						return;
					}
				}
			}

			await ` + "$`crew edit {{.TaskID}} --status in_progress --if-status in_progress`" + `;
		}

		// Permission replied: keep in_progress
		if (event.type === "permission.replied") {
			await ` + "$`crew edit {{.TaskID}} --status in_progress --if-status in_progress`" + `;
		}
	}
  }
}
EOF

# Add exclude pattern to git (use git rev-parse for worktree support)
GIT_COMMON_DIR=$(git rev-parse --git-common-dir 2>/dev/null) && \
  echo ".opencode/plugin/crew-hooks.ts" >> "${GIT_COMMON_DIR}/info/exclude" || true
`
