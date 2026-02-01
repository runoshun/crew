package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ACPPeekInput contains the parameters for peeking at an ACP session.
type ACPPeekInput struct {
	TaskID int  // Task ID to peek
	Lines  int  // Number of lines to display (0 uses default)
	Escape bool // Include ANSI escape sequences
}

// ACPPeekOutput contains the result of peeking at an ACP session.
type ACPPeekOutput struct {
	Output string // Captured session output
}

// ACPPeek is the use case for viewing ACP session output non-interactively.
// Fields are ordered to minimize memory padding.
type ACPPeek struct {
	tasks    domain.TaskRepository
	sessions domain.SessionManager
}

// NewACPPeek creates a new ACPPeek use case.
func NewACPPeek(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
) *ACPPeek {
	return &ACPPeek{
		tasks:    tasks,
		sessions: sessions,
	}
}

// Execute captures and returns the last N lines from a running ACP session.
func (uc *ACPPeek) Execute(_ context.Context, in ACPPeekInput) (*ACPPeekOutput, error) {
	// Get task
	if _, err := shared.GetTask(uc.tasks, in.TaskID); err != nil {
		return nil, err
	}

	sessionName := domain.ACPSessionName(in.TaskID)
	running, err := uc.sessions.IsRunning(sessionName)
	if err != nil {
		return nil, fmt.Errorf("check session: %w", err)
	}
	if !running {
		return nil, domain.ErrNoSession
	}

	lines := in.Lines
	if lines <= 0 {
		lines = DefaultPeekLines
	}

	output, err := uc.sessions.Peek(sessionName, lines, in.Escape)
	if err != nil {
		return nil, fmt.Errorf("peek session: %w", err)
	}

	return &ACPPeekOutput{Output: output}, nil
}
