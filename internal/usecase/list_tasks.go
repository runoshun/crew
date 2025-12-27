package usecase

import (
	"context"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ListTasksInput contains the parameters for listing tasks.
type ListTasksInput struct {
	ParentID *int     // Filter by parent task ID (nil = all tasks)
	Labels   []string // Filter by labels (AND condition)
}

// ListTasksOutput contains the result of listing tasks.
type ListTasksOutput struct {
	Tasks []*domain.Task // List of tasks matching the filter
}

// ListTasks is the use case for listing tasks.
type ListTasks struct {
	tasks domain.TaskRepository
}

// NewListTasks creates a new ListTasks use case.
func NewListTasks(tasks domain.TaskRepository) *ListTasks {
	return &ListTasks{
		tasks: tasks,
	}
}

// Execute lists tasks matching the given input criteria.
func (uc *ListTasks) Execute(_ context.Context, in ListTasksInput) (*ListTasksOutput, error) {
	filter := domain.TaskFilter{
		ParentID: in.ParentID,
		Labels:   in.Labels,
	}

	tasks, err := uc.tasks.List(filter)
	if err != nil {
		return nil, err
	}

	return &ListTasksOutput{Tasks: tasks}, nil
}
