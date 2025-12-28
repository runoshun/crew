package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupGitRepo creates a temporary git repository for testing.
func setupGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize git repository
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")

	// Create initial commit
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# Test\n"), 0o644))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "Initial commit")

	return dir
}

// runGit executes a git command and fails the test if it errors.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, out)
}

// =============================================================================
// NewClient Tests
// =============================================================================

func TestNewClient_Success(t *testing.T) {
	dir := setupGitRepo(t)

	client, err := NewClient(dir)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, dir, client.RepoRoot())
	assert.Equal(t, filepath.Join(dir, ".git"), client.GitDir())
}

func TestNewClient_NotGitRepo(t *testing.T) {
	dir := t.TempDir() // Not a git repository

	client, err := NewClient(dir)
	assert.ErrorIs(t, err, domain.ErrNotGitRepository)
	assert.Nil(t, client)
}

func TestNewClient_FromWorktree(t *testing.T) {
	// Setup main repo
	mainRepo := setupGitRepo(t)

	// Create a worktree
	worktreeDir := filepath.Join(t.TempDir(), "worktree")
	runGit(t, mainRepo, "worktree", "add", "-b", "feature", worktreeDir)

	// NewClient from worktree should find the common .git directory
	client, err := NewClient(worktreeDir)
	require.NoError(t, err)
	assert.Equal(t, worktreeDir, client.RepoRoot())
	// GitDir should point to the main repo's .git directory
	assert.Equal(t, filepath.Join(mainRepo, ".git"), client.GitDir())
}

// =============================================================================
// RepoRoot / GitDir Tests
// =============================================================================

func TestClient_RepoRoot(t *testing.T) {
	dir := setupGitRepo(t)

	client, err := NewClient(dir)
	require.NoError(t, err)

	assert.Equal(t, dir, client.RepoRoot())
}

func TestClient_GitDir(t *testing.T) {
	dir := setupGitRepo(t)

	client, err := NewClient(dir)
	require.NoError(t, err)

	expectedGitDir := filepath.Join(dir, ".git")
	assert.Equal(t, expectedGitDir, client.GitDir())
}

// =============================================================================
// CurrentBranch Tests
// =============================================================================

func TestClient_CurrentBranch_Main(t *testing.T) {
	dir := setupGitRepo(t)

	client, err := NewClient(dir)
	require.NoError(t, err)

	branch, err := client.CurrentBranch()
	require.NoError(t, err)
	// Default branch name depends on git config, could be "main" or "master"
	assert.True(t, branch == "main" || branch == "master", "expected main or master, got %s", branch)
}

func TestClient_CurrentBranch_FeatureBranch(t *testing.T) {
	dir := setupGitRepo(t)

	// Create and checkout feature branch
	runGit(t, dir, "checkout", "-b", "feature/test-branch")

	client, err := NewClient(dir)
	require.NoError(t, err)

	branch, err := client.CurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature/test-branch", branch)
}

func TestClient_CurrentBranch_AfterSwitch(t *testing.T) {
	dir := setupGitRepo(t)

	client, err := NewClient(dir)
	require.NoError(t, err)

	// Get initial branch
	initialBranch, err := client.CurrentBranch()
	require.NoError(t, err)

	// Create and switch to new branch
	runGit(t, dir, "checkout", "-b", "new-branch")

	// Verify branch changed
	newBranch, err := client.CurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "new-branch", newBranch)
	assert.NotEqual(t, initialBranch, newBranch)
}
