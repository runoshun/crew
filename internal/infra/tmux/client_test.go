package tmux

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnv creates a temporary directory for tmux socket and config.
func setupTestEnv(t *testing.T) (socketPath, crewDir string, cleanup func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tmux-test-*")
	require.NoError(t, err)

	crewDir = tmpDir
	socketPath = filepath.Join(tmpDir, "tmux.sock")

	// Create tmux.conf with minimal configuration
	configPath := filepath.Join(tmpDir, "tmux.conf")
	configContent := `# Minimal tmux config for testing
unbind-key -a
bind-key -n C-g detach-client
set -g status off
set -g escape-time 0
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cleanup = func() {
		// Kill any remaining sessions using this socket
		cmd := exec.Command("tmux", "-S", socketPath, "kill-server")
		_ = cmd.Run() // Ignore errors - server might not be running
		_ = os.RemoveAll(tmpDir)
	}

	return socketPath, crewDir, cleanup
}

func TestNewClient(t *testing.T) {
	socketPath := "/path/to/socket"
	crewDir := "/path/to/crew"

	client := NewClient(socketPath, crewDir)

	assert.Equal(t, socketPath, client.socketPath)
	assert.Equal(t, filepath.Join(crewDir, "tmux.conf"), client.configPath)
	assert.Equal(t, crewDir, client.crewDir)
}

func TestClient_Start_And_IsRunning(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)
	sessionName := "test-session"

	// Initially not running
	running, err := client.IsRunning(sessionName)
	require.NoError(t, err)
	assert.False(t, running)

	// Start session
	err = client.Start(context.Background(), domain.StartSessionOptions{
		Name:    sessionName,
		Dir:     crewDir, // Use crewDir as working directory
		Command: "sleep 60",
		TaskID:  1,
	})
	require.NoError(t, err)

	// Now it should be running
	running, err = client.IsRunning(sessionName)
	require.NoError(t, err)
	assert.True(t, running)
}

func TestClient_Start_AlreadyRunning(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)
	sessionName := "test-session"

	// Start first session
	err := client.Start(context.Background(), domain.StartSessionOptions{
		Name:    sessionName,
		Dir:     crewDir,
		Command: "sleep 60",
		TaskID:  1,
	})
	require.NoError(t, err)

	// Try to start again - should fail
	err = client.Start(context.Background(), domain.StartSessionOptions{
		Name:    sessionName,
		Dir:     crewDir,
		Command: "sleep 60",
		TaskID:  1,
	})
	assert.ErrorIs(t, err, domain.ErrSessionRunning)
}

func TestClient_Stop(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)
	sessionName := "test-session"

	// Start session
	err := client.Start(context.Background(), domain.StartSessionOptions{
		Name:    sessionName,
		Dir:     crewDir,
		Command: "sleep 60",
		TaskID:  1,
	})
	require.NoError(t, err)

	// Verify it's running
	running, err := client.IsRunning(sessionName)
	require.NoError(t, err)
	assert.True(t, running)

	// Stop session
	err = client.Stop(sessionName)
	require.NoError(t, err)

	// Verify it's stopped
	running, err = client.IsRunning(sessionName)
	require.NoError(t, err)
	assert.False(t, running)
}

func TestClient_Stop_NotRunning(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)

	// Stop non-existent session - should not error
	err := client.Stop("non-existent")
	assert.NoError(t, err)
}

func TestClient_Peek(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)
	sessionName := "test-session"

	// Start session that echoes something
	err := client.Start(context.Background(), domain.StartSessionOptions{
		Name:    sessionName,
		Dir:     crewDir,
		Command: "echo 'Hello World'; sleep 60",
		TaskID:  1,
	})
	require.NoError(t, err)

	// Wait a bit for the echo to complete
	time.Sleep(100 * time.Millisecond)

	// Peek at output
	output, err := client.Peek(sessionName, 10)
	require.NoError(t, err)
	assert.Contains(t, output, "Hello World")
}

func TestClient_Peek_NoSession(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)

	// Peek at non-existent session
	_, err := client.Peek("non-existent", 10)
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestClient_Send(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)
	sessionName := "test-session"

	// Start a bash session that reads input
	err := client.Start(context.Background(), domain.StartSessionOptions{
		Name:    sessionName,
		Dir:     crewDir,
		Command: "bash",
		TaskID:  1,
	})
	require.NoError(t, err)

	// Wait for bash to start
	time.Sleep(100 * time.Millisecond)

	// Send a command
	err = client.Send(sessionName, "echo 'Sent from test'")
	require.NoError(t, err)

	// Send Enter key
	err = client.Send(sessionName, "Enter")
	require.NoError(t, err)

	// Wait for command to execute
	time.Sleep(100 * time.Millisecond)

	// Verify the output
	output, err := client.Peek(sessionName, 20)
	require.NoError(t, err)
	assert.Contains(t, output, "Sent from test")
}

func TestClient_Send_NoSession(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)

	// Send to non-existent session
	err := client.Send("non-existent", "test")
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestClient_IsRunning_NoSocket(t *testing.T) {
	// Test with non-existent socket
	client := NewClient("/non/existent/socket", "/tmp")

	running, err := client.IsRunning("any-session")
	require.NoError(t, err)
	assert.False(t, running)
}

func TestClient_MultipleSessions(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)

	// Start multiple sessions with unique names
	sessionNames := []string{"crew-1", "crew-2", "crew-3"}
	for i, name := range sessionNames {
		err := client.Start(context.Background(), domain.StartSessionOptions{
			Name:    name,
			Dir:     crewDir,
			Command: "sleep 60",
			TaskID:  i + 1,
		})
		require.NoError(t, err)
	}

	// Verify all are running
	for _, name := range sessionNames {
		running, err := client.IsRunning(name)
		require.NoError(t, err)
		assert.True(t, running, "session %s should be running", name)
	}

	// Stop middle session
	err := client.Stop("crew-2")
	require.NoError(t, err)

	// Verify middle is stopped, others still running
	running, err := client.IsRunning("crew-1")
	require.NoError(t, err)
	assert.True(t, running)

	running, err = client.IsRunning("crew-2")
	require.NoError(t, err)
	assert.False(t, running)

	running, err = client.IsRunning("crew-3")
	require.NoError(t, err)
	assert.True(t, running)
}

func TestClient_Attach(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)
	sessionName := "crew-1"

	// Start session first
	err := client.Start(context.Background(), domain.StartSessionOptions{
		Name:    sessionName,
		Dir:     crewDir,
		Command: "sleep 60",
		TaskID:  1,
	})
	require.NoError(t, err)

	// Capture the exec call instead of actually executing
	var capturedPath string
	var capturedArgs []string
	client.SetExecFunc(func(argv0 string, argv []string, envv []string) error {
		capturedPath = argv0
		capturedArgs = argv
		return nil // Simulate successful exec (though it would normally not return)
	})

	// Call Attach
	err = client.Attach(sessionName)
	require.NoError(t, err)

	// Verify the exec arguments
	assert.Contains(t, capturedPath, "tmux")
	assert.Equal(t, []string{
		"tmux",
		"-S", socketPath,
		"-f", filepath.Join(crewDir, "tmux.conf"),
		"attach",
		"-t", sessionName,
	}, capturedArgs)
}

func TestClient_Attach_NoSession(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)

	// Attach to non-existent session
	err := client.Attach("non-existent")
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestClient_Attach_ExecError(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)
	sessionName := "crew-1"

	// Start session first
	err := client.Start(context.Background(), domain.StartSessionOptions{
		Name:    sessionName,
		Dir:     crewDir,
		Command: "sleep 60",
		TaskID:  1,
	})
	require.NoError(t, err)

	// Simulate exec failure
	execErr := os.ErrPermission
	client.SetExecFunc(func(argv0 string, argv []string, envv []string) error {
		return execErr
	})

	// Call Attach
	err = client.Attach(sessionName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attach session")
}
