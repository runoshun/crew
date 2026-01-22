package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// EditCommentInput specifies the input for the EditComment UseCase.
type EditCommentInput struct {
	Message string
	TaskID  int
	Index   int
}

// EditComment UseCase handles updating an existing comment.
type EditComment struct {
	tasks domain.TaskRepository
	clock domain.Clock
}

// NewEditComment creates a new EditComment UseCase.
func NewEditComment(tasks domain.TaskRepository, clock domain.Clock) *EditComment {
	return &EditComment{
		tasks: tasks,
		clock: clock,
	}
}

// Execute updates an existing comment of a task.
func (uc *EditComment) Execute(_ context.Context, in EditCommentInput) error {
	// Validate message
	message, err := shared.ValidateMessage(in.Message)
	if err != nil {
		return err
	}

	// Verify task exists
	_, err = shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return err
	}

	// Create updated comment
	comment := domain.Comment{
		Text: message,
		Time: uc.clock.Now(),
	}

	// Update comment
	if err := uc.tasks.UpdateComment(in.TaskID, in.Index, comment); err != nil {
		return fmt.Errorf("update comment: %w", err)
	}

	return nil
}
