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

type mockCommentNamespaceLister struct {
	*testutil.MockTaskRepository
	listAllCalled bool
	listAllTasks  []*domain.Task
	listAllErr    error
}

func (m *mockCommentNamespaceLister) ListAll(filter domain.TaskFilter) ([]*domain.Task, error) {
	m.listAllCalled = true
	return m.listAllTasks, m.listAllErr
}

func TestListComments_Execute_FilterByTypeAndTags(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Task A", Status: domain.StatusTodo}
	repo.Tasks[2] = &domain.Task{ID: 2, Title: "Task B", Status: domain.StatusTodo}

	base := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	repo.Comments[1] = []domain.Comment{
		{Text: "A1", Time: base, Type: domain.CommentTypeFriction, Tags: []string{"docs", "testing"}},
		{Text: "A2", Time: base.Add(time.Minute), Type: domain.CommentTypeReport, Tags: []string{"docs"}},
	}
	repo.Comments[2] = []domain.Comment{
		{Text: "B1", Time: base.Add(2 * time.Minute), Type: domain.CommentTypeFriction, Tags: []string{"docs"}},
	}

	uc := NewListComments(repo)
	out, err := uc.Execute(context.Background(), ListCommentsInput{
		Type: domain.CommentTypeFriction,
		Tags: []string{"docs"},
	})

	require.NoError(t, err)
	require.Len(t, out.Comments, 2)
	assert.Equal(t, "B1", out.Comments[0].Comment.Text)
	assert.Equal(t, 2, out.Comments[0].Task.ID)
	assert.Equal(t, "A1", out.Comments[1].Comment.Text)
	assert.Equal(t, 1, out.Comments[1].Task.ID)
}

func TestListComments_Execute_AllNamespaces(t *testing.T) {
	repo := &mockCommentNamespaceLister{MockTaskRepository: testutil.NewMockTaskRepository()}
	repo.listAllTasks = []*domain.Task{
		{ID: 1, Title: "Alpha", Status: domain.StatusTodo, Namespace: "alpha"},
		{ID: 2, Title: "Beta", Status: domain.StatusTodo, Namespace: "beta"},
	}
	repo.Comments[1] = []domain.Comment{{Text: "Alpha comment", Time: time.Now()}}
	repo.Comments[2] = []domain.Comment{{Text: "Beta comment", Time: time.Now()}}

	uc := NewListComments(repo)
	out, err := uc.Execute(context.Background(), ListCommentsInput{AllNamespaces: true})

	require.NoError(t, err)
	assert.True(t, repo.listAllCalled)
	require.Len(t, out.Comments, 2)
}

func TestListComments_Execute_InvalidType(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	uc := NewListComments(repo)

	_, err := uc.Execute(context.Background(), ListCommentsInput{Type: domain.CommentType("invalid")})
	assert.ErrorIs(t, err, domain.ErrInvalidCommentType)
}
