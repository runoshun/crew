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
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Task 2",
		Status: domain.StatusForReview, // Target status
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, clock, &stdout)

	// Execute
	ctx := context.Background()
	start := time.Now()
	out, err := uc.Execute(ctx, PollStatusInput{
		Status:   domain.StatusForReview,
		Interval: 10,
		Timeout:  0,
	})

	// Assert - should exit immediately
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 2, out.TaskID)
	assert.Equal(t, domain.StatusForReview, out.Status)

	// Should exit very quickly
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 100*time.Millisecond)

	// Check output format
	assert.Equal(t, "for_review 2\n", stdout.String())
}

func TestPollStatus_Execute_WaitForStatus(t *testing.T) {
	// Setup - no task has the target status initially
	repo := NewThreadSafeTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusInProgress,
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, clock, &stdout)

	// Change task status after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		repo.UpdateStatus(1, domain.StatusForReview)
	}()

	// Execute with short interval
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out, err := uc.Execute(ctx, PollStatusInput{
		Status:   domain.StatusForReview,
		Interval: 1,
		Timeout:  0,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 1, out.TaskID)
	assert.Equal(t, domain.StatusForReview, out.Status)
	assert.Equal(t, "for_review 1\n", stdout.String())
}

func TestPollStatus_Execute_Timeout(t *testing.T) {
	// Setup - no task has the target status
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusInProgress,
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, clock, &stdout)

	// Execute with timeout
	ctx := context.Background()
	start := time.Now()
	out, err := uc.Execute(ctx, PollStatusInput{
		Status:   domain.StatusForReview,
		Interval: 1,
		Timeout:  1, // 1 second timeout
	})

	// Assert - should timeout without finding task
	require.NoError(t, err)
	assert.Nil(t, out)
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 1*time.Second)
	assert.Less(t, elapsed, 2*time.Second)

	// No output when not found
	assert.Empty(t, stdout.String())
}

func TestPollStatus_Execute_InvalidStatus(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, clock, &stdout)

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
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusInProgress,
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, clock, &stdout)

	// Create context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Execute
	out, err := uc.Execute(ctx, PollStatusInput{
		Status:   domain.StatusForReview,
		Interval: 1,
		Timeout:  0,
	})

	// Assert - Ctrl+C (context.Canceled) should be treated as normal exit
	assert.NoError(t, err)
	assert.Nil(t, out)
}

func TestPollStatus_Execute_DefaultInterval(t *testing.T) {
	// Setup
	repo := NewThreadSafeTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusInProgress,
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, clock, &stdout)

	// Change to target status after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		repo.UpdateStatus(1, domain.StatusForReview)
	}()

	// Execute with interval <= 0 (should use default)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	out, err := uc.Execute(ctx, PollStatusInput{
		Status:   domain.StatusForReview,
		Interval: 0, // Should use default (10)
		Timeout:  0,
	})

	// Assert - should still work with default interval
	require.NoError(t, err)
	require.NotNil(t, out)
}

func TestPollStatus_Execute_ListError(t *testing.T) {
	// Setup
	repo := &testutil.MockTaskRepositoryWithListError{
		MockTaskRepository: testutil.NewMockTaskRepository(),
		ListErr:            errors.New("database error"),
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, clock, &stdout)

	// Execute
	_, err := uc.Execute(context.Background(), PollStatusInput{
		Status:   domain.StatusForReview,
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
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, clock, &stdout)

	// Execute with timeout
	out, err := uc.Execute(context.Background(), PollStatusInput{
		Status:   domain.StatusForReview,
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
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusForReview,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Task 2",
		Status: domain.StatusForReview,
	}

	var stdout bytes.Buffer
	uc := NewPollStatus(repo, clock, &stdout)

	// Execute
	ctx := context.Background()
	out, err := uc.Execute(ctx, PollStatusInput{
		Status:   domain.StatusForReview,
		Interval: 10,
		Timeout:  0,
	})

	// Assert - should return one of the matching tasks
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusForReview, out.Status)
	// TaskID should be either 1 or 2
	assert.True(t, out.TaskID == 1 || out.TaskID == 2)
}

func TestPollStatus_Execute_AllStatuses(t *testing.T) {
	// Test that we can poll for various valid statuses
	statuses := []domain.Status{
		domain.StatusTodo,
		domain.StatusInProgress,
		domain.StatusNeedsInput,
		domain.StatusForReview,
		domain.StatusReviewing,
		domain.StatusReviewed,
		domain.StatusStopped,
		domain.StatusError,
		domain.StatusClosed,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			repo := NewThreadSafeTaskRepository()
			clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
			repo.Tasks[1] = &domain.Task{
				ID:     1,
				Title:  "Task 1",
				Status: status,
			}

			var stdout bytes.Buffer
			uc := NewPollStatus(repo, clock, &stdout)

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
