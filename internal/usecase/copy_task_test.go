package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyTask_Execute_Success(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Original task",
		Description: "Task description",
		Status:      domain.StatusInProgress,
		Labels:      []string{"bug", "urgent"},
		Issue:       42,
		PR:          10,
		BaseBranch:  "main",
	}
	repo.nextID = 2
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock)

	// Execute
	out, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, out.TaskID)

	// Verify new task
	task := repo.tasks[2]
	require.NotNil(t, task)
	assert.Equal(t, 2, task.ID)
	assert.Equal(t, "Original task (copy)", task.Title)
	assert.Equal(t, "Task description", task.Description)
	assert.Equal(t, domain.StatusTodo, task.Status)
	assert.Equal(t, []string{"bug", "urgent"}, task.Labels)
	assert.Equal(t, clock.now, task.Created)
	// Base branch should be source task's branch name
	assert.Equal(t, "crew-1-gh-42", task.BaseBranch)
	// Issue and PR should NOT be copied
	assert.Equal(t, 0, task.Issue)
	assert.Equal(t, 0, task.PR)
}

func TestCopyTask_Execute_WithCustomTitle(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	repo.nextID = 2
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock)

	// Execute with custom title
	customTitle := "Custom new title"
	out, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
		Title:    &customTitle,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, out.TaskID)

	task := repo.tasks[2]
	assert.Equal(t, "Custom new title", task.Title)
}

func TestCopyTask_Execute_WithParent(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	parentID := 1
	repo.tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Parent task",
		Status:     domain.StatusInProgress,
		BaseBranch: "main",
	}
	repo.tasks[2] = &domain.Task{
		ID:         2,
		ParentID:   &parentID,
		Title:      "Child task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	repo.nextID = 3
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock)

	// Execute - copy child task
	out, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 2,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 3, out.TaskID)

	// Verify parent is inherited
	task := repo.tasks[3]
	require.NotNil(t, task.ParentID)
	assert.Equal(t, 1, *task.ParentID)
}

func TestCopyTask_Execute_SourceNotFound(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock)

	// Execute with non-existent source
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestCopyTask_Execute_GetError(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	repo.getErr = assert.AnError
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock)

	// Execute
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get source task")
}

func TestCopyTask_Execute_SaveError(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	repo.nextID = 2
	repo.saveErr = assert.AnError
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock)

	// Execute
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")
}

func TestCopyTask_Execute_NextIDError(t *testing.T) {
	// Setup
	repo := &mockTaskRepositoryWithNextIDError{
		mockTaskRepository: newMockTaskRepository(),
		nextIDErr:          assert.AnError,
	}
	repo.tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock)

	// Execute
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generate task ID")
}

func TestCopyTask_Execute_LabelsAreCopied(t *testing.T) {
	// Setup - verify labels are deep copied (not shared)
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		Labels:     []string{"original"},
		BaseBranch: "main",
	}
	repo.nextID = 2
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock)

	// Execute
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})
	require.NoError(t, err)

	// Modify original task's labels
	repo.tasks[1].Labels[0] = "modified"

	// Verify copied task's labels are not affected
	assert.Equal(t, []string{"original"}, repo.tasks[2].Labels)
}

func TestCopyTask_Execute_EmptyLabels(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		Labels:     nil,
		BaseBranch: "main",
	}
	repo.nextID = 2
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock)

	// Execute
	out, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})

	// Assert
	require.NoError(t, err)
	task := repo.tasks[out.TaskID]
	assert.Nil(t, task.Labels)
}
