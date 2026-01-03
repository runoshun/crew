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

func TestEditComment_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	repo.Comments[1] = []domain.Comment{
		{Text: "Original comment", Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
	uc := NewEditComment(repo, clock)

	// Execute
	err := uc.Execute(context.Background(), EditCommentInput{
		TaskID:  1,
		Index:   0,
		Message: "Updated comment",
	})

	// Assert
	require.NoError(t, err)

	// Verify comment updated
	comments := repo.Comments[1]
	require.Len(t, comments, 1)
	assert.Equal(t, "Updated comment", comments[0].Text)
	assert.Equal(t, clock.NowTime, comments[0].Time)
}

func TestEditComment_Execute_EmptyMessage(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1}
	clock := &testutil.MockClock{NowTime: time.Now()}
	uc := NewEditComment(repo, clock)

	// Execute with empty message
	err := uc.Execute(context.Background(), EditCommentInput{
		TaskID:  1,
		Index:   0,
		Message: "",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrEmptyMessage)
}

func TestEditComment_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Now()}
	uc := NewEditComment(repo, clock)

	// Execute with non-existent task
	err := uc.Execute(context.Background(), EditCommentInput{
		TaskID:  999,
		Index:   0,
		Message: "Updated comment",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestEditComment_Execute_CommentNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1}
	repo.Comments[1] = []domain.Comment{
		{Text: "Original comment", Time: time.Now()},
	}
	clock := &testutil.MockClock{NowTime: time.Now()}
	uc := NewEditComment(repo, clock)

	// Execute with invalid index
	err := uc.Execute(context.Background(), EditCommentInput{
		TaskID:  1,
		Index:   1, // Out of range
		Message: "Updated comment",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrCommentNotFound)
}

func TestEditComment_Execute_UpdateCommentError(t *testing.T) {
	// Setup
	repo := &testutil.MockTaskRepositoryWithUpdateCommentError{
		MockTaskRepository: testutil.NewMockTaskRepository(),
		UpdateCommentErr:   assert.AnError,
	}
	repo.Tasks[1] = &domain.Task{ID: 1}
	repo.Comments[1] = []domain.Comment{
		{Text: "Original comment", Time: time.Now()},
	}
	clock := &testutil.MockClock{NowTime: time.Now()}
	uc := NewEditComment(repo, clock)

	// Execute
	err := uc.Execute(context.Background(), EditCommentInput{
		TaskID:  1,
		Index:   0,
		Message: "Updated comment",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update comment")
}

func TestEditComment_Execute_GetTaskError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	clock := &testutil.MockClock{NowTime: time.Now()}
	uc := NewEditComment(repo, clock)

	// Execute
	err := uc.Execute(context.Background(), EditCommentInput{
		TaskID:  1,
		Index:   0,
		Message: "Updated comment",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}
