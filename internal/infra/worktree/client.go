// Package worktree provides git worktree operations.
package worktree

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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

	// Build the git worktree add command
	var args []string
	if branchExists {
		// Branch exists, create worktree for existing branch
		args = []string{"worktree", "add", path, branch}
	} else {
		// Branch doesn't exist, create new branch from baseBranch
		args = []string{"worktree", "add", "-b", branch, path, baseBranch}
	}

	// Execute the command
	cmd := exec.Command("git", args...)
	cmd.Dir = c.repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Check if this is an "already registered" error (orphaned worktree)
		outStr := string(out)
		if strings.Contains(outStr, "already registered") {
			// Worktree is registered but directory is missing
			// Prune stale worktree entries and retry
			if pruneErr := c.prune(); pruneErr != nil {
				return "", fmt.Errorf("prune stale worktrees: %w", pruneErr)
			}
			// Retry the command with a fresh exec.Cmd
			cmd = exec.Command("git", args...)
			cmd.Dir = c.repoRoot
			out, err = cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("create worktree after prune: %w: %s", err, string(out))
			}
		} else {
			return "", fmt.Errorf("create worktree: %w: %s", err, outStr)
		}
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
// Returns true only if both git registration and directory exist.
func (c *Client) Exists(branch string) (bool, error) {
	worktrees, err := c.List()
	if err != nil {
		return false, err
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			// Found in git worktree list, but verify directory exists
			if _, err := os.Stat(wt.Path); err != nil {
				if os.IsNotExist(err) {
					// Directory doesn't exist - treat as not existing
					return false, nil
				}
				return false, fmt.Errorf("check worktree directory: %w", err)
			}
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

// prune removes stale worktree entries.
// This cleans up worktree registrations where the directory no longer exists.
func (c *Client) prune() error {
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = c.repoRoot

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("prune worktrees: %w: %s", err, string(out))
	}

	return nil
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
