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
			want: "---\ntitle: Test Task\n---\n\nThis is a test description",
		},
		{
			name: "with title only",
			task: &Task{
				Title:       "Title Only",
				Description: "",
			},
			want: "---\ntitle: Title Only\n---\n\n",
		},
		{
			name: "with multiline description",
			task: &Task{
				Title:       "Multi-line",
				Description: "Line 1\nLine 2\nLine 3",
			},
			want: "---\ntitle: Multi-line\n---\n\nLine 1\nLine 2\nLine 3",
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
		wantErr     bool
		errContains string
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
			}
		})
	}
}

func TestTask_RoundTrip(t *testing.T) {
	// Test that ToMarkdown -> FromMarkdown preserves data
	original := &Task{
		Title:       "Round Trip Test",
		Description: "This should survive the round trip",
	}

	markdown := original.ToMarkdown()

	parsed := &Task{}
	err := parsed.FromMarkdown(markdown)

	require.NoError(t, err)
	assert.Equal(t, original.Title, parsed.Title)
	assert.Equal(t, original.Description, parsed.Description)
}
