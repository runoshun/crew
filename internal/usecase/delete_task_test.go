package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteTask_Execute_Success(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
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
	_, exists := repo.tasks[1]
	assert.False(t, exists, "task should be deleted from repository")
}

func TestDeleteTask_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
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
	repo := newMockTaskRepository()
	repo.getErr = assert.AnError
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
	repo := &mockTaskRepositoryWithDeleteError{
		mockTaskRepository: newMockTaskRepository(),
		deleteErr:          assert.AnError,
	}
	repo.tasks[1] = &domain.Task{
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

// mockTaskRepositoryWithDeleteError extends mockTaskRepository to return error on Delete.
type mockTaskRepositoryWithDeleteError struct {
	*mockTaskRepository
	deleteErr error
}

func (m *mockTaskRepositoryWithDeleteError) Delete(_ int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return m.mockTaskRepository.Delete(0)
}
