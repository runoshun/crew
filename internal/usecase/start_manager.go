// Package usecase contains application use cases.
package usecase

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// StartManagerInput contains the parameters for starting a manager.
type StartManagerInput struct {
	Name  string // Manager name
	Model string // Model name override (optional)
}

// StartManagerOutput contains the result of starting a manager.
type StartManagerOutput struct {
	Command string // The full command to execute
	Prompt  string // The prompt content
}

// StartManager is the use case for starting a manager agent.
type StartManager struct {
	configLoader domain.ConfigLoader
	repoRoot     string
	gitDir       string
}

// NewStartManager creates a new StartManager use case.
func NewStartManager(
	configLoader domain.ConfigLoader,
	repoRoot string,
	gitDir string,
) *StartManager {
	return &StartManager{
		configLoader: configLoader,
		repoRoot:     repoRoot,
		gitDir:       gitDir,
	}
}

// Execute starts a manager with the given input.
// Returns the command and prompt to execute; the caller is responsible for
// actually running the command (e.g., via syscall.Exec).
func (uc *StartManager) Execute(_ context.Context, in StartManagerInput) (*StartManagerOutput, error) {
	// Load config
	cfg, err := uc.configLoader.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Resolve manager by name
	name := in.Name
	if name == "" {
		name = cfg.ManagersConfig.Default
	}
	manager, ok := cfg.Managers[name]
	if !ok {
		return nil, fmt.Errorf("manager %q: %w", name, domain.ErrManagerNotFound)
	}

	// Resolve manager configuration (inherit from Agent/Worker as needed)
	resolvedManager, defaultModel, err := uc.resolveManager(manager, cfg)
	if err != nil {
		return nil, fmt.Errorf("resolve manager: %w", err)
	}

	// Resolve model priority: CLI flag > manager config > builtin default
	model := in.Model
	if model == "" && resolvedManager.Model != "" {
		model = resolvedManager.Model
	}
	if model == "" {
		model = defaultModel
	}

	// Build command data for template expansion
	cmdData := domain.CommandData{
		GitDir:   uc.gitDir,
		RepoRoot: uc.repoRoot,
		Model:    model,
	}

	// Build the system prompt with priority: manager > managers config > default
	defaultSystemPrompt := cfg.ManagersConfig.SystemPrompt

	// Build the user prompt with priority: manager > managers config
	defaultPrompt := cfg.ManagersConfig.Prompt

	// Render command and prompt using Manager.RenderCommand
	result, err := resolvedManager.RenderCommand(cmdData, `"$PROMPT"`, defaultSystemPrompt, defaultPrompt)
	if err != nil {
		return nil, fmt.Errorf("render command: %w", err)
	}

	return &StartManagerOutput{
		Command: result.Command,
		Prompt:  result.Prompt,
	}, nil
}

// resolveManager resolves a Manager configuration by inheriting from Agent/Worker as needed.
// Returns the resolved Manager with all fields populated, default model, and any error.
func (uc *StartManager) resolveManager(manager domain.Manager, cfg *domain.Config) (domain.Manager, string, error) {
	// Manager must have an Agent reference
	if manager.Agent == "" {
		return domain.Manager{}, "", fmt.Errorf("manager has no agent reference")
	}

	agentRef := manager.Agent
	resolved := manager // Copy the original manager

	// Inherit SystemArgs from builtin manager with same name as Agent if not set
	if resolved.SystemArgs == "" {
		if refManager, ok := cfg.Managers[agentRef]; ok {
			resolved.SystemArgs = refManager.SystemArgs
		}
	}

	// Get Command, CommandTemplate, and Args from Worker or Agent
	defaultModel := ""
	if worker, ok := cfg.Workers[agentRef]; ok {
		// Inherit from Worker
		if resolved.Command == "" {
			resolved.Command = worker.Command
		}
		if resolved.CommandTemplate == "" {
			resolved.CommandTemplate = worker.CommandTemplate
		}
		// Combine worker Args with manager Args (worker first, then manager)
		if worker.Args != "" {
			if resolved.Args != "" {
				resolved.Args = worker.Args + " " + resolved.Args
			} else {
				resolved.Args = worker.Args
			}
		}
		// Get default model from Agent definition
		if agentDef, ok := cfg.Agents[agentRef]; ok {
			defaultModel = agentDef.DefaultModel
		}
	} else if agentDef, ok := cfg.Agents[agentRef]; ok {
		// Inherit from Agent directly
		if resolved.Command == "" {
			resolved.Command = agentDef.Command
		}
		if resolved.CommandTemplate == "" {
			resolved.CommandTemplate = agentDef.CommandTemplate
		}
		defaultModel = agentDef.DefaultModel
	} else {
		return domain.Manager{}, "", fmt.Errorf("agent %q not found", agentRef)
	}

	return resolved, defaultModel, nil
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
}

// BuildScript creates a shell script that sets PROMPT and executes the command.
// This is used when the caller wants to write a script file instead of using syscall.Exec.
func (out *StartManagerOutput) BuildScript() string {
	shell := "/bin/bash"

	const scriptTemplate = `#!{{.Shell}}
set -o pipefail

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
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return ""
	}

	return buf.String()
}
