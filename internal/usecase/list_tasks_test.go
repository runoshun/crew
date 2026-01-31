package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockNamespaceLister struct {
	*testutil.MockTaskRepository
	listAllCalled bool
	listAllTasks  []*domain.Task
	listAllErr    error
}

func (m *mockNamespaceLister) ListAll(filter domain.TaskFilter) ([]*domain.Task, error) {
	m.listAllCalled = true
	return m.listAllTasks, m.listAllErr
}

func TestListTasks_Execute_AllTasks(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task 1",
		Status: domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Task 2",
		Status: domain.StatusInProgress,
	}
	repo.Tasks[3] = &domain.Task{
		ID:     3,
		Title:  "Task 3",
		Status: domain.StatusClosed,
	}

	uc := NewListTasks(repo, nil)

	// Execute with IncludeTerminal = true
	out, err := uc.Execute(context.Background(), ListTasksInput{
		IncludeTerminal: true,
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, out.Tasks, 3)
}

func TestListTasks_Execute_AllNamespaces(t *testing.T) {
	repo := &mockNamespaceLister{MockTaskRepository: testutil.NewMockTaskRepository()}
	repo.listAllTasks = []*domain.Task{
		{ID: 1, Title: "Alpha", Status: domain.StatusTodo, Namespace: "alpha"},
		{ID: 2, Title: "Beta", Status: domain.StatusTodo, Namespace: "beta"},
	}

	uc := NewListTasks(repo, nil)
	out, err := uc.Execute(context.Background(), ListTasksInput{AllNamespaces: true})

	require.NoError(t, err)
	assert.True(t, repo.listAllCalled)
	require.Len(t, out.Tasks, 2)
}

func TestListTasks_Execute_ExcludeTerminal(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Active Task",
		Status: domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Done Task",
		Status: domain.StatusClosed,
	}
	repo.Tasks[3] = &domain.Task{
		ID:     3,
		Title:  "Closed Task",
		Status: domain.StatusClosed,
	}

	uc := NewListTasks(repo, nil)

	// Execute with IncludeTerminal = false (default)
	out, err := uc.Execute(context.Background(), ListTasksInput{
		IncludeTerminal: false,
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, out.Tasks, 1) // Only active task
	assert.Equal(t, 1, out.Tasks[0].ID)
}

func TestListTasks_Execute_Empty(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	uc := NewListTasks(repo, nil)

	// Execute
	out, err := uc.Execute(context.Background(), ListTasksInput{})

	// Assert
	require.NoError(t, err)
	require.Empty(t, out.Tasks)
}

func TestListTasks_Execute_WithParentFilter(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	parentID := 1
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Parent Task",
		Status: domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:       2,
		Title:    "Child Task 1",
		ParentID: &parentID,
		Status:   domain.StatusTodo,
	}
	repo.Tasks[3] = &domain.Task{
		ID:       3,
		Title:    "Child Task 2",
		ParentID: &parentID,
		Status:   domain.StatusTodo,
	}
	repo.Tasks[4] = &domain.Task{
		ID:     4,
		Title:  "Orphan Task",
		Status: domain.StatusTodo,
	}

	uc := NewListTasks(repo, nil)

	// Execute with parent filter
	out, err := uc.Execute(context.Background(), ListTasksInput{
		ParentID: &parentID,
	})

	// Assert
	require.NoError(t, err)
	// mockTaskRepository.List doesn't implement filtering, so we just test the usecase passes the filter.
	// Filtering behavior is covered in filestore tests.
	require.NotNil(t, out.Tasks)
}

func TestListTasks_Execute_WithLabelFilter(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task with bug label",
		Status: domain.StatusTodo,
		Labels: []string{"bug"},
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Task with feature label",
		Status: domain.StatusTodo,
		Labels: []string{"feature"},
	}
	repo.Tasks[3] = &domain.Task{
		ID:     3,
		Title:  "Task with both labels",
		Status: domain.StatusTodo,
		Labels: []string{"bug", "feature"},
	}

	uc := NewListTasks(repo, nil)

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
	repo := &testutil.MockTaskRepositoryWithListError{
		MockTaskRepository: testutil.NewMockTaskRepository(),
		ListErr:            errors.New("database error"),
	}
	uc := NewListTasks(repo, nil)

	// Execute
	_, err := uc.Execute(context.Background(), ListTasksInput{})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestListTasks_Execute_PreservesTaskData(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	now := time.Now()
	parentID := 1
	repo.Tasks[1] = &domain.Task{
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
	repo.Tasks[2] = &domain.Task{
		ID:         2,
		Title:      "Child Task",
		ParentID:   &parentID,
		Status:     domain.StatusTodo,
		Created:    now,
		BaseBranch: "main",
		Labels:     []string{"bug"},
	}

	uc := NewListTasks(repo, nil)

	// Execute with IncludeTerminal = true to get all tasks
	out, err := uc.Execute(context.Background(), ListTasksInput{
		IncludeTerminal: true,
	})

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
