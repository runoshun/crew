package shared

import (
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// GetTask retrieves a task by ID and returns domain.ErrTaskNotFound if not found.
// This centralizes the common pattern of:
//
//	task, err := repo.Get(taskID)
//	if err != nil { return nil, fmt.Errorf("get task: %w", err) }
//	if task == nil { return nil, domain.ErrTaskNotFound }
func GetTask(repo domain.TaskRepository, taskID int) (*domain.Task, error) {
	task, err := repo.Get(taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}
	return task, nil
}
