package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
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
	tasks domain.TaskRepository
}

// NewCloseTask creates a new CloseTask use case.
func NewCloseTask(tasks domain.TaskRepository) *CloseTask {
	return &CloseTask{
		tasks: tasks,
	}
}

// Execute closes a task by transitioning its status to closed.
// Note: In this phase, this only updates the status.
// Session termination and worktree cleanup will be added in later phases.
func (uc *CloseTask) Execute(_ context.Context, in CloseTaskInput) (*CloseTaskOutput, error) {
	// Get the task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Validate status transition
	if !task.Status.CanTransitionTo(domain.StatusClosed) {
		return nil, fmt.Errorf("cannot close task in %s status: %w", task.Status, domain.ErrInvalidTransition)
	}

	// Update status
	task.Status = domain.StatusClosed

	// Clear agent info (in case session was running)
	task.Agent = ""
	task.Session = ""

	// Save task
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &CloseTaskOutput{Task: task}, nil
}
