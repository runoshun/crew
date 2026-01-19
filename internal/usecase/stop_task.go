package usecase

import (
	"context"
	"fmt"
	"os"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// StopTaskInput contains the parameters for stopping a task session.
type StopTaskInput struct {
	Review bool // Stop review session instead of work session
	TaskID int  // Task ID to stop
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
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	workSessionName := domain.SessionName(task.ID)
	reviewSessionName := domain.ReviewSessionName(task.ID)

	if in.Review {
		sessionStopped, stopErr := uc.stopSession(reviewSessionName)
		if stopErr != nil {
			return nil, stopErr
		}
		uc.cleanupReviewScriptFiles(task.ID)
		return &StopTaskOutput{Task: task, SessionName: sessionStopped}, nil
	}

	sessionStopped, err := uc.stopSession(workSessionName)
	if err != nil {
		return nil, err
	}
	if sessionStopped != "" {
		if task.Status != domain.StatusReviewing {
			task.Status = domain.StatusStopped
		}
		uc.cleanupScriptFiles(task.ID)
		return uc.saveStoppedTask(task, sessionStopped)
	}

	if task.Session != "" {
		if task.Status != domain.StatusReviewing {
			task.Status = domain.StatusStopped
			uc.cleanupScriptFiles(task.ID)
			return uc.saveStoppedTask(task, "")
		}
	}

	reviewStopped, err := uc.stopSession(reviewSessionName)
	if err != nil {
		return nil, err
	}
	if reviewStopped != "" {
		uc.cleanupReviewScriptFiles(task.ID)
		return &StopTaskOutput{Task: task, SessionName: reviewStopped}, nil
	}

	uc.cleanupScriptFiles(task.ID)
	return uc.saveStoppedTask(task, "")
}

func (uc *StopTask) stopSession(sessionName string) (string, error) {
	running, err := uc.sessions.IsRunning(sessionName)
	if err != nil {
		return "", fmt.Errorf("check session running: %w", err)
	}
	if !running {
		return "", nil
	}

	if stopErr := uc.sessions.Stop(sessionName); stopErr != nil {
		return "", fmt.Errorf("stop session: %w", stopErr)
	}
	return sessionName, nil
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

// cleanupReviewScriptFiles removes the generated review script files.
func (uc *StopTask) cleanupReviewScriptFiles(taskID int) {
	scriptPath := domain.ReviewScriptPath(uc.crewDir, taskID)
	_ = os.Remove(scriptPath)
	promptPath := domain.ReviewPromptPath(uc.crewDir, taskID)
	_ = os.Remove(promptPath)
}
