package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// DeleteTaskInput contains the parameters for deleting a task.
type DeleteTaskInput struct {
	TaskID int // Task ID to delete
}

// DeleteTaskOutput contains the result of deleting a task.
type DeleteTaskOutput struct {
	// Empty for now; may include deleted task info in the future
}

// DeleteTask is the use case for deleting a task.
type DeleteTask struct {
	tasks domain.TaskRepository
}

// NewDeleteTask creates a new DeleteTask use case.
func NewDeleteTask(tasks domain.TaskRepository) *DeleteTask {
	return &DeleteTask{
		tasks: tasks,
	}
}

// Execute deletes a task with the given ID.
// Note: In Phase 2, this only deletes from the store.
// Session termination and worktree cleanup will be added in later phases.
func (uc *DeleteTask) Execute(_ context.Context, in DeleteTaskInput) (*DeleteTaskOutput, error) {
	// Verify task exists
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Delete task from store
	if err := uc.tasks.Delete(in.TaskID); err != nil {
		return nil, fmt.Errorf("delete task: %w", err)
	}

	return &DeleteTaskOutput{}, nil
}
