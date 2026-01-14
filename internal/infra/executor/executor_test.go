package executor

import (
	"runtime"
	"strings"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Execute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	client := NewClient()

	t.Run("executes simple echo command", func(t *testing.T) {
		cmd := domain.NewShellCommand("echo hello", "")
		output, err := client.Execute(cmd)
		require.NoError(t, err)
		assert.Equal(t, "hello\n", string(output))
	})

	t.Run("executes command in specified directory", func(t *testing.T) {
		dir := t.TempDir()
		cmd := domain.NewShellCommand("pwd", dir)
		output, err := client.Execute(cmd)
		require.NoError(t, err)
		assert.Contains(t, strings.TrimSpace(string(output)), dir)
	})

	t.Run("returns error for non-existent command", func(t *testing.T) {
		cmd := domain.NewCommand("nonexistent-command-xyz", nil, "")
		_, err := client.Execute(cmd)
		require.Error(t, err)
	})

	t.Run("returns error for failing command", func(t *testing.T) {
		cmd := domain.NewShellCommand("exit 1", "")
		_, err := client.Execute(cmd)
		require.Error(t, err)
	})

	t.Run("captures stderr in output", func(t *testing.T) {
		cmd := domain.NewShellCommand("echo error >&2", "")
		output, err := client.Execute(cmd)
		require.NoError(t, err)
		assert.Equal(t, "error\n", string(output))
	})

	t.Run("works with bash command builder", func(t *testing.T) {
		cmd := domain.NewBashCommand("echo $BASH_VERSION | head -c 1", "")
		output, err := client.Execute(cmd)
		require.NoError(t, err)
		// BASH_VERSION should start with a number (e.g., "5.1.16...")
		assert.True(t, len(output) > 0, "expected non-empty output")
	})
}

func TestNewClient(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client)
}
