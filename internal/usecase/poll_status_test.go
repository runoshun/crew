package usecase

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPollStatus_Execute_ImmediateMatch(t *testing.T) {
	// Setup - task already has the target status
	repo := NewThreadSafeTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Task 2",
		Status: domain.StatusDone, // Target status
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, &stdout)

	// Execute
	ctx := context.Background()
	out, err := uc.Execute(ctx, PollStatusInput{
		Status:   domain.StatusDone,
		Interval: 10,
		Timeout:  0,
	})

	// Assert - should exit immediately
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 2, out.TaskID)
	assert.Equal(t, domain.StatusDone, out.Status)

	// Check output format
	assert.Equal(t, "done 2\n", stdout.String())
}

func TestPollStatus_Execute_WaitForStatus(t *testing.T) {
	// Setup - no task has the target status initially
	repo := NewThreadSafeTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusInProgress,
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, &stdout)

	// Change task status after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		repo.UpdateStatus(1, domain.StatusDone)
	}()

	// Execute with short interval
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out, err := uc.Execute(ctx, PollStatusInput{
		Status:   domain.StatusDone,
		Interval: 1,
		Timeout:  0,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 1, out.TaskID)
	assert.Equal(t, domain.StatusDone, out.Status)
	assert.Equal(t, "done 1\n", stdout.String())
}

func TestPollStatus_Execute_Timeout(t *testing.T) {
	// Setup - no task has the target status
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusInProgress,
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, &stdout)

	// Execute with context timeout (avoids real-time dependency)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	out, err := uc.Execute(ctx, PollStatusInput{
		Status:   domain.StatusDone,
		Interval: 10, // Long interval, but context will timeout first
		Timeout:  0,  // No usecase timeout, rely on context
	})

	// Assert - should exit via context timeout without finding task
	require.NoError(t, err)
	assert.Nil(t, out)

	// No output when not found
	assert.Empty(t, stdout.String())
}

func TestPollStatus_Execute_InvalidStatus(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, &stdout)

	// Execute with invalid status
	_, err := uc.Execute(context.Background(), PollStatusInput{
		Status:   domain.Status("invalid_status"),
		Interval: 1,
		Timeout:  0,
	})

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidStatus)
}

func TestPollStatus_Execute_ContextCanceled(t *testing.T) {
	// Setup
	repo := NewThreadSafeTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusInProgress,
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, &stdout)

	// Create context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Execute
	out, err := uc.Execute(ctx, PollStatusInput{
		Status:   domain.StatusDone,
		Interval: 1,
		Timeout:  0,
	})

	// Assert - Ctrl+C (context.Canceled) should be treated as normal exit
	assert.NoError(t, err)
	assert.Nil(t, out)
}

func TestPollStatus_Execute_DefaultInterval(t *testing.T) {
	// Test that interval <= 0 uses default (10s) and still works
	// by having a task already match the target status (immediate exit)
	repo := NewThreadSafeTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusDone, // Already matches target
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, &stdout)

	// Execute with interval <= 0 (should use default, but exits immediately due to match)
	out, err := uc.Execute(context.Background(), PollStatusInput{
		Status:   domain.StatusDone,
		Interval: 0, // Should use default (10), but won't matter since immediate match
		Timeout:  0,
	})

	// Assert - should still work with default interval (exits immediately)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 1, out.TaskID)
}

func TestPollStatus_Execute_ListError(t *testing.T) {
	// Setup
	repo := &testutil.MockTaskRepositoryWithListError{
		MockTaskRepository: testutil.NewMockTaskRepository(),
		ListErr:            errors.New("database error"),
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, &stdout)

	// Execute
	_, err := uc.Execute(context.Background(), PollStatusInput{
		Status:   domain.StatusDone,
		Interval: 1,
		Timeout:  0,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find task")
}

func TestPollStatus_Execute_EmptyTaskList(t *testing.T) {
	// Setup - no tasks at all
	repo := testutil.NewMockTaskRepository()

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, &stdout)

	// Execute with timeout
	out, err := uc.Execute(context.Background(), PollStatusInput{
		Status:   domain.StatusDone,
		Interval: 1,
		Timeout:  1,
	})

	// Assert - should timeout without finding task
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestPollStatus_Execute_MultipleMatchingTasks(t *testing.T) {
	// Setup - multiple tasks with same status
	repo := NewThreadSafeTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusDone,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Task 2",
		Status: domain.StatusDone,
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, &stdout)

	// Execute
	ctx := context.Background()
	out, err := uc.Execute(ctx, PollStatusInput{
		Status:   domain.StatusDone,
		Interval: 10,
		Timeout:  0,
	})

	// Assert - should return one of the matching tasks
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusDone, out.Status)
	// TaskID should be either 1 or 2
	assert.True(t, out.TaskID == 1 || out.TaskID == 2)
}

func TestPollStatus_Execute_AllStatuses(t *testing.T) {
	// Test that we can poll for various valid statuses
	statuses := []domain.Status{
		domain.StatusTodo,
		domain.StatusInProgress,
		domain.StatusInProgress,
		domain.StatusDone,
		domain.StatusInProgress,
		domain.StatusDone,
		domain.StatusError,
		domain.StatusError,
		domain.StatusClosed,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			repo := NewThreadSafeTaskRepository()
			repo.Tasks[1] = &domain.Task{
				ID:     1,
				Title:  "Task 1",
				Status: status,
			}

			var stdout bytes.Buffer
			uc := NewPollStatus(repo, &stdout)

			out, err := uc.Execute(context.Background(), PollStatusInput{
				Status:   status,
				Interval: 10,
				Timeout:  0,
			})

			require.NoError(t, err)
			require.NotNil(t, out)
			assert.Equal(t, 1, out.TaskID)
			assert.Equal(t, status, out.Status)
		})
	}
}
