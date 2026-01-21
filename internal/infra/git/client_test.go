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
// Merge Tests
// =============================================================================

func TestClient_Merge_Success(t *testing.T) {
	dir := setupGitRepo(t)

	// Create a feature branch with a new file
	runGit(t, dir, "checkout", "-b", "feature")
	featureFile := filepath.Join(dir, "feature.txt")
	require.NoError(t, os.WriteFile(featureFile, []byte("feature content\n"), 0o644))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "Add feature")

	// Switch back to main and merge
	runGit(t, dir, "checkout", "-")
	client, err := NewClient(dir)
	require.NoError(t, err)

	err = client.Merge("feature", false)
	require.NoError(t, err)

	// Verify merge was successful
	_, err = os.Stat(featureFile)
	assert.NoError(t, err, "feature file should exist after merge")
}

func TestClient_Merge_Conflict_AutoAbort(t *testing.T) {
	dir := setupGitRepo(t)

	// Get the initial branch name first
	client, err := NewClient(dir)
	require.NoError(t, err)
	mainBranch, err := client.CurrentBranch()
	require.NoError(t, err)

	// Modify README.md on main branch
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# Main Branch\n"), 0o644))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "Update README on main")

	// Create feature branch from the initial commit (before main's change)
	runGit(t, dir, "checkout", "HEAD~1")
	runGit(t, dir, "checkout", "-b", "feature")

	// Modify the same file differently on feature branch
	require.NoError(t, os.WriteFile(readme, []byte("# Feature Branch\n"), 0o644))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "Update README on feature")

	// Switch back to main branch (use explicit branch name)
	runGit(t, dir, "checkout", mainBranch)

	// Attempt to merge feature into main (will conflict)
	err = client.Merge("feature", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to merge")

	// Verify that we're still on the main branch
	currentBranch, err := client.CurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, mainBranch, currentBranch)

	// Verify that git status is clean (merge was aborted)
	hasChanges, err := client.HasUncommittedChanges(dir)
	require.NoError(t, err)
	assert.False(t, hasChanges, "working tree should be clean after auto-abort")

	// Verify no MERGE_HEAD exists (merge state was aborted)
	mergeHead := filepath.Join(dir, ".git", "MERGE_HEAD")
	_, err = os.Stat(mergeHead)
	assert.True(t, os.IsNotExist(err), "MERGE_HEAD should not exist after abort")
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

	// Get current branch name (depends on git init.defaultBranch setting)
	currentBranch, _ := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	branch := strings.TrimSpace(string(currentBranch))

	// Push to origin
	runGit(t, dir, "push", "-u", "origin", branch)

	// Set origin/HEAD explicitly (git push doesn't set this automatically)
	runGit(t, dir, "remote", "set-head", "origin", branch)

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
// HasMergeConflict / GetMergeConflictFiles Tests
// =============================================================================

func TestClient_HasMergeConflict_NoConflict(t *testing.T) {
	dir := setupGitRepo(t)

	// Get the initial branch name
	client, err := NewClient(dir)
	require.NoError(t, err)
	mainBranch, err := client.CurrentBranch()
	require.NoError(t, err)

	// Create a feature branch and add a new file (no conflict)
	runGit(t, dir, "checkout", "-b", "feature")
	featureFile := filepath.Join(dir, "feature.txt")
	require.NoError(t, os.WriteFile(featureFile, []byte("feature content\n"), 0o644))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "Add feature")

	// Switch back to main
	runGit(t, dir, "checkout", mainBranch)

	// Check for conflicts - should be none
	hasConflict, err := client.HasMergeConflict("feature", mainBranch)
	require.NoError(t, err)
	assert.False(t, hasConflict)

	files, err := client.GetMergeConflictFiles("feature", mainBranch)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestClient_HasMergeConflict_WithConflict(t *testing.T) {
	dir := setupGitRepo(t)

	// Get the initial branch name
	client, err := NewClient(dir)
	require.NoError(t, err)
	mainBranch, err := client.CurrentBranch()
	require.NoError(t, err)

	// Modify README.md on main branch
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# Main Branch Content\n"), 0o644))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "Update README on main")

	// Create feature branch from initial commit (before main's change)
	runGit(t, dir, "checkout", "HEAD~1")
	runGit(t, dir, "checkout", "-b", "feature")

	// Modify the same file differently on feature branch
	require.NoError(t, os.WriteFile(readme, []byte("# Feature Branch Content\n"), 0o644))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "Update README on feature")

	// Switch back to main
	runGit(t, dir, "checkout", mainBranch)

	// Check for conflicts - should detect conflict
	hasConflict, err := client.HasMergeConflict("feature", mainBranch)
	require.NoError(t, err)
	assert.True(t, hasConflict)

	files, err := client.GetMergeConflictFiles("feature", mainBranch)
	require.NoError(t, err)
	assert.Contains(t, files, "README.md")
}

func TestClient_GetMergeConflictFiles_MultipleFiles(t *testing.T) {
	dir := setupGitRepo(t)

	// Get the initial branch name
	client, err := NewClient(dir)
	require.NoError(t, err)
	mainBranch, err := client.CurrentBranch()
	require.NoError(t, err)

	// Add another file on main
	readme := filepath.Join(dir, "README.md")
	file2 := filepath.Join(dir, "file2.txt")
	require.NoError(t, os.WriteFile(readme, []byte("# Main README\n"), 0o644))
	require.NoError(t, os.WriteFile(file2, []byte("main file2\n"), 0o644))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "Update files on main")

	// Create feature branch from initial commit
	runGit(t, dir, "checkout", "HEAD~1")
	runGit(t, dir, "checkout", "-b", "feature")

	// Modify both files differently on feature
	require.NoError(t, os.WriteFile(readme, []byte("# Feature README\n"), 0o644))
	require.NoError(t, os.WriteFile(file2, []byte("feature file2\n"), 0o644))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "Update files on feature")

	// Switch back to main
	runGit(t, dir, "checkout", mainBranch)

	// Check for conflicts - should detect both files
	files, err := client.GetMergeConflictFiles("feature", mainBranch)
	require.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "README.md")
	assert.Contains(t, files, "file2.txt")
}

// =============================================================================
// parseMergeTreeConflicts Tests
// =============================================================================

func TestParseMergeTreeConflicts(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []string
	}{
		{
			name:     "no conflicts",
			output:   "abc123\n",
			expected: nil,
		},
		{
			name: "single conflict",
			output: `abc123
CONFLICT (content): Merge conflict in README.md
`,
			expected: []string{"README.md"},
		},
		{
			name: "multiple conflicts",
			output: `abc123
CONFLICT (content): Merge conflict in file1.txt
CONFLICT (content): Merge conflict in dir/file2.txt
`,
			expected: []string{"file1.txt", "dir/file2.txt"},
		},
		{
			name: "add/add conflict",
			output: `abc123
CONFLICT (add/add): Merge conflict in newfile.txt
`,
			expected: []string{"newfile.txt"},
		},
		{
			name: "modify/delete conflict",
			output: `abc123
CONFLICT (modify/delete): config.json deleted in feature and modified in main. Version main of config.json left in tree.
`,
			expected: []string{"config.json"},
		},
		{
			name: "delete/modify conflict",
			output: `abc123
CONFLICT (delete/modify): settings.yaml deleted in main and modified in feature. Version feature of settings.yaml left in tree.
`,
			expected: []string{"settings.yaml"},
		},
		{
			name: "rename/delete conflict",
			output: `abc123
CONFLICT (rename/delete): old.txt renamed to new.txt in feature, but deleted in main.
`,
			expected: []string{"old.txt"},
		},
		{
			name: "rename/rename conflict",
			output: `abc123
CONFLICT (rename/rename): original.txt renamed to feature.txt in feature but renamed to main.txt in main.
`,
			expected: []string{"original.txt"},
		},
		{
			name: "file/directory conflict",
			output: `abc123
CONFLICT (file/directory): directory in the way of myfile.txt
`,
			expected: []string{"myfile.txt"},
		},
		{
			name: "mixed conflict types",
			output: `abc123
CONFLICT (content): Merge conflict in README.md
CONFLICT (modify/delete): config.json deleted in feature and modified in main. Version main of config.json left in tree.
CONFLICT (rename/delete): old.txt renamed to new.txt in feature, but deleted in main.
`,
			expected: []string{"README.md", "config.json", "old.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMergeTreeConflicts(tt.output)
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
