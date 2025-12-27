// Package domain contains core business entities and interfaces.
package domain

import "time"

// Task represents a work unit managed by git-crew.
// Fields are ordered to minimize memory padding.
type Task struct {
	Created     time.Time // Creation time
	Started     time.Time // When status became in_progress
	ParentID    *int      // Parent task ID (nil = root task)
	Description string    // Description (optional)
	Agent       string    // Running agent name (empty if not running)
	Session     string    // tmux session name (empty if not running)
	BaseBranch  string    // Base branch for worktree creation
	Status      Status    // Current status
	Title       string    // Title (required)
	Labels      []string  // Labels
	ID          int       // Task ID (monotonic, no reuse)
	Issue       int       // GitHub issue number (0 = not linked)
	PR          int       // GitHub PR number (0 = not created)
}

// IsRoot returns true if this is a root task (no parent).
func (t *Task) IsRoot() bool {
	return t.ParentID == nil
}

// IsRunning returns true if the task has an active session.
func (t *Task) IsRunning() bool {
	return t.Session != ""
}

// Comment represents a note attached to a task.
// Fields are ordered to minimize memory padding.
type Comment struct {
	Time time.Time // Creation time
	Text string    // Comment text
}
