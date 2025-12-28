//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Close Command Integration Tests
// =============================================================================

func TestIntegration_Close(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	// Create a task
	crewMust(t, dir, "new", "--title", "Task to close")

	// Close the task
	out := crewMust(t, dir, "close", "1")
	assert.Contains(t, out, "Closed task #1")

	// Verify task is closed
	out = crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Status: closed")
}

func TestIntegration_Close_NotFound(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	_, err := crew(t, dir, "close", "999")
	assert.Error(t, err)
}

func TestIntegration_Close_AlreadyClosed(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	crewMust(t, dir, "new", "--title", "Task")
	crewMust(t, dir, "close", "1")

	// Closing again should fail
	_, err := crew(t, dir, "close", "1")
	assert.Error(t, err)
}

// =============================================================================
// Session Commands Integration Tests (require tmux)
// =============================================================================

// cleanupTmuxSocket removes the tmux socket after the test.
func cleanupTmuxSocket(t *testing.T, dir string) {
	t.Helper()
	socketPath := filepath.Join(dir, ".git", "crew", "tmux.sock")
	// Kill any sessions using this socket
	cmd := exec.Command("tmux", "-S", socketPath, "kill-server")
	_ = cmd.Run() // Ignore errors if no server running
}

func TestIntegration_Start_WithAgent(t *testing.T) {
	dir := testRepo(t)
	t.Cleanup(func() { cleanupTmuxSocket(t, dir) })

	crewMust(t, dir, "init")
	crewMust(t, dir, "new", "--title", "Start test task")

	// Start with "echo" as a simple agent that exits immediately
	out := crewMust(t, dir, "start", "1", "echo")
	assert.Contains(t, out, "Started task #1")
	assert.Contains(t, out, "session: crew-1")

	// Verify task status changed to in_progress
	out = crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Status: in_progress")
	assert.Contains(t, out, "Agent: echo")
}

func TestIntegration_Start_WithDefaultAgent(t *testing.T) {
	dir := testRepo(t)
	t.Cleanup(func() { cleanupTmuxSocket(t, dir) })

	crewMust(t, dir, "init")

	// Create config with default_agent
	crewDir := filepath.Join(dir, ".git", "crew")
	configPath := filepath.Join(crewDir, "config.toml")
	configContent := `default_agent = "echo"
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	crewMust(t, dir, "new", "--title", "Default agent test")

	// Start without specifying agent - should use default_agent from config
	out := crewMust(t, dir, "start", "1")
	assert.Contains(t, out, "Started task #1")

	// Verify task uses default agent
	out = crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Agent: echo")
}

func TestIntegration_Start_NoAgentNoConfig(t *testing.T) {
	dir := testRepo(t)
	t.Cleanup(func() { cleanupTmuxSocket(t, dir) })

	crewMust(t, dir, "init")
	crewMust(t, dir, "new", "--title", "No agent test")

	// Start without agent and no default_agent in config should fail
	_, err := crew(t, dir, "start", "1")
	assert.Error(t, err)
}

func TestIntegration_Start_TaskNotFound(t *testing.T) {
	dir := testRepo(t)

	crewMust(t, dir, "init")

	_, err := crew(t, dir, "start", "999", "echo")
	assert.Error(t, err)
}

func TestIntegration_Start_InvalidStatus(t *testing.T) {
	dir := testRepo(t)
	t.Cleanup(func() { cleanupTmuxSocket(t, dir) })

	crewMust(t, dir, "init")
	crewMust(t, dir, "new", "--title", "Closed task")
	crewMust(t, dir, "close", "1")

	// Starting a closed task should fail
	_, err := crew(t, dir, "start", "1", "echo")
	assert.Error(t, err)
}

func TestIntegration_Attach_NoSession(t *testing.T) {
	dir := testRepo(t)

	crewMust(t, dir, "init")
	crewMust(t, dir, "new", "--title", "No session task")

	// Attach without starting should fail
	_, err := crew(t, dir, "attach", "1")
	assert.Error(t, err)
}

func TestIntegration_Attach_TaskNotFound(t *testing.T) {
	dir := testRepo(t)

	crewMust(t, dir, "init")

	_, err := crew(t, dir, "attach", "999")
	assert.Error(t, err)
}

// =============================================================================
// Session Workflow Integration Test
// =============================================================================

func TestIntegration_SessionWorkflow(t *testing.T) {
	dir := testRepo(t)
	t.Cleanup(func() { cleanupTmuxSocket(t, dir) })

	crewMust(t, dir, "init")

	// Create config with default agent
	crewDir := filepath.Join(dir, ".git", "crew")
	configPath := filepath.Join(crewDir, "config.toml")
	configContent := `default_agent = "true"
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Create task
	crewMust(t, dir, "new", "--title", "Workflow test", "--desc", "Test session workflow")

	// Verify initial status
	out := crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Status: todo")

	// Start the task (uses default_agent "true" which exits immediately with code 0)
	out = crewMust(t, dir, "start", "1")
	assert.Contains(t, out, "Started task #1")

	// Verify status changed
	out = crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Status: in_progress")
	assert.Contains(t, out, "Agent: true")
}
