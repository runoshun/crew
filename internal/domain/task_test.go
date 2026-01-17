package domain

import (
	"testing"

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
			want: "---\ntitle: Test Task\nlabels:\n---\n\nThis is a test description",
		},
		{
			name: "with title only",
			task: &Task{
				Title:       "Title Only",
				Description: "",
			},
			want: "---\ntitle: Title Only\nlabels:\n---\n\n",
		},
		{
			name: "with multiline description",
			task: &Task{
				Title:       "Multi-line",
				Description: "Line 1\nLine 2\nLine 3",
			},
			want: "---\ntitle: Multi-line\nlabels:\n---\n\nLine 1\nLine 2\nLine 3",
		},
		{
			name: "with single label",
			task: &Task{
				Title:       "Task with label",
				Description: "Description",
				Labels:      []string{"bug"},
			},
			want: "---\ntitle: Task with label\nlabels: bug\n---\n\nDescription",
		},
		{
			name: "with multiple labels",
			task: &Task{
				Title:       "Task with labels",
				Description: "Description",
				Labels:      []string{"bug", "urgent", "frontend"},
			},
			want: "---\ntitle: Task with labels\nlabels: bug, urgent, frontend\n---\n\nDescription",
		},
		{
			name: "with labels and no description",
			task: &Task{
				Title:  "Labels only",
				Labels: []string{"feature"},
			},
			want: "---\ntitle: Labels only\nlabels: feature\n---\n\n",
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
		wantLabels  []string
		errContains string
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
		name     string
		original *Task
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
