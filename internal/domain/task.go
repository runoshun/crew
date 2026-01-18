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
	return t.ToMarkdownWithComments(nil)
}

// ToMarkdownWithComments converts the task to a Markdown format with frontmatter and comments.
// Comments are appended as separate blocks after the description.
func (t *Task) ToMarkdownWithComments(comments []Comment) string {
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

	// Append comments if any
	for i, comment := range comments {
		result += "\n\n---\n"
		result += "# Comment: " + intToStr(i) + "\n"
		result += "# Author: " + comment.Author + "\n"
		result += "# Time: " + comment.Time.Format(time.RFC3339) + "\n"
		result += "\n" + comment.Text
	}

	return result
}

// intToStr converts an integer to a string without importing strconv.
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
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

// ParsedComment represents a comment parsed from editor markdown format.
// Index is used to identify which original comment this corresponds to.
// Fields are ordered to minimize memory padding.
type ParsedComment struct {
	Text  string // Edited comment text
	Index int    // Original 0-based index
}

// EditorContent represents the parsed content from editor markdown format.
type EditorContent struct {
	Title       string
	Description string
	Labels      []string
	Comments    []ParsedComment
	LabelsFound bool // True if labels field was present in frontmatter
}

// ParseEditorContent parses the editor markdown format and extracts task info and comments.
// This is used when editing a task via editor to parse both task data and comments.
func ParseEditorContent(content string) (*EditorContent, error) {
	// Find frontmatter boundaries
	if len(content) < 4 || content[:4] != "---\n" {
		return nil, errors.New("invalid frontmatter: missing opening ---")
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
		return nil, errors.New("invalid frontmatter: missing closing ---")
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
				labelsStr = trimSpace(line[7:])
			}
		}
	}

	if title == "" {
		return nil, ErrEmptyTitle
	}

	// Parse labels
	var labels []string
	if labelsFound && labelsStr != "" {
		parts := splitByComma(labelsStr)
		labelSet := make(map[string]bool)
		for _, part := range parts {
			trimmed := trimSpace(part)
			if trimmed != "" {
				labelSet[trimmed] = true
			}
		}
		if len(labelSet) > 0 {
			labels = make([]string, 0, len(labelSet))
			for label := range labelSet {
				labels = append(labels, label)
			}
			slices.Sort(labels)
		}
	}

	// Get body after frontmatter (description + comments)
	bodyStartIdx := 4 // Skip "---\n"
	for i := 0; i <= endIdx; i++ {
		if i < len(lines) {
			bodyStartIdx += len(lines[i]) + 1
		}
	}

	body := ""
	if bodyStartIdx < len(content) {
		body = trimLeadingNewlines(content[bodyStartIdx:])
	}

	// Split body into description and comment blocks
	// Comment blocks start with "\n\n---\n# Comment:" or "---\n# Comment:" if description is empty
	description := body
	var comments []ParsedComment

	// Find first comment block
	commentSeparator := "\n\n---\n# Comment:"
	sepIdx := indexOf(body, commentSeparator)
	if sepIdx >= 0 {
		description = body[:sepIdx]
		// Parse comment blocks
		commentSection := body[sepIdx+2:] // Skip "\n\n", keep "---\n# Comment:..."
		var err error
		comments, err = parseCommentBlocks(commentSection)
		if err != nil {
			return nil, err
		}
	} else {
		// Check if comment section starts immediately (empty description)
		directSeparator := "---\n# Comment:"
		if len(body) >= len(directSeparator) && body[:len(directSeparator)] == directSeparator {
			description = ""
			var err error
			comments, err = parseCommentBlocks(body)
			if err != nil {
				return nil, err
			}
		}
	}

	return &EditorContent{
		Title:       title,
		Description: description,
		Labels:      labels,
		Comments:    comments,
		LabelsFound: labelsFound,
	}, nil
}

// parseCommentBlocks parses the comment section of editor content.
// Each block starts with "---\n# Comment: <index>\n..."
// Returns parsed comments and any validation error.
func parseCommentBlocks(section string) ([]ParsedComment, error) {
	var comments []ParsedComment

	// Split by comment block separator
	blocks := splitByCommentSeparator(section)

	for _, block := range blocks {
		block = trimSpace(block)
		if block == "" {
			continue
		}

		// Parse comment block
		comment, err := parseCommentBlock(block)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}

	return comments, nil
}

// splitByCommentSeparator splits the section by "---\n# Comment:" separator.
func splitByCommentSeparator(s string) []string {
	var blocks []string
	separator := "---\n# Comment:"
	start := 0
	for {
		idx := indexOf(s[start:], separator)
		if idx < 0 {
			if start < len(s) {
				blocks = append(blocks, s[start:])
			}
			break
		}
		if idx > 0 {
			blocks = append(blocks, s[start:start+idx])
		}
		start = start + idx
		// Find next separator or end
		nextIdx := indexOf(s[start+len(separator):], separator)
		if nextIdx < 0 {
			blocks = append(blocks, s[start:])
			break
		}
		blocks = append(blocks, s[start:start+len(separator)+nextIdx])
		start = start + len(separator) + nextIdx
	}
	return blocks
}

// parseCommentBlock parses a single comment block.
// Expected format:
// ---
// # Comment: <index>
// # Author: <author>
// # Time: <time>
//
// <text>
func parseCommentBlock(block string) (ParsedComment, error) {
	lines := splitLines(block)
	if len(lines) < 4 {
		return ParsedComment{}, ErrInvalidCommentMeta
	}

	// Parse header
	if lines[0] != "---" {
		return ParsedComment{}, ErrInvalidCommentMeta
	}

	// Parse "# Comment: <index>"
	if len(lines[1]) < 12 || lines[1][:11] != "# Comment: " {
		return ParsedComment{}, ErrInvalidCommentMeta
	}
	indexStr := lines[1][11:]
	index, err := strToInt(indexStr)
	if err != nil {
		return ParsedComment{}, ErrInvalidCommentMeta
	}

	// Parse "# Author: <author>" (can be empty after colon)
	if len(lines[2]) < 9 || lines[2][:9] != "# Author:" {
		return ParsedComment{}, ErrInvalidCommentMeta
	}

	// Parse "# Time: <time>" and validate RFC3339 format
	if len(lines[3]) < 8 || lines[3][:7] != "# Time:" {
		return ParsedComment{}, ErrInvalidCommentMeta
	}
	timeStr := trimSpace(lines[3][7:])
	if _, err := time.Parse(time.RFC3339, timeStr); err != nil {
		return ParsedComment{}, ErrInvalidCommentMeta
	}

	// Find text (after empty line following meta)
	textStartIdx := 4
	// Skip any empty lines
	for textStartIdx < len(lines) && lines[textStartIdx] == "" {
		textStartIdx++
	}

	// Build text from remaining lines
	var textLines []string
	for i := textStartIdx; i < len(lines); i++ {
		textLines = append(textLines, lines[i])
	}
	text := joinLines(textLines)

	// Validate text is not empty
	if trimSpace(text) == "" {
		return ParsedComment{}, ErrCommentTextEmpty
	}

	return ParsedComment{
		Index: index,
		Text:  text,
	}, nil
}

// indexOf returns the index of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// joinLines joins lines with newline separator.
func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\n" + lines[i]
	}
	return result
}

// strToInt parses a string to an integer.
func strToInt(s string) (int, error) {
	s = trimSpace(s)
	if s == "" {
		return 0, errors.New("empty string")
	}
	negative := false
	if s[0] == '-' {
		negative = true
		s = s[1:]
	}
	if len(s) == 0 {
		return 0, errors.New("invalid number")
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, errors.New("invalid character")
		}
		n = n*10 + int(s[i]-'0')
	}
	if negative {
		n = -n
	}
	return n, nil
}
