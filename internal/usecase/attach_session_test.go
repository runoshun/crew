package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachSession_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true

	uc := NewAttachSession(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), AttachSessionInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	assert.True(t, sessions.AttachCalled)
}

func TestAttachSession_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	sessions := testutil.NewMockSessionManager()

	uc := NewAttachSession(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), AttachSessionInput{
		TaskID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestAttachSession_Execute_NoSession(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false

	uc := NewAttachSession(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), AttachSessionInput{
		TaskID: 1,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestAttachSession_Execute_IsRunningError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningErr = assert.AnError

	uc := NewAttachSession(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), AttachSessionInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check session")
}

func TestAttachSession_Execute_AttachError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.AttachErr = assert.AnError

	uc := NewAttachSession(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), AttachSessionInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attach session")
}

func TestAttachSession_Execute_GetTaskError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	sessions := testutil.NewMockSessionManager()

	uc := NewAttachSession(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), AttachSessionInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}
