package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendKeys_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true

	uc := NewSendKeys(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), SendKeysInput{
		TaskID: 1,
		Keys:   "Enter",
	})

	// Assert
	require.NoError(t, err)
	assert.True(t, sessions.SendCalled)
	assert.Equal(t, "Enter", sessions.SentKeys)
}

func TestSendKeys_Execute_TextKeys(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true

	uc := NewSendKeys(repo, sessions)

	// Execute with text keys
	_, err := uc.Execute(context.Background(), SendKeysInput{
		TaskID: 1,
		Keys:   "hello world",
	})

	// Assert
	require.NoError(t, err)
	assert.True(t, sessions.SendCalled)
	assert.Equal(t, "hello world", sessions.SentKeys)
}

func TestSendKeys_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	sessions := testutil.NewMockSessionManager()

	uc := NewSendKeys(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), SendKeysInput{
		TaskID: 999,
		Keys:   "Enter",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestSendKeys_Execute_NoSession(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false

	uc := NewSendKeys(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), SendKeysInput{
		TaskID: 1,
		Keys:   "Enter",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestSendKeys_Execute_IsRunningError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningErr = assert.AnError

	uc := NewSendKeys(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), SendKeysInput{
		TaskID: 1,
		Keys:   "Enter",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check session")
}

func TestSendKeys_Execute_SendError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.SendErr = assert.AnError

	uc := NewSendKeys(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), SendKeysInput{
		TaskID: 1,
		Keys:   "Enter",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send keys")
}

func TestSendKeys_Execute_GetTaskError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	sessions := testutil.NewMockSessionManager()

	uc := NewSendKeys(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), SendKeysInput{
		TaskID: 1,
		Keys:   "Enter",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}
