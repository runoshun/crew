// Package git provides git operations.
package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Client provides git operations.
type Client struct {
	repoRoot   string // Main repository root (parent of .git)
	gitDir     string // Common .git directory
	workingDir string // Current working directory (may be worktree)
}

// NewClient creates a new git client by detecting the repository root from the given directory.
// It handles both regular repositories and worktrees.
func NewClient(dir string) (*Client, error) {
	repoRoot, gitDir, workingDir, err := findGitRoot(dir)
	if err != nil {
		return nil, err
	}
	return &Client{
		repoRoot:   repoRoot,
		gitDir:     gitDir,
		workingDir: workingDir,
	}, nil
}

// RepoRoot returns the repository root directory.
func (c *Client) RepoRoot() string {
	return c.repoRoot
}

// GitDir returns the .git directory path.
func (c *Client) GitDir() string {
	return c.gitDir
}

// CurrentBranch returns the name of the current branch.
// Uses workingDir to correctly detect branch in worktrees.
func (c *Client) CurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = c.workingDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// BranchExists checks if a branch exists.
// TODO: implement in later phase
func (c *Client) BranchExists(_ string) (bool, error) {
	panic("not implemented")
}

// HasUncommittedChanges checks for uncommitted changes in a directory.
// Returns true if there are uncommitted changes (staged or unstaged).
func (c *Client) HasUncommittedChanges(dir string) (bool, error) {
	// Use git status --porcelain to check for any uncommitted changes
	// This returns empty output if the working tree is clean
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check uncommitted changes: %w", err)
	}
	// If output is non-empty, there are uncommitted changes
	return len(out) > 0, nil
}

// HasMergeConflict checks if merging branch into target would conflict.
// TODO: implement in later phase
func (c *Client) HasMergeConflict(_, _ string) (bool, error) {
	panic("not implemented")
}

// Merge merges a branch into the current branch.
// If noFF is true, a merge commit is always created (--no-ff).
func (c *Client) Merge(branch string, noFF bool) error {
	args := []string{"merge"}
	if noFF {
		args = append(args, "--no-ff")
	}
	args = append(args, branch)

	cmd := exec.Command("git", args...)
	cmd.Dir = c.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to merge branch %s: %w: %s", branch, err, string(out))
	}
	return nil
}

// DeleteBranch deletes a branch.
// Uses -d flag (safe delete, only if fully merged).
func (c *Client) DeleteBranch(branch string) error {
	cmd := exec.Command("git", "branch", "-d", branch)
	cmd.Dir = c.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w: %s", branch, err, string(out))
	}
	return nil
}

// Ensure Client implements domain.Git interface.
var _ domain.Git = (*Client)(nil)

// findGitRoot finds the git repository root and .git directory from the given directory.
// This works correctly both in the main repository and inside worktrees.
// Returns:
//   - repoRoot: main repository root (parent of .git)
//   - gitDir: common .git directory
//   - workingDir: current working directory (toplevel of current worktree or main repo)
func findGitRoot(dir string) (repoRoot, gitDir, workingDir string, err error) {
	// First check if we're in a git repository at all
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", "", "", domain.ErrNotGitRepository
	}
	gitDir = strings.TrimSpace(string(out))

	// Get the toplevel (this returns the worktree root if in a worktree)
	cmd = exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	toplevel, err := cmd.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to find toplevel: %w", err)
	}
	workingDir = strings.TrimSpace(string(toplevel))

	// Make gitDir absolute if it's relative
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(workingDir, gitDir)
	}

	// Clean the path
	gitDir = filepath.Clean(gitDir)

	// repoRoot is the parent of .git directory
	repoRoot = filepath.Dir(gitDir)

	return repoRoot, gitDir, workingDir, nil
}
