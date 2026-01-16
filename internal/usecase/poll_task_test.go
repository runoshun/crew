package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ThreadSafeTaskRepository wraps MockTaskRepository with mutex for concurrent access
type ThreadSafeTaskRepository struct {
	*testutil.MockTaskRepository
	mu sync.RWMutex
}

func NewThreadSafeTaskRepository() *ThreadSafeTaskRepository {
	return &ThreadSafeTaskRepository{
		MockTaskRepository: testutil.NewMockTaskRepository(),
	}
}

func (r *ThreadSafeTaskRepository) Get(id int) (*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, err := r.MockTaskRepository.Get(id)
	if err != nil || task == nil {
		return task, err
	}
	// Return a copy to avoid race conditions
	taskCopy := *task
	return &taskCopy, nil
}

func (r *ThreadSafeTaskRepository) Save(task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.MockTaskRepository.Save(task)
}

func (r *ThreadSafeTaskRepository) UpdateStatus(id int, status domain.Status) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if task, ok := r.Tasks[id]; ok {
		// Create a copy to avoid race on Status field
		updatedTask := *task
		updatedTask.Status = status
		r.Tasks[id] = &updatedTask
	}
}

func TestPollTask_Execute_StatusChange(t *testing.T) {
	// Setup
	repo := NewThreadSafeTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}

	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

	// Start a goroutine to change status after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		repo.UpdateStatus(1, domain.StatusInProgress)
		time.Sleep(50 * time.Millisecond)
		repo.UpdateStatus(1, domain.StatusDone)
	}()

	// Execute with short interval
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := uc.Execute(ctx, PollTaskInput{
		TaskID:          1,
		Interval:        1, // Short interval for faster test
		Timeout:         0,
		CommandTemplate: "",
	})

	// Assert - should exit when status becomes done (terminal state)
	require.NoError(t, err)
}

func TestPollTask_Execute_Timeout(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}

	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

	// Execute with timeout
	ctx := context.Background()
	start := time.Now()
	_, err := uc.Execute(ctx, PollTaskInput{
		TaskID:   1,
		Interval: 1,
		Timeout:  1, // 1 second timeout
	})

	// Assert
	require.NoError(t, err)
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 1*time.Second)
	assert.Less(t, elapsed, 2*time.Second)
}

func TestPollTask_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

	// Execute
	_, err := uc.Execute(context.Background(), PollTaskInput{
		TaskID:   999,
		Interval: 1,
		Timeout:  0,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestPollTask_Execute_GetTaskError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = errors.New("database error")
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

	// Execute
	_, err := uc.Execute(context.Background(), PollTaskInput{
		TaskID:   1,
		Interval: 1,
		Timeout:  0,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestPollTask_Execute_ContextCanceled(t *testing.T) {
	// Setup
	repo := NewThreadSafeTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}

	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

	// Create context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Execute
	_, err := uc.Execute(ctx, PollTaskInput{
		TaskID:   1,
		Interval: 1,
		Timeout:  0,
	})

	// Assert - Ctrl+C (context.Canceled) should be treated as normal exit
	assert.NoError(t, err)
}

func TestPollTask_Execute_TerminalStates(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
	}{
		{"done", domain.StatusDone},
		{"closed", domain.StatusClosed},
		{"error", domain.StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repo := NewThreadSafeTaskRepository()
			clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
			repo.Tasks[1] = &domain.Task{
				ID:     1,
				Title:  "Test task",
				Status: domain.StatusTodo,
			}

			executor := testutil.NewMockCommandExecutor()
			uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

			// Change to terminal status after short delay
			go func() {
				time.Sleep(50 * time.Millisecond)
				repo.UpdateStatus(1, tt.status)
			}()

			// Execute
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			start := time.Now()
			_, err := uc.Execute(ctx, PollTaskInput{
				TaskID:   1,
				Interval: 1,
				Timeout:  0,
			})

			// Assert - should exit quickly when terminal state is reached
			require.NoError(t, err)
			elapsed := time.Since(start)
			// Should exit within 2 seconds (1 polling interval + some buffer)
			assert.Less(t, elapsed, 2*time.Second)
		})
	}
}

func TestPollTask_Execute_DefaultInterval(t *testing.T) {
	// Setup
	repo := NewThreadSafeTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}

	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

	// Change status to terminal after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		repo.UpdateStatus(1, domain.StatusDone)
	}()

	// Execute with interval <= 0 (should use default)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := uc.Execute(ctx, PollTaskInput{
		TaskID:   1,
		Interval: 0, // Should use default (10)
		Timeout:  0,
	})

	// Assert
	require.NoError(t, err)
}

func TestPollTask_Execute_ImmediateTerminalState(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
	}{
		{"done", domain.StatusDone},
		{"closed", domain.StatusClosed},
		{"error", domain.StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repo := NewThreadSafeTaskRepository()
			clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
			repo.Tasks[1] = &domain.Task{
				ID:     1,
				Title:  "Test task",
				Status: tt.status, // Already in terminal state
			}

			executor := testutil.NewMockCommandExecutor()
			uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

			// Execute
			ctx := context.Background()
			start := time.Now()
			_, err := uc.Execute(ctx, PollTaskInput{
				TaskID:   1,
				Interval: 10,
				Timeout:  0,
			})

			// Assert - should exit immediately without polling
			require.NoError(t, err)
			elapsed := time.Since(start)
			assert.Less(t, elapsed, 100*time.Millisecond)
		})
	}
}

func TestPollTask_isTerminalStatus(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

	tests := []struct {
		name     string
		status   domain.Status
		expected bool
	}{
		{"done is terminal", domain.StatusDone, true},
		{"closed is terminal", domain.StatusClosed, true},
		{"error is terminal", domain.StatusError, true},
		{"todo is not terminal", domain.StatusTodo, false},
		{"in_progress is not terminal", domain.StatusInProgress, false},
		{"stopped is not terminal", domain.StatusStopped, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.isTerminalStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPollTask_Execute_CommandOutput(t *testing.T) {
	// Setup
	repo := NewThreadSafeTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, &stdout, &stderr)

	// Start a goroutine to change status to terminal state after a delay
	// Use delay longer than polling interval to ensure we catch the change
	go func() {
		time.Sleep(1500 * time.Millisecond)
		repo.UpdateStatus(1, domain.StatusDone)
	}()

	// Execute with command template
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := uc.Execute(ctx, PollTaskInput{
		TaskID:          1,
		Interval:        1,
		Timeout:         0,
		CommandTemplate: `echo "{{.TaskID}}: {{.OldStatus}} -> {{.NewStatus}}"`,
	})

	// Assert
	require.NoError(t, err)

	// Check that command was executed with correct template expansion
	assert.True(t, executor.ExecuteWithContextCalled)
	assert.NotNil(t, executor.ExecutedCmd)
	assert.Equal(t, "sh", executor.ExecutedCmd.Program)
	assert.Len(t, executor.ExecutedCmd.Args, 2)
	assert.Equal(t, "-c", executor.ExecutedCmd.Args[0])
	assert.Contains(t, executor.ExecutedCmd.Args[1], "1: todo -> done")
}

func TestPollTask_Execute_ExpectedStatus_Match(t *testing.T) {
	// Setup - task is in expected status
	repo := NewThreadSafeTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress, // Current status matches expected
	}

	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

	// Change to terminal state after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		repo.UpdateStatus(1, domain.StatusDone)
	}()

	// Execute with expected status that matches current status
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := uc.Execute(ctx, PollTaskInput{
		TaskID:           1,
		ExpectedStatuses: []domain.Status{domain.StatusInProgress},
		Interval:         1,
		Timeout:          0,
		CommandTemplate:  "",
	})

	// Assert - should continue polling (not exit immediately)
	require.NoError(t, err)
}

func TestPollTask_Execute_ExpectedStatus_Mismatch(t *testing.T) {
	// Setup - task is NOT in expected status
	repo := NewThreadSafeTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInReview, // Current status differs from expected
	}

	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

	// Execute with expected status that differs from current status
	ctx := context.Background()
	start := time.Now()

	_, err := uc.Execute(ctx, PollTaskInput{
		TaskID:           1,
		ExpectedStatuses: []domain.Status{domain.StatusInProgress},
		Interval:         1,
		Timeout:          0,
		CommandTemplate:  `echo "{{.TaskID}}: {{.OldStatus}} -> {{.NewStatus}}"`,
	})

	// Assert - should exit immediately (not wait for polling interval)
	require.NoError(t, err)
	elapsed := time.Since(start)
	// Use more lenient threshold to avoid flakiness (should be well under 1 second)
	assert.Less(t, elapsed, 500*time.Millisecond)

	// Check that command was executed
	assert.True(t, executor.ExecuteWithContextCalled)
	assert.NotNil(t, executor.ExecutedCmd)
	assert.Contains(t, executor.ExecutedCmd.Args[1], "1: in_progress")
}

func TestPollTask_Execute_ExpectedStatus_Multiple(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus domain.Status
		expected      []domain.Status
		shouldMatch   bool
	}{
		{
			name:          "matches first expected status",
			currentStatus: domain.StatusInProgress,
			expected:      []domain.Status{domain.StatusInProgress, domain.StatusNeedsInput},
			shouldMatch:   true,
		},
		{
			name:          "matches second expected status",
			currentStatus: domain.StatusNeedsInput,
			expected:      []domain.Status{domain.StatusInProgress, domain.StatusNeedsInput},
			shouldMatch:   true,
		},
		{
			name:          "does not match any expected status",
			currentStatus: domain.StatusInReview,
			expected:      []domain.Status{domain.StatusInProgress, domain.StatusNeedsInput},
			shouldMatch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repo := NewThreadSafeTaskRepository()
			clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
			repo.Tasks[1] = &domain.Task{
				ID:     1,
				Title:  "Test task",
				Status: tt.currentStatus,
			}

			executor := testutil.NewMockCommandExecutor()
			uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

			// Change to terminal state after short delay if should match
			if tt.shouldMatch {
				go func() {
					time.Sleep(50 * time.Millisecond)
					repo.UpdateStatus(1, domain.StatusDone)
				}()
			}

			// Execute
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			start := time.Now()
			_, err := uc.Execute(ctx, PollTaskInput{
				TaskID:           1,
				ExpectedStatuses: tt.expected,
				Interval:         1,
				Timeout:          0,
				CommandTemplate:  `echo "{{.TaskID}}: {{.OldStatus}} -> {{.NewStatus}}"`,
			})

			// Assert
			require.NoError(t, err)

			if tt.shouldMatch {
				// Should continue polling (not exit immediately)
				elapsed := time.Since(start)
				assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
			} else {
				// Should exit immediately (use lenient threshold to avoid flakiness)
				elapsed := time.Since(start)
				assert.Less(t, elapsed, 500*time.Millisecond)
				assert.True(t, executor.ExecuteWithContextCalled)
			}
		})
	}
}

func TestPollTask_Execute_ExpectedStatus_Empty(t *testing.T) {
	// Setup - empty expected statuses (should behave like before)
	repo := NewThreadSafeTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}

	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

	// Change to terminal state after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		repo.UpdateStatus(1, domain.StatusDone)
	}()

	// Execute with empty expected statuses
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := uc.Execute(ctx, PollTaskInput{
		TaskID:           1,
		ExpectedStatuses: []domain.Status{}, // Empty
		Interval:         1,
		Timeout:          0,
	})

	// Assert - should behave like normal polling
	require.NoError(t, err)
}

func TestPollTask_containsStatus(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	executor := testutil.NewMockCommandExecutor()
	uc := NewPollTask(repo, clock, executor, io.Discard, io.Discard)

	tests := []struct {
		name     string
		statuses []domain.Status
		target   domain.Status
		expected bool
	}{
		{
			name:     "contains status",
			statuses: []domain.Status{domain.StatusTodo, domain.StatusInProgress, domain.StatusDone},
			target:   domain.StatusInProgress,
			expected: true,
		},
		{
			name:     "does not contain status",
			statuses: []domain.Status{domain.StatusTodo, domain.StatusInProgress},
			target:   domain.StatusDone,
			expected: false,
		},
		{
			name:     "empty slice",
			statuses: []domain.Status{},
			target:   domain.StatusTodo,
			expected: false,
		},
		{
			name:     "single element match",
			statuses: []domain.Status{domain.StatusInProgress},
			target:   domain.StatusInProgress,
			expected: true,
		},
		{
			name:     "single element no match",
			statuses: []domain.Status{domain.StatusInProgress},
			target:   domain.StatusTodo,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.containsStatus(tt.statuses, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}
