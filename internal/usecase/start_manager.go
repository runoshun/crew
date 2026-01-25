// Package usecase contains application use cases.
package usecase

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// StartManagerInput contains the parameters for starting a manager.
type StartManagerInput struct {
	Name             string // Manager agent name
	Model            string // Model name override (optional)
	AdditionalPrompt string // Additional prompt to append (optional)
	Session          bool   // Start in tmux session (--session flag)
}

// StartManagerOutput contains the result of starting a manager.
type StartManagerOutput struct {
	Command     string // The full command to execute
	Prompt      string // The prompt content
	SessionName string // Session name (only set when Session=true)
}

// StartManager is the use case for starting a manager agent.
type StartManager struct {
	sessions     domain.SessionManager
	configLoader domain.ConfigLoader
	repoRoot     string
	gitDir       string
	crewDir      string
}

// NewStartManager creates a new StartManager use case.
func NewStartManager(
	sessions domain.SessionManager,
	configLoader domain.ConfigLoader,
	repoRoot string,
	gitDir string,
	crewDir string,
) *StartManager {
	return &StartManager{
		sessions:     sessions,
		configLoader: configLoader,
		repoRoot:     repoRoot,
		gitDir:       gitDir,
		crewDir:      crewDir,
	}
}

// Execute starts a manager with the given input.
// Returns the command and prompt to execute; the caller is responsible for
// actually running the command (e.g., via syscall.Exec).
// When Session=true, starts a tmux session and returns the session name.
func (uc *StartManager) Execute(ctx context.Context, in StartManagerInput) (*StartManagerOutput, error) {
	// Load config
	cfg, err := uc.configLoader.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Resolve manager agent by name
	name := in.Name
	if name == "" {
		name = cfg.AgentsConfig.DefaultManager
	}
	// Get agent configuration from enabled agents only
	agent, ok := cfg.EnabledAgents()[name]
	if !ok {
		// Check if agent exists but is disabled
		if _, exists := cfg.Agents[name]; exists {
			return nil, fmt.Errorf("agent %q is disabled: %w", name, domain.ErrAgentDisabled)
		}
		return nil, fmt.Errorf("agent %q: %w", name, domain.ErrAgentNotFound)
	}

	// Verify agent role
	if agent.Role != domain.RoleManager {
		return nil, fmt.Errorf("agent %q is not a manager (role: %s): %w", name, agent.Role, domain.ErrAgentRoleMismatch)
	}

	// Resolve model priority: CLI flag > agent config > builtin default
	model := in.Model
	if model == "" {
		model = agent.DefaultModel
	}

	// Build command data for template expansion
	cmdData := domain.CommandData{
		GitDir:   uc.gitDir,
		RepoRoot: uc.repoRoot,
		Model:    model,
	}

	// Build the default prompts
	// Priority: Agent.Prompt > AgentsConfig.ManagerPrompt > empty
	defaultSystemPrompt := domain.DefaultManagerSystemPrompt
	defaultPrompt := cfg.AgentsConfig.ManagerPrompt

	// Render command and prompt using Agent.RenderCommand
	result, err := agent.RenderCommand(cmdData, `"$PROMPT"`, defaultSystemPrompt, defaultPrompt)
	if err != nil {
		return nil, fmt.Errorf("render command: %w", err)
	}

	// Append additional prompt if provided
	finalPrompt := result.Prompt
	if in.AdditionalPrompt != "" {
		if result.Prompt != "" {
			finalPrompt = result.Prompt + "\n\n" + in.AdditionalPrompt
		} else {
			finalPrompt = in.AdditionalPrompt
		}
	}

	output := &StartManagerOutput{
		Command: result.Command,
		Prompt:  finalPrompt,
	}

	// If session mode, start a tmux session
	if in.Session {
		sessionName := domain.ManagerSessionName()

		// Check if session is already running
		running, err := uc.sessions.IsRunning(sessionName)
		if err != nil {
			return nil, fmt.Errorf("check session: %w", err)
		}
		if running {
			return nil, fmt.Errorf("manager session %q: %w", sessionName, domain.ErrSessionRunning)
		}

		// Generate script file
		scriptPath, err := uc.generateManagerScript(output)
		if err != nil {
			return nil, fmt.Errorf("generate script: %w", err)
		}

		// Start session
		if err := uc.sessions.Start(ctx, domain.StartSessionOptions{
			Name:      sessionName,
			Dir:       uc.repoRoot,
			Command:   scriptPath,
			TaskTitle: "Manager",
			TaskAgent: name,
			TaskID:    0, // Manager has no task ID
		}); err != nil {
			_ = os.Remove(scriptPath) // Cleanup on failure
			return nil, fmt.Errorf("start session: %w", err)
		}

		output.SessionName = sessionName
	}

	return output, nil
}

// generateManagerScript creates the manager script file.
func (uc *StartManager) generateManagerScript(out *StartManagerOutput) (string, error) {
	scriptsDir := filepath.Join(uc.crewDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0750); err != nil {
		return "", fmt.Errorf("create scripts directory: %w", err)
	}

	scriptPath := domain.ManagerScriptPath(uc.crewDir)
	script := out.BuildScript()

	// Write script file (executable)
	// G306: We intentionally use 0700 because this is an executable script
	if err := os.WriteFile(scriptPath, []byte(script), 0700); err != nil { //nolint:gosec // executable script requires execute permission
		return "", fmt.Errorf("write script file: %w", err)
	}

	return scriptPath, nil
}

// GetCommand returns the executable path and arguments for the manager command.
// This is a convenience method for callers that need to exec the command.
func (out *StartManagerOutput) GetCommand() (path string, args []string) {
	// Parse the command string into path and args
	// The command format is typically: command [args...] "$PROMPT"
	// We need to split this carefully
	parts := splitCommand(out.Command)
	if len(parts) == 0 {
		return "", []string{}
	}

	if len(parts) == 1 {
		return parts[0], []string{}
	}

	return parts[0], parts[1:]
}

// splitCommand splits a command string into parts, respecting quotes.
// This is a simplified version that handles the common cases.
func splitCommand(cmd string) []string {
	var parts []string
	var current bytes.Buffer
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]

		if inQuote {
			if c == quoteChar {
				inQuote = false
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteByte(c)
			}
		} else {
			switch c {
			case ' ', '\t':
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			case '"', '\'':
				inQuote = true
				quoteChar = c
			default:
				current.WriteByte(c)
			}
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// ManagerScriptData holds data for manager script template execution.
type ManagerScriptData struct {
	AgentCommand string
	Prompt       string
	Shell        string
	SessionName  string
	TaskCommand  string
}

// BuildScript creates a shell script that sets PROMPT and executes the command.
// This is used when the caller wants to write a script file instead of using syscall.Exec.
func (out *StartManagerOutput) BuildScript() string {
	shell := "/bin/bash"

	const scriptTemplate = `#!{{.Shell}}
set -o pipefail

# Session log (stderr only)
LOG="$(git rev-parse --git-common-dir)/crew/logs/{{.SessionName}}.log"
STARTED_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
{
  printf '================================================================================\n'
  printf 'Session: %s\n' '{{.SessionName}}'
  printf 'Started: %s\n' "$STARTED_AT"
  printf 'Directory: %s\n' "$PWD"
  printf 'Command: %s\n' '{{.TaskCommand}}'
  printf '================================================================================\n\n'
} >"$LOG"
exec 2>>"$LOG"

# Embedded prompt
read -r -d '' PROMPT << 'END_OF_PROMPT'
{{.Prompt}}
END_OF_PROMPT

# Run manager agent
{{.AgentCommand}}
`

	tmpl := template.Must(template.New("script").Parse(scriptTemplate))
	data := ManagerScriptData{
		AgentCommand: out.Command,
		Prompt:       out.Prompt,
		Shell:        shell,
		SessionName:  domain.ManagerSessionName(),
		TaskCommand:  out.Command,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return ""
	}

	return buf.String()
}
