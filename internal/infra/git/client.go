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

// WorkingDir returns the toplevel of the current working directory.
// This is the worktree root if running inside a git worktree.
func (c *Client) WorkingDir() string {
	return c.workingDir
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

// UserEmail returns the configured git user.email value.
// Returns empty string if not set.
func (c *Client) UserEmail() (string, error) {
	cmd := exec.Command("git", "config", "user.email")
	cmd.Dir = c.repoRoot
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("failed to get user.email: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// BranchExists checks if a branch exists.
func (c *Client) BranchExists(branch string) (bool, error) {
	//nolint:gosec // branch name is used as argument, not shell command
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = c.repoRoot
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	// Exit code 1 means ref not found
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("failed to check branch existence: %w", err)
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
// Uses git merge-tree --write-tree for dry-run conflict detection.
func (c *Client) HasMergeConflict(branch, target string) (bool, error) {
	files, err := c.GetMergeConflictFiles(branch, target)
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

// GetMergeConflictFiles returns the list of files that would conflict
// when merging branch into target. Returns empty slice if no conflicts.
// Uses git merge-tree --write-tree for dry-run conflict detection.
func (c *Client) GetMergeConflictFiles(branch, target string) ([]string, error) {
	// git merge-tree --write-tree returns exit code 0 if no conflicts,
	// exit code 1 if there are conflicts with conflicting files listed in output.
	cmd := exec.Command("git", "merge-tree", "--write-tree", target, branch)
	cmd.Dir = c.repoRoot
	out, err := cmd.CombinedOutput()

	if err == nil {
		// No conflicts
		return nil, nil
	}

	// Check if it's exit code 1 (conflicts) or some other error
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		return nil, fmt.Errorf("failed to run merge-tree: %w: %s", err, string(out))
	}

	// Parse output to find conflicting files
	// Format includes lines like: "CONFLICT (content): Merge conflict in <file>"
	return parseMergeTreeConflicts(string(out)), nil
}

// parseMergeTreeConflicts extracts conflicting file names from git merge-tree output.
// Handles various conflict formats:
//   - CONFLICT (content): Merge conflict in <file>
//   - CONFLICT (add/add): Merge conflict in <file>
//   - CONFLICT (modify/delete): <file> deleted in ... and modified in ...
//   - CONFLICT (rename/delete): <file> renamed to <file2> in ..., but deleted in ...
//   - CONFLICT (rename/rename): <file> renamed to <file2> in ... but renamed to <file3> in ...
//   - CONFLICT (file/directory): directory in the way of <file>
func parseMergeTreeConflicts(output string) []string {
	var files []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "CONFLICT (") {
			continue
		}

		var file string

		// Pattern 1: "Merge conflict in <file>" (content, add/add)
		if idx := strings.Index(line, "Merge conflict in "); idx >= 0 {
			file = strings.TrimSpace(line[idx+len("Merge conflict in "):])
		}

		// Pattern 2: "<file> renamed to" (rename/delete, rename/rename)
		// Check this before "deleted in" since rename lines may contain both patterns
		if file == "" {
			if idx := strings.Index(line, " renamed to "); idx >= 0 {
				startIdx := strings.Index(line, "): ")
				if startIdx >= 0 && startIdx < idx {
					file = strings.TrimSpace(line[startIdx+3 : idx])
				}
			}
		}

		// Pattern 3: "<file> deleted in" (modify/delete, delete/modify)
		if file == "" {
			if idx := strings.Index(line, " deleted in "); idx >= 0 {
				startIdx := strings.Index(line, "): ")
				if startIdx >= 0 && startIdx < idx {
					file = strings.TrimSpace(line[startIdx+3 : idx])
				}
			}
		}

		// Pattern 4: "directory in the way of <file>" (file/directory)
		if file == "" {
			if idx := strings.Index(line, "directory in the way of "); idx >= 0 {
				file = strings.TrimSpace(line[idx+len("directory in the way of "):])
				// Remove trailing period if present
				file = strings.TrimSuffix(file, ".")
			}
		}

		// Add unique files
		if file != "" && !seen[file] {
			seen[file] = true
			files = append(files, file)
		}
	}
	return files
}

// Merge merges a branch into the current branch.
// If noFF is true, a merge commit is always created (--no-ff).
// On conflict, automatically runs git merge --abort to restore clean state.
func (c *Client) Merge(branch string, noFF bool) error {
	args := []string{"merge"}
	if noFF {
		args = append(args, "--no-ff")
	}
	args = append(args, branch)

	cmd := exec.Command("git", args...)
	cmd.Dir = c.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		// On merge failure (including conflicts), abort the merge to restore clean state
		abortCmd := exec.Command("git", "merge", "--abort")
		abortCmd.Dir = c.repoRoot
		_ = abortCmd.Run() // Ignore abort errors - merge may have failed before creating merge state

		return fmt.Errorf("failed to merge branch %s: %w: %s", branch, err, string(out))
	}
	return nil
}

// DeleteBranch deletes a branch.
// If force is true, it uses -D (force delete), otherwise -d.
func (c *Client) DeleteBranch(branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	cmd := exec.Command("git", "branch", flag, branch)
	cmd.Dir = c.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w: %s", branch, err, string(out))
	}
	return nil
}

// ListBranches returns a list of all local branches.
func (c *Client) ListBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = c.repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var branches []string
	for _, line := range lines {
		if line != "" {
			branches = append(branches, strings.TrimSpace(line))
		}
	}
	return branches, nil
}

// GetDefaultBranch returns the default branch name.
// Priority: git config crew.defaultBranch > refs/remotes/origin/HEAD > "main"
func (c *Client) GetDefaultBranch() (string, error) {
	// 1. Try git config crew.defaultBranch
	cmd := exec.Command("git", "config", "crew.defaultBranch")
	cmd.Dir = c.repoRoot
	if out, err := cmd.Output(); err == nil {
		branch := strings.TrimSpace(string(out))
		if branch != "" {
			return branch, nil
		}
	}

	// 2. Try refs/remotes/origin/HEAD
	cmd = exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = c.repoRoot
	if out, err := cmd.Output(); err == nil {
		ref := strings.TrimSpace(string(out))
		// Extract branch name from refs/remotes/origin/xxx
		if strings.HasPrefix(ref, "refs/remotes/origin/") {
			branch := strings.TrimPrefix(ref, "refs/remotes/origin/")
			if branch != "" {
				return branch, nil
			}
		}
	}

	// 3. Fallback to "main"
	return "main", nil
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
		gitDir = filepath.Join(dir, gitDir)
	}

	// Clean the path
	gitDir = filepath.Clean(gitDir)

	// repoRoot is the parent of .git directory
	repoRoot = filepath.Dir(gitDir)

	return repoRoot, gitDir, workingDir, nil
}
