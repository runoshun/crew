package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditTask_Execute_UpdateTitle(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Original title",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute
	newTitle := "Updated title"
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID: 1,
		Title:  &newTitle,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, out.Task.ID)
	assert.Equal(t, "Updated title", out.Task.Title)
}

func TestEditTask_Execute_UpdateDescription(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Test task",
		Description: "Original description",
		Status:      domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute
	newDesc := "Updated description"
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:      1,
		Description: &newDesc,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Updated description", out.Task.Description)
	assert.Equal(t, "Test task", out.Task.Title) // Title unchanged
}

func TestEditTask_Execute_AddLabels(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Labels: []string{"existing"},
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:    1,
		AddLabels: []string{"new", "urgent"},
	})

	// Assert
	require.NoError(t, err)
	assert.Len(t, out.Task.Labels, 3)
	assert.Contains(t, out.Task.Labels, "existing")
	assert.Contains(t, out.Task.Labels, "new")
	assert.Contains(t, out.Task.Labels, "urgent")
}

func TestEditTask_Execute_RemoveLabels(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Labels: []string{"keep", "remove-me", "also-keep"},
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:       1,
		RemoveLabels: []string{"remove-me"},
	})

	// Assert
	require.NoError(t, err)
	assert.Len(t, out.Task.Labels, 2)
	assert.Contains(t, out.Task.Labels, "keep")
	assert.Contains(t, out.Task.Labels, "also-keep")
	assert.NotContains(t, out.Task.Labels, "remove-me")
}

func TestEditTask_Execute_AddAndRemoveLabels(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Labels: []string{"old"},
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:       1,
		AddLabels:    []string{"new"},
		RemoveLabels: []string{"old"},
	})

	// Assert
	require.NoError(t, err)
	assert.Len(t, out.Task.Labels, 1)
	assert.Contains(t, out.Task.Labels, "new")
	assert.NotContains(t, out.Task.Labels, "old")
}

func TestEditTask_Execute_MultipleUpdates(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Original",
		Description: "Old desc",
		Labels:      []string{"old"},
		Status:      domain.StatusTodo,
		Created:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	uc := NewEditTask(repo)

	// Execute
	newTitle := "Updated"
	newDesc := "New desc"
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:       1,
		Title:        &newTitle,
		Description:  &newDesc,
		AddLabels:    []string{"new"},
		RemoveLabels: []string{"old"},
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Updated", out.Task.Title)
	assert.Equal(t, "New desc", out.Task.Description)
	assert.Len(t, out.Task.Labels, 1)
	assert.Contains(t, out.Task.Labels, "new")
}

func TestEditTask_Execute_NoFieldsToUpdate(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with no updates
	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrNoFieldsToUpdate)
}

func TestEditTask_Execute_EmptyTitle(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with empty title
	emptyTitle := ""
	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID: 1,
		Title:  &emptyTitle,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrEmptyTitle)
}

func TestEditTask_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	uc := NewEditTask(repo)

	// Execute for non-existent task
	newTitle := "Updated"
	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID: 999,
		Title:  &newTitle,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestEditTask_Execute_SaveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	repo.SaveErr = assert.AnError
	uc := NewEditTask(repo)

	// Execute
	newTitle := "Updated"
	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID: 1,
		Title:  &newTitle,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")
}

func TestEditTask_Execute_GetError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	uc := NewEditTask(repo)

	// Execute
	newTitle := "Updated"
	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID: 1,
		Title:  &newTitle,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestEditTask_Execute_ClearDescription(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Test task",
		Description: "Has description",
		Status:      domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with empty description (to clear it)
	emptyDesc := ""
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:      1,
		Description: &emptyDesc,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "", out.Task.Description)
}

func TestEditTask_Execute_RemoveAllLabels(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Labels: []string{"a", "b", "c"},
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:       1,
		RemoveLabels: []string{"a", "b", "c"},
	})

	// Assert
	require.NoError(t, err)
	assert.Nil(t, out.Task.Labels)
}

func TestEditTask_Execute_AddDuplicateLabel(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Labels: []string{"existing"},
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - adding a label that already exists
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:    1,
		AddLabels: []string{"existing", "new"},
	})

	// Assert
	require.NoError(t, err)
	assert.Len(t, out.Task.Labels, 2)
	assert.Contains(t, out.Task.Labels, "existing")
	assert.Contains(t, out.Task.Labels, "new")
}

func TestEditTask_Execute_ReplaceLabels(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Labels: []string{"old1", "old2"},
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - replace all labels
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:    1,
		Labels:    []string{"new1", "new2", "new3"},
		LabelsSet: true,
	})

	// Assert
	require.NoError(t, err)
	assert.Len(t, out.Task.Labels, 3)
	assert.ElementsMatch(t, []string{"new1", "new2", "new3"}, out.Task.Labels)
}

func TestEditTask_Execute_ClearLabels(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Labels: []string{"a", "b", "c"},
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - clear all labels by setting empty slice with LabelsSet=true
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:    1,
		Labels:    []string{},
		LabelsSet: true,
	})

	// Assert
	require.NoError(t, err)
	assert.Nil(t, out.Task.Labels)
}

func TestEditTask_Execute_LabelsWithAddLabelsConflict(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Labels: []string{"old"},
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - Labels takes precedence, AddLabels/RemoveLabels are ignored
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:    1,
		Labels:    []string{"replaced"},
		LabelsSet: true,
		AddLabels: []string{"should-be-ignored"},
	})

	// Assert
	require.NoError(t, err)
	assert.Len(t, out.Task.Labels, 1)
	assert.Contains(t, out.Task.Labels, "replaced")
	assert.NotContains(t, out.Task.Labels, "should-be-ignored")
}

func TestEditTask_Execute_LabelsDeduplicated(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Labels: []string{"old"},
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - duplicate labels in input should be deduplicated
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:    1,
		Labels:    []string{"a", "b", "a", "c", "b"},
		LabelsSet: true,
	})

	// Assert
	require.NoError(t, err)
	assert.Len(t, out.Task.Labels, 3)
	assert.ElementsMatch(t, []string{"a", "b", "c"}, out.Task.Labels)
}

func TestUpdateLabels(t *testing.T) {
	tests := []struct {
		name    string
		current []string
		add     []string
		remove  []string
		want    []string
	}{
		{
			name:    "add to empty",
			current: nil,
			add:     []string{"a"},
			remove:  nil,
			want:    []string{"a"},
		},
		{
			name:    "remove from existing",
			current: []string{"a", "b"},
			add:     nil,
			remove:  []string{"a"},
			want:    []string{"b"},
		},
		{
			name:    "add and remove",
			current: []string{"a"},
			add:     []string{"b"},
			remove:  []string{"a"},
			want:    []string{"b"},
		},
		{
			name:    "remove non-existent",
			current: []string{"a"},
			add:     nil,
			remove:  []string{"b"},
			want:    []string{"a"},
		},
		{
			name:    "add duplicate",
			current: []string{"a"},
			add:     []string{"a"},
			remove:  nil,
			want:    []string{"a"},
		},
		{
			name:    "remove all",
			current: []string{"a", "b"},
			add:     nil,
			remove:  []string{"a", "b"},
			want:    nil,
		},
		{
			name:    "add while removing same label",
			current: []string{"a"},
			add:     []string{"a"},
			remove:  []string{"a"},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateLabels(tt.current, tt.add, tt.remove)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				assert.ElementsMatch(t, tt.want, got)
			}
		})
	}
}

func TestEditTask_Execute_UpdateStatus(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute
	newStatus := domain.StatusInProgress
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID: 1,
		Status: &newStatus,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, domain.StatusInProgress, out.Task.Status)
}

func TestEditTask_Execute_UpdateStatusWithOtherFields(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Original title",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - update both status and title
	newStatus := domain.StatusInProgress
	newTitle := "Updated title"
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID: 1,
		Status: &newStatus,
		Title:  &newTitle,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, domain.StatusInProgress, out.Task.Status)
	assert.Equal(t, "Updated title", out.Task.Title)
}

func TestEditTask_Execute_InvalidStatus(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with invalid status
	invalidStatus := domain.Status("invalid")
	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID: 1,
		Status: &invalidStatus,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrInvalidStatus)
}

func TestEditTask_Execute_StatusTransitions(t *testing.T) {
	tests := []struct {
		name       string
		fromStatus domain.Status
		toStatus   domain.Status
		wantErr    bool
	}{
		// Valid transitions
		{"todo to in_progress", domain.StatusTodo, domain.StatusInProgress, false},
		{"in_progress to for_review", domain.StatusInProgress, domain.StatusForReview, false},
		{"for_review to closed", domain.StatusForReview, domain.StatusClosed, false},
		{"any to closed", domain.StatusTodo, domain.StatusClosed, false},
		// Same status (no change)
		{"todo to todo", domain.StatusTodo, domain.StatusTodo, false},
		// Backward transitions (allowed for manual edit)
		{"in_progress to todo", domain.StatusInProgress, domain.StatusTodo, false},
		{"done to in_progress", domain.StatusClosed, domain.StatusInProgress, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testutil.NewMockTaskRepository()
			repo.Tasks[1] = &domain.Task{
				ID:     1,
				Title:  "Test task",
				Status: tt.fromStatus,
			}
			uc := NewEditTask(repo)

			out, err := uc.Execute(context.Background(), EditTaskInput{
				TaskID: 1,
				Status: &tt.toStatus,
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.toStatus, out.Task.Status)
			}
		})
	}
}

func TestEditTask_Execute_EditorMode_UpdateTitleAndDescription(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Original Title",
		Description: "Original description",
		Status:      domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with editor mode
	markdown := `---
title: Updated Title
---

Updated description text`

	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", out.Task.Title)
	assert.Equal(t, "Updated description text", out.Task.Description)
}

func TestEditTask_Execute_EditorMode_EmptyDescription(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Original Title",
		Description: "Original description",
		Status:      domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with editor mode (empty description)
	markdown := `---
title: Title Only
---

`

	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Title Only", out.Task.Title)
	assert.Equal(t, "", out.Task.Description)
}

func TestEditTask_Execute_EditorMode_InvalidFrontmatter(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Original Title",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with invalid frontmatter
	markdown := `title: No frontmatter delimiters

Description`

	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "frontmatter")
}

func TestEditTask_Execute_EditorMode_MissingTitle(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Original Title",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with missing title in frontmatter
	markdown := `---
---

Description only`

	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrEmptyTitle)
}

func TestEditTask_Execute_EditorMode_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	uc := NewEditTask(repo)

	// Execute for non-existent task
	markdown := `---
title: Test
---

Description`

	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     999,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestEditTask_Execute_ConditionalStatusUpdate_MatchingStatus(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	uc := NewEditTask(repo)

	// Execute - update status with matching condition
	newStatus := domain.StatusNeedsInput
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:   1,
		Status:   &newStatus,
		IfStatus: []domain.Status{domain.StatusInProgress},
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, domain.StatusNeedsInput, out.Task.Status)
}

func TestEditTask_Execute_ConditionalStatusUpdate_NonMatchingStatus(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - update status with non-matching condition
	newStatus := domain.StatusNeedsInput
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:   1,
		Status:   &newStatus,
		IfStatus: []domain.Status{domain.StatusInProgress},
	})

	// Assert - no error, but status should not change
	require.NoError(t, err)
	assert.Equal(t, domain.StatusTodo, out.Task.Status, "status should not change when condition not met")
}

func TestEditTask_Execute_ConditionalStatusUpdate_MultipleConditions(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusNeedsInput,
	}
	uc := NewEditTask(repo)

	// Execute - update status with multiple conditions (one matches)
	newStatus := domain.StatusInProgress
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:   1,
		Status:   &newStatus,
		IfStatus: []domain.Status{domain.StatusInProgress, domain.StatusNeedsInput},
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, domain.StatusInProgress, out.Task.Status)
}

func TestEditTask_Execute_ConditionalStatusUpdate_WithOtherFields(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Original title",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - conditional status update with title change
	// Status condition not met, but title should still be updated
	newStatus := domain.StatusNeedsInput
	newTitle := "Updated title"
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:   1,
		Status:   &newStatus,
		IfStatus: []domain.Status{domain.StatusInProgress},
		Title:    &newTitle,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, domain.StatusTodo, out.Task.Status, "status should not change when condition not met")
	assert.Equal(t, "Updated title", out.Task.Title, "title should be updated even when status condition not met")
}

func TestEditTask_Execute_InvalidIfStatus(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with invalid IfStatus
	newStatus := domain.StatusInProgress
	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:   1,
		Status:   &newStatus,
		IfStatus: []domain.Status{domain.Status("invalid")},
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrInvalidStatus)
}

func TestEditTask_Execute_EditorMode_WithComments(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Original Title",
		Description: "Description",
		Status:      domain.StatusTodo,
	}
	repo.Comments[1] = []domain.Comment{
		{Text: "Original comment", Author: "worker", Time: now},
	}
	uc := NewEditTask(repo)

	// Execute with editor mode - edit comment text
	markdown := `---
title: Updated Title
labels:
---

Updated description

---
# Comment: 0
# Author: worker
# Time: 2026-01-18T10:00:00Z

Edited comment text`

	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", out.Task.Title)
	assert.Equal(t, "Updated description", out.Task.Description)

	// Verify comment was updated
	comments := repo.Comments[1]
	require.Len(t, comments, 1)
	assert.Equal(t, "Edited comment text", comments[0].Text)
	// Author and time should be preserved
	assert.Equal(t, "worker", comments[0].Author)
	assert.Equal(t, now, comments[0].Time)
}

func TestEditTask_Execute_EditorMode_MultipleComments(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Title",
		Description: "Description",
		Status:      domain.StatusTodo,
	}
	repo.Comments[1] = []domain.Comment{
		{Text: "First comment", Author: "worker", Time: now},
		{Text: "Second comment", Author: "manager", Time: later},
	}
	uc := NewEditTask(repo)

	// Execute with editor mode - edit only second comment
	markdown := `---
title: Title
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

Edited second comment`

	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Title", out.Task.Title)

	// Verify comments
	comments := repo.Comments[1]
	require.Len(t, comments, 2)
	assert.Equal(t, "First comment", comments[0].Text) // unchanged
	assert.Equal(t, "Edited second comment", comments[1].Text)
	// Metadata preserved
	assert.Equal(t, "manager", comments[1].Author)
	assert.Equal(t, later, comments[1].Time)
}

func TestEditTask_Execute_EditorMode_CommentUnchanged(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Title",
		Description: "Description",
		Status:      domain.StatusTodo,
	}
	repo.Comments[1] = []domain.Comment{
		{Text: "Same comment", Author: "worker", Time: now},
	}
	uc := NewEditTask(repo)

	// Execute with editor mode - comment text unchanged
	markdown := `---
title: Title
labels:
---

Description

---
# Comment: 0
# Author: worker
# Time: 2026-01-18T10:00:00Z

Same comment`

	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Title", out.Task.Title)
	// Comment should still be unchanged
	comments := repo.Comments[1]
	require.Len(t, comments, 1)
	assert.Equal(t, "Same comment", comments[0].Text)
}

func TestEditTask_Execute_EditorMode_InvalidCommentIndex(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Title",
		Description: "Description",
		Status:      domain.StatusTodo,
	}
	repo.Comments[1] = []domain.Comment{
		{Text: "Only comment", Author: "worker", Time: time.Now()},
	}
	uc := NewEditTask(repo)

	// Execute with editor mode - reference non-existent comment index
	markdown := `---
title: Title
labels:
---

Description

---
# Comment: 5
# Author: worker
# Time: 2026-01-18T10:00:00Z

Invalid comment`

	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid comment index")
}

func TestEditTask_Execute_EditorMode_MissingComment(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Title",
		Description: "Description",
		Status:      domain.StatusTodo,
	}
	repo.Comments[1] = []domain.Comment{
		{Text: "First comment", Author: "worker", Time: now},
		{Text: "Second comment", Author: "manager", Time: later},
	}
	uc := NewEditTask(repo)

	// Execute with editor mode - only include one comment when two exist
	markdown := `---
title: Title
labels:
---

Description

---
# Comment: 0
# Author: worker
# Time: 2026-01-18T10:00:00Z

First comment`

	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert - should error because comment was removed
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "comment count mismatch")

	// Verify task was NOT saved (atomic behavior)
	assert.Equal(t, "Title", repo.Tasks[1].Title)
}

func TestEditTask_Execute_EditorMode_NoCommentsWhenOriginalHasComments(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Title",
		Description: "Description",
		Status:      domain.StatusTodo,
	}
	repo.Comments[1] = []domain.Comment{
		{Text: "Existing comment", Author: "worker", Time: now},
	}
	uc := NewEditTask(repo)

	// Execute with editor mode - no comments section when original has comments
	markdown := `---
title: Updated Title
labels:
---

Updated description`

	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert - should error because all comments were removed
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "comment count mismatch")

	// Verify task was NOT saved (atomic behavior)
	assert.Equal(t, "Title", repo.Tasks[1].Title)
}

func TestEditTask_Execute_EditorMode_DuplicateCommentIndex(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Title",
		Description: "Description",
		Status:      domain.StatusTodo,
	}
	repo.Comments[1] = []domain.Comment{
		{Text: "First comment", Author: "worker", Time: now},
		{Text: "Second comment", Author: "manager", Time: later},
	}
	uc := NewEditTask(repo)

	// Execute with editor mode - duplicate index 0
	markdown := `---
title: Title
labels:
---

Description

---
# Comment: 0
# Author: worker
# Time: 2026-01-18T10:00:00Z

First comment

---
# Comment: 0
# Author: manager
# Time: 2026-01-18T11:00:00Z

Second comment with wrong index`

	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert - should error because of duplicate index
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate comment index")

	// Verify nothing was saved (atomic behavior)
	assert.Equal(t, "First comment", repo.Comments[1][0].Text)
	assert.Equal(t, "Second comment", repo.Comments[1][1].Text)
}

func TestEditTask_Execute_SetParent(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Child task",
		Status: domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Parent task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - set parent
	parentID := 2
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:   1,
		ParentID: &parentID,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out.Task.ParentID)
	assert.Equal(t, 2, *out.Task.ParentID)
}

func TestEditTask_Execute_RemoveParent(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	parentID := 2
	repo.Tasks[1] = &domain.Task{
		ID:       1,
		Title:    "Child task",
		ParentID: &parentID,
		Status:   domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Parent task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - remove parent using RemoveParent flag
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:       1,
		RemoveParent: true,
	})

	// Assert
	require.NoError(t, err)
	assert.Nil(t, out.Task.ParentID)
}

func TestEditTask_Execute_RemoveParentWithZero(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	parentID := 2
	repo.Tasks[1] = &domain.Task{
		ID:       1,
		Title:    "Child task",
		ParentID: &parentID,
		Status:   domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Parent task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - remove parent using ParentID=0
	zeroParentID := 0
	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:   1,
		ParentID: &zeroParentID,
	})

	// Assert
	require.NoError(t, err)
	assert.Nil(t, out.Task.ParentID)
}

func TestEditTask_Execute_ParentNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Child task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - set non-existent parent
	parentID := 999
	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:   1,
		ParentID: &parentID,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrParentNotFound)
}

func TestEditTask_Execute_CircularReference_SelfReference(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - try to set self as parent
	parentID := 1
	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:   1,
		ParentID: &parentID,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrCircularReference)
}

func TestEditTask_Execute_CircularReference_ChildAsParent(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	parentID := 1
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Parent task",
		Status: domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:       2,
		Title:    "Child task",
		ParentID: &parentID,
		Status:   domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - try to set child as parent (creates cycle)
	newParentID := 2
	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:   1,
		ParentID: &newParentID,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrCircularReference)
}

func TestEditTask_Execute_CircularReference_GrandchildAsParent(t *testing.T) {
	// Setup: 1 -> 2 -> 3 (grandchild)
	repo := testutil.NewMockTaskRepository()
	parentID1 := 1
	parentID2 := 2
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Root task",
		Status: domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:       2,
		Title:    "Child task",
		ParentID: &parentID1,
		Status:   domain.StatusTodo,
	}
	repo.Tasks[3] = &domain.Task{
		ID:       3,
		Title:    "Grandchild task",
		ParentID: &parentID2,
		Status:   domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute - try to set grandchild (3) as parent of root (1)
	newParentID := 3
	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:   1,
		ParentID: &newParentID,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrCircularReference)
}

func TestEditTask_Execute_EditorMode_SetParent(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Child task",
		Description: "Description",
		Status:      domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Parent task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with editor mode - set parent
	markdown := `---
title: Child task
parent: 2
labels:
---

Description`

	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out.Task.ParentID)
	assert.Equal(t, 2, *out.Task.ParentID)
}

func TestEditTask_Execute_EditorMode_RemoveParent(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	parentID := 2
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Child task",
		Description: "Description",
		ParentID:    &parentID,
		Status:      domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Parent task",
		Status: domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with editor mode - remove parent (empty parent field)
	markdown := `---
title: Child task
parent:
labels:
---

Description`

	out, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	require.NoError(t, err)
	assert.Nil(t, out.Task.ParentID)
}

func TestEditTask_Execute_EditorMode_CircularReference(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	parentID := 1
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Parent task",
		Description: "Description",
		Status:      domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:       2,
		Title:    "Child task",
		ParentID: &parentID,
		Status:   domain.StatusTodo,
	}
	uc := NewEditTask(repo)

	// Execute with editor mode - try to set child as parent
	markdown := `---
title: Parent task
parent: 2
labels:
---

Description`

	_, err := uc.Execute(context.Background(), EditTaskInput{
		TaskID:     1,
		EditorEdit: true,
		EditorText: markdown,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrCircularReference)
}
