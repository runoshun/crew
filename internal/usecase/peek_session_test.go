package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeekSession_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.PeekOutput = "line1\nline2\nline3"

	uc := NewPeekSession(repo, sessions)

	// Execute
	out, err := uc.Execute(context.Background(), PeekSessionInput{
		TaskID: 1,
		Lines:  30,
	})

	// Assert
	require.NoError(t, err)
	assert.True(t, sessions.PeekCalled)
	assert.Equal(t, 30, sessions.PeekLines)
	assert.Equal(t, "line1\nline2\nline3", out.Output)
}

func TestPeekSession_Execute_DefaultLines(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.PeekOutput = "output"

	uc := NewPeekSession(repo, sessions)

	// Execute with Lines=0 (should use default)
	out, err := uc.Execute(context.Background(), PeekSessionInput{
		TaskID: 1,
		Lines:  0,
	})

	// Assert
	require.NoError(t, err)
	assert.True(t, sessions.PeekCalled)
	assert.Equal(t, DefaultPeekLines, sessions.PeekLines)
	assert.Equal(t, "output", out.Output)
}

func TestPeekSession_Execute_NegativeLines(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.PeekOutput = "output"

	uc := NewPeekSession(repo, sessions)

	// Execute with negative Lines (should use default)
	out, err := uc.Execute(context.Background(), PeekSessionInput{
		TaskID: 1,
		Lines:  -5,
	})

	// Assert
	require.NoError(t, err)
	assert.True(t, sessions.PeekCalled)
	assert.Equal(t, DefaultPeekLines, sessions.PeekLines)
	assert.Equal(t, "output", out.Output)
}

func TestPeekSession_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	sessions := testutil.NewMockSessionManager()

	uc := NewPeekSession(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), PeekSessionInput{
		TaskID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestPeekSession_Execute_NoSession(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false

	uc := NewPeekSession(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), PeekSessionInput{
		TaskID: 1,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestPeekSession_Execute_IsRunningError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningErr = assert.AnError

	uc := NewPeekSession(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), PeekSessionInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check session")
}

func TestPeekSession_Execute_PeekError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.PeekErr = assert.AnError

	uc := NewPeekSession(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), PeekSessionInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "peek session")
}

func TestPeekSession_Execute_GetTaskError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	sessions := testutil.NewMockSessionManager()

	uc := NewPeekSession(repo, sessions)

	// Execute
	_, err := uc.Execute(context.Background(), PeekSessionInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}
