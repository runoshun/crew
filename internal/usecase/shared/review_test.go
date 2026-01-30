package shared

import (
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
	executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\n" + domain.ReviewLGTMPrefix + "\nLooks good.")
	clock := &testutil.MockClock{NowTime: time.Now()}

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
		input := ReviewInput{
			Task: &domain.Task{
				ID:     2,
				Title:  "Review task",
				Status: domain.StatusInProgress, // Valid for review
			},
			WorktreePath: "/tmp/worktree",
			Result: domain.RenderCommandResult{
				Prompt:  "prompt",
				Command: "echo review",
			},
		}
		reviewOut, err := ExecuteReview(context.Background(), deps, input)
		require.NoError(t, err)
		require.NotNil(t, reviewOut)
		assert.True(t, reviewOut.IsLGTM)
	})

	t.Run("skip status check", func(t *testing.T) {
		input := ReviewInput{
			Task: &domain.Task{
				ID:     3,
				Title:  "Review task",
				Status: domain.StatusError, // Would be invalid without skip
			},
			WorktreePath:    "/tmp/worktree",
			Result:          domain.RenderCommandResult{Prompt: "prompt", Command: "echo review"},
			SkipStatusCheck: true,
		}
		reviewOut, err := ExecuteReview(context.Background(), deps, input)
		require.NoError(t, err)
		require.NotNil(t, reviewOut)
		assert.True(t, reviewOut.IsLGTM)
	})
}
