// Package domain contains core business entities and interfaces.
package domain

import (
	"errors"
	"slices"
	"time"
)

// CloseReason specifies why a task was closed.
type CloseReason string

const (
	CloseReasonNone      CloseReason = ""          // Not closed yet
	CloseReasonMerged    CloseReason = "merged"    // Task was merged
	CloseReasonAbandoned CloseReason = "abandoned" // Task was abandoned without merge
)

// Task represents a work unit managed by git-crew.
// Fields are ordered to minimize memory padding.
type Task struct {
	Created     time.Time   `json:"created"`               // Creation time
	Started     time.Time   `json:"started,omitempty"`     // When status became in_progress
	ParentID    *int        `json:"parentID"`              // Parent task ID (nil = root task)
	SkipReview  *bool       `json:"skipReview,omitempty"`  // Skip review on completion (nil=use config, true=skip, false=require review)
	Description string      `json:"description,omitempty"` // Description (optional)
	Agent       string      `json:"agent,omitempty"`       // Running agent name (empty if not running)
	Session     string      `json:"session,omitempty"`     // tmux session name (empty if not running)
	BaseBranch  string      `json:"baseBranch"`            // Base branch for worktree creation
	Status      Status      `json:"status"`                // Current status
	CloseReason CloseReason `json:"closeReason,omitempty"` // Why the task was closed
	Title       string      `json:"title"`                 // Title (required)
	Labels      []string    `json:"labels,omitempty"`      // Labels
	ID          int         `json:"-"`                     // Task ID (stored as map key, not in value)
	Issue       int         `json:"issue,omitempty"`       // GitHub issue number (0 = not linked)
	PR          int         `json:"pr,omitempty"`          // GitHub PR number (0 = not created)
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
// Title, labels, and description are included for editing purposes.
func (t *Task) ToMarkdown() string {
	result := "---\ntitle: " + t.Title + "\n"

	// Add labels field (comma-separated, empty if no labels)
	if len(t.Labels) > 0 {
		result += "labels: "
		for i, label := range t.Labels {
			if i > 0 {
				result += ", "
			}
			result += label
		}
		result += "\n"
	} else {
		result += "labels:\n"
	}

	result += "---\n\n" + t.Description
	return result
}

// FromMarkdown parses Markdown with frontmatter and updates the task's title, labels, and description.
// Returns an error if parsing fails.
func (t *Task) FromMarkdown(content string) error {
	// Simple frontmatter parser
	// Expected format:
	// ---
	// title: <title>
	// labels: <label1>, <label2>, ...
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
	labelsStr := ""
	labelsFound := false
	for i := 0; i < endIdx; i++ {
		line := lines[i]
		if len(line) > 7 && line[:7] == "title: " {
			title = line[7:]
		} else if len(line) >= 7 && line[:7] == "labels:" {
			labelsFound = true
			if len(line) > 7 {
				// Skip "labels:" (7 chars) and trim leading whitespace
				labelsStr = trimSpace(line[7:])
			}
		}
	}

	if title == "" {
		return ErrEmptyTitle
	}

	// Parse labels (comma-separated, trim whitespace, deduplicate, sort)
	var labels []string
	if labelsFound {
		if labelsStr != "" {
			parts := splitByComma(labelsStr)
			// Use a map to deduplicate
			labelSet := make(map[string]bool)
			for _, part := range parts {
				trimmed := trimSpace(part)
				if trimmed != "" {
					labelSet[trimmed] = true
				}
			}
			// Convert map to slice
			if len(labelSet) > 0 {
				labels = make([]string, 0, len(labelSet))
				for label := range labelSet {
					labels = append(labels, label)
				}
				slices.Sort(labels)
			}
		}
		// If labelsFound but labelsStr is empty or all whitespace, labels = nil (cleared)
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
	if labelsFound {
		t.Labels = labels
	}
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

// splitByComma splits a string by commas.
func splitByComma(s string) []string {
	if s == "" {
		return nil
	}
	parts := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

// trimSpace removes leading and trailing whitespace.
func trimSpace(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Trim trailing whitespace
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

// Comment represents a note attached to a task.
// Fields are ordered to minimize memory padding.
type Comment struct {
	Time   time.Time `json:"time"`             // Creation time
	Text   string    `json:"text"`             // Comment text
	Author string    `json:"author,omitempty"` // "manager", "worker", or empty
}
