// Package worktree provides git worktree operations.
package worktree

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Client manages git worktrees.
type Client struct {
	repoRoot    string // Main repository root
	worktreeDir string // Directory where worktrees are created
}

// NewClient creates a new worktree client.
// repoRoot is the main repository root directory.
// worktreeDir is the directory where worktrees will be created (typically .git/crew/worktrees).
func NewClient(repoRoot, worktreeDir string) *Client {
	return &Client{
		repoRoot:    repoRoot,
		worktreeDir: worktreeDir,
	}
}

// Ensure Client implements domain.WorktreeManager interface.
var _ domain.WorktreeManager = (*Client)(nil)

// SetupWorktree performs post-creation setup tasks (file copying and command execution).
// This should be called after Create to apply worktree customization.
func (c *Client) SetupWorktree(wtPath string, config *domain.WorktreeConfig) error {
	if config == nil {
		return nil
	}

	// Copy files/directories (with CoW support)
	for _, item := range config.Copy {
		src := filepath.Join(c.repoRoot, item)
		dst := filepath.Join(wtPath, item)

		if err := c.copyWithCoW(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", item, err)
		}
	}

	// Execute setup command
	if config.SetupCommand != "" {
		// G204: Setup command is from config file (trusted source)
		cmd := exec.Command("sh", "-c", config.SetupCommand) //nolint:gosec // Command from config file
		cmd.Dir = wtPath

		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("setup command failed: %w: %s", err, string(out))
		}
	}

	return nil
}

// copyWithCoW copies a file or directory using Copy-on-Write if available.
// Falls back to regular copy if CoW is not supported.
func (c *Client) copyWithCoW(src, dst string) error {
	// Try CoW copy first (cp --reflink=auto)
	cmd := exec.Command("cp", "--reflink=auto", "-r", src, dst)
	if err := cmd.Run(); err != nil {
		// If CoW fails, we might need to create parent directories
		if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
			return fmt.Errorf("create parent directory: %w", err)
		}
		// Retry
		cmd = exec.Command("cp", "--reflink=auto", "-r", src, dst)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("copy failed: %w", err)
		}
	}
	return nil
}

// Create creates a new worktree for the given branch.
// If the branch doesn't exist, it creates a new branch from baseBranch.
// The worktree directory is named by task ID (extracted from branch name).
func (c *Client) Create(branch, baseBranch string) (string, error) {
	// Extract task ID from branch name for directory naming
	taskID, ok := domain.ParseBranchTaskID(branch)
	if !ok {
		return "", fmt.Errorf("invalid crew branch name: %s", branch)
	}

	// Use task ID for directory name
	path := domain.WorktreePath(c.worktreeDir, taskID)

	// Check if worktree already exists
	exists, err := c.Exists(branch)
	if err != nil {
		return "", fmt.Errorf("check worktree exists: %w", err)
	}
	if exists {
		return path, nil // Already exists, return path
	}

	// Check if branch exists
	branchExists, err := c.branchExists(branch)
	if err != nil {
		return "", fmt.Errorf("check branch exists: %w", err)
	}

	var cmd *exec.Cmd
	if branchExists {
		// Branch exists, create worktree for existing branch
		cmd = exec.Command("git", "worktree", "add", path, branch)
	} else {
		// Branch doesn't exist, create new branch from baseBranch
		cmd = exec.Command("git", "worktree", "add", "-b", branch, path, baseBranch)
	}
	cmd.Dir = c.repoRoot

	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("create worktree: %w: %s", err, string(out))
	}

	return path, nil
}

// Resolve returns the path of an existing worktree for the branch.
func (c *Client) Resolve(branch string) (string, error) {
	worktrees, err := c.List()
	if err != nil {
		return "", err
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			return wt.Path, nil
		}
	}

	return "", domain.ErrWorktreeNotFound
}

// Remove deletes a worktree.
// Returns ErrUncommittedChanges if the worktree has uncommitted changes.
// Use prune command to force cleanup of dirty worktrees.
func (c *Client) Remove(branch string) error {
	// First resolve the path
	path, err := c.Resolve(branch)
	if err != nil {
		return err
	}

	// Remove the worktree (no --force, fails if dirty)
	cmd := exec.Command("git", "worktree", "remove", path)
	cmd.Dir = c.repoRoot

	out, err := cmd.CombinedOutput()
	if err != nil {
		// Check if the error is due to uncommitted changes
		outStr := string(out)
		if strings.Contains(outStr, "contains modified or untracked files") ||
			strings.Contains(outStr, "is dirty") {
			return domain.ErrUncommittedChanges
		}
		return fmt.Errorf("remove worktree: %w: %s", err, outStr)
	}

	return nil
}

// Exists checks if a worktree exists for the branch.
func (c *Client) Exists(branch string) (bool, error) {
	worktrees, err := c.List()
	if err != nil {
		return false, err
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			return true, nil
		}
	}

	return false, nil
}

// List returns all worktrees.
func (c *Client) List() ([]domain.WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = c.repoRoot

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}

	return parseWorktreeList(string(out))
}

// parseWorktreeList parses the porcelain output of git worktree list.
// Format:
//
//	worktree /path/to/worktree
//	HEAD abc123
//	branch refs/heads/branch-name
//	<blank line>
func parseWorktreeList(output string) ([]domain.WorktreeInfo, error) {
	var worktrees []domain.WorktreeInfo
	var current domain.WorktreeInfo

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "worktree "):
			current.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			// Extract branch name from refs/heads/branch-name
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "":
			// End of entry
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = domain.WorktreeInfo{}
		}
	}

	// Handle last entry if no trailing newline
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse worktree list: %w", err)
	}

	return worktrees, nil
}

// branchExists checks if a branch exists in the repository.
func (c *Client) branchExists(branch string) (bool, error) {
	ref := "refs/heads/" + branch
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", ref)
	cmd.Dir = c.repoRoot

	err := cmd.Run()
	if err != nil {
		// Exit code 1 means branch doesn't exist
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("check branch exists: %w", err)
	}

	return true, nil
}
