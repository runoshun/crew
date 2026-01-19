package usecase

import (
	"context"
	"fmt"
	"os"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// StopTaskInput contains the parameters for stopping a task session.
type StopTaskInput struct {
	TaskID int // Task ID to stop
}

// StopTaskOutput contains the result of stopping a task session.
type StopTaskOutput struct {
	Task *domain.Task // The stopped task
}

// StopTask is the use case for stopping a task session.
type StopTask struct {
	tasks    domain.TaskRepository
	sessions domain.SessionManager
	crewDir  string
}

// NewStopTask creates a new StopTask use case.
func NewStopTask(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	crewDir string,
) *StopTask {
	return &StopTask{
		tasks:    tasks,
		sessions: sessions,
		crewDir:  crewDir,
	}
}

// Execute stops a task session by:
// 1. Terminating the tmux session
// 2. Deleting the task script
// 3. Clearing agent info
// 4. Updating status to stopped (if session exists)
func (uc *StopTask) Execute(_ context.Context, in StopTaskInput) (*StopTaskOutput, error) {
	// Get the task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Get session name
	sessionName := domain.SessionName(task.ID)

	// Stop session if running
	hadSession := task.Session != ""
	running, err := uc.sessions.IsRunning(sessionName)
	if err != nil {
		return nil, fmt.Errorf("check session running: %w", err)
	}
	if running {
		if stopErr := uc.sessions.Stop(sessionName); stopErr != nil {
			return nil, fmt.Errorf("stop session: %w", stopErr)
		}
	}

	// Update status to stopped if a session was associated or is running
	if hadSession || running {
		task.Status = domain.StatusStopped
	}

	// Delete task script (ignore errors)
	uc.cleanupScriptFiles(task.ID)

	// Clear agent info
	task.Agent = ""
	task.Session = ""

	// Save task
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &StopTaskOutput{Task: task}, nil
}

// cleanupScriptFiles removes the generated script file.
func (uc *StopTask) cleanupScriptFiles(taskID int) {
	scriptPath := domain.ScriptPath(uc.crewDir, taskID)
	_ = os.Remove(scriptPath)
}
