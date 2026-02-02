package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ShowTaskInput contains the parameters for showing a task.
type ShowTaskInput struct {
	CommentsBy string // Filter comments by author (optional)
	TaskID     int    // Task ID (required)
	LastReview bool   // Show only the latest review comment (author="reviewer")
}

// ShowTaskOutput contains the result of showing a task.
type ShowTaskOutput struct {
	Task     *domain.Task     // The task details
	Children []*domain.Task   // Direct child tasks
	Comments []domain.Comment // Comments on the task
}

// ShowTask is the use case for displaying task details.
type ShowTask struct {
	tasks     domain.TaskRepository
	acpStates domain.ACPStateStore
}

// NewShowTask creates a new ShowTask use case.
func NewShowTask(tasks domain.TaskRepository, acpStates domain.ACPStateStore) *ShowTask {
	return &ShowTask{
		tasks:     tasks,
		acpStates: acpStates,
	}
}

// Execute retrieves and returns the task details.
func (uc *ShowTask) Execute(ctx context.Context, in ShowTaskInput) (*ShowTaskOutput, error) {
	// Get task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}
	if attachErr := uc.attachExecutionSubstate(ctx, task); attachErr != nil {
		return nil, attachErr
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

	// Filter comments
	if in.LastReview {
		// Find latest comment by "reviewer" based on Time
		var latestReview *domain.Comment
		for i := range comments {
			if comments[i].Author == "reviewer" {
				if latestReview == nil || comments[i].Time.After(latestReview.Time) {
					latestReview = &comments[i]
				}
			}
		}

		if latestReview != nil {
			comments = []domain.Comment{*latestReview}
		} else {
			comments = []domain.Comment{}
		}
	} else if in.CommentsBy != "" {
		// Filter by author
		filtered := make([]domain.Comment, 0, len(comments))
		for _, c := range comments {
			if c.Author == in.CommentsBy {
				filtered = append(filtered, c)
			}
		}
		comments = filtered
	}

	return &ShowTaskOutput{
		Task:     task,
		Children: children,
		Comments: comments,
	}, nil
}

func (uc *ShowTask) attachExecutionSubstate(ctx context.Context, task *domain.Task) error {
	if uc.acpStates == nil || task == nil {
		return nil
	}
	namespace := task.Namespace
	if namespace == "" {
		namespace = domain.DefaultNamespace
	}
	state, err := uc.acpStates.Load(ctx, namespace, task.ID)
	if err != nil {
		if errors.Is(err, domain.ErrACPStateNotFound) {
			return nil
		}
		return fmt.Errorf("load acp state for %s#%d: %w", namespace, task.ID, err)
	}
	task.ExecutionSubstate = state.ExecutionSubstate
	return nil
}
