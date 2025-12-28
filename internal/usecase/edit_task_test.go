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
