package config

const (
	claudeAllowedToolsForWorker  = `--allowedTools='Bash(git add:*) Bash(git commit:*) Bash(crew complete) Bash(crew show) Bash(crew edit:*)'`
	claudeAllowedToolsForManager = `--allowedTools='Bash(crew:*)'`
)

// claudeAgents contains the built-in configuration for the Claude CLI.
var claudeAgents = builtinAgentSet{
	Worker: builtinAgentDef{
		CommandTemplate:   "claude --model {{.Model}} --permission-mode acceptEdits " + claudeAllowedToolsForWorker + " {{.Args}}{{if .Continue}} -c{{end}} {{.Prompt}}",
		DefaultModel:      "opus",
		Description:       "Claude model via Anthropic CLI",
		WorkerSetupScript: claudeSetupScript,
	},
	Manager: builtinAgentDef{
		Description: "Claude manager agent for task orchestration",
	},
	Reviewer: builtinAgentDef{
		// Non-interactive mode: -p (print) for synchronous execution
		CommandTemplate: "claude -p --model {{.Model}} {{.Args}} {{.Prompt}}",
		DefaultModel:    "sonnet",
		Description:     "Code review agent via Claude CLI",
	},
}

const claudeSetupScript = `#!/bin/bash
cd {{.Worktree}}

PLUGIN_DIR=.claude/crew-plugin

mkdir -p ${PLUGIN_DIR}/.claude-plugin
cat > ${PLUGIN_DIR}/plugin.json << 'EOF'
{
  "name": "crew-claude-worker-plugin",
  "description": "plugin for crew worker",
  "version": "0.1.0",
  "author": {
  	"name": "runoshun"
  }
}
EOF

mkdir -p ${PLUGIN_DIR}/hooks
cat > ${PLUGIN_DIR}/hooks/hooks.json << 'EOF'
{
  "hooks": {
    "Notification": [
      {
        "matcher": "permission_prompt|idle_prompt",
        "hooks": [
          {
            "type": "command",
            "command": "crew show {{.TaskID}} | grep -q '^Status: in_review' || crew edit {{.TaskID}} --status needs_input"
          }
        ]
      }
    ],
	  "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "crew edit {{.TaskID}} --status in_progress"
          }
        ]
      }
		]
  }
}
EOF

# Add exclude pattern to git (use git rev-parse for worktree support)
GIT_DIR=$(git rev-parse --git-dir 2>/dev/null) && \
  echo ".claude/crew-plugin/" >> "${GIT_DIR}/info/exclude" || true
`
