package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// SendKeysInput contains the parameters for sending keys to a session.
// Fields are ordered to minimize memory padding.
type SendKeysInput struct {
	Keys   string // Keys to send (Tab, Escape, Enter, or any text)
	TaskID int    // Task ID
}

// SendKeysOutput contains the result of sending keys.
type SendKeysOutput struct{}

// SendKeys is the use case for sending keys to a running session.
type SendKeys struct {
	tasks    domain.TaskRepository
	sessions domain.SessionManager
}

// NewSendKeys creates a new SendKeys use case.
func NewSendKeys(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
) *SendKeys {
	return &SendKeys{
		tasks:    tasks,
		sessions: sessions,
	}
}

// Execute sends keys to a running session for the given task.
func (uc *SendKeys) Execute(_ context.Context, in SendKeysInput) (*SendKeysOutput, error) {
	// Get task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	// Check if session is running and get session name
	sessionName, err := shared.RequireRunningSession(uc.sessions, task.ID)
	if err != nil {
		return nil, err
	}

	// Send keys to session
	if err := uc.sessions.Send(sessionName, in.Keys); err != nil {
		return nil, fmt.Errorf("send keys: %w", err)
	}

	return &SendKeysOutput{}, nil
}
