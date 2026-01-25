package usecase

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// PollStatusInput contains the parameters for polling tasks by status.
type PollStatusInput struct {
	Status   domain.Status // Status to wait for
	Interval int           // Polling interval in seconds (default: 10)
	Timeout  int           // Timeout in seconds (0 = no timeout)
}

// PollStatusOutput contains the result of polling for status.
type PollStatusOutput struct {
	Status domain.Status // Status of the found task
	TaskID int           // ID of the found task
}

// PollStatus is the use case for polling tasks by status.
type PollStatus struct {
	tasks  domain.TaskRepository
	clock  domain.Clock
	stdout io.Writer
}

// NewPollStatus creates a new PollStatus use case.
func NewPollStatus(tasks domain.TaskRepository, clock domain.Clock, stdout io.Writer) *PollStatus {
	return &PollStatus{
		tasks:  tasks,
		clock:  clock,
		stdout: stdout,
	}
}

// Execute polls for tasks with the specified status.
// Returns immediately if a task with the status exists,
// otherwise waits until one appears or timeout is reached.
func (uc *PollStatus) Execute(ctx context.Context, in PollStatusInput) (*PollStatusOutput, error) {
	// Validate input
	if !in.Status.IsValid() {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidStatus, in.Status)
	}

	// Validate interval
	if in.Interval <= 0 {
		in.Interval = 10
	}

	// Check immediately for existing task with status
	task, err := uc.findTaskWithStatus(in.Status)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}
	if task != nil {
		uc.output(task.ID, task.Status)
		return &PollStatusOutput{
			TaskID: task.ID,
			Status: task.Status,
		}, nil
	}

	ticker := time.NewTicker(time.Duration(in.Interval) * time.Second)
	defer ticker.Stop()

	// Setup timeout if specified
	var timeoutChan <-chan time.Time
	if in.Timeout > 0 {
		timer := time.NewTimer(time.Duration(in.Timeout) * time.Second)
		defer timer.Stop()
		timeoutChan = timer.C
	}

	for {
		select {
		case <-ctx.Done():
			// Context canceled is a normal exit
			if ctx.Err() == context.Canceled {
				return nil, nil
			}
			return nil, ctx.Err()
		case <-timeoutChan:
			// Timeout reached - exit without finding task
			return nil, nil
		case <-ticker.C:
			task, err := uc.findTaskWithStatus(in.Status)
			if err != nil {
				return nil, fmt.Errorf("find task: %w", err)
			}
			if task != nil {
				uc.output(task.ID, task.Status)
				return &PollStatusOutput{
					TaskID: task.ID,
					Status: task.Status,
				}, nil
			}
		}
	}
}

// findTaskWithStatus finds a task with the specified status.
// Returns the first matching task, or nil if none found.
func (uc *PollStatus) findTaskWithStatus(status domain.Status) (*domain.Task, error) {
	tasks, err := uc.tasks.List(domain.TaskFilter{})
	if err != nil {
		return nil, err
	}

	for _, t := range tasks {
		if t.Status == status {
			return t, nil
		}
	}

	return nil, nil
}

// output writes the task info to stdout.
func (uc *PollStatus) output(taskID int, status domain.Status) {
	_, _ = fmt.Fprintf(uc.stdout, "%s %d\n", status, taskID)
}
