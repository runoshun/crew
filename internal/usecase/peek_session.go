package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// DefaultPeekLines is the default number of lines to display.
const DefaultPeekLines = 30

// PeekSessionInput contains the parameters for peeking at a session.
type PeekSessionInput struct {
	TaskID int  // Task ID to peek
	Lines  int  // Number of lines to display (0 uses default)
	Escape bool // Include ANSI escape sequences
}

// PeekSessionOutput contains the result of peeking at a session.
type PeekSessionOutput struct {
	Output string // Captured session output
}

// PeekSession is the use case for viewing session output non-interactively.
type PeekSession struct {
	tasks    domain.TaskRepository
	sessions domain.SessionManager
}

// NewPeekSession creates a new PeekSession use case.
func NewPeekSession(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
) *PeekSession {
	return &PeekSession{
		tasks:    tasks,
		sessions: sessions,
	}
}

// Execute captures and returns the last N lines from a running session.
func (uc *PeekSession) Execute(_ context.Context, in PeekSessionInput) (*PeekSessionOutput, error) {
	// Get task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Get session name
	sessionName := domain.SessionName(task.ID)

	// Check if session is running
	running, err := uc.sessions.IsRunning(sessionName)
	if err != nil {
		return nil, fmt.Errorf("check session: %w", err)
	}
	if !running {
		return nil, domain.ErrNoSession
	}

	// Determine number of lines
	lines := in.Lines
	if lines <= 0 {
		lines = DefaultPeekLines
	}

	// Peek at session
	output, err := uc.sessions.Peek(sessionName, lines, in.Escape)
	if err != nil {
		return nil, fmt.Errorf("peek session: %w", err)
	}

	return &PeekSessionOutput{
		Output: output,
	}, nil
}
