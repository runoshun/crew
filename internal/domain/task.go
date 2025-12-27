// Package domain contains core business entities and interfaces.
package domain

import "time"

// Task represents a work unit managed by git-crew.
// Fields are ordered to minimize memory padding.
type Task struct {
	Created     time.Time `json:"created"`               // Creation time
	Started     time.Time `json:"started,omitempty"`     // When status became in_progress
	ParentID    *int      `json:"parentID"`              // Parent task ID (nil = root task)
	Description string    `json:"description,omitempty"` // Description (optional)
	Agent       string    `json:"agent,omitempty"`       // Running agent name (empty if not running)
	Session     string    `json:"session,omitempty"`     // tmux session name (empty if not running)
	BaseBranch  string    `json:"baseBranch"`            // Base branch for worktree creation
	Status      Status    `json:"status"`                // Current status
	Title       string    `json:"title"`                 // Title (required)
	Labels      []string  `json:"labels,omitempty"`      // Labels
	ID          int       `json:"-"`                     // Task ID (stored as map key, not in value)
	Issue       int       `json:"issue,omitempty"`       // GitHub issue number (0 = not linked)
	PR          int       `json:"pr,omitempty"`          // GitHub PR number (0 = not created)
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
	Time time.Time `json:"time"` // Creation time
	Text string    `json:"text"` // Comment text
}
