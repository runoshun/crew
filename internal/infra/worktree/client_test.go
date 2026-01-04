package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a temporary git repository for testing.
func setupTestRepo(t *testing.T) (repoRoot, worktreeDir string, cleanup func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "worktree-test-*")
	require.NoError(t, err)

	repoRoot = filepath.Join(tmpDir, "repo")
	worktreeDir = filepath.Join(tmpDir, "worktrees")

	// Create directories
	require.NoError(t, os.MkdirAll(repoRoot, 0755))
	require.NoError(t, os.MkdirAll(worktreeDir, 0755))

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoRoot
	require.NoError(t, cmd.Run())

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoRoot
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoRoot
	require.NoError(t, cmd.Run())

	// Create initial commit (required for worktrees)
	testFile := filepath.Join(repoRoot, "README.md")
	require.NoError(t, os.WriteFile(testFile, []byte("# Test"), 0644))

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoRoot
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoRoot
	require.NoError(t, cmd.Run())

	cleanup = func() {
		_ = os.RemoveAll(tmpDir)
	}

	return repoRoot, worktreeDir, cleanup
}

func TestClient_Create_NewBranch(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Create worktree with new branch (crew branch format)
	path, err := client.Create("crew-1", "main")

	require.NoError(t, err)
	// Directory should be named by task ID, not branch name
	assert.Equal(t, filepath.Join(worktreeDir, "1"), path)

	// Verify worktree exists
	exists, err := client.Exists("crew-1")
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify directory exists
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestClient_Create_ExistingBranch(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a crew branch first
	cmd := exec.Command("git", "branch", "crew-2")
	cmd.Dir = repoRoot
	require.NoError(t, cmd.Run())

	client := NewClient(repoRoot, worktreeDir)

	// Create worktree for existing branch
	path, err := client.Create("crew-2", "main")

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(worktreeDir, "2"), path)

	// Verify worktree exists
	exists, err := client.Exists("crew-2")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestClient_Create_AlreadyExists(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Create worktree first time
	path1, err := client.Create("crew-1", "main")
	require.NoError(t, err)

	// Create again - should return same path without error
	path2, err := client.Create("crew-1", "main")
	require.NoError(t, err)
	assert.Equal(t, path1, path2)
}

func TestClient_Create_InvalidBranchName(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Try to create with non-crew branch name
	_, err := client.Create("feature-1", "main")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid crew branch name")
}

func TestClient_Create_WithGitHubIssue(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Create worktree with crew-N-gh-M format
	path, err := client.Create("crew-5-gh-123", "main")

	require.NoError(t, err)
	// Directory should be named by task ID (5), not the full branch name
	assert.Equal(t, filepath.Join(worktreeDir, "5"), path)
}

func TestClient_Resolve(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Create worktree
	expectedPath, err := client.Create("crew-1", "main")
	require.NoError(t, err)

	// Resolve it
	path, err := client.Resolve("crew-1")
	require.NoError(t, err)
	assert.Equal(t, expectedPath, path)
}

func TestClient_Resolve_NotFound(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Try to resolve non-existent worktree
	_, err := client.Resolve("non-existent")

	assert.ErrorIs(t, err, domain.ErrWorktreeNotFound)
}

func TestClient_Remove(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Create worktree
	path, err := client.Create("crew-1", "main")
	require.NoError(t, err)

	// Verify it exists
	exists, err := client.Exists("crew-1")
	require.NoError(t, err)
	assert.True(t, exists)

	// Remove it
	err = client.Remove("crew-1")
	require.NoError(t, err)

	// Verify it's gone
	exists, err = client.Exists("crew-1")
	require.NoError(t, err)
	assert.False(t, exists)

	// Verify directory is gone
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestClient_Remove_WithUncommittedChanges(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Create worktree
	path, err := client.Create("crew-1", "main")
	require.NoError(t, err)

	// Create uncommitted changes in the worktree
	testFile := filepath.Join(path, "dirty.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("uncommitted"), 0644))

	// Try to remove - should fail with ErrUncommittedChanges
	err = client.Remove("crew-1")
	assert.ErrorIs(t, err, domain.ErrUncommittedChanges)

	// Verify worktree still exists
	exists, err := client.Exists("crew-1")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestClient_Remove_NotFound(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Try to remove non-existent worktree
	err := client.Remove("non-existent")

	assert.ErrorIs(t, err, domain.ErrWorktreeNotFound)
}

func TestClient_List(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Create multiple worktrees
	_, err := client.Create("crew-1", "main")
	require.NoError(t, err)
	_, err = client.Create("crew-2", "main")
	require.NoError(t, err)

	// List worktrees
	worktrees, err := client.List()
	require.NoError(t, err)

	// Should have main repo + 2 worktrees = 3 entries
	assert.Len(t, worktrees, 3)

	// Verify our worktrees are in the list
	branches := make(map[string]bool)
	for _, wt := range worktrees {
		branches[wt.Branch] = true
	}
	assert.True(t, branches["crew-1"])
	assert.True(t, branches["crew-2"])
}

func TestClient_Exists(t *testing.T) {
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Initially doesn't exist
	exists, err := client.Exists("crew-1")
	require.NoError(t, err)
	assert.False(t, exists)

	// Create it
	_, err = client.Create("crew-1", "main")
	require.NoError(t, err)

	// Now it exists
	exists, err = client.Exists("crew-1")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestParseWorktreeList(t *testing.T) {
	input := `worktree /path/to/main
HEAD abc123def456
branch refs/heads/main

worktree /path/to/feature
HEAD def456abc123
branch refs/heads/feature-branch

`

	worktrees, err := parseWorktreeList(input)

	require.NoError(t, err)
	require.Len(t, worktrees, 2)

	assert.Equal(t, "/path/to/main", worktrees[0].Path)
	assert.Equal(t, "main", worktrees[0].Branch)

	assert.Equal(t, "/path/to/feature", worktrees[1].Path)
	assert.Equal(t, "feature-branch", worktrees[1].Branch)
}

func TestParseWorktreeList_Empty(t *testing.T) {
	worktrees, err := parseWorktreeList("")

	require.NoError(t, err)
	assert.Empty(t, worktrees)
}

func TestParseWorktreeList_DetachedHead(t *testing.T) {
	// Detached HEAD doesn't have a branch line
	input := `worktree /path/to/detached
HEAD abc123def456
detached

`

	worktrees, err := parseWorktreeList(input)

	require.NoError(t, err)
	require.Len(t, worktrees, 1)
	assert.Equal(t, "/path/to/detached", worktrees[0].Path)
	assert.Equal(t, "", worktrees[0].Branch) // No branch for detached HEAD
}

func TestClient_Create_OrphanedWorktree(t *testing.T) {
	// This tests the scenario where:
	// 1. A worktree was created
	// 2. The worktree directory was manually deleted (or by external tool)
	// 3. Git still has the worktree registered
	// 4. Attempting to create a new worktree for the same branch should auto-recover

	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	client := NewClient(repoRoot, worktreeDir)

	// Create a worktree
	path, err := client.Create("crew-1", "main")
	require.NoError(t, err)
	expectedPath := filepath.Join(worktreeDir, "1")
	assert.Equal(t, expectedPath, path)

	// Verify it exists
	exists, err := client.Exists("crew-1")
	require.NoError(t, err)
	assert.True(t, exists)

	// Simulate orphaned worktree: manually remove the directory
	// but leave git's worktree registration intact
	err = os.RemoveAll(path)
	require.NoError(t, err)

	// Verify directory is gone but still registered
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))

	// Try to create worktree again - should auto-recover via prune
	path2, err := client.Create("crew-1", "main")
	require.NoError(t, err, "Create should auto-recover from orphaned worktree")
	assert.Equal(t, expectedPath, path2)

	// Verify worktree is functional
	exists, err = client.Exists("crew-1")
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify directory exists
	_, err = os.Stat(path2)
	assert.NoError(t, err)
}

func TestClient_SetupWorktree_CopyFiles(t *testing.T) {
	// Setup test repo
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create test files in main repo
	testFile := filepath.Join(repoRoot, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

	testDir := filepath.Join(repoRoot, "testdir")
	require.NoError(t, os.MkdirAll(testDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("file content"), 0644))

	// Create client and worktree
	client := NewClient(repoRoot, worktreeDir)
	branch := "crew-1"
	wtPath, err := client.Create(branch, "main")
	require.NoError(t, err)

	// Setup worktree with copy config
	config := &domain.WorktreeConfig{
		Copy: []string{"test.txt", "testdir"},
	}

	err = client.SetupWorktree(wtPath, config)
	require.NoError(t, err)

	// Verify files were copied
	copiedFile := filepath.Join(wtPath, "test.txt")
	assert.FileExists(t, copiedFile)
	content, err := os.ReadFile(copiedFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	copiedDir := filepath.Join(wtPath, "testdir", "file.txt")
	assert.FileExists(t, copiedDir)
}

func TestClient_SetupWorktree_SetupCommand(t *testing.T) {
	// Setup test repo
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create client and worktree
	client := NewClient(repoRoot, worktreeDir)
	branch := "crew-1"
	wtPath, err := client.Create(branch, "main")
	require.NoError(t, err)

	// Setup worktree with setup command
	config := &domain.WorktreeConfig{
		SetupCommand: "echo 'test' > setup_test.txt",
	}

	err = client.SetupWorktree(wtPath, config)
	require.NoError(t, err)

	// Verify command was executed
	testFile := filepath.Join(wtPath, "setup_test.txt")
	assert.FileExists(t, testFile)
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "test\n", string(content))
}

func TestClient_SetupWorktree_NilConfig(t *testing.T) {
	// Setup test repo
	repoRoot, worktreeDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create client and worktree
	client := NewClient(repoRoot, worktreeDir)
	branch := "crew-1"
	wtPath, err := client.Create(branch, "main")
	require.NoError(t, err)

	// Setup with nil config should not error
	err = client.SetupWorktree(wtPath, nil)
	assert.NoError(t, err)
}
