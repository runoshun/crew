package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartManager_Execute_Success(t *testing.T) {
	repoRoot := t.TempDir()
	gitDir := repoRoot + "/.git"

	configLoader := testutil.NewMockConfigLoader()
	// Use builtin claude-manager agent (already resolved by NewMockConfigLoader)

	uc := NewStartManager(configLoader, repoRoot, gitDir)

	// Execute
	out, err := uc.Execute(context.Background(), StartManagerInput{
		Name: "claude-manager",
	})

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, out.Command)
	assert.NotEmpty(t, out.Prompt)
	assert.Contains(t, out.Command, "claude")
	// Prompt should contain the default manager system prompt
	assert.Contains(t, out.Prompt, "crew --help-manager")
}

func TestStartManager_Execute_ManagerNotFound(t *testing.T) {
	repoRoot := t.TempDir()
	gitDir := repoRoot + "/.git"

	configLoader := testutil.NewMockConfigLoader()
	// No managers configured

	uc := NewStartManager(configLoader, repoRoot, gitDir)

	// Execute
	_, err := uc.Execute(context.Background(), StartManagerInput{
		Name: "nonexistent",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrAgentNotFound)
}

func TestStartManager_Execute_ConfigLoadError(t *testing.T) {
	repoRoot := t.TempDir()
	gitDir := repoRoot + "/.git"

	configLoader := testutil.NewMockConfigLoader()
	configLoader.LoadErr = assert.AnError

	uc := NewStartManager(configLoader, repoRoot, gitDir)

	// Execute
	_, err := uc.Execute(context.Background(), StartManagerInput{
		Name: "default",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestStartManager_Execute_WithModelOverride(t *testing.T) {
	repoRoot := t.TempDir()
	gitDir := repoRoot + "/.git"

	configLoader := testutil.NewMockConfigLoader()
	// Configure a custom manager agent (fully resolved, no inheritance)
	configLoader.Config.Agents["custom-manager"] = domain.Agent{
		Role:            domain.RoleManager,
		CommandTemplate: "claude --model {{.Model}} {{.Args}} {{.Prompt}}",
		DefaultModel:    "config-model",
		SystemPrompt:    domain.DefaultManagerSystemPrompt,
	}

	uc := NewStartManager(configLoader, repoRoot, gitDir)

	// Execute with model override
	out, err := uc.Execute(context.Background(), StartManagerInput{
		Name:  "custom-manager",
		Model: "sonnet",
	})

	// Assert
	require.NoError(t, err)
	// CLI flag should take precedence
	assert.Contains(t, out.Command, "--model sonnet")
	assert.NotContains(t, out.Command, "--model config-model")
}

func TestStartManager_Execute_WithConfigModel(t *testing.T) {
	repoRoot := t.TempDir()
	gitDir := repoRoot + "/.git"

	configLoader := testutil.NewMockConfigLoader()
	// Configure a custom manager agent (fully resolved, no inheritance)
	configLoader.Config.Agents["custom-manager"] = domain.Agent{
		Role:            domain.RoleManager,
		CommandTemplate: "claude --model {{.Model}} {{.Args}} {{.Prompt}}",
		DefaultModel:    "config-model",
		SystemPrompt:    domain.DefaultManagerSystemPrompt,
	}

	uc := NewStartManager(configLoader, repoRoot, gitDir)

	// Execute without model override
	out, err := uc.Execute(context.Background(), StartManagerInput{
		Name: "custom-manager",
	})

	// Assert
	require.NoError(t, err)
	assert.Contains(t, out.Command, "--model config-model")
}

func TestStartManager_Execute_WithManagerArgs(t *testing.T) {
	repoRoot := t.TempDir()
	gitDir := repoRoot + "/.git"

	configLoader := testutil.NewMockConfigLoader()
	// Configure a custom manager agent (fully resolved, no inheritance)
	configLoader.Config.Agents["custom-manager"] = domain.Agent{
		Role:            domain.RoleManager,
		CommandTemplate: "claude {{.Args}} {{.Prompt}}",
		Args:            "--extra-flag",
		SystemPrompt:    domain.DefaultManagerSystemPrompt,
	}

	uc := NewStartManager(configLoader, repoRoot, gitDir)

	// Execute
	out, err := uc.Execute(context.Background(), StartManagerInput{
		Name: "custom-manager",
	})

	// Assert
	require.NoError(t, err)
	// Manager args should be present
	assert.Contains(t, out.Command, "--extra-flag")
}

func TestStartManager_Execute_WithBuiltinAgent(t *testing.T) {
	repoRoot := t.TempDir()
	gitDir := repoRoot + "/.git"

	configLoader := testutil.NewMockConfigLoader()
	// Use builtin manager agent (opencode-manager)

	uc := NewStartManager(configLoader, repoRoot, gitDir)

	// Execute
	out, err := uc.Execute(context.Background(), StartManagerInput{
		Name: "opencode-manager",
	})

	// Assert
	require.NoError(t, err)
	assert.Contains(t, out.Command, "opencode")
}

func TestStartManager_Execute_DefaultManager(t *testing.T) {
	repoRoot := t.TempDir()
	gitDir := repoRoot + "/.git"

	configLoader := testutil.NewMockConfigLoader()
	// Default manager is set to "opencode-manager" by builtin.Register

	uc := NewStartManager(configLoader, repoRoot, gitDir)

	// Execute with empty name (should use default)
	out, err := uc.Execute(context.Background(), StartManagerInput{
		Name: "", // Empty - should use default
	})

	// Assert
	require.NoError(t, err)
	assert.Contains(t, out.Command, "opencode")
}

func TestStartManagerOutput_GetCommand(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		expectedPath string
		expectedArgs []string
	}{
		{
			name:         "simple command",
			command:      "claude",
			expectedPath: "claude",
			expectedArgs: []string{},
		},
		{
			name:         "command with args",
			command:      `claude --model opus "$PROMPT"`,
			expectedPath: "claude",
			expectedArgs: []string{"--model", "opus", "$PROMPT"},
		},
		{
			name:         "command with quoted args",
			command:      `opencode -m "gpt-4o" --prompt "$PROMPT"`,
			expectedPath: "opencode",
			expectedArgs: []string{"-m", "gpt-4o", "--prompt", "$PROMPT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &StartManagerOutput{
				Command: tt.command,
				Prompt:  "test prompt",
			}

			path, args := out.GetCommand()

			assert.Equal(t, tt.expectedPath, path)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestStartManagerOutput_BuildScript(t *testing.T) {
	out := &StartManagerOutput{
		Command: "claude --model opus \"$PROMPT\"",
		Prompt:  "This is the manager prompt.",
	}

	script := out.BuildScript()

	// Assert script structure
	assert.Contains(t, script, "#!/bin/bash")
	assert.Contains(t, script, "set -o pipefail")
	assert.Contains(t, script, "END_OF_PROMPT")
	assert.Contains(t, script, "This is the manager prompt.")
	assert.Contains(t, script, "claude --model opus")
}

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected []string
	}{
		{
			name:     "simple",
			cmd:      "cmd",
			expected: []string{"cmd"},
		},
		{
			name:     "with spaces",
			cmd:      "cmd arg1 arg2",
			expected: []string{"cmd", "arg1", "arg2"},
		},
		{
			name:     "with double quotes",
			cmd:      `cmd "arg with spaces" arg2`,
			expected: []string{"cmd", "arg with spaces", "arg2"},
		},
		{
			name:     "with single quotes",
			cmd:      `cmd 'arg with spaces' arg2`,
			expected: []string{"cmd", "arg with spaces", "arg2"},
		},
		{
			name:     "mixed quotes",
			cmd:      `cmd "double" 'single' normal`,
			expected: []string{"cmd", "double", "single", "normal"},
		},
		{
			name:     "prompt variable",
			cmd:      `claude "$PROMPT"`,
			expected: []string{"claude", "$PROMPT"},
		},
		{
			name:     "empty string",
			cmd:      "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCommand(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}
