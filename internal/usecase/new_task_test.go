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

func TestNewTask_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewNewTask(repo, clock)

	// Execute
	out, err := uc.Execute(context.Background(), NewTaskInput{
		Title:       "Test task",
		Description: "Test description",
		Labels:      []string{"bug", "urgent"},
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, out.TaskID)

	// Verify saved task
	task := repo.Tasks[1]
	require.NotNil(t, task)
	assert.Equal(t, 1, task.ID)
	assert.Equal(t, "Test task", task.Title)
	assert.Equal(t, "Test description", task.Description)
	assert.Equal(t, domain.StatusTodo, task.Status)
	assert.Nil(t, task.ParentID)
	assert.Equal(t, clock.NowTime, task.Created)
	assert.Equal(t, 0, task.Issue)
	assert.Equal(t, []string{"bug", "urgent"}, task.Labels)
	assert.Equal(t, "main", task.BaseBranch)
}

func TestNewTask_Execute_WithParent(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewNewTask(repo, clock)

	// Create parent task first
	parentID := 1
	repo.Tasks[parentID] = &domain.Task{
		ID:     parentID,
		Title:  "Parent task",
		Status: domain.StatusTodo,
	}
	repo.NextIDN = 2

	// Execute
	out, err := uc.Execute(context.Background(), NewTaskInput{
		Title:    "Child task",
		ParentID: &parentID,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, out.TaskID)

	// Verify saved task
	task := repo.Tasks[2]
	require.NotNil(t, task)
	assert.NotNil(t, task.ParentID)
	assert.Equal(t, parentID, *task.ParentID)
}

func TestNewTask_Execute_WithIssue(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewNewTask(repo, clock)

	// Execute
	out, err := uc.Execute(context.Background(), NewTaskInput{
		Title: "Fix issue",
		Issue: 123,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, out.TaskID)

	// Verify issue is set
	task := repo.Tasks[1]
	require.NotNil(t, task)
	assert.Equal(t, 123, task.Issue)
}

func TestNewTask_Execute_EmptyTitle(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewNewTask(repo, clock)

	// Execute
	_, err := uc.Execute(context.Background(), NewTaskInput{
		Title: "",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrEmptyTitle)
}

func TestNewTask_Execute_ParentNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewNewTask(repo, clock)

	// Execute with non-existent parent
	nonExistentParent := 999
	_, err := uc.Execute(context.Background(), NewTaskInput{
		Title:    "Child task",
		ParentID: &nonExistentParent,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrParentNotFound)
}

func TestNewTask_Execute_SaveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.SaveErr = assert.AnError
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewNewTask(repo, clock)

	// Execute
	_, err := uc.Execute(context.Background(), NewTaskInput{
		Title: "Test task",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")
}

func TestNewTask_Execute_GetParentError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewNewTask(repo, clock)

	// Execute with parent that causes Get error
	parentID := 1
	_, err := uc.Execute(context.Background(), NewTaskInput{
		Title:    "Child task",
		ParentID: &parentID,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get parent task")
}

func TestNewTask_Execute_NextIDError(t *testing.T) {
	// Setup
	repo := &testutil.MockTaskRepositoryWithNextIDError{
		MockTaskRepository: testutil.NewMockTaskRepository(),
		NextIDErr:          assert.AnError,
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewNewTask(repo, clock)

	// Execute
	_, err := uc.Execute(context.Background(), NewTaskInput{
		Title: "Test task",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generate task ID")
}
