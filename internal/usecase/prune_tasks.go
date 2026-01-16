package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// PruneTasksInput contains the parameters for pruning tasks.
type PruneTasksInput struct {
	All    bool // Unused (kept for backward compatibility)
	DryRun bool // If true, only list what would be pruned
}

// PruneTasksOutput contains the result of pruning tasks.
type PruneTasksOutput struct {
	DeletedBranches  []string // Branches that were (or would be) deleted
	DeletedWorktrees []string // Worktrees that were (or would be) deleted
}

// PruneTasks is the use case for pruning completed tasks and orphan resources.
type PruneTasks struct {
	tasks     domain.TaskRepository
	worktrees domain.WorktreeManager
	git       domain.Git
}

// NewPruneTasks creates a new PruneTasks use case.
func NewPruneTasks(tasks domain.TaskRepository, worktrees domain.WorktreeManager, git domain.Git) *PruneTasks {
	return &PruneTasks{
		tasks:     tasks,
		worktrees: worktrees,
		git:       git,
	}
}

// Execute prunes tasks and resources.
func (uc *PruneTasks) Execute(_ context.Context, in PruneTasksInput) (*PruneTasksOutput, error) {
	out := &PruneTasksOutput{
		DeletedBranches:  []string{},
		DeletedWorktrees: []string{},
	}

	// 1. Identify tasks with closed status (for branch/worktree deletion criteria)
	// NOTE: Tasks themselves are NOT deleted, only branches and worktrees
	tasks, err := uc.tasks.List(domain.TaskFilter{})
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	// Build a set of task IDs that should have their resources cleaned up
	shouldCleanup := make(map[int]bool)
	for _, task := range tasks {
		if task.Status == domain.StatusClosed {
			shouldCleanup[task.ID] = true
		}
	}

	// 2. Identify branches to delete
	// Rules:
	// - Must match crew branch pattern
	// - Task either doesn't exist OR has closed status (in cleanup set)
	branches, err := uc.git.ListBranches()
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	for _, branch := range branches {
		taskID, isCrewBranch := domain.ParseBranchTaskID(branch)
		if !isCrewBranch {
			continue
		}

		// Check if task exists
		taskExists, existsErr := uc.taskExists(taskID)
		if existsErr != nil {
			return nil, existsErr
		}

		// Delete branch if:
		// - Task doesn't exist (orphan), OR
		// - Task exists and is in cleanup set (closed/done)
		if !taskExists || shouldCleanup[taskID] {
			out.DeletedBranches = append(out.DeletedBranches, branch)
		}
	}

	// 3. Identify worktrees to delete
	// Delete worktree if its branch is being deleted OR if it's an orphan crew worktree
	worktrees, err := uc.worktrees.List()
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}

	for _, wt := range worktrees {
		// If worktree's branch is in DeletedBranches, include it
		if uc.contains(out.DeletedBranches, wt.Branch) {
			out.DeletedWorktrees = append(out.DeletedWorktrees, wt.Path)
			continue
		}

		// Also check for orphan crew worktrees (branch doesn't match deletion list but task is gone/closed)
		taskID, isCrewBranch := domain.ParseBranchTaskID(wt.Branch)
		if !isCrewBranch {
			continue
		}

		taskExists, err := uc.taskExists(taskID)
		if err != nil {
			return nil, err
		}

		// If task doesn't exist OR should be cleaned up, but branch wasn't marked for deletion
		// (edge case: branch manually deleted but worktree remains)
		if (!taskExists || shouldCleanup[taskID]) && !uc.contains(out.DeletedBranches, wt.Branch) {
			out.DeletedWorktrees = append(out.DeletedWorktrees, wt.Path)
		}
	}

	// 4. Perform deletions (if not dry run)
	// Order: Worktrees -> Branches (Tasks are NOT deleted)
	if !in.DryRun {
		// 4.1 Delete Worktrees first (to free up branches)
		for _, path := range out.DeletedWorktrees {
			// Find the branch for this path to call Remove(branch)
			var branchToRemove string
			for _, wt := range worktrees {
				if wt.Path == path {
					branchToRemove = wt.Branch
					break
				}
			}

			if branchToRemove != "" {
				if err := uc.worktrees.Remove(branchToRemove); err != nil {
					// Log error but continue? Or return partial error?
					// For CLI tools, usually failing fast or logging is good.
					// Let's return error for now to be safe.
					return nil, fmt.Errorf("remove worktree %s: %w", path, err)
				}
			}
		}

		// 4.2 Delete Branches
		for _, branch := range out.DeletedBranches {
			if err := uc.git.DeleteBranch(branch, true); err != nil {
				return nil, fmt.Errorf("delete branch %s: %w", branch, err)
			}
		}
	}

	return out, nil
}

func (uc *PruneTasks) taskExists(id int) (bool, error) {
	task, err := uc.tasks.Get(id)
	if err != nil {
		return false, fmt.Errorf("get task %d: %w", id, err)
	}
	return task != nil, nil
}

func (uc *PruneTasks) contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
