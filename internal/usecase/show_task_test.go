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

func TestShowTask_Execute_Success(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Test task",
		Description: "Test description",
		Status:      domain.StatusTodo,
		Created:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Labels:      []string{"bug"},
		BaseBranch:  "main",
	}
	uc := NewShowTask(repo)

	// Execute
	out, err := uc.Execute(context.Background(), ShowTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, out.Task.ID)
	assert.Equal(t, "Test task", out.Task.Title)
	assert.Equal(t, "Test description", out.Task.Description)
	assert.Equal(t, domain.StatusTodo, out.Task.Status)
	assert.Empty(t, out.Children)
	assert.Empty(t, out.Comments)
}

func TestShowTask_Execute_WithChildren(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	parentID := 1
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Parent task",
		Status: domain.StatusInProgress,
	}
	repo.tasks[2] = &domain.Task{
		ID:       2,
		ParentID: &parentID,
		Title:    "Child task 1",
		Status:   domain.StatusTodo,
	}
	repo.tasks[3] = &domain.Task{
		ID:       3,
		ParentID: &parentID,
		Title:    "Child task 2",
		Status:   domain.StatusDone,
	}
	uc := NewShowTask(repo)

	// Execute
	out, err := uc.Execute(context.Background(), ShowTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	assert.Len(t, out.Children, 2)
}

func TestShowTask_Execute_WithComments(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	repo.comments[1] = []domain.Comment{
		{Text: "First comment", Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Text: "Second comment", Time: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
	}
	uc := NewShowTask(repo)

	// Execute
	out, err := uc.Execute(context.Background(), ShowTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	assert.Len(t, out.Comments, 2)
	assert.Equal(t, "First comment", out.Comments[0].Text)
	assert.Equal(t, "Second comment", out.Comments[1].Text)
}

func TestShowTask_Execute_FullDetails(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	parentID := 10
	repo.tasks[1] = &domain.Task{
		ID:          1,
		ParentID:    &parentID,
		Title:       "OAuth implementation",
		Description: "Implement OAuth2.0 support",
		Status:      domain.StatusInProgress,
		Created:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Started:     time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Agent:       "claude",
		Session:     "crew-1",
		Issue:       123,
		PR:          456,
		BaseBranch:  "main",
		Labels:      []string{"feature", "oauth"},
	}
	uc := NewShowTask(repo)

	// Execute
	out, err := uc.Execute(context.Background(), ShowTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, out.Task.ID)
	assert.NotNil(t, out.Task.ParentID)
	assert.Equal(t, 10, *out.Task.ParentID)
	assert.Equal(t, "OAuth implementation", out.Task.Title)
	assert.Equal(t, "Implement OAuth2.0 support", out.Task.Description)
	assert.Equal(t, domain.StatusInProgress, out.Task.Status)
	assert.Equal(t, "claude", out.Task.Agent)
	assert.Equal(t, "crew-1", out.Task.Session)
	assert.Equal(t, 123, out.Task.Issue)
	assert.Equal(t, 456, out.Task.PR)
	assert.Equal(t, []string{"feature", "oauth"}, out.Task.Labels)
}

func TestShowTask_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	uc := NewShowTask(repo)

	// Execute
	_, err := uc.Execute(context.Background(), ShowTaskInput{
		TaskID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestShowTask_Execute_GetError(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	repo.getErr = errors.New("database error")
	uc := NewShowTask(repo)

	// Execute
	_, err := uc.Execute(context.Background(), ShowTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestShowTask_Execute_GetChildrenError(t *testing.T) {
	// Setup
	repo := &mockTaskRepositoryWithChildrenError{
		mockTaskRepository: newMockTaskRepository(),
		childrenErr:        errors.New("children error"),
	}
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	uc := NewShowTask(repo)

	// Execute
	_, err := uc.Execute(context.Background(), ShowTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get children")
}

func TestShowTask_Execute_GetCommentsError(t *testing.T) {
	// Setup
	repo := &mockTaskRepositoryWithCommentsError{
		mockTaskRepository: newMockTaskRepository(),
		commentsErr:        errors.New("comments error"),
	}
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	uc := NewShowTask(repo)

	// Execute
	_, err := uc.Execute(context.Background(), ShowTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get comments")
}

// mockTaskRepositoryWithChildrenError extends mockTaskRepository to return error on GetChildren.
type mockTaskRepositoryWithChildrenError struct {
	*mockTaskRepository
	childrenErr error
}

func (m *mockTaskRepositoryWithChildrenError) GetChildren(_ int) ([]*domain.Task, error) {
	if m.childrenErr != nil {
		return nil, m.childrenErr
	}
	return m.mockTaskRepository.GetChildren(0)
}

// mockTaskRepositoryWithCommentsError extends mockTaskRepository to return error on GetComments.
type mockTaskRepositoryWithCommentsError struct {
	*mockTaskRepository
	commentsErr error
}

func (m *mockTaskRepositoryWithCommentsError) GetComments(_ int) ([]domain.Comment, error) {
	if m.commentsErr != nil {
		return nil, m.commentsErr
	}
	return m.mockTaskRepository.GetComments(0)
}
