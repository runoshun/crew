package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// MergeTaskInput contains the parameters for merging a task.
type MergeTaskInput struct {
	TaskID int  // Task ID to merge
	Force  bool // Force stop session if running
}

// MergeTaskOutput contains the result of merging a task.
type MergeTaskOutput struct {
	Task *domain.Task // The merged task
}

// MergeTask is the use case for merging a task branch into main.
type MergeTask struct {
	tasks     domain.TaskRepository
	sessions  domain.SessionManager
	worktrees domain.WorktreeManager
	git       domain.Git
}

// NewMergeTask creates a new MergeTask use case.
func NewMergeTask(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	worktrees domain.WorktreeManager,
	git domain.Git,
) *MergeTask {
	return &MergeTask{
		tasks:     tasks,
		sessions:  sessions,
		worktrees: worktrees,
		git:       git,
	}
}

// Execute merges a task branch into main.
// Preconditions:
// - Current branch is main
// - main's working tree is clean
// - No session running (unless --force)
//
// Processing:
// 1. If session is running and --force, stop it
// 2. Delete worktree
// 3. Execute git merge --no-ff
// 4. Update status to done
func (uc *MergeTask) Execute(_ context.Context, in MergeTaskInput) (*MergeTaskOutput, error) {
	// Get the task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Check current branch is main
	currentBranch, err := uc.git.CurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}
	if currentBranch != "main" {
		return nil, domain.ErrNotOnMainBranch
	}

	// Check main's working tree is clean
	// Use repo root for checking main's working tree (empty string will be handled by the caller)
	hasChanges, err := uc.git.HasUncommittedChanges("")
	if err != nil {
		return nil, fmt.Errorf("check uncommitted changes: %w", err)
	}
	if hasChanges {
		return nil, domain.ErrUncommittedChanges
	}

	// Get branch name
	branch := domain.BranchName(task.ID, task.Issue)

	// Check if session is running
	sessionName := domain.SessionName(task.ID)
	running, err := uc.sessions.IsRunning(sessionName)
	if err != nil {
		return nil, fmt.Errorf("check session running: %w", err)
	}

	if running {
		if !in.Force {
			return nil, domain.ErrSessionRunning
		}
		// Force stop session
		if stopErr := uc.sessions.Stop(sessionName); stopErr != nil {
			return nil, fmt.Errorf("stop session: %w", stopErr)
		}
	}

	// Execute git merge --no-ff first (before deleting worktree)
	// This way, if merge fails due to conflict, worktree is preserved for resolution
	if mergeErr := uc.git.Merge(branch, true); mergeErr != nil {
		return nil, fmt.Errorf("merge branch: %w", mergeErr)
	}

	// Delete worktree if it exists (after successful merge)
	exists, err := uc.worktrees.Exists(branch)
	if err != nil {
		return nil, fmt.Errorf("check worktree exists: %w", err)
	}
	if exists {
		if err := uc.worktrees.Remove(branch); err != nil {
			return nil, fmt.Errorf("remove worktree: %w", err)
		}
	}

	// Delete the branch after merge
	if err := uc.git.DeleteBranch(branch, false); err != nil {
		return nil, fmt.Errorf("delete branch: %w", err)
	}

	// Update status to done
	task.Status = domain.StatusDone
	task.Agent = ""
	task.Session = ""

	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &MergeTaskOutput{Task: task}, nil
}
