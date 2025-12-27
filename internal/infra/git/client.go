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
	repoRoot string
	gitDir   string
}

// NewClient creates a new git client by detecting the repository root from the given directory.
// It handles both regular repositories and worktrees.
func NewClient(dir string) (*Client, error) {
	repoRoot, gitDir, err := findGitRoot(dir)
	if err != nil {
		return nil, err
	}
	return &Client{
		repoRoot: repoRoot,
		gitDir:   gitDir,
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
func (c *Client) CurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = c.repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// findGitRoot finds the git repository root and .git directory from the given directory.
func findGitRoot(dir string) (repoRoot, gitDir string, err error) {
	// Get the repository root
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", "", domain.ErrNotGitRepository
	}
	repoRoot = strings.TrimSpace(string(out))

	// Get the .git directory (handles both regular repos and worktrees)
	cmd = exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = dir
	out, err = cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to find .git directory: %w", err)
	}
	gitDir = strings.TrimSpace(string(out))

	// Make gitDir absolute if it's relative
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoRoot, gitDir)
	}

	// Clean the path
	gitDir = filepath.Clean(gitDir)

	return repoRoot, gitDir, nil
}
