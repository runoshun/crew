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
	Command    string // Command name for retry message (e.g., "complete", "merge")
	TaskID     int    // Task ID
}

// ConflictCheckOutput contains the result of conflict checking.
type ConflictCheckOutput struct {
	Message string // Conflict message to display (empty if no conflict)
}

// CheckAndHandle checks for merge conflicts and handles them if found.
// If conflicts exist:
// - Transitions task status to in_progress
// - Notifies the running session (if any)
// - Returns a ConflictCheckOutput with the message and ErrMergeConflict
//
// The caller is responsible for displaying the conflict message to stdout.
// Always returns a non-nil ConflictCheckOutput (empty on error or no conflict).
func (h *ConflictHandler) CheckAndHandle(in ConflictCheckInput) (*ConflictCheckOutput, error) {
	// Get conflicting files
	conflictFiles, err := h.git.GetMergeConflictFiles(in.Branch, in.BaseBranch)
	if err != nil {
		return &ConflictCheckOutput{}, fmt.Errorf("check merge conflict: %w", err)
	}

	if len(conflictFiles) == 0 {
		// No conflicts
		return &ConflictCheckOutput{}, nil
	}

	// Get task
	task, err := GetTask(h.tasks, in.TaskID)
	if err != nil {
		return &ConflictCheckOutput{}, err
	}

	// Build conflict message
	message := buildConflictMessage(conflictFiles, in.BaseBranch, in.Command)

	// Transition status to in_progress
	task.Status = domain.StatusInProgress
	if err := h.tasks.Save(task); err != nil {
		return &ConflictCheckOutput{}, fmt.Errorf("save task: %w", err)
	}

	// Notify session if running
	notificationMsg := fmt.Sprintf(conflictNotificationTemplate, in.Command, task.ID)
	_ = SendSessionNotification(h.sessions, task.ID, notificationMsg)

	return &ConflictCheckOutput{Message: message}, domain.ErrMergeConflict
}

// conflictNotificationTemplate is the notification message for conflict resolution.
const conflictNotificationTemplate = "Merge conflict detected. Please resolve the conflicts and run 'crew %s %d'."

// buildConflictMessage creates a user-friendly conflict message.
func buildConflictMessage(files []string, baseBranch, command string) string {
	var sb strings.Builder
	sb.WriteString("Merge conflict detected with base branch.\n\n")
	sb.WriteString("Conflicting files:\n")
	for _, f := range files {
		sb.WriteString("- ")
		sb.WriteString(f)
		sb.WriteString("\n")
	}
	sb.WriteString("\nPlease resolve the conflicts:\n")
	sb.WriteString(fmt.Sprintf("1. Run 'git merge %s' (use local branch directly - no fetch needed)\n", baseBranch))
	sb.WriteString("2. Resolve conflicts in the listed files\n")
	sb.WriteString("3. git add <files> && git commit\n")
	sb.WriteString(fmt.Sprintf("4. Run 'crew %s' again", command))
	return sb.String()
}
