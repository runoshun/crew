// Package domain contains core business entities and interfaces.
package domain

import (
	"fmt"
)

// TaskDraft represents a task to be created from file input.
// Parent can be either a relative index (1-based, within the same file)
// or an absolute task ID.
// Fields are ordered to minimize memory padding.
type TaskDraft struct {
	Title       string
	Description string
	ParentRef   string
	Labels      []string
}

// ParseTaskDrafts parses a markdown file containing one or more task definitions.
// Tasks are separated by frontmatter blocks starting with "---".
//
// Format:
//
//	---
//	title: Task Title
//	labels: [label1, label2]
//	parent: 1
//	---
//	Task description here.
//
//	---
//	title: Second Task
//	parent: 1
//	---
//	Second task description.
//
// Parent references:
//   - Relative: "parent: 1" refers to the 1st task in this file
//   - Absolute: "parent: #123" refers to existing task ID 123 (use # prefix)
func ParseTaskDrafts(content string) ([]TaskDraft, error) {
	if content == "" {
		return nil, ErrEmptyFile
	}

	// Split content by task blocks
	blocks := splitTaskBlocks(content)
	if len(blocks) == 0 {
		return nil, ErrNoTasksInFile
	}

	drafts := make([]TaskDraft, 0, len(blocks))
	for i, block := range blocks {
		draft, err := parseTaskBlock(block)
		if err != nil {
			return nil, fmt.Errorf("task %d: %w", i+1, err)
		}
		drafts = append(drafts, draft)
	}

	return drafts, nil
}

// splitTaskBlocks splits content into separate task blocks.
// Each block starts with "---" on a new line.
func splitTaskBlocks(content string) []string {
	var blocks []string
	lines := splitLines(content)

	blockStart := -1
	var currentBlock []string

	for i, line := range lines {
		if line == "---" {
			if blockStart == -1 {
				// Start of first block
				blockStart = i
				currentBlock = []string{}
			} else if len(currentBlock) == 0 {
				// This is the closing "---" of frontmatter
				currentBlock = append(currentBlock, line)
			} else {
				// Check if this is a new block start or just a separator within content
				// If we have content and see "---", check if next line looks like frontmatter
				if i+1 < len(lines) && isFrontmatterKey(lines[i+1]) {
					// End current block and start new one
					if len(currentBlock) > 0 {
						blocks = append(blocks, joinLines(currentBlock))
					}
					blockStart = i
					currentBlock = []string{}
				} else {
					// Just a separator within description
					currentBlock = append(currentBlock, line)
				}
			}
		} else if blockStart != -1 {
			currentBlock = append(currentBlock, line)
		}
	}

	// Add last block
	if len(currentBlock) > 0 {
		blocks = append(blocks, joinLines(currentBlock))
	}

	return blocks
}

// isFrontmatterKey checks if a line looks like a frontmatter key.
func isFrontmatterKey(line string) bool {
	// Common frontmatter keys
	keys := []string{"title:", "labels:", "parent:"}
	for _, key := range keys {
		if len(line) >= len(key) && line[:len(key)] == key {
			return true
		}
	}
	return false
}

// parseTaskBlock parses a single task block.
// Expected format:
// title: Task Title
// labels: [label1, label2]
// parent: 1
// ---
// Description here
func parseTaskBlock(block string) (TaskDraft, error) {
	lines := splitLines(block)
	if len(lines) == 0 {
		return TaskDraft{}, ErrEmptyTitle
	}

	// Parse frontmatter lines until we hit "---"
	title := ""
	labelsStr := ""
	parentRef := ""
	frontmatterEnd := -1

	for i, line := range lines {
		if line == "---" {
			frontmatterEnd = i
			break
		}

		if len(line) > 7 && line[:7] == "title: " {
			title = trimSpace(line[7:])
		} else if len(line) >= 7 && line[:7] == "labels:" {
			if len(line) > 7 {
				labelsStr = trimSpace(line[7:])
			}
		} else if len(line) >= 7 && line[:7] == "parent:" {
			if len(line) > 7 {
				parentRef = trimSpace(line[7:])
			}
		}
	}

	if title == "" {
		return TaskDraft{}, ErrEmptyTitle
	}

	// Parse labels
	var labels []string
	if labelsStr != "" {
		labels = parseLabelsValue(labelsStr)
	}

	// Get description (everything after frontmatter closing "---")
	description := ""
	if frontmatterEnd >= 0 && frontmatterEnd+1 < len(lines) {
		descLines := lines[frontmatterEnd+1:]
		description = trimLeadingNewlines(joinLines(descLines))
	}

	return TaskDraft{
		Title:       title,
		Description: description,
		Labels:      labels,
		ParentRef:   parentRef,
	}, nil
}

// parseLabelsValue parses labels from frontmatter value.
// Supports formats:
// - [label1, label2] (YAML array style)
// - label1, label2 (comma-separated)
func parseLabelsValue(value string) []string {
	value = trimSpace(value)
	if value == "" {
		return nil
	}

	// Remove surrounding brackets if present
	if len(value) >= 2 && value[0] == '[' && value[len(value)-1] == ']' {
		value = value[1 : len(value)-1]
	}

	// Split by comma
	parts := splitByComma(value)
	var labels []string
	seen := make(map[string]bool)
	for _, part := range parts {
		label := trimSpace(part)
		if label != "" && !seen[label] {
			labels = append(labels, label)
			seen[label] = true
		}
	}

	return labels
}

// ResolveParentRef resolves a parent reference to an actual task ID.
// ref can be:
// - A relative index (1-based) referring to a task in the same file: "1", "2", etc.
// - An absolute task ID with # prefix: "#123", "#1", etc.
//
// The # prefix explicitly marks a reference as absolute, avoiding ambiguity
// when the existing task ID falls within the range of relative indices.
//
// createdIDs maps relative index (1-based) to created task ID.
func ResolveParentRef(ref string, createdIDs map[int]int) (*int, error) {
	if ref == "" {
		return nil, nil
	}

	// Check for absolute reference with # prefix
	if len(ref) > 1 && ref[0] == '#' {
		n, err := strToInt(ref[1:])
		if err != nil {
			return nil, ErrInvalidParentRef
		}
		if n <= 0 {
			return nil, ErrInvalidParentRef
		}
		// Return as absolute ID (skip relative lookup)
		return &n, nil
	}

	// Parse as relative reference
	n, err := strToInt(ref)
	if err != nil {
		return nil, ErrInvalidParentRef
	}

	if n <= 0 {
		return nil, ErrInvalidParentRef
	}

	// Check if it's a relative reference (within createdIDs range)
	if id, ok := createdIDs[n]; ok {
		return &id, nil
	}

	// Not in createdIDs - treat as absolute ID for backwards compatibility
	// (allows referencing existing tasks with IDs larger than file task count)
	return &n, nil
}
