package gitstore

import (
	"os"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

func setupTestRepo(t *testing.T) *git.Repository {
	t.Helper()

	dir, err := os.MkdirTemp("", "gitstore-test-*")
	require.NoError(t, err)

	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	// Create an initial commit (required for some git operations)
	wt, err := repo.Worktree()
	require.NoError(t, err)

	// Create a dummy file
	dummyFile := dir + "/README.md"
	err = os.WriteFile(dummyFile, []byte("# Test"), 0o644)
	require.NoError(t, err)

	_, err = wt.Add("README.md")
	require.NoError(t, err)

	_, err = wt.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return repo
}

func TestStore_Initialize(t *testing.T) {
	repo := setupTestRepo(t)
	store := NewWithRepo(repo, "crew-test")

	err := store.Initialize()
	require.NoError(t, err)

	// Second call should be idempotent
	err = store.Initialize()
	require.NoError(t, err)
}

func TestStore_NextID(t *testing.T) {
	repo := setupTestRepo(t)
	store := NewWithRepo(repo, "crew-test")

	id1, err := store.NextID()
	require.NoError(t, err)
	assert.Equal(t, 1, id1)

	id2, err := store.NextID()
	require.NoError(t, err)
	assert.Equal(t, 2, id2)

	id3, err := store.NextID()
	require.NoError(t, err)
	assert.Equal(t, 3, id3)
}

func TestStore_SaveAndGet(t *testing.T) {
	repo := setupTestRepo(t)
	store := NewWithRepo(repo, "crew-test")

	task := &domain.Task{
		ID:          1,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      domain.StatusTodo,
		Labels:      []string{"bug", "urgent"},
	}

	// Save
	err := store.Save(task)
	require.NoError(t, err)

	// Get
	got, err := store.Get(1)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, task.ID, got.ID)
	assert.Equal(t, task.Title, got.Title)
	assert.Equal(t, task.Description, got.Description)
	assert.Equal(t, task.Status, got.Status)
	assert.Equal(t, task.Labels, got.Labels)

	// Get non-existent
	got, err = store.Get(999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestStore_List(t *testing.T) {
	repo := setupTestRepo(t)
	store := NewWithRepo(repo, "crew-test")

	// Create tasks
	tasks := []*domain.Task{
		{ID: 1, Title: "Task 1", Status: domain.StatusTodo, Labels: []string{"bug"}},
		{ID: 2, Title: "Task 2", Status: domain.StatusInProgress, Labels: []string{"feature"}},
		{ID: 3, Title: "Task 3", Status: domain.StatusTodo, Labels: []string{"bug", "urgent"}},
	}
	for _, task := range tasks {
		err := store.Save(task)
		require.NoError(t, err)
	}

	// List all
	got, err := store.List(domain.TaskFilter{})
	require.NoError(t, err)
	assert.Len(t, got, 3)

	// List with label filter
	got, err = store.List(domain.TaskFilter{Labels: []string{"bug"}})
	require.NoError(t, err)
	assert.Len(t, got, 2)

	// List with multiple labels (AND)
	got, err = store.List(domain.TaskFilter{Labels: []string{"bug", "urgent"}})
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, 3, got[0].ID)
}

func TestStore_GetChildren(t *testing.T) {
	repo := setupTestRepo(t)
	store := NewWithRepo(repo, "crew-test")

	parentID := 1
	tasks := []*domain.Task{
		{ID: 1, Title: "Parent"},
		{ID: 2, Title: "Child 1", ParentID: &parentID},
		{ID: 3, Title: "Child 2", ParentID: &parentID},
		{ID: 4, Title: "Unrelated"},
	}
	for _, task := range tasks {
		err := store.Save(task)
		require.NoError(t, err)
	}

	children, err := store.GetChildren(1)
	require.NoError(t, err)
	assert.Len(t, children, 2)
}

func TestStore_Delete(t *testing.T) {
	repo := setupTestRepo(t)
	store := NewWithRepo(repo, "crew-test")

	task := &domain.Task{ID: 1, Title: "To Delete"}
	err := store.Save(task)
	require.NoError(t, err)

	// Add comment
	err = store.AddComment(1, domain.Comment{Text: "Test comment", Time: time.Now()})
	require.NoError(t, err)

	// Verify exists
	got, err := store.Get(1)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Delete
	err = store.Delete(1)
	require.NoError(t, err)

	// Verify deleted
	got, err = store.Get(1)
	require.NoError(t, err)
	assert.Nil(t, got)

	// Verify comments deleted
	comments, err := store.GetComments(1)
	require.NoError(t, err)
	assert.Empty(t, comments)

	// Delete non-existent should not error
	err = store.Delete(999)
	require.NoError(t, err)
}

func TestStore_Comments(t *testing.T) {
	repo := setupTestRepo(t)
	store := NewWithRepo(repo, "crew-test")

	task := &domain.Task{ID: 1, Title: "Task with comments"}
	err := store.Save(task)
	require.NoError(t, err)

	// Initially no comments
	comments, err := store.GetComments(1)
	require.NoError(t, err)
	assert.Empty(t, comments)

	// Add comments
	now := time.Now()
	err = store.AddComment(1, domain.Comment{Text: "First comment", Time: now})
	require.NoError(t, err)

	err = store.AddComment(1, domain.Comment{Text: "Second comment", Time: now.Add(time.Hour)})
	require.NoError(t, err)

	// Get comments
	comments, err = store.GetComments(1)
	require.NoError(t, err)
	assert.Len(t, comments, 2)
	assert.Equal(t, "First comment", comments[0].Text)
	assert.Equal(t, "Second comment", comments[1].Text)
}

func TestStore_Update(t *testing.T) {
	repo := setupTestRepo(t)
	store := NewWithRepo(repo, "crew-test")

	task := &domain.Task{ID: 1, Title: "Original", Status: domain.StatusTodo}
	err := store.Save(task)
	require.NoError(t, err)

	// Update
	task.Title = "Updated"
	task.Status = domain.StatusInProgress
	err = store.Save(task)
	require.NoError(t, err)

	// Verify
	got, err := store.Get(1)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title)
	assert.Equal(t, domain.StatusInProgress, got.Status)
}

func TestStore_NamespaceIsolation(t *testing.T) {
	repo := setupTestRepo(t)
	store1 := NewWithRepo(repo, "crew-user1")
	store2 := NewWithRepo(repo, "crew-user2")

	// Save to store1
	task := &domain.Task{ID: 1, Title: "User1 Task"}
	err := store1.Save(task)
	require.NoError(t, err)

	// Should not be visible in store2
	got, err := store2.Get(1)
	require.NoError(t, err)
	assert.Nil(t, got)

	// Save to store2
	task2 := &domain.Task{ID: 1, Title: "User2 Task"}
	err = store2.Save(task2)
	require.NoError(t, err)

	// Each store sees its own task
	got1, err := store1.Get(1)
	require.NoError(t, err)
	require.NotNil(t, got1)
	assert.Equal(t, "User1 Task", got1.Title)

	got2, err := store2.Get(1)
	require.NoError(t, err)
	require.NotNil(t, got2)
	assert.Equal(t, "User2 Task", got2.Title)
}
