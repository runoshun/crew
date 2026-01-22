// Package shared provides shared utilities for use cases.
package shared

import (
	"fmt"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ConflictHandler handles merge conflict detection and notification.
type ConflictHandler struct {
	tasks    domain.TaskRepository
	sessions domain.SessionManager
	git      domain.Git
	clock    domain.Clock
}

// NewConflictHandler creates a new ConflictHandler.
func NewConflictHandler(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	git domain.Git,
	clock domain.Clock,
) *ConflictHandler {
	return &ConflictHandler{
		tasks:    tasks,
		sessions: sessions,
		git:      git,
		clock:    clock,
	}
}

// ConflictCheckInput contains the parameters for conflict checking.
// Fields are ordered to minimize memory padding.
type ConflictCheckInput struct {
	Branch     string // Task branch to merge
	BaseBranch string // Target branch to merge into
	TaskID     int    // Task ID
}

// CheckAndHandle checks for merge conflicts and handles them if found.
// If conflicts exist:
// - Transitions task status to in_progress
// - Adds a comment with conflict file list
// - Notifies the running session (if any)
// - Returns ErrMergeConflict
//
// If no conflicts, returns nil.
func (h *ConflictHandler) CheckAndHandle(in ConflictCheckInput) error {
	// Get conflicting files
	conflictFiles, err := h.git.GetMergeConflictFiles(in.Branch, in.BaseBranch)
	if err != nil {
		return fmt.Errorf("check merge conflict: %w", err)
	}

	if len(conflictFiles) == 0 {
		// No conflicts
		return nil
	}

	// Get task
	task, err := GetTask(h.tasks, in.TaskID)
	if err != nil {
		return err
	}

	// Build conflict message
	message := buildConflictMessage(conflictFiles, in.BaseBranch)

	// Create comment
	comment := domain.Comment{
		Text: message,
		Time: h.clock.Now(),
	}

	// Transition status to in_progress and add comment atomically
	task.Status = domain.StatusInProgress
	comments, err := h.tasks.GetComments(task.ID)
	if err != nil {
		return fmt.Errorf("get comments: %w", err)
	}
	comments = append(comments, comment)

	if err := h.tasks.SaveTaskWithComments(task, comments); err != nil {
		return fmt.Errorf("save task with comments: %w", err)
	}

	// Notify session if running
	notificationMsg := fmt.Sprintf(conflictNotificationTemplate, task.ID, task.ID)
	_ = SendSessionNotification(h.sessions, task.ID, notificationMsg)

	return domain.ErrMergeConflict
}

// conflictNotificationTemplate is the notification message for conflict resolution.
const conflictNotificationTemplate = "Merge conflict detected. Please check the comment with 'crew show %d' and resolve the conflicts. When finished, run 'crew complete %d'."

// buildConflictMessage creates a user-friendly conflict message.
func buildConflictMessage(files []string, baseBranch string) string {
	var sb strings.Builder
	sb.WriteString("Merge conflict detected with base branch.\n\n")
	sb.WriteString("Conflicting files:\n")
	for _, f := range files {
		sb.WriteString("- ")
		sb.WriteString(f)
		sb.WriteString("\n")
	}
	sb.WriteString("\nPlease resolve the conflicts:\n")
	sb.WriteString(fmt.Sprintf("1. git fetch origin %s:%s\n", baseBranch, baseBranch))
	sb.WriteString(fmt.Sprintf("2. git merge %s\n", baseBranch))
	sb.WriteString("3. Resolve conflicts in the listed files\n")
	sb.WriteString("4. git add <files> && git commit\n")
	sb.WriteString("5. Run 'crew complete' again")
	return sb.String()
}
