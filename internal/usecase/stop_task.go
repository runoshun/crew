package usecase

import (
	"context"
	"fmt"
	"os"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// StopTaskInput contains the parameters for stopping a task session.
type StopTaskInput struct {
	TaskID int // Task ID to stop
}

// StopTaskOutput contains the result of stopping a task session.
type StopTaskOutput struct {
	Task        *domain.Task // The stopped task
	SessionName string       // Name of the session that was stopped (if any)
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
// 3. Clearing agent info (work sessions only)
// 4. Updating status to stopped (if session exists)
func (uc *StopTask) Execute(_ context.Context, in StopTaskInput) (*StopTaskOutput, error) {
	// Get the task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	sessionStopped, err := shared.StopSession(uc.sessions, task.ID)
	if err != nil {
		return nil, err
	}
	if sessionStopped != "" {
		// Set status to error when stopping (indicates manual stop or abnormal termination)
		task.Status = domain.StatusError
		uc.cleanupScriptFiles(task.ID)
		return uc.saveStoppedTask(task, sessionStopped)
	}

	if task.Session != "" {
		// Set status to error when stopping
		task.Status = domain.StatusError
		uc.cleanupScriptFiles(task.ID)
		return uc.saveStoppedTask(task, "")
	}

	uc.cleanupScriptFiles(task.ID)
	return uc.saveStoppedTask(task, "")
}

func (uc *StopTask) saveStoppedTask(task *domain.Task, sessionStopped string) (*StopTaskOutput, error) {
	// Clear agent info
	task.Agent = ""
	task.Session = ""

	// Save task
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &StopTaskOutput{Task: task, SessionName: sessionStopped}, nil
}

// cleanupScriptFiles removes the generated script file.
func (uc *StopTask) cleanupScriptFiles(taskID int) {
	scriptPath := domain.ScriptPath(uc.crewDir, taskID)
	_ = os.Remove(scriptPath)
}
