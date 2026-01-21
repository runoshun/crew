package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTaskDrafts(t *testing.T) {
	tests := []struct {
		wantErr error
		name    string
		content string
		want    []TaskDraft
	}{
		{
			name: "single task",
			content: `---
title: First Task
---
Task description here.`,
			want: []TaskDraft{
				{
					Title:       "First Task",
					Description: "Task description here.",
				},
			},
		},
		{
			name: "single task with labels array",
			content: `---
title: Task with Labels
labels: [backend, urgent]
---
Description.`,
			want: []TaskDraft{
				{
					Title:       "Task with Labels",
					Description: "Description.",
					Labels:      []string{"backend", "urgent"},
				},
			},
		},
		{
			name: "single task with labels comma",
			content: `---
title: Task with Labels
labels: backend, urgent
---
Description.`,
			want: []TaskDraft{
				{
					Title:       "Task with Labels",
					Description: "Description.",
					Labels:      []string{"backend", "urgent"},
				},
			},
		},
		{
			name: "multiple tasks",
			content: `---
title: Phase 1: Foundation
labels: [backend]
---
First task body.

## Requirements
- Requirement 1
- Requirement 2

---
title: Phase 2: UseCase
parent: 1
---
Second task body.

Parent is relative index (1-based).`,
			want: []TaskDraft{
				{
					Title:       "Phase 1: Foundation",
					Description: "First task body.\n\n## Requirements\n- Requirement 1\n- Requirement 2",
					Labels:      []string{"backend"},
				},
				{
					Title:       "Phase 2: UseCase",
					Description: "Second task body.\n\nParent is relative index (1-based).",
					ParentRef:   "1",
				},
			},
		},
		{
			name: "task with absolute parent",
			content: `---
title: Sub Task
parent: 123
---
Task with existing parent.`,
			want: []TaskDraft{
				{
					Title:       "Sub Task",
					Description: "Task with existing parent.",
					ParentRef:   "123",
				},
			},
		},
		{
			name: "task without description",
			content: `---
title: No Description
labels: [quick]
---`,
			want: []TaskDraft{
				{
					Title:  "No Description",
					Labels: []string{"quick"},
				},
			},
		},
		{
			name: "three tasks with hierarchy",
			content: `---
title: Root Task
---
Root description.

---
title: Child Task 1
parent: 1
---
First child.

---
title: Child Task 2
parent: 1
---
Second child.`,
			want: []TaskDraft{
				{
					Title:       "Root Task",
					Description: "Root description.",
				},
				{
					Title:       "Child Task 1",
					Description: "First child.",
					ParentRef:   "1",
				},
				{
					Title:       "Child Task 2",
					Description: "Second child.",
					ParentRef:   "1",
				},
			},
		},
		{
			name:    "empty content",
			content: "",
			wantErr: ErrEmptyFile,
		},
		{
			name: "missing title",
			content: `---
labels: [bug]
---
No title here.`,
			wantErr: ErrEmptyTitle,
		},
		{
			name: "description with markdown separator",
			content: `---
title: Task with Separator
---
Some content here.

---

More content after separator (not a new task).`,
			want: []TaskDraft{
				{
					Title:       "Task with Separator",
					Description: "Some content here.\n\n---\n\nMore content after separator (not a new task).",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTaskDrafts(tt.content)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Len(t, got, len(tt.want))

			for i, want := range tt.want {
				assert.Equal(t, want.Title, got[i].Title, "task %d title", i)
				assert.Equal(t, want.Description, got[i].Description, "task %d description", i)
				assert.Equal(t, want.Labels, got[i].Labels, "task %d labels", i)
				assert.Equal(t, want.ParentRef, got[i].ParentRef, "task %d parentRef", i)
			}
		})
	}
}

func TestResolveParentRef(t *testing.T) {
	createdIDs := map[int]int{
		1: 100, // Relative index 1 -> Created task ID 100
		2: 101, // Relative index 2 -> Created task ID 101
	}

	tests := []struct {
		wantErr    error
		createdIDs map[int]int
		want       *int
		name       string
		ref        string
	}{
		{
			name:       "empty ref returns nil",
			ref:        "",
			createdIDs: createdIDs,
			want:       nil,
		},
		{
			name:       "relative ref to first task",
			ref:        "1",
			createdIDs: createdIDs,
			want:       intPtr(100),
		},
		{
			name:       "relative ref to second task",
			ref:        "2",
			createdIDs: createdIDs,
			want:       intPtr(101),
		},
		{
			name:       "absolute ref (not in createdIDs)",
			ref:        "500",
			createdIDs: createdIDs,
			want:       intPtr(500),
		},
		{
			name:       "explicit absolute ref with # prefix",
			ref:        "#123",
			createdIDs: createdIDs,
			want:       intPtr(123),
		},
		{
			name:       "explicit absolute ref #1 bypasses relative lookup",
			ref:        "#1",
			createdIDs: createdIDs,
			want:       intPtr(1), // Returns 1, not 100 (the mapped value)
		},
		{
			name:       "explicit absolute ref #2 bypasses relative lookup",
			ref:        "#2",
			createdIDs: createdIDs,
			want:       intPtr(2), // Returns 2, not 101 (the mapped value)
		},
		{
			name:       "invalid # ref (non-numeric after #)",
			ref:        "#abc",
			createdIDs: createdIDs,
			wantErr:    ErrInvalidParentRef,
		},
		{
			name:       "invalid # ref (zero)",
			ref:        "#0",
			createdIDs: createdIDs,
			wantErr:    ErrInvalidParentRef,
		},
		{
			name:       "invalid # ref (negative)",
			ref:        "#-1",
			createdIDs: createdIDs,
			wantErr:    ErrInvalidParentRef,
		},
		{
			name:       "invalid ref (non-numeric)",
			ref:        "abc",
			createdIDs: createdIDs,
			wantErr:    ErrInvalidParentRef,
		},
		{
			name:       "invalid ref (zero)",
			ref:        "0",
			createdIDs: createdIDs,
			wantErr:    ErrInvalidParentRef,
		},
		{
			name:       "invalid ref (negative)",
			ref:        "-1",
			createdIDs: createdIDs,
			wantErr:    ErrInvalidParentRef,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveParentRef(tt.ref, tt.createdIDs)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseLabelsValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{
			name:  "empty value",
			value: "",
			want:  nil,
		},
		{
			name:  "single label",
			value: "backend",
			want:  []string{"backend"},
		},
		{
			name:  "comma-separated",
			value: "backend, frontend, urgent",
			want:  []string{"backend", "frontend", "urgent"},
		},
		{
			name:  "array syntax",
			value: "[backend, frontend]",
			want:  []string{"backend", "frontend"},
		},
		{
			name:  "with duplicates",
			value: "bug, bug, urgent",
			want:  []string{"bug", "urgent"},
		},
		{
			name:  "extra whitespace",
			value: "  backend  ,  frontend  ",
			want:  []string{"backend", "frontend"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLabelsValue(tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}
