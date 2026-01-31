package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// CloseTaskInput contains the parameters for closing a task.
type CloseTaskInput struct {
	TaskID int // Task ID to close
}

// CloseTaskOutput contains the result of closing a task.
type CloseTaskOutput struct {
	Task *domain.Task // The closed task
}

// CloseTask is the use case for closing a task without merging.
type CloseTask struct {
	tasks     domain.TaskRepository
	sessions  domain.SessionManager
	worktrees domain.WorktreeManager
}

// NewCloseTask creates a new CloseTask use case.
func NewCloseTask(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	worktrees domain.WorktreeManager,
) *CloseTask {
	return &CloseTask{
		tasks:     tasks,
		sessions:  sessions,
		worktrees: worktrees,
	}
}

// Execute closes a task by:
// 1. Stopping the session if running
// 2. Deleting the worktree if it exists
// 3. Updating status to closed
// 4. Clearing agent info
func (uc *CloseTask) Execute(_ context.Context, in CloseTaskInput) (*CloseTaskOutput, error) {
	// Get the task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	// Validate status transition
	if !task.Status.CanTransitionTo(domain.StatusClosed) {
		return nil, fmt.Errorf("cannot close task in %s status: %w", task.Status.Display(), domain.ErrInvalidTransition)
	}

	// Get branch name for worktree operations
	branch := domain.BranchName(task.ID, task.Issue)

	// Stop session if running
	if _, stopErr := shared.StopSession(uc.sessions, task.ID); stopErr != nil {
		return nil, stopErr
	}

	// Delete worktree if it exists
	exists, err := uc.worktrees.Exists(branch)
	if err != nil {
		return nil, fmt.Errorf("check worktree exists: %w", err)
	}
	if exists {
		if removeErr := uc.worktrees.Remove(branch); removeErr != nil {
			return nil, fmt.Errorf("remove worktree: %w", removeErr)
		}
	}

	// Update status with abandoned reason
	task.Status = domain.StatusClosed
	task.CloseReason = domain.CloseReasonAbandoned

	// Clear agent info
	task.Agent = ""
	task.Session = ""

	// Save task
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &CloseTaskOutput{Task: task}, nil
}
