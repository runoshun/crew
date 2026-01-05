// Package domain contains core business entities and interfaces.
package domain

import (
	"errors"
	"time"
)

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

// ToMarkdown converts the task to a Markdown format with frontmatter.
// Only title and description are included for editing purposes.
func (t *Task) ToMarkdown() string {
	return "---\ntitle: " + t.Title + "\n---\n\n" + t.Description
}

// FromMarkdown parses Markdown with frontmatter and updates the task's title and description.
// Returns an error if parsing fails.
func (t *Task) FromMarkdown(content string) error {
	// Simple frontmatter parser
	// Expected format:
	// ---
	// title: <title>
	// ---
	//
	// <description>

	// Find frontmatter boundaries
	if len(content) < 4 || content[:4] != "---\n" {
		return errors.New("invalid frontmatter: missing opening ---")
	}

	// Find closing ---
	endIdx := -1
	lines := splitLines(content[4:])
	for i, line := range lines {
		if line == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return errors.New("invalid frontmatter: missing closing ---")
	}

	// Parse frontmatter
	title := ""
	for i := 0; i < endIdx; i++ {
		line := lines[i]
		if len(line) > 7 && line[:7] == "title: " {
			title = line[7:]
		}
	}

	if title == "" {
		return ErrEmptyTitle
	}

	// Get description (everything after frontmatter)
	descStartIdx := 4 // Skip "---\n"
	for i := 0; i <= endIdx; i++ {
		if i < len(lines) {
			descStartIdx += len(lines[i]) + 1 // +1 for newline
		}
	}

	description := ""
	if descStartIdx < len(content) {
		description = trimLeadingNewlines(content[descStartIdx:])
	}

	// Update task fields
	t.Title = title
	t.Description = description

	return nil
}

// splitLines splits content by newlines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// trimLeadingNewlines removes leading newline characters.
func trimLeadingNewlines(s string) string {
	start := 0
	for start < len(s) && s[start] == '\n' {
		start++
	}
	return s[start:]
}

// Comment represents a note attached to a task.
// Fields are ordered to minimize memory padding.
type Comment struct {
	Time time.Time `json:"time"` // Creation time
	Text string    `json:"text"` // Comment text
}
