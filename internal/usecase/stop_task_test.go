package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopTask_Execute_Success_InProgress(t *testing.T) {
	// Setup - task in progress with running session
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task in progress",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	uc := NewStopTask(repo, sessions, t.TempDir())

	// Execute
	out, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusError, out.Task.Status)
	assert.Empty(t, out.Task.Agent, "agent should be cleared")
	assert.Empty(t, out.Task.Session, "session should be cleared")
	assert.True(t, sessions.StopCalled, "session should be stopped")

	// Verify task is updated in repository
	savedTask := repo.Tasks[1]
	assert.Equal(t, domain.StatusError, savedTask.Status)
}

func TestStopTask_Execute_NoRunningSession(t *testing.T) {
	// Setup - task with no running session
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task with no session",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false // No running session
	uc := NewStopTask(repo, sessions, t.TempDir())

	// Execute
	out, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusError, out.Task.Status)
	assert.Empty(t, out.Task.Agent, "agent should be cleared")
	assert.Empty(t, out.Task.Session, "session should be cleared")
	assert.False(t, sessions.StopCalled, "stop should not be called when no session running")
}

func TestStopTask_Execute_StopReviewSession(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task in review",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	uc := NewStopTask(repo, sessions, t.TempDir())

	out, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
		Review: true,
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusInProgress, out.Task.Status)
	assert.Equal(t, domain.ReviewSessionName(1), out.SessionName)
	assert.True(t, sessions.StopCalled)
}

func TestStopTask_Execute_StopReviewSessionFallback(t *testing.T) {
	// Test: when work session is not running and task.Session is empty,
	// but review session is running, stop the review session
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task in review",
		Status:  domain.StatusInProgress,
		Agent:   "", // No agent
		Session: "", // No work session
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	sessionsByName := map[string]bool{
		domain.ReviewSessionName(1): true, // Review session is running
		"crew-1":                    false,
	}
	sessions.IsRunningFunc = func(name string) (bool, error) {
		return sessionsByName[name], nil
	}
	uc := NewStopTask(repo, sessions, t.TempDir())

	out, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.ReviewSessionName(1), out.SessionName)
	assert.True(t, sessions.StopCalled)
}

func TestStopTask_Execute_NotInProgress(t *testing.T) {
	// Setup - task in todo status (no transition to in_review)
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task in todo",
		Status:  domain.StatusTodo,
		Agent:   "",
		Session: "",
	}
	sessions := testutil.NewMockSessionManager()
	uc := NewStopTask(repo, sessions, t.TempDir())

	// Execute
	out, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
	})

	// Assert - should succeed but not change status
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusTodo, out.Task.Status, "status should not change when no session")
}

func TestStopTask_Execute_AlreadyInReview(t *testing.T) {
	// Setup - task already in review
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task in review",
		Status:  domain.StatusDone,
		Agent:   "",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	uc := NewStopTask(repo, sessions, t.TempDir())

	// Execute
	out, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
	})

	// Assert - should succeed and move to stopped
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusError, out.Task.Status, "status should change to stopped when session exists")
}

func TestStopTask_Execute_ErrorStatus(t *testing.T) {
	// Setup - task in error status
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task in error",
		Status:  domain.StatusError,
		Agent:   "",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	uc := NewStopTask(repo, sessions, t.TempDir())

	// Execute
	out, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
	})

	// Assert - should succeed and move to stopped
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusError, out.Task.Status, "status should change to stopped when session exists")
}

func TestStopTask_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	sessions := testutil.NewMockSessionManager()
	uc := NewStopTask(repo, sessions, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestStopTask_Execute_GetError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	sessions := testutil.NewMockSessionManager()
	uc := NewStopTask(repo, sessions, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestStopTask_Execute_SessionCheckError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task to stop",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningErr = assert.AnError
	uc := NewStopTask(repo, sessions, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check session running")
}

func TestStopTask_Execute_SessionStopError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task to stop",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.StopErr = assert.AnError
	uc := NewStopTask(repo, sessions, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stop session")
}

func TestStopTask_Execute_SaveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task to stop",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}
	repo.SaveErr = assert.AnError
	sessions := testutil.NewMockSessionManager()
	uc := NewStopTask(repo, sessions, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")
}

func TestStopTask_Execute_ClearsAgentInfo(t *testing.T) {
	// Setup - verify agent info is cleared even when status doesn't change
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task with agent info",
		Status:  domain.StatusDone, // Not in_progress
		Agent:   "claude",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	uc := NewStopTask(repo, sessions, t.TempDir())

	// Execute
	out, err := uc.Execute(context.Background(), StopTaskInput{
		TaskID: 1,
	})

	// Assert - agent info should be cleared even though status changes
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusError, out.Task.Status)
	assert.Empty(t, out.Task.Agent, "agent should be cleared")
	assert.Empty(t, out.Task.Session, "session should be cleared")
	assert.True(t, sessions.StopCalled, "session should be stopped")
}
