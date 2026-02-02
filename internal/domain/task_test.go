package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_ToMarkdown(t *testing.T) {
	tests := []struct {
		name string
		task *Task
		want string
	}{
		{
			name: "with title and description",
			task: &Task{
				Title:       "Test Task",
				Description: "This is a test description",
			},
			want: "---\ntitle: Test Task\nparent:\nlabels:\nskip_review:\n---\n\nThis is a test description",
		},
		{
			name: "with title only",
			task: &Task{
				Title:       "Title Only",
				Description: "",
			},
			want: "---\ntitle: Title Only\nparent:\nlabels:\nskip_review:\n---\n\n",
		},
		{
			name: "with multiline description",
			task: &Task{
				Title:       "Multi-line",
				Description: "Line 1\nLine 2\nLine 3",
			},
			want: "---\ntitle: Multi-line\nparent:\nlabels:\nskip_review:\n---\n\nLine 1\nLine 2\nLine 3",
		},
		{
			name: "with single label",
			task: &Task{
				Title:       "Task with label",
				Description: "Description",
				Labels:      []string{"bug"},
			},
			want: "---\ntitle: Task with label\nparent:\nlabels: bug\nskip_review:\n---\n\nDescription",
		},
		{
			name: "with multiple labels",
			task: &Task{
				Title:       "Task with labels",
				Description: "Description",
				Labels:      []string{"bug", "urgent", "frontend"},
			},
			want: "---\ntitle: Task with labels\nparent:\nlabels: bug, urgent, frontend\nskip_review:\n---\n\nDescription",
		},
		{
			name: "with labels and no description",
			task: &Task{
				Title:  "Labels only",
				Labels: []string{"feature"},
			},
			want: "---\ntitle: Labels only\nparent:\nlabels: feature\nskip_review:\n---\n\n",
		},
		{
			name: "with parent",
			task: &Task{
				Title:    "Sub task",
				ParentID: intPtr(5),
			},
			want: "---\ntitle: Sub task\nparent: 5\nlabels:\nskip_review:\n---\n\n",
		},
		{
			name: "with skip_review true",
			task: &Task{
				Title:      "Skip review",
				SkipReview: boolPtr(true),
			},
			want: "---\ntitle: Skip review\nparent:\nlabels:\nskip_review: true\n---\n\n",
		},
		{
			name: "with skip_review false",
			task: &Task{
				Title:      "Require review",
				SkipReview: boolPtr(false),
			},
			want: "---\ntitle: Require review\nparent:\nlabels:\nskip_review: false\n---\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.task.ToMarkdown()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTask_FromMarkdown(t *testing.T) {
	tests := []struct {
		name        string
		markdown    string
		wantTitle   string
		wantDesc    string
		errContains string
		wantLabels  []string
		wantErr     bool
	}{
		{
			name: "valid markdown with title and description",
			markdown: `---
title: Test Title
---

Test description`,
			wantTitle: "Test Title",
			wantDesc:  "Test description",
			wantErr:   false,
		},
		{
			name: "valid markdown with title only",
			markdown: `---
title: Title Only
---

`,
			wantTitle: "Title Only",
			wantDesc:  "",
			wantErr:   false,
		},
		{
			name: "valid markdown with multiline description",
			markdown: `---
title: Multi-line Task
---

Line 1
Line 2
Line 3`,
			wantTitle: "Multi-line Task",
			wantDesc:  "Line 1\nLine 2\nLine 3",
			wantErr:   false,
		},
		{
			name: "valid markdown with extra newlines",
			markdown: `---
title: Test
---


Description after empty lines`,
			wantTitle: "Test",
			wantDesc:  "Description after empty lines",
			wantErr:   false,
		},
		{
			name: "with single label",
			markdown: `---
title: Task with label
labels: bug
---

Description`,
			wantTitle:  "Task with label",
			wantDesc:   "Description",
			wantLabels: []string{"bug"},
			wantErr:    false,
		},
		{
			name: "with multiple labels",
			markdown: `---
title: Task with labels
labels: bug, urgent, frontend
---

Description`,
			wantTitle:  "Task with labels",
			wantDesc:   "Description",
			wantLabels: []string{"bug", "frontend", "urgent"},
			wantErr:    false,
		},
		{
			name: "with empty labels field",
			markdown: `---
title: No labels
labels:
---

Description`,
			wantTitle:  "No labels",
			wantDesc:   "Description",
			wantLabels: nil,
			wantErr:    false,
		},
		{
			name: "with labels with extra whitespace",
			markdown: `---
title: Whitespace test
labels:  bug ,  urgent  , frontend
---

Description`,
			wantTitle:  "Whitespace test",
			wantDesc:   "Description",
			wantLabels: []string{"bug", "frontend", "urgent"},
			wantErr:    false,
		},
		{
			name: "with labels without space after colon",
			markdown: `---
title: No space after colon
labels:bug
---

Description`,
			wantTitle:  "No space after colon",
			wantDesc:   "Description",
			wantLabels: []string{"bug"},
			wantErr:    false,
		},
		{
			name: "with duplicate labels",
			markdown: `---
title: Duplicate labels
labels: bug, urgent, bug, frontend, urgent
---

Description`,
			wantTitle:  "Duplicate labels",
			wantDesc:   "Description",
			wantLabels: []string{"bug", "frontend", "urgent"},
			wantErr:    false,
		},
		{
			name: "without labels field (backward compatibility)",
			markdown: `---
title: Old format
---

Description`,
			wantTitle:  "Old format",
			wantDesc:   "Description",
			wantLabels: nil, // Labels should not be modified if field is not present
			wantErr:    false,
		},
		{
			name: "missing opening frontmatter delimiter",
			markdown: `title: No delimiter
---

Description`,
			wantErr:     true,
			errContains: "missing opening ---",
		},
		{
			name: "missing closing frontmatter delimiter",
			markdown: `---
title: No closing delimiter

Description`,
			wantErr:     true,
			errContains: "missing closing ---",
		},
		{
			name: "empty title",
			markdown: `---
---

Description`,
			wantErr: true,
		},
		{
			name: "title field missing",
			markdown: `---
other: field
---

Description`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{}
			err := task.FromMarkdown(tt.markdown)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTitle, task.Title)
				assert.Equal(t, tt.wantDesc, task.Description)
				if tt.wantLabels != nil {
					assert.Equal(t, tt.wantLabels, task.Labels)
				}
			}
		})
	}
}

func TestTask_RoundTrip(t *testing.T) {
	tests := []struct {
		original *Task
		name     string
	}{
		{
			name: "without labels",
			original: &Task{
				Title:       "Round Trip Test",
				Description: "This should survive the round trip",
			},
		},
		{
			name: "with labels",
			original: &Task{
				Title:       "Round Trip with Labels",
				Description: "This should survive with labels",
				Labels:      []string{"bug", "urgent"},
			},
		},
		{
			name: "with empty labels",
			original: &Task{
				Title:       "Round Trip no labels",
				Description: "No labels set",
				Labels:      nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that ToMarkdown -> FromMarkdown preserves data
			markdown := tt.original.ToMarkdown()

			parsed := &Task{}
			err := parsed.FromMarkdown(markdown)

			require.NoError(t, err)
			assert.Equal(t, tt.original.Title, parsed.Title)
			assert.Equal(t, tt.original.Description, parsed.Description)
			assert.Equal(t, tt.original.Labels, parsed.Labels)
		})
	}
}

func TestTask_ToMarkdownWithComments(t *testing.T) {
	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	tests := []struct {
		task     *Task
		name     string
		want     string
		comments []Comment
	}{
		{
			name: "no comments",
			task: &Task{
				Title:       "Test Task",
				Description: "Description",
			},
			comments: nil,
			want:     "---\ntitle: Test Task\nparent:\nlabels:\nskip_review:\n---\n\nDescription",
		},
		{
			name: "single comment",
			task: &Task{
				Title:       "Test Task",
				Description: "Description",
			},
			comments: []Comment{
				{Text: "First comment", Author: "worker", Time: now},
			},
			want: "---\ntitle: Test Task\nparent:\nlabels:\nskip_review:\n---\n\nDescription\n\n---\n# Comment: 0\n# Author: worker\n# Time: 2026-01-18T10:00:00Z\n\nFirst comment",
		},
		{
			name: "comment with type tags metadata",
			task: &Task{
				Title:       "Test Task",
				Description: "Description",
			},
			comments: []Comment{
				{
					Text:     "Comment text",
					Author:   "worker",
					Time:     now,
					Type:     CommentTypeFriction,
					Tags:     []string{"testing", "docs"},
					Metadata: map[string]string{"source": "cli", "priority": "high"},
				},
			},
			want: "---\ntitle: Test Task\nparent:\nlabels:\nskip_review:\n---\n\nDescription\n\n---\n# Comment: 0\n# Author: worker\n# Type: friction\n# Tags: docs, testing\n# Metadata: priority=high, source=cli\n# Time: 2026-01-18T10:00:00Z\n\nComment text",
		},
		{
			name: "multiple comments",
			task: &Task{
				Title:       "Test Task",
				Description: "Description",
				Labels:      []string{"bug"},
			},
			comments: []Comment{
				{Text: "First comment", Author: "worker", Time: now},
				{Text: "Second comment", Author: "manager", Time: later},
			},
			want: "---\ntitle: Test Task\nparent:\nlabels: bug\nskip_review:\n---\n\nDescription\n\n---\n# Comment: 0\n# Author: worker\n# Time: 2026-01-18T10:00:00Z\n\nFirst comment\n\n---\n# Comment: 1\n# Author: manager\n# Time: 2026-01-18T11:00:00Z\n\nSecond comment",
		},
		{
			name: "comment with empty author",
			task: &Task{
				Title:       "Test Task",
				Description: "Description",
			},
			comments: []Comment{
				{Text: "Comment without author", Author: "", Time: now},
			},
			want: "---\ntitle: Test Task\nparent:\nlabels:\nskip_review:\n---\n\nDescription\n\n---\n# Comment: 0\n# Author: \n# Time: 2026-01-18T10:00:00Z\n\nComment without author",
		},
		{
			name: "multiline comment",
			task: &Task{
				Title:       "Test Task",
				Description: "Description",
			},
			comments: []Comment{
				{Text: "Line 1\nLine 2\nLine 3", Author: "worker", Time: now},
			},
			want: "---\ntitle: Test Task\nparent:\nlabels:\nskip_review:\n---\n\nDescription\n\n---\n# Comment: 0\n# Author: worker\n# Time: 2026-01-18T10:00:00Z\n\nLine 1\nLine 2\nLine 3",
		},
		{
			name: "task with parent",
			task: &Task{
				Title:       "Sub Task",
				Description: "Description",
				ParentID:    intPtr(10),
			},
			comments: nil,
			want:     "---\ntitle: Sub Task\nparent: 10\nlabels:\nskip_review:\n---\n\nDescription",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.task.ToMarkdownWithComments(tt.comments)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseEditorContent(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		wantTitle     string
		wantDesc      string
		errContains   string
		wantLabels    []string
		wantComments  []ParsedComment
		wantLabelsNil bool
		wantErr       bool
	}{
		{
			name: "task only (no comments)",
			content: `---
title: Test Task
labels: bug
---

Description text`,
			wantTitle:    "Test Task",
			wantDesc:     "Description text",
			wantLabels:   []string{"bug"},
			wantComments: nil,
		},
		{
			name: "task with single comment",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment: 0
# Author: worker
# Time: 2026-01-18T10:00:00Z

First comment`,
			wantTitle: "Test Task",
			wantDesc:  "Description",
			wantComments: []ParsedComment{
				{Index: 0, Text: "First comment"},
			},
		},
		{
			name: "comment with type tags metadata",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment: 0
# Author: worker
# Type: friction
# Tags: docs, testing
# Metadata: source=cli, priority=high
# Time: 2026-01-18T10:00:00Z

Comment text`,
			wantTitle: "Test Task",
			wantDesc:  "Description",
			wantComments: []ParsedComment{
				{Index: 0, Text: "Comment text"},
			},
		},
		{
			name: "task with multiple comments",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment: 0
# Author: worker
# Time: 2026-01-18T10:00:00Z

First comment

---
# Comment: 1
# Author: manager
# Time: 2026-01-18T11:00:00Z

Second comment`,
			wantTitle: "Test Task",
			wantDesc:  "Description",
			wantComments: []ParsedComment{
				{Index: 0, Text: "First comment"},
				{Index: 1, Text: "Second comment"},
			},
		},
		{
			name: "multiline comment text",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment: 0
# Author: worker
# Time: 2026-01-18T10:00:00Z

Line 1
Line 2
Line 3`,
			wantTitle: "Test Task",
			wantDesc:  "Description",
			wantComments: []ParsedComment{
				{Index: 0, Text: "Line 1\nLine 2\nLine 3"},
			},
		},
		{
			name: "empty description with comments",
			content: `---
title: Test Task
labels:
---



---
# Comment: 0
# Author: worker
# Time: 2026-01-18T10:00:00Z

Comment text`,
			wantTitle: "Test Task",
			wantDesc:  "",
			wantComments: []ParsedComment{
				{Index: 0, Text: "Comment text"},
			},
		},
		{
			name: "comment with empty author",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment: 0
# Author:
# Time: 2026-01-18T10:00:00Z

Comment text`,
			wantTitle: "Test Task",
			wantDesc:  "Description",
			wantComments: []ParsedComment{
				{Index: 0, Text: "Comment text"},
			},
		},
		{
			name: "invalid comment meta - missing index",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment:
# Author: worker
# Time: 2026-01-18T10:00:00Z

Comment text`,
			wantErr:     true,
			errContains: "invalid comment metadata",
		},
		{
			name: "invalid comment meta - bad index",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment: abc
# Author: worker
# Time: 2026-01-18T10:00:00Z

Comment text`,
			wantErr:     true,
			errContains: "invalid comment metadata",
		},
		{
			name: "empty comment text",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment: 0
# Author: worker
# Time: 2026-01-18T10:00:00Z

`,
			wantErr:     true,
			errContains: "comment text cannot be empty",
		},
		{
			name: "whitespace only comment text",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment: 0
# Author: worker
# Time: 2026-01-18T10:00:00Z

   `,
			wantErr:     true,
			errContains: "comment text cannot be empty",
		},
		{
			name: "invalid time format - not RFC3339",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment: 0
# Author: worker
# Time: 2026/01/18 10:00:00

Comment text`,
			wantErr:     true,
			errContains: "invalid comment metadata",
		},
		{
			name: "invalid time format - empty",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment: 0
# Author: worker
# Time:

Comment text`,
			wantErr:     true,
			errContains: "invalid comment metadata",
		},
		{
			name: "invalid time format - random string",
			content: `---
title: Test Task
labels:
---

Description

---
# Comment: 0
# Author: worker
# Time: not a valid time

Comment text`,
			wantErr:     true,
			errContains: "invalid comment metadata",
		},
		{
			name: "invalid parent - non-numeric",
			content: `---
title: Test Task
parent: abc
labels:
---

Description`,
			wantErr:     true,
			errContains: "invalid parent ID",
		},
		{
			name: "valid parent - numeric",
			content: `---
title: Test Task
parent: 5
labels:
---

Description`,
			wantTitle: "Test Task",
			wantDesc:  "Description",
		},
		{
			name: "valid parent - zero removes parent",
			content: `---
title: Test Task
parent: 0
labels:
---

Description`,
			wantTitle: "Test Task",
			wantDesc:  "Description",
		},
		{
			name: "invalid parent - negative number",
			content: `---
title: Test Task
parent: -1
labels:
---

Description`,
			wantErr:     true,
			errContains: "invalid parent ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEditorContent(tt.content)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTitle, got.Title)
			assert.Equal(t, tt.wantDesc, got.Description)
			if tt.wantLabelsNil {
				assert.Nil(t, got.Labels)
			} else if tt.wantLabels != nil {
				assert.Equal(t, tt.wantLabels, got.Labels)
			}
			assert.Equal(t, tt.wantComments, got.Comments)
		})
	}
}

func TestParseEditorContent_SkipReview(t *testing.T) {
	content := `---
title: Test Task
skip_review: true
labels:
---

Description`
	got, err := ParseEditorContent(content)
	require.NoError(t, err)
	require.True(t, got.SkipReviewFound)
	require.NotNil(t, got.SkipReview)
	assert.True(t, *got.SkipReview)

	content = `---
title: Test Task
skip_review: false
labels:
---

Description`
	got, err = ParseEditorContent(content)
	require.NoError(t, err)
	require.True(t, got.SkipReviewFound)
	require.NotNil(t, got.SkipReview)
	assert.False(t, *got.SkipReview)

	content = `---
title: Test Task
skip_review:
labels:
---

Description`
	got, err = ParseEditorContent(content)
	require.NoError(t, err)
	require.True(t, got.SkipReviewFound)
	assert.Nil(t, got.SkipReview)

	content = `---
title: Test Task
skip_review: yes
labels:
---

Description`
	_, err = ParseEditorContent(content)
	require.Error(t, err)
}

func TestRoundTripWithComments(t *testing.T) {
	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	tests := []struct {
		name     string
		task     *Task
		comments []Comment
	}{
		{
			name: "task with single comment",
			task: &Task{
				Title:       "Round Trip Test",
				Description: "Description",
			},
			comments: []Comment{
				{Text: "First comment", Author: "worker", Time: now},
			},
		},
		{
			name: "task with multiple comments",
			task: &Task{
				Title:       "Multiple Comments",
				Description: "Description",
				Labels:      []string{"bug", "urgent"},
			},
			comments: []Comment{
				{Text: "Comment 1", Author: "worker", Time: now},
				{Text: "Comment 2", Author: "manager", Time: later},
			},
		},
		{
			name: "multiline description and comments",
			task: &Task{
				Title:       "Complex Content",
				Description: "Line 1\nLine 2\nLine 3",
			},
			comments: []Comment{
				{Text: "Multi\nLine\nComment", Author: "worker", Time: now},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			markdown := tt.task.ToMarkdownWithComments(tt.comments)

			// Parse
			content, err := ParseEditorContent(markdown)
			require.NoError(t, err)

			// Verify task fields
			assert.Equal(t, tt.task.Title, content.Title)
			assert.Equal(t, tt.task.Description, content.Description)
			assert.Equal(t, tt.task.Labels, content.Labels)

			// Verify comments
			require.Len(t, content.Comments, len(tt.comments))
			for i, expected := range tt.comments {
				assert.Equal(t, i, content.Comments[i].Index)
				assert.Equal(t, expected.Text, content.Comments[i].Text)
			}
		})
	}
}

// intPtr returns a pointer to the given int.
func intPtr(n int) *int {
	return &n
}

func boolPtr(v bool) *bool {
	return &v
}

func TestTask_IsBlocked(t *testing.T) {
	tests := []struct {
		name        string
		blockReason string
		want        bool
	}{
		{
			name:        "empty block reason - not blocked",
			blockReason: "",
			want:        false,
		},
		{
			name:        "non-empty block reason - blocked",
			blockReason: "Parent task",
			want:        true,
		},
		{
			name:        "block reason with whitespace only is still blocked",
			blockReason: " ",
			want:        true,
		},
		{
			name:        "dependency block reason",
			blockReason: "Depends on #42",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				Title:       "Test Task",
				BlockReason: tt.blockReason,
			}
			got := task.IsBlocked()
			assert.Equal(t, tt.want, got)
		})
	}
}
