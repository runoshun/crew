package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloseTask_Execute_Success(t *testing.T) {
	tests := []struct {
		name         string
		initialState domain.Status
	}{
		{"from todo", domain.StatusTodo},
		{"from in_progress", domain.StatusInProgress},
		{"from in_review", domain.StatusInReview},
		{"from error", domain.StatusError},
		{"from done", domain.StatusDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repo := testutil.NewMockTaskRepository()
			repo.Tasks[1] = &domain.Task{
				ID:      1,
				Title:   "Task to close",
				Status:  tt.initialState,
				Agent:   "claude",
				Session: "crew-1",
			}
			uc := NewCloseTask(repo)

			// Execute
			out, err := uc.Execute(context.Background(), CloseTaskInput{
				TaskID: 1,
			})

			// Assert
			require.NoError(t, err)
			require.NotNil(t, out)
			assert.Equal(t, domain.StatusClosed, out.Task.Status)
			assert.Empty(t, out.Task.Agent, "agent should be cleared")
			assert.Empty(t, out.Task.Session, "session should be cleared")

			// Verify task is updated in repository
			savedTask := repo.Tasks[1]
			assert.Equal(t, domain.StatusClosed, savedTask.Status)
		})
	}
}

func TestCloseTask_Execute_AlreadyClosed(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Already closed task",
		Status: domain.StatusClosed,
	}
	uc := NewCloseTask(repo)

	// Execute
	_, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert - closed cannot transition to closed
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)
}

func TestCloseTask_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	uc := NewCloseTask(repo)

	// Execute
	_, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestCloseTask_Execute_GetError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	uc := NewCloseTask(repo)

	// Execute
	_, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestCloseTask_Execute_SaveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to close",
		Status: domain.StatusTodo,
	}
	repo.SaveErr = assert.AnError
	uc := NewCloseTask(repo)

	// Execute
	_, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")
}
