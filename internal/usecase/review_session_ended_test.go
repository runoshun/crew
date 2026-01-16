package usecase

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewSessionEnded_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Now()}
	crewDir := t.TempDir()

	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task being reviewed",
		Status: domain.StatusReviewing,
	}

	uc := NewReviewSessionEnded(repo, clock, crewDir, io.Discard)

	// Execute
	out, err := uc.Execute(context.Background(), ReviewSessionEndedInput{
		TaskID:   1,
		ExitCode: 0,
		Output:   "Review passed: looks good!",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.Ignored)

	// Verify task was updated
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusReviewed, task.Status)

	// Verify comment was added via repository
	comments, err := repo.GetComments(1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "reviewer", comments[0].Author)
	assert.Equal(t, "Review passed: looks good!", comments[0].Text)
}

func TestReviewSessionEnded_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Now()}
	crewDir := t.TempDir()

	uc := NewReviewSessionEnded(repo, clock, crewDir, io.Discard)

	// Execute
	out, err := uc.Execute(context.Background(), ReviewSessionEndedInput{
		TaskID:   999,
		ExitCode: 0,
		Output:   "Review output",
	})

	// Assert - should be ignored (not an error)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.Ignored)
}

func TestReviewSessionEnded_Execute_NotReviewing(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Now()}
	crewDir := t.TempDir()

	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task not in reviewing status",
		Status: domain.StatusInProgress,
	}

	uc := NewReviewSessionEnded(repo, clock, crewDir, io.Discard)

	// Execute
	out, err := uc.Execute(context.Background(), ReviewSessionEndedInput{
		TaskID:   1,
		ExitCode: 0,
		Output:   "Review output",
	})

	// Assert - should be ignored
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.Ignored)

	// Verify task status was not changed
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusInProgress, task.Status)
}

func TestReviewSessionEnded_Execute_NonZeroExitCode(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Now()}
	crewDir := t.TempDir()

	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task being reviewed",
		Status: domain.StatusReviewing,
	}

	uc := NewReviewSessionEnded(repo, clock, crewDir, io.Discard)

	// Execute with non-zero exit code
	out, err := uc.Execute(context.Background(), ReviewSessionEndedInput{
		TaskID:   1,
		ExitCode: 1,
		Output:   "Review failed with error",
	})

	// Assert - should still succeed and save comment
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.Ignored)

	// Verify task was reverted to for_review for retry (not reviewed)
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusForReview, task.Status)

	// Verify comment was added (even with non-zero exit)
	comments, err := repo.GetComments(1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
}

func TestReviewSessionEnded_Execute_CleanupsScriptFiles(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Now()}
	crewDir := t.TempDir()

	// Create script and prompt files
	scriptsDir := filepath.Join(crewDir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0o755))

	scriptPath := filepath.Join(scriptsDir, "review-1.sh")
	promptPath := filepath.Join(scriptsDir, "review-1-prompt.txt")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/bash"), 0o755))
	require.NoError(t, os.WriteFile(promptPath, []byte("Review prompt"), 0o644))

	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task being reviewed",
		Status: domain.StatusReviewing,
	}

	uc := NewReviewSessionEnded(repo, clock, crewDir, io.Discard)

	// Execute
	out, err := uc.Execute(context.Background(), ReviewSessionEndedInput{
		TaskID:   1,
		ExitCode: 0,
		Output:   "Review complete",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.Ignored)

	// Verify script files were cleaned up
	_, err = os.Stat(scriptPath)
	assert.True(t, os.IsNotExist(err), "script file should be deleted")
	_, err = os.Stat(promptPath)
	assert.True(t, os.IsNotExist(err), "prompt file should be deleted")
}
