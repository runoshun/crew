package config

import "github.com/runoshun/git-crew/v2/internal/domain"

const (
	claudeAllowedToolsForWorker  = `--allowedTools='Bash(git add:*) Bash(git commit:*) Bash(crew complete) Bash(crew show) Bash(crew edit:*) Bash(crew --help-worker) Skill(dev-workflow)'`
	claudeAllowedToolsForManager = `--allowedTools='Bash(crew:*)'`
)

// claudeAgents contains the built-in configuration for the Claude CLI.
var claudeAgents = builtinAgentSet{
	Worker: domain.Agent{
		CommandTemplate: "claude --model {{.Model}} --permission-mode acceptEdits --plugin-dir .claude/crew-plugin " + claudeAllowedToolsForWorker + " {{.Args}}{{if .Continue}} -c{{end}} {{.Prompt}}",
		DefaultModel:    "opus",
		Description:     "Claude model via Anthropic CLI",
		SetupScript:     claudeSetupScript,
	},
	Manager: domain.Agent{
		CommandTemplate: "claude --model {{.Model}} " + claudeAllowedToolsForManager + " {{.Args}} {{.Prompt}}",
		Description:     "Claude manager agent for task orchestration",
	},
	Reviewer: domain.Agent{
		// Non-interactive mode: -p (print) for synchronous execution
		CommandTemplate: "claude -p --model {{.Model}} {{.Args}} {{.Prompt}}",
		DefaultModel:    "opus",
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
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "jq -r '(.cwd) as $cwd | .tool_input.file_path // \"\" | if startswith($cwd) then {permissionDecision: \"allow\"} else {} end'"
          }
        ]
      }
    ],
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
GIT_COMMON_DIR=$(git rev-parse --git-common-dir 2>/dev/null) && \
  echo ".claude/crew-plugin/" >> "${GIT_COMMON_DIR}/info/exclude" || true
`
