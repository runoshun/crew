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

	input := ReviewInput{
		Task: &domain.Task{
			ID:     1,
			Title:  "Review task",
			Status: domain.StatusInProgress,
		},
		WorktreePath: "/tmp/worktree",
		Result: domain.RenderCommandResult{
			Prompt:  "prompt",
			Command: "echo review",
		},
	}

	deps := ReviewDeps{
		Tasks:    repo,
		Executor: executor,
		Clock:    clock,
	}

	t.Run("invalid status without skip", func(t *testing.T) {
		_, err := ExecuteReview(context.Background(), deps, input)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("skip status check", func(t *testing.T) {
		reviewOut, err := ExecuteReview(context.Background(), deps, ReviewInput{
			Task:            input.Task,
			WorktreePath:    input.WorktreePath,
			Result:          input.Result,
			SkipStatusCheck: true,
		})
		require.NoError(t, err)
		require.NotNil(t, reviewOut)
		assert.True(t, reviewOut.IsLGTM)
		assert.NotEmpty(t, repo.Comments[1])
	})
}
