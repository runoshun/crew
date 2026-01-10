package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// NewClient from worktree should find the main repository
	client, err := NewClient(worktreeDir)
	require.NoError(t, err)
	// RepoRoot should be the main repo (parent of .git), not the worktree
	// This allows crew commands to work from within worktrees
	assert.Equal(t, mainRepo, client.RepoRoot())
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

// =============================================================================
// GetDefaultBranch Tests
// =============================================================================

func TestClient_GetDefaultBranch_FromConfig(t *testing.T) {
	dir := setupGitRepo(t)

	// Set crew.defaultBranch config
	runGit(t, dir, "config", "crew.defaultBranch", "develop")

	client, err := NewClient(dir)
	require.NoError(t, err)

	branch, err := client.GetDefaultBranch()
	require.NoError(t, err)
	assert.Equal(t, "develop", branch)
}

func TestClient_GetDefaultBranch_FromOriginHEAD(t *testing.T) {
	dir := setupGitRepo(t)

	// Create a bare remote repository
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, t.TempDir(), "init", "--bare", remoteDir)

	// Set up remote
	runGit(t, dir, "remote", "add", "origin", remoteDir)

	// Check if 'main' branch already exists
	currentBranch, _ := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	branch := strings.TrimSpace(string(currentBranch))

	// If not on 'main', create it
	if branch != "main" {
		runGit(t, dir, "checkout", "-b", "main")
	}

	// Push to origin
	runGit(t, dir, "push", "-u", "origin", branch)

	client, err := NewClient(dir)
	require.NoError(t, err)

	defaultBranch, err := client.GetDefaultBranch()
	require.NoError(t, err)
	assert.Equal(t, branch, defaultBranch)
}

func TestClient_GetDefaultBranch_Fallback(t *testing.T) {
	dir := setupGitRepo(t)

	// Don't set any config or remote
	client, err := NewClient(dir)
	require.NoError(t, err)

	branch, err := client.GetDefaultBranch()
	require.NoError(t, err)
	assert.Equal(t, "main", branch)
}

// =============================================================================
// GetNewTaskBaseBranch Tests
// =============================================================================

func TestClient_GetNewTaskBaseBranch_Default(t *testing.T) {
	dir := setupGitRepo(t)

	// Set crew.defaultBranch config
	runGit(t, dir, "config", "crew.defaultBranch", "develop")

	client, err := NewClient(dir)
	require.NoError(t, err)

	// When crew.newTaskBase is not set, should use default branch
	branch, err := client.GetNewTaskBaseBranch()
	require.NoError(t, err)
	assert.Equal(t, "develop", branch)
}

func TestClient_GetNewTaskBaseBranch_Current(t *testing.T) {
	dir := setupGitRepo(t)

	// Set crew.newTaskBase to "current"
	runGit(t, dir, "config", "crew.newTaskBase", "current")

	// Create and checkout a feature branch
	runGit(t, dir, "checkout", "-b", "feature/test")

	client, err := NewClient(dir)
	require.NoError(t, err)

	// Should return the current branch
	branch, err := client.GetNewTaskBaseBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature/test", branch)
}

func TestClient_GetNewTaskBaseBranch_CurrentOnMain(t *testing.T) {
	dir := setupGitRepo(t)

	// Set crew.newTaskBase to "current"
	runGit(t, dir, "config", "crew.newTaskBase", "current")
	// Set crew.defaultBranch to something else
	runGit(t, dir, "config", "crew.defaultBranch", "develop")

	// Stay on the initial branch (main/master)
	client, err := NewClient(dir)
	require.NoError(t, err)

	currentBranch, err := client.CurrentBranch()
	require.NoError(t, err)

	// Should return the current branch (not develop)
	branch, err := client.GetNewTaskBaseBranch()
	require.NoError(t, err)
	assert.Equal(t, currentBranch, branch)
}
