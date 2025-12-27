package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ShowTaskInput contains the parameters for showing a task.
type ShowTaskInput struct {
	TaskID int // Task ID (required)
}

// ShowTaskOutput contains the result of showing a task.
type ShowTaskOutput struct {
	Task     *domain.Task     // The task details
	Children []*domain.Task   // Direct child tasks
	Comments []domain.Comment // Comments on the task
}

// ShowTask is the use case for displaying task details.
type ShowTask struct {
	tasks domain.TaskRepository
}

// NewShowTask creates a new ShowTask use case.
func NewShowTask(tasks domain.TaskRepository) *ShowTask {
	return &ShowTask{
		tasks: tasks,
	}
}

// Execute retrieves and returns the task details.
func (uc *ShowTask) Execute(_ context.Context, in ShowTaskInput) (*ShowTaskOutput, error) {
	// Get task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Get children
	children, err := uc.tasks.GetChildren(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get children: %w", err)
	}

	// Get comments
	comments, err := uc.tasks.GetComments(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get comments: %w", err)
	}

	return &ShowTaskOutput{
		Task:     task,
		Children: children,
		Comments: comments,
	}, nil
}
