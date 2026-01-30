package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"text/template"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// PollTaskInput contains the parameters for polling a task.
type PollTaskInput struct {
	CommandTemplate  string          // Command template to execute on status change
	ExpectedStatuses []domain.Status // Expected statuses (if current status differs, notify immediately)
	TaskIDs          []int           // Task IDs to poll (monitors multiple tasks)
	Interval         int             // Polling interval in seconds (default: 10)
	Timeout          int             // Timeout in seconds (0 = no timeout)
}

// PollTaskOutput contains the result of polling a task.
type PollTaskOutput struct {
	// No output needed - this is a long-running operation
}

// PollTask is the use case for polling task status changes.
type PollTask struct {
	tasks    domain.TaskRepository
	clock    domain.Clock
	executor domain.CommandExecutor
	stdout   io.Writer
	stderr   io.Writer
}

// NewPollTask creates a new PollTask use case.
func NewPollTask(tasks domain.TaskRepository, clock domain.Clock, executor domain.CommandExecutor, stdout, stderr io.Writer) *PollTask {
	return &PollTask{
		tasks:    tasks,
		clock:    clock,
		executor: executor,
		stdout:   stdout,
		stderr:   stderr,
	}
}

// CommandData holds data for command template expansion.
type CommandData struct {
	OldStatus string
	NewStatus string
	TaskID    int
}

// Execute polls the task status and executes the command on status change.
func (uc *PollTask) Execute(ctx context.Context, in PollTaskInput) (*PollTaskOutput, error) {
	// Validate input
	if len(in.TaskIDs) == 0 {
		return nil, fmt.Errorf("at least one task ID is required")
	}

	// Validate interval
	if in.Interval <= 0 {
		in.Interval = 10
	}

	// Get initial state for all tasks
	previousStatuses := make(map[int]domain.Status)
	var activeTaskIDs []int // Tasks that are not in terminal state
	for _, taskID := range in.TaskIDs {
		task, err := uc.tasks.Get(taskID)
		if err != nil {
			return nil, fmt.Errorf("get task %d: %w", taskID, err)
		}
		if task == nil {
			return nil, fmt.Errorf("task %d: %w", taskID, domain.ErrTaskNotFound)
		}

		// Check if current status matches expected statuses (immediate notification if different)
		// This must come before terminal check to ensure notification even for terminal states
		if len(in.ExpectedStatuses) > 0 {
			if !uc.containsStatus(in.ExpectedStatuses, task.Status) {
				// Current status differs from expected - notify immediately
				if in.CommandTemplate != "" {
					// Join expected statuses for display
					expectedStr := ""
					for i, s := range in.ExpectedStatuses {
						if i > 0 {
							expectedStr += ","
						}
						expectedStr += string(s)
					}
					data := CommandData{
						TaskID:    taskID,
						OldStatus: expectedStr, // Show all expected statuses
						NewStatus: string(task.Status),
					}
					if err := uc.executeCommand(ctx, in.CommandTemplate, data); err != nil {
						return nil, fmt.Errorf("execute command: %w", err)
					}
				}
				return &PollTaskOutput{}, nil
			}
		}

		previousStatuses[taskID] = task.Status

		// Track non-terminal tasks for monitoring
		if !uc.isTerminalStatus(task.Status) {
			activeTaskIDs = append(activeTaskIDs, taskID)
		}
	}

	// If ALL tasks are already in terminal state, exit immediately
	if len(activeTaskIDs) == 0 {
		return &PollTaskOutput{}, nil
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
			// Ctrl+C (context.Canceled) is a normal exit, not an error
			if ctx.Err() == context.Canceled {
				return &PollTaskOutput{}, nil
			}
			return nil, ctx.Err()
		case <-timeoutChan:
			// Timeout reached
			return &PollTaskOutput{}, nil
		case <-ticker.C:
			// Poll only active (non-terminal) tasks
			var stillActiveTaskIDs []int
			for _, taskID := range activeTaskIDs {
				task, err := uc.tasks.Get(taskID)
				if err != nil {
					return nil, fmt.Errorf("get task %d: %w", taskID, err)
				}
				if task == nil {
					return nil, fmt.Errorf("task %d: %w", taskID, domain.ErrTaskNotFound)
				}

				previousStatus := previousStatuses[taskID]
				// Check for status change
				if task.Status != previousStatus {
					// Execute command
					if in.CommandTemplate != "" {
						data := CommandData{
							TaskID:    taskID,
							OldStatus: string(previousStatus),
							NewStatus: string(task.Status),
						}
						if err := uc.executeCommand(ctx, in.CommandTemplate, data); err != nil {
							return nil, fmt.Errorf("execute command: %w", err)
						}
					}
					// Exit immediately after one change detection
					return &PollTaskOutput{}, nil
				}

				// Keep tracking non-terminal tasks
				if !uc.isTerminalStatus(task.Status) {
					stillActiveTaskIDs = append(stillActiveTaskIDs, taskID)
				}
			}

			// If all tasks have become terminal, exit
			if len(stillActiveTaskIDs) == 0 {
				return &PollTaskOutput{}, nil
			}
			activeTaskIDs = stillActiveTaskIDs
		}
	}
}

// isTerminalStatus returns true if the status is a terminal state.
func (uc *PollTask) isTerminalStatus(status domain.Status) bool {
	return status.IsTerminal()
}

// containsStatus returns true if the slice contains the specified status.
func (uc *PollTask) containsStatus(statuses []domain.Status, target domain.Status) bool {
	for _, s := range statuses {
		if s == target {
			return true
		}
	}
	return false
}

// executeCommand executes the command template with the given data.
func (uc *PollTask) executeCommand(ctx context.Context, cmdTemplate string, data CommandData) error {
	// Parse template
	tmpl, err := template.New("command").Parse(cmdTemplate)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// Run command via CommandExecutor
	execCmd := &domain.ExecCommand{
		Program: "sh",
		Args:    []string{"-c", buf.String()},
	}
	if err := uc.executor.ExecuteWithContext(ctx, execCmd, uc.stdout, uc.stderr); err != nil {
		return fmt.Errorf("run command: %w", err)
	}

	return nil
}
