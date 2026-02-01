package shared

import (
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// CheckSessionRunning checks if a session is running for the given task ID.
// It centralizes the common pattern of:
//
//	sessionName := domain.SessionName(task.ID)
//	running, err := sessions.IsRunning(sessionName)
//	if err != nil { return nil, fmt.Errorf("check session running: %w", err) }
func CheckSessionRunning(sessions domain.SessionManager, taskID int) (bool, error) {
	sessionName := domain.SessionName(taskID)
	running, err := sessions.IsRunning(sessionName)
	if err != nil {
		return false, fmt.Errorf("check session running: %w", err)
	}
	return running, nil
}

// StopSession stops a session for the given task ID if it is running.
// Returns the stopped session name (empty if not running).
func StopSession(sessions domain.SessionManager, taskID int) (string, error) {
	sessionName := domain.SessionName(taskID)
	running, err := sessions.IsRunning(sessionName)
	if err != nil {
		return "", fmt.Errorf("check session running: %w", err)
	}
	if !running {
		return "", nil
	}

	if stopErr := sessions.Stop(sessionName); stopErr != nil {
		return "", fmt.Errorf("stop session: %w", stopErr)
	}
	return sessionName, nil
}

// SendSessionNotification sends a notification message to a running session.
// If the session is not running, it does nothing (no error).
// The message is followed by an Enter key to submit it.
func SendSessionNotification(sessions domain.SessionManager, taskID int, message string) error {
	sessionName := domain.SessionName(taskID)
	running, _ := sessions.IsRunning(sessionName)
	if !running {
		return nil
	}

	if err := sessions.Send(sessionName, message); err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	if err := sessions.Send(sessionName, "Enter"); err != nil {
		return fmt.Errorf("send enter: %w", err)
	}
	return nil
}

// RequireRunningSession checks if a session is running and returns ErrNoSession if not.
// This is used by usecases that require an active session (send_keys, peek_session, attach_session).
func RequireRunningSession(sessions domain.SessionManager, taskID int) (string, error) {
	sessionName := domain.SessionName(taskID)
	running, err := sessions.IsRunning(sessionName)
	if err != nil {
		return "", fmt.Errorf("check session: %w", err)
	}
	if !running {
		return "", domain.ErrNoSession
	}
	return sessionName, nil
}

// EnsureNoRunningSession checks that no session is running for the given task.
// Returns ErrSessionRunning if session is already running.
// This is used by usecases that need to start a new session (start_task).
func EnsureNoRunningSession(sessions domain.SessionManager, taskID int) error {
	sessionName := domain.SessionName(taskID)
	running, err := sessions.IsRunning(sessionName)
	if err != nil {
		return fmt.Errorf("check session: %w", err)
	}
	if running {
		return fmt.Errorf("task #%d session is already running: %w", taskID, domain.ErrSessionRunning)
	}
	return nil
}
