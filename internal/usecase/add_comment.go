package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// AddCommentInput contains the parameters for adding a comment.
// Fields are ordered to minimize memory padding.
type AddCommentInput struct {
	Message string // Comment text (required)
	TaskID  int    // Task ID (required)
}

// AddCommentOutput contains the result of adding a comment.
type AddCommentOutput struct {
	Comment domain.Comment // The created comment
}

// AddComment is the use case for adding a comment to a task.
type AddComment struct {
	tasks domain.TaskRepository
	clock domain.Clock
}

// NewAddComment creates a new AddComment use case.
func NewAddComment(tasks domain.TaskRepository, clock domain.Clock) *AddComment {
	return &AddComment{
		tasks: tasks,
		clock: clock,
	}
}

// Execute adds a comment to a task.
func (uc *AddComment) Execute(_ context.Context, in AddCommentInput) (*AddCommentOutput, error) {
	// Validate message
	message := strings.TrimSpace(in.Message)
	if message == "" {
		return nil, domain.ErrEmptyMessage
	}

	// Verify task exists
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Create comment
	comment := domain.Comment{
		Text: message,
		Time: uc.clock.Now(),
	}

	// Save comment
	if err := uc.tasks.AddComment(in.TaskID, comment); err != nil {
		return nil, fmt.Errorf("add comment: %w", err)
	}

	return &AddCommentOutput{
		Comment: comment,
	}, nil
}
