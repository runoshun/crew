// Package usecase contains application use cases.
package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// NewTaskInput contains the parameters for creating a new task.
// Fields are ordered to minimize memory padding.
type NewTaskInput struct {
	ParentID    *int     // Parent task ID (optional, nil = root task)
	Title       string   // Task title (required)
	Description string   // Task description (optional)
	BaseBranch  string   // Base branch (optional, empty = use default)
	Labels      []string // Labels (optional)
	Issue       int      // Linked GitHub issue number (0 = not linked)
	SkipReview  bool     // Skip review on completion (go directly to reviewed)
}

// NewTaskOutput contains the result of creating a new task.
type NewTaskOutput struct {
	TaskID int // The ID of the created task
}

// NewTask is the use case for creating a new task.
type NewTask struct {
	tasks  domain.TaskRepository
	git    domain.Git
	clock  domain.Clock
	logger domain.Logger
}

// NewNewTask creates a new NewTask use case.
func NewNewTask(tasks domain.TaskRepository, git domain.Git, clock domain.Clock, logger domain.Logger) *NewTask {
	return &NewTask{
		tasks:  tasks,
		git:    git,
		clock:  clock,
		logger: logger,
	}
}

// Execute creates a new task with the given input.
func (uc *NewTask) Execute(_ context.Context, in NewTaskInput) (*NewTaskOutput, error) {
	// Validate title
	if in.Title == "" {
		return nil, domain.ErrEmptyTitle
	}

	// Validate parent exists if specified
	if in.ParentID != nil {
		parent, err := uc.tasks.Get(*in.ParentID)
		if err != nil {
			return nil, fmt.Errorf("get parent task: %w", err)
		}
		if parent == nil {
			return nil, domain.ErrParentNotFound
		}
	}

	// Get next task ID
	id, err := uc.tasks.NextID()
	if err != nil {
		return nil, fmt.Errorf("generate task ID: %w", err)
	}

	// Create task
	now := uc.clock.Now()
	baseBranch, err := resolveNewTaskBaseBranch(in.BaseBranch, uc.git)
	if err != nil {
		return nil, err
	}
	task := &domain.Task{
		ID:          id,
		ParentID:    in.ParentID,
		Title:       in.Title,
		Description: in.Description,
		Status:      domain.StatusTodo,
		Created:     now,
		Issue:       in.Issue,
		Labels:      in.Labels,
		BaseBranch:  baseBranch,
		SkipReview:  in.SkipReview,
	}

	// Save task
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	// Log task creation
	if uc.logger != nil {
		uc.logger.Info(id, "task", fmt.Sprintf("created: %q", in.Title))
	}

	return &NewTaskOutput{TaskID: id}, nil
}
