package cli

import (
	"testing"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestNewRootCommand_NoArgs_LaunchesTUI(t *testing.T) {
	// Save original function and restore after test
	originalFunc := launchTUIFunc
	defer func() {
		launchTUIFunc = originalFunc
	}()

	// Mock launchTUIFunc to track if it was called
	called := false
	launchTUIFunc = func(c *app.Container) error {
		called = true
		return nil
	}

	// Create root command with nil container (not used in this test)
	root := NewRootCommand(nil, "test-version")

	// Execute root command without arguments
	root.SetArgs([]string{})
	err := root.Execute()

	// Verify launchTUIFunc was called
	assert.NoError(t, err)
	assert.True(t, called, "launchTUIFunc should be called when no arguments are provided")
}

func TestNewRootCommand_WithHelp_ShowsHelp(t *testing.T) {
	// Save original function and restore after test
	originalFunc := launchTUIFunc
	defer func() {
		launchTUIFunc = originalFunc
	}()

	// Mock launchTUIFunc to ensure it's NOT called
	called := false
	launchTUIFunc = func(c *app.Container) error {
		called = true
		return nil
	}

	// Create root command with nil container
	root := NewRootCommand(nil, "test-version")

	// Execute root command with --help
	root.SetArgs([]string{"--help"})
	err := root.Execute()

	// Verify launchTUIFunc was NOT called (help is handled by cobra)
	// Note: --help causes cobra to exit early, so we expect an error or successful help display
	// In practice, cobra's --help doesn't return an error, it just displays help and returns nil
	assert.NoError(t, err)
	assert.False(t, called, "launchTUIFunc should NOT be called when --help is provided")
}
