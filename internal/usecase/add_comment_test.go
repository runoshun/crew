package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddComment_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	sessions := testutil.NewMockSessionManager()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
	uc := NewAddComment(repo, sessions, clock)

	// Execute
	out, err := uc.Execute(context.Background(), AddCommentInput{
		TaskID:  1,
		Message: "This is a test comment",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "This is a test comment", out.Comment.Text)
	assert.Equal(t, clock.NowTime, out.Comment.Time)

	// Verify comment saved
	comments := repo.Comments[1]
	require.Len(t, comments, 1)
	assert.Equal(t, "This is a test comment", comments[0].Text)
	assert.Equal(t, clock.NowTime, comments[0].Time)
}

func TestAddComment_Execute_TrimsWhitespace(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	sessions := testutil.NewMockSessionManager()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
	uc := NewAddComment(repo, sessions, clock)

	// Execute with whitespace
	out, err := uc.Execute(context.Background(), AddCommentInput{
		TaskID:  1,
		Message: "  trimmed message  ",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "trimmed message", out.Comment.Text)
}

func TestAddComment_Execute_EmptyMessage(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	sessions := testutil.NewMockSessionManager()
	clock := &testutil.MockClock{NowTime: time.Now()}
	uc := NewAddComment(repo, sessions, clock)

	// Execute with empty message
	_, err := uc.Execute(context.Background(), AddCommentInput{
		TaskID:  1,
		Message: "",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrEmptyMessage)
}

func TestAddComment_Execute_WhitespaceOnlyMessage(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	sessions := testutil.NewMockSessionManager()
	clock := &testutil.MockClock{NowTime: time.Now()}
	uc := NewAddComment(repo, sessions, clock)

	// Execute with whitespace-only message
	_, err := uc.Execute(context.Background(), AddCommentInput{
		TaskID:  1,
		Message: "   ",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrEmptyMessage)
}

func TestAddComment_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	sessions := testutil.NewMockSessionManager()
	clock := &testutil.MockClock{NowTime: time.Now()}
	uc := NewAddComment(repo, sessions, clock)

	// Execute with non-existent task
	_, err := uc.Execute(context.Background(), AddCommentInput{
		TaskID:  999,
		Message: "Comment on missing task",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestAddComment_Execute_GetTaskError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	sessions := testutil.NewMockSessionManager()
	clock := &testutil.MockClock{NowTime: time.Now()}
	uc := NewAddComment(repo, sessions, clock)

	// Execute
	_, err := uc.Execute(context.Background(), AddCommentInput{
		TaskID:  1,
		Message: "Comment",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestAddComment_Execute_AddCommentError(t *testing.T) {
	// Setup
	repo := &testutil.MockTaskRepositoryWithAddCommentError{
		MockTaskRepository: testutil.NewMockTaskRepository(),
		AddCommentErr:      assert.AnError,
	}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	sessions := testutil.NewMockSessionManager()
	clock := &testutil.MockClock{NowTime: time.Now()}
	uc := NewAddComment(repo, sessions, clock)

	// Execute
	_, err := uc.Execute(context.Background(), AddCommentInput{
		TaskID:  1,
		Message: "Comment",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "add comment")
}

func TestAddComment_Execute_MultipleComments(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	sessions := testutil.NewMockSessionManager()
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := &testutil.MockClock{NowTime: baseTime}
	uc := NewAddComment(repo, sessions, clock)

	// Add first comment
	_, err := uc.Execute(context.Background(), AddCommentInput{
		TaskID:  1,
		Message: "First comment",
	})
	require.NoError(t, err)

	// Add second comment (with different time)
	clock.NowTime = baseTime.Add(time.Hour)
	_, err = uc.Execute(context.Background(), AddCommentInput{
		TaskID:  1,
		Message: "Second comment",
	})
	require.NoError(t, err)

	// Verify both comments saved
	comments := repo.Comments[1]
	require.Len(t, comments, 2)
	assert.Equal(t, "First comment", comments[0].Text)
	assert.Equal(t, baseTime, comments[0].Time)
	assert.Equal(t, "Second comment", comments[1].Text)
	assert.Equal(t, baseTime.Add(time.Hour), comments[1].Time)
}

func TestAddComment_Execute_RequestChanges(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
	uc := NewAddComment(repo, sessions, clock)

	// Execute with RequestChanges
	out, err := uc.Execute(context.Background(), AddCommentInput{
		TaskID:         1,
		Message:        "修正してください",
		RequestChanges: true,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "修正してください", out.Comment.Text)

	// Verify status changed to needs_changes
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusNeedsChanges, task.Status)

	// Verify notification was sent (Send called)
	assert.True(t, sessions.SendCalled)
}

func TestAddComment_Execute_RequestChanges_SessionNotRunning(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
	uc := NewAddComment(repo, sessions, clock)

	// Execute with RequestChanges
	out, err := uc.Execute(context.Background(), AddCommentInput{
		TaskID:         1,
		Message:        "修正してください",
		RequestChanges: true,
	})

	// Assert - should still succeed even if session not running
	require.NoError(t, err)
	assert.Equal(t, "修正してください", out.Comment.Text)

	// Verify status changed to needs_changes
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusNeedsChanges, task.Status)

	// Verify no notification was sent
	assert.False(t, sessions.SendCalled)
}
