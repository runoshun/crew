package usecase

import (
	"context"
	"os"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// SessionEndedInput contains the parameters for handling session termination.
type SessionEndedInput struct {
	TaskID   int // Task ID
	ExitCode int // Exit code of the agent process
}

// SessionEndedOutput contains the result of handling session termination.
type SessionEndedOutput struct {
	Ignored bool // True if the callback was ignored (already cleaned up)
}

// SessionEnded is the use case for handling session termination.
// This is called by the task script's EXIT trap.
type SessionEnded struct {
	tasks   domain.TaskRepository
	crewDir string
}

// NewSessionEnded creates a new SessionEnded use case.
func NewSessionEnded(tasks domain.TaskRepository, crewDir string) *SessionEnded {
	return &SessionEnded{
		tasks:   tasks,
		crewDir: crewDir,
	}
}

// Execute handles session termination.
// It clears agent info, deletes script files, and updates status based on exit code.
func (uc *SessionEnded) Execute(_ context.Context, in SessionEndedInput) (*SessionEndedOutput, error) {
	// Get task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		// Task doesn't exist, nothing to do
		return &SessionEndedOutput{Ignored: true}, nil
	}

	// Check if agent info is already cleared (race condition prevention)
	// This happens when user runs "git crew stop" before the script's EXIT trap fires
	if task.Agent == "" && task.Session == "" {
		return &SessionEndedOutput{Ignored: true}, nil
	}

	// Update status based on exit code and determine whether to clear session info
	// - Normal exit (0) from in_progress: transition to in_review and keep session
	// - Normal exit (0) from other states: maintain current status and keep session
	// - Abnormal exit (non-zero): transition to error and clear session
	shouldClearSession := false
	if in.ExitCode != 0 && task.Status == domain.StatusInProgress {
		task.Status = domain.StatusError
		shouldClearSession = true
	} else if task.Status == domain.StatusInProgress {
		// Normal exit from in_progress -> in_review (keep session for review/merge)
		task.Status = domain.StatusInReview
	}
	// If already in_review, done, or closed, don't change status or session

	// Clear agent info only if abnormal exit
	if shouldClearSession {
		task.Agent = ""
		task.Session = ""
	}

	// Save task
	if err := uc.tasks.Save(task); err != nil {
		return nil, err
	}

	// Cleanup script files (ignore errors)
	uc.cleanupScriptFiles(in.TaskID)

	return &SessionEndedOutput{Ignored: false}, nil
}

// cleanupScriptFiles removes the generated script file.
func (uc *SessionEnded) cleanupScriptFiles(taskID int) {
	scriptPath := domain.ScriptPath(uc.crewDir, taskID)
	_ = os.Remove(scriptPath)
}
