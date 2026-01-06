package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockInitializer is a test double for domain.StoreInitializer.
type mockInitializer struct {
	initErr       error
	initCalled    bool
	isInitialized bool
	repaired      bool
}

func (m *mockInitializer) Initialize() (bool, error) {
	m.initCalled = true
	return m.repaired, m.initErr
}

func (m *mockInitializer) IsInitialized() bool {
	return m.isInitialized
}

func TestInitRepo_Execute_Success(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".git", "crew")
	storePath := filepath.Join(crewDir, "tasks.json")

	// Create mock initializer
	mock := &mockInitializer{}
	uc := NewInitRepo(mock)

	// Execute
	out, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		StorePath: storePath,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, crewDir, out.CrewDir)
	assert.True(t, mock.initCalled, "Initialize should be called")

	// Verify directory structure
	assertDirExists(t, crewDir)
	assertDirExists(t, filepath.Join(crewDir, "scripts"))
	assertDirExists(t, filepath.Join(crewDir, "logs"))

	// Verify tmux.conf
	tmuxConf := filepath.Join(crewDir, "tmux.conf")
	assertFileExists(t, tmuxConf)
	content, err := os.ReadFile(tmuxConf)
	require.NoError(t, err)
	assert.Contains(t, string(content), "unbind-key -a")
	assert.Contains(t, string(content), "C-g detach-client")
}

func TestInitRepo_Execute_AlreadyInitialized(t *testing.T) {
	// Setup temp directory with existing crew dir
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".git", "crew")
	require.NoError(t, os.MkdirAll(crewDir, 0o750))

	mock := &mockInitializer{isInitialized: true}
	uc := NewInitRepo(mock)

	// Execute
	out, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		StorePath: filepath.Join(crewDir, "tasks.json"),
	})

	// Assert - now returns success with AlreadyInitialized flag
	require.NoError(t, err)
	assert.True(t, out.AlreadyInitialized)
	assert.True(t, mock.initCalled, "Initialize should be called for repair")
}

func TestInitRepo_Execute_InitializerError(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".git", "crew")

	// Create mock that returns error
	mock := &mockInitializer{initErr: assert.AnError}
	uc := NewInitRepo(mock)

	// Execute
	_, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		StorePath: filepath.Join(crewDir, "tasks.json"),
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initialize task store")
}

// Helper functions

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err, "directory should exist: %s", path)
	assert.True(t, info.IsDir(), "should be a directory: %s", path)
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err, "file should exist: %s", path)
	assert.False(t, info.IsDir(), "should be a file: %s", path)
}
