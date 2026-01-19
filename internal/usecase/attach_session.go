package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// AttachSessionInput contains the parameters for attaching to a session.
type AttachSessionInput struct {
	TaskID int  // Task ID to attach to
	Review bool // If true, attach to review session instead of work session
}

// AttachSessionOutput contains the result of attaching to a session.
// This use case replaces the current process, so the output is not used.
type AttachSessionOutput struct{}

// AttachSession is the use case for attaching to a running session.
type AttachSession struct {
	tasks    domain.TaskRepository
	sessions domain.SessionManager
}

// NewAttachSession creates a new AttachSession use case.
func NewAttachSession(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
) *AttachSession {
	return &AttachSession{
		tasks:    tasks,
		sessions: sessions,
	}
}

// Execute attaches to a running session for the given task.
// This replaces the current process and does not return on success.
func (uc *AttachSession) Execute(_ context.Context, in AttachSessionInput) (*AttachSessionOutput, error) {
	// Get task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Get session name based on mode
	var sessionName string
	if in.Review {
		sessionName = domain.ReviewSessionName(task.ID)
	} else {
		sessionName = domain.SessionName(task.ID)
	}

	// Check if session is running
	running, err := uc.sessions.IsRunning(sessionName)
	if err != nil {
		return nil, fmt.Errorf("check session: %w", err)
	}
	if !running {
		return nil, fmt.Errorf("%w: session stopped or missing. Recover: crew start %d --continue, crew peek %d, or crew exec %d -- <cmd>", domain.ErrNoSession, task.ID, task.ID, task.ID)
	}

	// Attach to session (this replaces the current process)
	if err := uc.sessions.Attach(sessionName); err != nil {
		return nil, fmt.Errorf("attach session: %w", err)
	}

	// This line should never be reached
	return &AttachSessionOutput{}, nil
}
