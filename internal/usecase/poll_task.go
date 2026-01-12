package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"text/template"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// PollTaskInput contains the parameters for polling a task.
type PollTaskInput struct {
	CommandTemplate string // Command template to execute on status change
	TaskID          int    // Task ID to poll
	Interval        int    // Polling interval in seconds (default: 10)
	Timeout         int    // Timeout in seconds (0 = no timeout)
}

// PollTaskOutput contains the result of polling a task.
type PollTaskOutput struct {
	// No output needed - this is a long-running operation
}

// PollTask is the use case for polling task status changes.
type PollTask struct {
	tasks  domain.TaskRepository
	clock  domain.Clock
	stdout io.Writer
	stderr io.Writer
}

// NewPollTask creates a new PollTask use case.
func NewPollTask(tasks domain.TaskRepository, clock domain.Clock, stdout, stderr io.Writer) *PollTask {
	return &PollTask{
		tasks:  tasks,
		clock:  clock,
		stdout: stdout,
		stderr: stderr,
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
	// Validate interval
	if in.Interval <= 0 {
		in.Interval = 10
	}

	// Get initial task state
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Check if task is already in terminal state (immediate exit)
	if uc.isTerminalStatus(task.Status) {
		return &PollTaskOutput{}, nil
	}

	previousStatus := task.Status
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
			// Poll the task
			task, err := uc.tasks.Get(in.TaskID)
			if err != nil {
				return nil, fmt.Errorf("get task: %w", err)
			}
			if task == nil {
				return nil, domain.ErrTaskNotFound
			}

			// Check for status change
			if task.Status != previousStatus {
				// Execute command
				if in.CommandTemplate != "" {
					data := CommandData{
						TaskID:    in.TaskID,
						OldStatus: string(previousStatus),
						NewStatus: string(task.Status),
					}
					if err := uc.executeCommand(in.CommandTemplate, data); err != nil {
						return nil, fmt.Errorf("execute command: %w", err)
					}
				}
				previousStatus = task.Status
			}

			// Check if task is in terminal state
			if uc.isTerminalStatus(task.Status) {
				return &PollTaskOutput{}, nil
			}
		}
	}
}

// isTerminalStatus returns true if the status is a terminal state.
func (uc *PollTask) isTerminalStatus(status domain.Status) bool {
	return status == domain.StatusDone || status == domain.StatusClosed || status == domain.StatusError
}

// executeCommand executes the command template with the given data.
func (uc *PollTask) executeCommand(cmdTemplate string, data CommandData) error {
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

	// Run command
	// #nosec G204 - Command template is user-controlled by design
	cmd := exec.Command("sh", "-c", buf.String())
	cmd.Stdout = uc.stdout
	cmd.Stderr = uc.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command: %w", err)
	}

	return nil
}
