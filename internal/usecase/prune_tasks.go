package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// PruneTasksInput contains the parameters for pruning tasks.
type PruneTasksInput struct {
	All    bool // If true, also prune 'done' tasks (in addition to 'closed')
	DryRun bool // If true, only list what would be pruned
}

// PruneTasksOutput contains the result of pruning tasks.
type PruneTasksOutput struct {
	DeletedTasks     []*domain.Task // Tasks that were (or would be) deleted
	DeletedBranches  []string       // Branches that were (or would be) deleted
	DeletedWorktrees []string       // Worktrees that were (or would be) deleted
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
		DeletedTasks:     []*domain.Task{},
		DeletedBranches:  []string{},
		DeletedWorktrees: []string{},
	}

	// 1. Identify candidate tasks for deletion
	tasks, err := uc.tasks.List(domain.TaskFilter{})
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	for _, task := range tasks {
		shouldDelete := false
		if task.Status == domain.StatusClosed {
			shouldDelete = true
		} else if in.All && task.Status == domain.StatusDone {
			shouldDelete = true
		}

		if shouldDelete {
			out.DeletedTasks = append(out.DeletedTasks, task)
			if !in.DryRun {
				if deleteErr := uc.tasks.Delete(task.ID); deleteErr != nil {
					return nil, fmt.Errorf("delete task #%d: %w", task.ID, deleteErr)
				}
			}
		}
	}

	// 2. Identify orphan branches
	// Rules:
	// - Must match crew branch pattern
	// - Task ID must NOT exist in the store (unless we just deleted it)
	// - If we just deleted the task, its branch is also a target
	branches, err := uc.git.ListBranches()
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	for _, branch := range branches {
		taskID, isCrewBranch := domain.ParseBranchTaskID(branch)
		if !isCrewBranch {
			continue
		}

		// Check if task exists in DB (refetch to be sure, or check if it was in our deleted list)
		// Simpler: check if we kept the task or if it's still in the DB (but if we deleted it, it's gone)
		// Logic:
		// If task exists in DB and was NOT deleted -> Keep branch
		// If task exists in DB but WAS deleted -> Delete branch
		// If task does NOT exist in DB -> Delete branch (orphan)

		taskExists, existsErr := uc.taskExists(taskID)
		if existsErr != nil {
			return nil, existsErr
		}

		// If task exists and was NOT just deleted, keep the branch
		if taskExists && !uc.wasDeleted(out.DeletedTasks, taskID) {
			continue
		}

		// Otherwise, it's a target
		out.DeletedBranches = append(out.DeletedBranches, branch)
		if !in.DryRun {
			// Force delete branch
			if deleteErr := uc.git.DeleteBranch(branch, true); deleteErr != nil {
				return nil, fmt.Errorf("delete branch %s: %w", branch, deleteErr)
			}
		}
	}

	// 3. Identify orphan worktrees
	// Logic similar to branches
	worktrees, err := uc.worktrees.List()
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}

	for _, wt := range worktrees {
		// Worktrees are usually named by ID or branch, but here we check the branch associated with it
		// If the branch is one we deleted (or would delete), then the worktree is also a target
		// OR if the worktree path corresponds to a crew task that doesn't exist

		// Check if this worktree is associated with a crew branch we are deleting
		if uc.contains(out.DeletedBranches, wt.Branch) {
			out.DeletedWorktrees = append(out.DeletedWorktrees, wt.Path)
			if !in.DryRun {
				if err := uc.worktrees.Remove(wt.Branch); err != nil {
					// Ignore "not found" errors for worktrees as they might have been cleaned up with the branch
					// But usually worktree remove cleans up the branch reference too?
					// git worktree remove <path>
					// Our interface takes branch... let's check implementation
					// Implementation resolves path from branch.
					return nil, fmt.Errorf("remove worktree for %s: %w", wt.Branch, err)
				}
			}
			continue
		}

		// Also check for worktrees that might have lost their branch (detached?) or other crew artifacts?
		// For now, let's stick to cleaning up worktrees associated with the pruned branches.
		// If there are other "orphan" worktrees (e.g. branch was deleted manually but worktree remains),
		// we might need more logic, but typically crew manages both.

		// Wait, if the branch is NOT a crew branch, we ignore.
		taskID, isCrewBranch := domain.ParseBranchTaskID(wt.Branch)
		if !isCrewBranch {
			continue
		}

		taskExists, err := uc.taskExists(taskID)
		if err != nil {
			return nil, err
		}

		if !taskExists || uc.wasDeleted(out.DeletedTasks, taskID) {
			// This case should be covered by the branch check above usually,
			// but if the branch was already gone but worktree remains?
			// ListBranches might not show it if it's checked out?
			// Actually, git branch lists checked out branches too.
			// So relying on DeletedBranches check is mostly sufficient,
			// UNLESS the branch was deleted manually but worktree remains.

			// Let's safe-guard: if we didn't mark branch for deletion but task is gone/deleted, prune worktree.
			if !uc.contains(out.DeletedBranches, wt.Branch) {
				out.DeletedWorktrees = append(out.DeletedWorktrees, wt.Path)
				if !in.DryRun {
					if err := uc.worktrees.Remove(wt.Branch); err != nil {
						return nil, fmt.Errorf("remove worktree for %s: %w", wt.Branch, err)
					}
				}
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

func (uc *PruneTasks) wasDeleted(deleted []*domain.Task, id int) bool {
	for _, t := range deleted {
		if t.ID == id {
			return true
		}
	}
	return false
}

func (uc *PruneTasks) contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
