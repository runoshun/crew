package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClock is a test double for domain.Clock.
type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

// mockTaskRepository is a test double for domain.TaskRepository.
// Fields are ordered to minimize memory padding.
type mockTaskRepository struct {
	tasks    map[int]*domain.Task
	comments map[int][]domain.Comment
	saveErr  error
	getErr   error
	nextID   int
}

func newMockTaskRepository() *mockTaskRepository {
	return &mockTaskRepository{
		tasks:    make(map[int]*domain.Task),
		nextID:   1,
		comments: make(map[int][]domain.Comment),
	}
}

func (m *mockTaskRepository) Get(id int) (*domain.Task, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	task, ok := m.tasks[id]
	if !ok {
		return nil, nil
	}
	return task, nil
}

func (m *mockTaskRepository) List(_ domain.TaskFilter) ([]*domain.Task, error) {
	tasks := make([]*domain.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (m *mockTaskRepository) GetChildren(parentID int) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for _, t := range m.tasks {
		if t.ParentID != nil && *t.ParentID == parentID {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

func (m *mockTaskRepository) Save(task *domain.Task) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepository) Delete(id int) error {
	delete(m.tasks, id)
	return nil
}

func (m *mockTaskRepository) NextID() (int, error) {
	id := m.nextID
	m.nextID++
	return id, nil
}

func (m *mockTaskRepository) GetComments(taskID int) ([]domain.Comment, error) {
	return m.comments[taskID], nil
}

func (m *mockTaskRepository) AddComment(taskID int, comment domain.Comment) error {
	m.comments[taskID] = append(m.comments[taskID], comment)
	return nil
}

func TestNewTask_Execute_Success(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
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
	task := repo.tasks[1]
	require.NotNil(t, task)
	assert.Equal(t, 1, task.ID)
	assert.Equal(t, "Test task", task.Title)
	assert.Equal(t, "Test description", task.Description)
	assert.Equal(t, domain.StatusTodo, task.Status)
	assert.Nil(t, task.ParentID)
	assert.Equal(t, clock.now, task.Created)
	assert.Equal(t, 0, task.Issue)
	assert.Equal(t, []string{"bug", "urgent"}, task.Labels)
	assert.Equal(t, "main", task.BaseBranch)
}

func TestNewTask_Execute_WithParent(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewNewTask(repo, clock)

	// Create parent task first
	parentID := 1
	repo.tasks[parentID] = &domain.Task{
		ID:     parentID,
		Title:  "Parent task",
		Status: domain.StatusTodo,
	}
	repo.nextID = 2

	// Execute
	out, err := uc.Execute(context.Background(), NewTaskInput{
		Title:    "Child task",
		ParentID: &parentID,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, out.TaskID)

	// Verify saved task
	task := repo.tasks[2]
	require.NotNil(t, task)
	assert.NotNil(t, task.ParentID)
	assert.Equal(t, parentID, *task.ParentID)
}

func TestNewTask_Execute_WithIssue(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
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
	task := repo.tasks[1]
	require.NotNil(t, task)
	assert.Equal(t, 123, task.Issue)
}

func TestNewTask_Execute_EmptyTitle(t *testing.T) {
	// Setup
	repo := newMockTaskRepository()
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
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
	repo := newMockTaskRepository()
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
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
	repo := newMockTaskRepository()
	repo.saveErr = assert.AnError
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
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
	repo := newMockTaskRepository()
	repo.getErr = assert.AnError
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
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
	repo := &mockTaskRepositoryWithNextIDError{
		mockTaskRepository: newMockTaskRepository(),
		nextIDErr:          assert.AnError,
	}
	clock := &mockClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewNewTask(repo, clock)

	// Execute
	_, err := uc.Execute(context.Background(), NewTaskInput{
		Title: "Test task",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generate task ID")
}

// mockTaskRepositoryWithNextIDError extends mockTaskRepository to return error on NextID.
type mockTaskRepositoryWithNextIDError struct {
	*mockTaskRepository
	nextIDErr error
}

func (m *mockTaskRepositoryWithNextIDError) NextID() (int, error) {
	if m.nextIDErr != nil {
		return 0, m.nextIDErr
	}
	return m.mockTaskRepository.NextID()
}
