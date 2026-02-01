package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ACPAttachInput contains the parameters for attaching to an ACP session.
type ACPAttachInput struct {
	TaskID int // Task ID to attach to
}

// ACPAttachOutput contains the result of attaching to an ACP session.
// This use case replaces the current process, so the output is not used.
type ACPAttachOutput struct{}

// ACPAttach is the use case for attaching to a running ACP session.
// Fields are ordered to minimize memory padding.
type ACPAttach struct {
	tasks    domain.TaskRepository
	sessions domain.SessionManager
}

// NewACPAttach creates a new ACPAttach use case.
func NewACPAttach(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
) *ACPAttach {
	return &ACPAttach{
		tasks:    tasks,
		sessions: sessions,
	}
}

// Execute attaches to a running ACP session for the given task.
// This replaces the current process and does not return on success.
func (uc *ACPAttach) Execute(_ context.Context, in ACPAttachInput) (*ACPAttachOutput, error) {
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
		return nil, fmt.Errorf("%w: acp session stopped or missing. Recover: crew acp start %d, crew acp peek %d", domain.ErrNoSession, in.TaskID, in.TaskID)
	}

	if err := uc.sessions.Attach(sessionName); err != nil {
		return nil, fmt.Errorf("attach session: %w", err)
	}

	return &ACPAttachOutput{}, nil
}
