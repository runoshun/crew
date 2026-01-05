package builtin

const (
	claudeAllowedToolsForWorker  = `--allowedTools='Bash(git add:*) Bash(git commit:*) Bash(crew complete) Bash(crew show) Bash(crew edit:*)'`
	claudeAllowedToolsForManager = `--allowedTools='Bash(crew:*)'`
)

// claudeAgent contains the built-in configuration for the Claude CLI.
var claudeAgent = agentConfig{
	Command:           "claude",
	CommandTemplate:   "{{.Command}} --model {{.Model}} {{.SystemArgs}} {{.Args}}{{if .Continue}} -c{{end}} {{.Prompt}}",
	WorkerSystemArgs:  "--permission-mode acceptEdits " + claudeAllowedToolsForWorker,
	ManagerSystemArgs: claudeAllowedToolsForManager,
	DefaultArgs:       "",
	DefaultModel:      "opus",
	Description:       "Claude model via Anthropic CLI",
	// WorktreeSetupScript and ExcludePatterns are Worker-only (Managers don't use worktrees)
	WorktreeSetupScript: claudeSetupScript,
	ExcludePatterns:     []string{".claude/crew-plugin/"},
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
            "command": "crew show {{.TaskID}} | grep -q '^Status: in_complete' || crew edit {{.TaskID}} --status needs_input"
          }
        ]
      }
    ]
  }
}
EOF
`
