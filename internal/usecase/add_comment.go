package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// AddCommentInput contains the parameters for adding a comment.
// Fields are ordered to minimize memory padding.
type AddCommentInput struct {
	Message        string // Comment text (required)
	TaskID         int    // Task ID (required)
	RequestChanges bool   // If true, change status to in_progress and notify session
}

// AddCommentOutput contains the result of adding a comment.
type AddCommentOutput struct {
	Comment        domain.Comment // The created comment
	SessionStarted bool           // True if a new session was started
}

// SessionStarter is an interface for starting task sessions.
// This allows AddComment to start sessions when needed while keeping testability.
type SessionStarter interface {
	Start(ctx context.Context, taskID int, continueFlag bool) error
}

// AddComment is the use case for adding a comment to a task.
type AddComment struct {
	tasks          domain.TaskRepository
	sessions       domain.SessionManager
	sessionStarter SessionStarter
	clock          domain.Clock
}

// NewAddComment creates a new AddComment use case.
func NewAddComment(tasks domain.TaskRepository, sessions domain.SessionManager, clock domain.Clock) *AddComment {
	return &AddComment{
		tasks:    tasks,
		sessions: sessions,
		clock:    clock,
	}
}

// WithSessionStarter sets the session starter for auto-starting sessions.
func (uc *AddComment) WithSessionStarter(starter SessionStarter) *AddComment {
	uc.sessionStarter = starter
	return uc
}

// Execute adds a comment to a task.
func (uc *AddComment) Execute(ctx context.Context, in AddCommentInput) (*AddCommentOutput, error) {
	// Validate message
	message := strings.TrimSpace(in.Message)
	if message == "" {
		return nil, domain.ErrEmptyMessage
	}

	// Verify task exists
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Create comment
	comment := domain.Comment{
		Text: message,
		Time: uc.clock.Now(),
	}

	// Save comment
	if err := uc.tasks.AddComment(in.TaskID, comment); err != nil {
		return nil, fmt.Errorf("add comment: %w", err)
	}

	var sessionStarted bool

	// If RequestChanges is true, update status to in_progress and notify session
	if in.RequestChanges {
		// Update status to in_progress
		task.Status = domain.StatusInProgress
		if err := uc.tasks.Save(task); err != nil {
			return nil, fmt.Errorf("update task status: %w", err)
		}

		// Check if session is running
		sessionName := domain.SessionName(task.ID)
		running, _ := uc.sessions.IsRunning(sessionName)
		if running {
			// Send notification to running session
			notificationMsg := fmt.Sprintf("crew show %d でコメントを確認して修正してください", task.ID)
			_ = uc.sessions.Send(sessionName, notificationMsg)
			_ = uc.sessions.Send(sessionName, "Enter")
		} else if uc.sessionStarter != nil {
			// Session not running - start it with --continue flag
			if err := uc.sessionStarter.Start(ctx, task.ID, true); err != nil {
				// Log but don't fail - comment was already saved
				// The caller can handle this as needed
				return nil, fmt.Errorf("start session: %w", err)
			}
			sessionStarted = true

			// After starting, send the notification
			sessionName := domain.SessionName(task.ID)
			notificationMsg := fmt.Sprintf("crew show %d でコメントを確認して修正してください", task.ID)
			_ = uc.sessions.Send(sessionName, notificationMsg)
			_ = uc.sessions.Send(sessionName, "Enter")
		}
	}

	return &AddCommentOutput{
		Comment:        comment,
		SessionStarted: sessionStarted,
	}, nil
}

// StartTaskAdapter adapts StartTask to implement SessionStarter interface.
type StartTaskAdapter struct {
	startTask *StartTask
}

// NewStartTaskAdapter creates a new adapter for StartTask.
func NewStartTaskAdapter(startTask *StartTask) *StartTaskAdapter {
	return &StartTaskAdapter{startTask: startTask}
}

// Start implements SessionStarter interface.
func (a *StartTaskAdapter) Start(ctx context.Context, taskID int, continueFlag bool) error {
	_, err := a.startTask.Execute(ctx, StartTaskInput{
		TaskID:   taskID,
		Continue: continueFlag,
	})
	return err
}
