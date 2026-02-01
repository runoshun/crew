package shared

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteReview_StatusCheck(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	executor := testutil.NewMockCommandExecutor()
	executor.ExecuteOutput = []byte("Review output")
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	clock := &testutil.MockClock{NowTime: now}

	deps := ReviewDeps{
		Tasks:    repo,
		Executor: executor,
		Clock:    clock,
	}

	t.Run("invalid status without skip", func(t *testing.T) {
		// Use todo status which is invalid for review
		input := ReviewInput{
			Task: &domain.Task{
				ID:     1,
				Title:  "Review task",
				Status: domain.StatusTodo, // Not in_progress or done
			},
			WorktreePath: "/tmp/worktree",
			Result: domain.RenderCommandResult{
				Prompt:  "prompt",
				Command: "echo review",
			},
		}
		_, err := ExecuteReview(context.Background(), deps, input)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("valid status - in_progress", func(t *testing.T) {
		task := &domain.Task{
			ID:     2,
			Title:  "Review task",
			Status: domain.StatusInProgress, // Valid for review
		}
		input := ReviewInput{
			Task:         task,
			WorktreePath: "/tmp/worktree",
			Result: domain.RenderCommandResult{
				Prompt:  "prompt",
				Command: "echo review",
			},
		}
		reviewOut, err := ExecuteReview(context.Background(), deps, input)
		require.NoError(t, err)
		require.NotNil(t, reviewOut)
		// ExecuteReview no longer updates task metadata - that's done by caller
	})

	t.Run("skip status check", func(t *testing.T) {
		task := &domain.Task{
			ID:     3,
			Title:  "Review task",
			Status: domain.StatusError, // Would be invalid without skip
		}
		input := ReviewInput{
			Task:            task,
			WorktreePath:    "/tmp/worktree",
			Result:          domain.RenderCommandResult{Prompt: "prompt", Command: "echo review"},
			SkipStatusCheck: true,
		}
		reviewOut, err := ExecuteReview(context.Background(), deps, input)
		require.NoError(t, err)
		require.NotNil(t, reviewOut)
		// ExecuteReview no longer updates task metadata - that's done by caller
	})
}

func TestExecuteReview_Verbose(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	clock := &testutil.MockClock{NowTime: now}

	t.Run("verbose mode streams both stdout and stderr to deps.Stderr", func(t *testing.T) {
		executor := testutil.NewMockCommandExecutor()
		executor.ExecuteOutput = []byte("Review output\n")
		executor.StderrOutput = []byte("stderr output\n")

		var stderrBuf bytes.Buffer
		deps := ReviewDeps{
			Tasks:    repo,
			Executor: executor,
			Clock:    clock,
			Stderr:   &stderrBuf,
		}

		task := &domain.Task{
			ID:     1,
			Title:  "Review task",
			Status: domain.StatusInProgress,
		}
		input := ReviewInput{
			Task:            task,
			WorktreePath:    "/tmp/worktree",
			Result:          domain.RenderCommandResult{Prompt: "prompt", Command: "echo review"},
			SkipStatusCheck: true,
			Verbose:         true,
		}
		reviewOut, err := ExecuteReview(context.Background(), deps, input)
		require.NoError(t, err)
		require.NotNil(t, reviewOut)

		// In verbose mode, both stdout and stderr should be written to deps.Stderr
		// stdout contains the reviewer output, stderr contains any error/debug output
		output := stderrBuf.String()
		assert.Contains(t, output, "Review output")
		assert.Contains(t, output, "stderr output")
	})

	t.Run("non-verbose mode does not stream to deps.Stderr", func(t *testing.T) {
		executor := testutil.NewMockCommandExecutor()
		executor.ExecuteOutput = []byte("Review output\n")
		executor.StderrOutput = []byte("stderr output\n")

		var stderrBuf bytes.Buffer
		deps := ReviewDeps{
			Tasks:    repo,
			Executor: executor,
			Clock:    clock,
			Stderr:   &stderrBuf,
		}

		task := &domain.Task{
			ID:     2,
			Title:  "Review task",
			Status: domain.StatusInProgress,
		}
		input := ReviewInput{
			Task:            task,
			WorktreePath:    "/tmp/worktree",
			Result:          domain.RenderCommandResult{Prompt: "prompt", Command: "echo review"},
			SkipStatusCheck: true,
			Verbose:         false, // Non-verbose mode
		}
		reviewOut, err := ExecuteReview(context.Background(), deps, input)
		require.NoError(t, err)
		require.NotNil(t, reviewOut)

		// In non-verbose mode, nothing should be written to deps.Stderr
		assert.Empty(t, stderrBuf.String())
	})
}

func TestUpdateReviewMetadata(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	clock := &testutil.MockClock{NowTime: now}

	t.Run("LGTM review", func(t *testing.T) {
		task := &domain.Task{ID: 1}
		UpdateReviewMetadata(clock, task, domain.ReviewLGTMPrefix+" Looks good!")

		assert.Equal(t, 1, task.ReviewCount)
		assert.Equal(t, now, task.LastReviewAt)
		require.NotNil(t, task.LastReviewIsLGTM)
		assert.True(t, *task.LastReviewIsLGTM)
	})

	t.Run("non-LGTM review", func(t *testing.T) {
		task := &domain.Task{ID: 2}
		UpdateReviewMetadata(clock, task, "❌ Needs changes")

		assert.Equal(t, 1, task.ReviewCount)
		assert.Equal(t, now, task.LastReviewAt)
		require.NotNil(t, task.LastReviewIsLGTM)
		assert.False(t, *task.LastReviewIsLGTM)
	})

	t.Run("increments review count", func(t *testing.T) {
		task := &domain.Task{ID: 3, ReviewCount: 2}
		UpdateReviewMetadata(clock, task, domain.ReviewLGTMPrefix)

		assert.Equal(t, 3, task.ReviewCount)
	})
}
