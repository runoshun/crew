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
			sessions := testutil.NewMockSessionManager()
			worktrees := testutil.NewMockWorktreeManager()
			uc := NewCloseTask(repo, sessions, worktrees)

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

func TestCloseTask_Execute_StopsRunningSession(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task with running session",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true // Session is running
	worktrees := testutil.NewMockWorktreeManager()
	uc := NewCloseTask(repo, sessions, worktrees)

	// Execute
	out, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, sessions.StopCalled, "session should be stopped")
	assert.Equal(t, domain.StatusClosed, out.Task.Status)
}

func TestCloseTask_Execute_DeletesWorktree(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task with worktree",
		Status:  domain.StatusInReview,
		Agent:   "",
		Session: "",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true // Worktree exists
	uc := NewCloseTask(repo, sessions, worktrees)

	// Execute
	out, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, worktrees.RemoveCalled, "worktree should be removed")
	assert.Equal(t, domain.StatusClosed, out.Task.Status)
}

func TestCloseTask_Execute_StopsSessionAndDeletesWorktree(t *testing.T) {
	// Setup - task with both running session and worktree
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task with session and worktree",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	uc := NewCloseTask(repo, sessions, worktrees)

	// Execute
	out, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, sessions.StopCalled, "session should be stopped")
	assert.True(t, worktrees.RemoveCalled, "worktree should be removed")
	assert.Equal(t, domain.StatusClosed, out.Task.Status)
}

func TestCloseTask_Execute_NoSessionOrWorktree(t *testing.T) {
	// Setup - task with no session and no worktree
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Fresh task",
		Status: domain.StatusTodo,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = false
	uc := NewCloseTask(repo, sessions, worktrees)

	// Execute
	out, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, sessions.StopCalled, "stop should not be called when no session")
	assert.False(t, worktrees.RemoveCalled, "remove should not be called when no worktree")
	assert.Equal(t, domain.StatusClosed, out.Task.Status)
}

func TestCloseTask_Execute_AlreadyClosed(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Already closed task",
		Status: domain.StatusClosed,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	uc := NewCloseTask(repo, sessions, worktrees)

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
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	uc := NewCloseTask(repo, sessions, worktrees)

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
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	uc := NewCloseTask(repo, sessions, worktrees)

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
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	uc := NewCloseTask(repo, sessions, worktrees)

	// Execute
	_, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")
}

func TestCloseTask_Execute_SessionCheckError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to close",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningErr = assert.AnError
	worktrees := testutil.NewMockWorktreeManager()
	uc := NewCloseTask(repo, sessions, worktrees)

	// Execute
	_, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check session running")
}

func TestCloseTask_Execute_SessionStopError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to close",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.StopErr = assert.AnError
	worktrees := testutil.NewMockWorktreeManager()
	uc := NewCloseTask(repo, sessions, worktrees)

	// Execute
	_, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stop session")
}

func TestCloseTask_Execute_WorktreeCheckError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to close",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsErr = assert.AnError
	uc := NewCloseTask(repo, sessions, worktrees)

	// Execute
	_, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check worktree exists")
}

func TestCloseTask_Execute_WorktreeRemoveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to close",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	worktrees.RemoveErr = assert.AnError
	uc := NewCloseTask(repo, sessions, worktrees)

	// Execute
	_, err := uc.Execute(context.Background(), CloseTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remove worktree")
}
