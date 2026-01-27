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

func TestInitRepo_Execute_GitignoreNeedsAdd(t *testing.T) {
	// Setup temp directory without .gitignore
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".crew")
	storePath := filepath.Join(crewDir, "tasks.json")

	mock := &mockInitializer{}
	uc := NewInitRepo(mock)

	// Execute
	out, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		RepoRoot:  tmpDir,
		StorePath: storePath,
	})

	// Assert
	require.NoError(t, err)
	assert.True(t, out.GitignoreNeedsAdd, "GitignoreNeedsAdd should be true when .gitignore doesn't exist")
}

func TestInitRepo_Execute_GitignoreAlreadyContainsCrew(t *testing.T) {
	// Setup temp directory with .gitignore containing .crew/
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".crew")
	storePath := filepath.Join(crewDir, "tasks.json")
	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	require.NoError(t, os.WriteFile(gitignorePath, []byte(".crew/\n"), 0o644))

	mock := &mockInitializer{}
	uc := NewInitRepo(mock)

	// Execute
	out, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		RepoRoot:  tmpDir,
		StorePath: storePath,
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.GitignoreNeedsAdd, "GitignoreNeedsAdd should be false when .crew/ is in .gitignore")
}

func TestInitRepo_Execute_GitignoreContainsCrewWithoutSlash(t *testing.T) {
	// Setup temp directory with .gitignore containing .crew (without slash)
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".crew")
	storePath := filepath.Join(crewDir, "tasks.json")
	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	require.NoError(t, os.WriteFile(gitignorePath, []byte(".crew\n"), 0o644))

	mock := &mockInitializer{}
	uc := NewInitRepo(mock)

	// Execute
	out, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		RepoRoot:  tmpDir,
		StorePath: storePath,
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.GitignoreNeedsAdd, "GitignoreNeedsAdd should be false when .crew is in .gitignore")
}

func TestInitRepo_Execute_GitignoreWithComment(t *testing.T) {
	// Setup temp directory with .gitignore containing comment before .crew/
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".crew")
	storePath := filepath.Join(crewDir, "tasks.json")
	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	require.NoError(t, os.WriteFile(gitignorePath, []byte("# crew directory\n.crew/\n"), 0o644))

	mock := &mockInitializer{}
	uc := NewInitRepo(mock)

	// Execute
	out, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		RepoRoot:  tmpDir,
		StorePath: storePath,
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.GitignoreNeedsAdd, "GitignoreNeedsAdd should be false when .crew/ is in .gitignore with comment")
}

func TestInitRepo_Execute_GitignoreWithAnchoredPattern(t *testing.T) {
	// Setup temp directory with .gitignore containing anchored pattern /.crew/
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".crew")
	storePath := filepath.Join(crewDir, "tasks.json")
	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	require.NoError(t, os.WriteFile(gitignorePath, []byte("/.crew/\n"), 0o644))

	mock := &mockInitializer{}
	uc := NewInitRepo(mock)

	// Execute
	out, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		RepoRoot:  tmpDir,
		StorePath: storePath,
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.GitignoreNeedsAdd, "GitignoreNeedsAdd should be false when /.crew/ is in .gitignore")
}

func TestInitRepo_Execute_GitignoreWithTrailingWhitespace(t *testing.T) {
	// Setup temp directory with .gitignore containing .crew with trailing space
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".crew")
	storePath := filepath.Join(crewDir, "tasks.json")
	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	require.NoError(t, os.WriteFile(gitignorePath, []byte(".crew  \n"), 0o644))

	mock := &mockInitializer{}
	uc := NewInitRepo(mock)

	// Execute
	out, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		RepoRoot:  tmpDir,
		StorePath: storePath,
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.GitignoreNeedsAdd, "GitignoreNeedsAdd should be false when .crew with trailing whitespace is in .gitignore")
}

func TestInitRepo_Execute_GitignoreWithWindowsLineEndings(t *testing.T) {
	// Setup temp directory with .gitignore containing CRLF line endings
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".crew")
	storePath := filepath.Join(crewDir, "tasks.json")
	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	require.NoError(t, os.WriteFile(gitignorePath, []byte("node_modules\r\n.crew/\r\n"), 0o644))

	mock := &mockInitializer{}
	uc := NewInitRepo(mock)

	// Execute
	out, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		RepoRoot:  tmpDir,
		StorePath: storePath,
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.GitignoreNeedsAdd, "GitignoreNeedsAdd should be false when .crew/ is in .gitignore with CRLF")
}

func TestInitRepo_Execute_AlreadyInitializedButGitignoreMissing(t *testing.T) {
	// Setup temp directory without .gitignore, already initialized (migration case)
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".crew")
	storePath := filepath.Join(crewDir, "tasks.json")

	// Create .crew directory (simulating already initialized)
	require.NoError(t, os.MkdirAll(crewDir, 0o750))

	mock := &mockInitializer{isInitialized: true}
	uc := NewInitRepo(mock)

	// Execute
	out, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		RepoRoot:  tmpDir,
		StorePath: storePath,
	})

	// Assert - even when already initialized, GitignoreNeedsAdd should be true
	require.NoError(t, err)
	assert.True(t, out.AlreadyInitialized)
	assert.True(t, out.GitignoreNeedsAdd, "GitignoreNeedsAdd should be true even when already initialized")
}

func TestInitRepo_Execute_GitignoreWithLeadingWhitespace(t *testing.T) {
	// Setup temp directory with .gitignore containing .crew with leading whitespace
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, ".crew")
	storePath := filepath.Join(crewDir, "tasks.json")
	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	require.NoError(t, os.WriteFile(gitignorePath, []byte("  .crew/\n"), 0o644))

	mock := &mockInitializer{}
	uc := NewInitRepo(mock)

	// Execute
	out, err := uc.Execute(context.Background(), InitRepoInput{
		CrewDir:   crewDir,
		RepoRoot:  tmpDir,
		StorePath: storePath,
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.GitignoreNeedsAdd, "GitignoreNeedsAdd should be false when .crew/ with leading whitespace is in .gitignore")
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
