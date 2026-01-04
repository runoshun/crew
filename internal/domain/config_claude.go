package domain

var claudeAgentConfig = BuiltinAgent{
	CommandTemplate:     "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
	Command:             "claude",
	SystemArgs:          "--model {{.Model}} --permission-mode acceptEdits " + claudeAllowedTools,
	DefaultArgs:         "",
	DefaultModel:        "opus",
	Description:         "Claude model via Anthropic CLI",
	WorktreeSetupScript: claudeSetupScript,
	ExcludePatterns:     []string{".claude/crew-plugin/"},
}

const claudeAllowedTools = `--allowedTools='Bash(git add:*) Bash(git commit:*) Bash(crew complete) Bash(crew show) Bash(crew edit:*)'`

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
            "command": "crew show {{.TaskID}} | grep -q '^Status: in_complete' || crew edit {{.TaskID}} --status needs_input"
          }
        ]
      }
    ]
  }
}
EOF
`
