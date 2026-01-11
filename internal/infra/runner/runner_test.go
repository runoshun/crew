package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Run_Success(t *testing.T) {
	dir := t.TempDir()
	client := NewClient()

	// Create a simple script that creates a file
	script := `echo "test" > output.txt`

	err := client.Run(dir, script)

	// Assert
	require.NoError(t, err)
	// Verify the script actually ran
	outputPath := filepath.Join(dir, "output.txt")
	assert.FileExists(t, outputPath)
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "test\n", string(content))
}

func TestClient_Run_ScriptError(t *testing.T) {
	dir := t.TempDir()
	client := NewClient()

	// Script that fails
	script := `exit 1`

	err := client.Run(dir, script)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execute script")
}

func TestClient_Run_InvalidCommand(t *testing.T) {
	dir := t.TempDir()
	client := NewClient()

	// Script with invalid command
	script := `nonexistent-command`

	err := client.Run(dir, script)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execute script")
}

func TestClient_Run_WorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	client := NewClient()

	// Script that prints current directory
	script := `pwd > output.txt`

	err := client.Run(dir, script)

	// Assert
	require.NoError(t, err)
	outputPath := filepath.Join(dir, "output.txt")
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	// The pwd output should match the dir
	assert.Contains(t, string(content), dir)
}
