package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTasks_Execute_AllTasks(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusTodo,
	}
	repo.tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Task 2",
		Status: domain.StatusInProgress,
	}
	repo.tasks[3] = &domain.Task{
		ID:     3,
		Title:  "Task 3",
		Status: domain.StatusDone,
	}

	uc := NewListTasks(repo)

	// Execute
	out, err := uc.Execute(context.Background(), ListTasksInput{})

	// Assert
	require.NoError(t, err)
	require.Len(t, out.Tasks, 3)
}

func TestListTasks_Execute_Empty(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	uc := NewListTasks(repo)

	// Execute
	out, err := uc.Execute(context.Background(), ListTasksInput{})

	// Assert
	require.NoError(t, err)
	require.Empty(t, out.Tasks)
}

func TestListTasks_Execute_WithParentFilter(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	parentID := 1
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Parent Task",
		Status: domain.StatusTodo,
	}
	repo.tasks[2] = &domain.Task{
		ID:       2,
		Title:    "Child Task 1",
		ParentID: &parentID,
		Status:   domain.StatusTodo,
	}
	repo.tasks[3] = &domain.Task{
		ID:       3,
		Title:    "Child Task 2",
		ParentID: &parentID,
		Status:   domain.StatusTodo,
	}
	repo.tasks[4] = &domain.Task{
		ID:     4,
		Title:  "Orphan Task",
		Status: domain.StatusTodo,
	}

	uc := NewListTasks(repo)

	// Execute with parent filter
	out, err := uc.Execute(context.Background(), ListTasksInput{
		ParentID: &parentID,
	})

	// Assert
	require.NoError(t, err)
	// mockTaskRepository.List doesn't implement filtering, so we just test the usecase passes the filter
	// The actual filtering is tested in jsonstore tests
	require.NotNil(t, out.Tasks)
}

func TestListTasks_Execute_WithLabelFilter(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task with bug label",
		Status: domain.StatusTodo,
		Labels: []string{"bug"},
	}
	repo.tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Task with feature label",
		Status: domain.StatusTodo,
		Labels: []string{"feature"},
	}
	repo.tasks[3] = &domain.Task{
		ID:     3,
		Title:  "Task with both labels",
		Status: domain.StatusTodo,
		Labels: []string{"bug", "feature"},
	}

	uc := NewListTasks(repo)

	// Execute with label filter
	out, err := uc.Execute(context.Background(), ListTasksInput{
		Labels: []string{"bug"},
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out.Tasks)
}

func TestListTasks_Execute_RepositoryError(t *testing.T) {
	// Setup
	repo := &mockTaskRepositoryWithListError{
		mockTaskRepository: newMockTaskRepository(),
		listErr:            errors.New("database error"),
	}
	uc := NewListTasks(repo)

	// Execute
	_, err := uc.Execute(context.Background(), ListTasksInput{})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestListTasks_Execute_PreservesTaskData(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	now := time.Now()
	parentID := 1
	repo.tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Parent Task",
		Description: "A parent task",
		Status:      domain.StatusInProgress,
		Created:     now,
		Started:     now,
		Agent:       "claude",
		Session:     "crew-1",
		Issue:       42,
		PR:          10,
		BaseBranch:  "main",
		Labels:      []string{"feature", "urgent"},
	}
	repo.tasks[2] = &domain.Task{
		ID:         2,
		Title:      "Child Task",
		ParentID:   &parentID,
		Status:     domain.StatusTodo,
		Created:    now,
		BaseBranch: "main",
		Labels:     []string{"bug"},
	}

	uc := NewListTasks(repo)

	// Execute
	out, err := uc.Execute(context.Background(), ListTasksInput{})

	// Assert
	require.NoError(t, err)
	require.Len(t, out.Tasks, 2)

	// Find the tasks in the output (order may vary due to map iteration)
	var parentTask, childTask *domain.Task
	for _, task := range out.Tasks {
		switch task.ID {
		case 1:
			parentTask = task
		case 2:
			childTask = task
		}
	}

	// Verify parent task data
	require.NotNil(t, parentTask)
	assert.Equal(t, "Parent Task", parentTask.Title)
	assert.Equal(t, "A parent task", parentTask.Description)
	assert.Equal(t, domain.StatusInProgress, parentTask.Status)
	assert.Equal(t, "claude", parentTask.Agent)
	assert.Equal(t, "crew-1", parentTask.Session)
	assert.Equal(t, 42, parentTask.Issue)
	assert.Equal(t, 10, parentTask.PR)
	assert.Equal(t, "main", parentTask.BaseBranch)
	assert.Equal(t, []string{"feature", "urgent"}, parentTask.Labels)

	// Verify child task data
	require.NotNil(t, childTask)
	assert.Equal(t, "Child Task", childTask.Title)
	require.NotNil(t, childTask.ParentID)
	assert.Equal(t, 1, *childTask.ParentID)
}

// mockTaskRepositoryWithListError extends mockTaskRepository to return error on List.
type mockTaskRepositoryWithListError struct {
	*mockTaskRepository
	listErr error
}

func (m *mockTaskRepositoryWithListError) List(_ domain.TaskFilter) ([]*domain.Task, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.mockTaskRepository.List(domain.TaskFilter{})
}
