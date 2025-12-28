package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteTask_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to delete",
		Status: domain.StatusTodo,
	}
	uc := NewDeleteTask(repo)

	// Execute
	out, err := uc.Execute(context.Background(), DeleteTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)

	// Verify task is deleted
	_, exists := repo.Tasks[1]
	assert.False(t, exists, "task should be deleted from repository")
}

func TestDeleteTask_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	uc := NewDeleteTask(repo)

	// Execute with non-existent task
	_, err := uc.Execute(context.Background(), DeleteTaskInput{
		TaskID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestDeleteTask_Execute_GetError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	uc := NewDeleteTask(repo)

	// Execute
	_, err := uc.Execute(context.Background(), DeleteTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestDeleteTask_Execute_DeleteError(t *testing.T) {
	// Setup
	repo := &testutil.MockTaskRepositoryWithDeleteError{
		MockTaskRepository: testutil.NewMockTaskRepository(),
		DeleteErr:          assert.AnError,
	}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to delete",
		Status: domain.StatusTodo,
	}
	uc := NewDeleteTask(repo)

	// Execute
	_, err := uc.Execute(context.Background(), DeleteTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete task")
}
