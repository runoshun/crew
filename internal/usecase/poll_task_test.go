package usecase

import (
	"context"
	"errors"
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

	uc := NewPollTask(repo, clock)

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

	uc := NewPollTask(repo, clock)

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
	uc := NewPollTask(repo, clock)

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
	uc := NewPollTask(repo, clock)

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

	uc := NewPollTask(repo, clock)

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

			uc := NewPollTask(repo, clock)

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

	uc := NewPollTask(repo, clock)

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

			uc := NewPollTask(repo, clock)

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
	uc := NewPollTask(repo, clock)

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
