package tui

import (
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     any
		want     string
		wantErr  bool
	}{
		{
			name:     "simple string interpolation",
			template: "Hello {{.Name}}",
			data: struct {
				Name string
			}{Name: "World"},
			want: "Hello World",
		},
		{
			name:     "nested field access",
			template: "Task: {{.Task.Title}}",
			data: struct {
				Task *domain.Task
			}{Task: &domain.Task{Title: "Test Task"}},
			want: "Task: Test Task",
		},
		{
			name:     "invalid template syntax",
			template: "{{.Invalid",
			data:     struct{}{},
			wantErr:  true,
		},
		{
			name:     "missing field",
			template: "{{.Missing}}",
			data:     struct{}{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderTemplate(tt.template, tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestKeybindingTemplateData(t *testing.T) {
	task := &domain.Task{
		ID:          42,
		Title:       "Test Task",
		Status:      domain.StatusInProgress,
		BaseBranch:  "main",
		Description: "Test description",
		Issue:       123,
		Labels:      []string{"bug", "urgent"},
		Created:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data := KeybindingTemplateData{
		Task:         task,
		TaskID:       task.ID,
		TaskTitle:    task.Title,
		TaskStatus:   string(task.Status),
		Branch:       "crew-42-gh-123",
		WorktreePath: "/path/to/worktree",
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		// Backward compatible fields
		{
			name:     "TaskID",
			template: "{{.TaskID}}",
			want:     "42",
		},
		{
			name:     "TaskTitle",
			template: "{{.TaskTitle}}",
			want:     "Test Task",
		},
		{
			name:     "TaskStatus",
			template: "{{.TaskStatus}}",
			want:     "in_progress",
		},
		{
			name:     "Branch",
			template: "{{.Branch}}",
			want:     "crew-42-gh-123",
		},
		{
			name:     "WorktreePath",
			template: "{{.WorktreePath}}",
			want:     "/path/to/worktree",
		},

		// New Task field access
		{
			name:     "Task.ID",
			template: "{{.Task.ID}}",
			want:     "42",
		},
		{
			name:     "Task.Title",
			template: "{{.Task.Title}}",
			want:     "Test Task",
		},
		{
			name:     "Task.BaseBranch",
			template: "{{.Task.BaseBranch}}",
			want:     "main",
		},
		{
			name:     "Task.Description",
			template: "{{.Task.Description}}",
			want:     "Test description",
		},
		{
			name:     "Task.Issue",
			template: "{{.Task.Issue}}",
			want:     "123",
		},
		{
			name:     "Task.Status",
			template: "{{.Task.Status}}",
			want:     "in_progress",
		},

		// Complex templates
		{
			name:     "git log with BaseBranch",
			template: "git log {{.Task.BaseBranch}}..{{.Branch}}",
			want:     "git log main..crew-42-gh-123",
		},
		{
			name:     "mixed old and new fields",
			template: "Task #{{.TaskID}}: {{.Task.Title}} (base: {{.Task.BaseBranch}})",
			want:     "Task #42: Test Task (base: main)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderTemplate(tt.template, data)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestKeybindingTemplateData_Labels(t *testing.T) {
	task := &domain.Task{
		ID:     1,
		Title:  "Test",
		Labels: []string{"bug", "urgent"},
	}

	data := KeybindingTemplateData{
		Task:       task,
		TaskID:     task.ID,
		TaskTitle:  task.Title,
		TaskStatus: string(task.Status),
		Branch:     "crew-1",
	}

	// Access first label using index
	got, err := renderTemplate("{{index .Task.Labels 0}}", data)
	require.NoError(t, err)
	assert.Equal(t, "bug", got)

	// Access labels length
	got, err = renderTemplate("{{len .Task.Labels}}", data)
	require.NoError(t, err)
	assert.Equal(t, "2", got)
}
