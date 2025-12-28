package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// CopyTaskInput contains the parameters for copying a task.
// Fields are ordered to minimize memory padding.
type CopyTaskInput struct {
	Title    *string // New title (optional, defaults to "<original> (copy)")
	SourceID int     // Source task ID to copy
}

// CopyTaskOutput contains the result of copying a task.
type CopyTaskOutput struct {
	TaskID int // The ID of the new task
}

// CopyTask is the use case for copying a task.
type CopyTask struct {
	tasks domain.TaskRepository
	clock domain.Clock
}

// NewCopyTask creates a new CopyTask use case.
func NewCopyTask(tasks domain.TaskRepository, clock domain.Clock) *CopyTask {
	return &CopyTask{
		tasks: tasks,
		clock: clock,
	}
}

// Execute copies a task with the given input.
// The new task copies: title (with " (copy)" suffix), description, labels.
// The new task does NOT copy: issue, PR, comments.
// The base branch is set to the source task's branch name.
func (uc *CopyTask) Execute(_ context.Context, in CopyTaskInput) (*CopyTaskOutput, error) {
	// Get source task
	source, err := uc.tasks.Get(in.SourceID)
	if err != nil {
		return nil, fmt.Errorf("get source task: %w", err)
	}
	if source == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Get next task ID
	id, err := uc.tasks.NextID()
	if err != nil {
		return nil, fmt.Errorf("generate task ID: %w", err)
	}

	// Determine title
	title := source.Title + " (copy)"
	if in.Title != nil {
		title = *in.Title
	}

	// Inherit base branch from source (copy uses same base)
	baseBranch := source.BaseBranch

	// Copy labels (create new slice to avoid sharing)
	var labels []string
	if len(source.Labels) > 0 {
		labels = make([]string, len(source.Labels))
		copy(labels, source.Labels)
	}

	// Create new task
	now := uc.clock.Now()
	task := &domain.Task{
		ID:          id,
		ParentID:    source.ParentID, // Inherit parent
		Title:       title,
		Description: source.Description,
		Status:      domain.StatusTodo,
		Created:     now,
		BaseBranch:  baseBranch,
		Labels:      labels,
		// NOT copied: Issue, PR, Agent, Session, Started
	}

	// Save task
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &CopyTaskOutput{TaskID: id}, nil
}
