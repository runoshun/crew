package cli

import (
	"bytes"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
)

// mockTaskRepository is a test double for domain.TaskRepository.
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

// mockStoreInitializer is a test double for domain.StoreInitializer.
type mockStoreInitializer struct{}

func (m *mockStoreInitializer) Initialize() error {
	return nil
}

// newTestContainer creates an app.Container with mock dependencies.
func newTestContainer(repo *mockTaskRepository) *app.Container {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return app.NewWithDeps(
		app.Config{},
		repo,
		&mockStoreInitializer{},
		&mockClock{now: time.Now()},
		logger,
	)
}

// mockClock is a test double for domain.Clock.
type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

func TestPrintTaskList_Empty(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	printTaskList(&buf, []*domain.Task{}, clock)

	// Should only have header
	expected := "ID   PARENT   STATUS   AGENT   LABELS   TITLE\n"
	assert.Equal(t, expected, buf.String())
}

func TestPrintTaskList_SingleTask(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	tasks := []*domain.Task{
		{
			ID:     1,
			Title:  "Test task",
			Status: domain.StatusTodo,
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "PARENT")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "AGENT")
	assert.Contains(t, output, "LABELS")
	assert.Contains(t, output, "TITLE")
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "-") // PARENT is nil
	assert.Contains(t, output, "todo")
	assert.Contains(t, output, "Test task")
}

func TestPrintTaskList_WithParent(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	parentID := 1
	tasks := []*domain.Task{
		{
			ID:       2,
			ParentID: &parentID,
			Title:    "Child task",
			Status:   domain.StatusTodo,
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "2")
	assert.Contains(t, output, "1") // Parent ID
	assert.Contains(t, output, "Child task")
}

func TestPrintTaskList_WithAgent(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	tasks := []*domain.Task{
		{
			ID:     1,
			Title:  "Task with agent",
			Status: domain.StatusInProgress,
			Agent:  "claude",
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "claude")
}

func TestPrintTaskList_WithLabels(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	tasks := []*domain.Task{
		{
			ID:     1,
			Title:  "Task with labels",
			Status: domain.StatusTodo,
			Labels: []string{"bug", "urgent"},
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "[bug,urgent]")
}

func TestPrintTaskList_InProgressWithElapsed(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	started := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC) // 1 hour ago

	var buf bytes.Buffer
	clock := &mockClock{now: now}

	tasks := []*domain.Task{
		{
			ID:      1,
			Title:   "In progress task",
			Status:  domain.StatusInProgress,
			Started: started,
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "in_progress (1h)")
}

func TestPrintTaskList_MultipleTasks(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	parentID := 1
	tasks := []*domain.Task{
		{
			ID:     1,
			Title:  "Parent task",
			Status: domain.StatusInProgress,
			Agent:  "claude",
			Labels: []string{"feature"},
		},
		{
			ID:       2,
			ParentID: &parentID,
			Title:    "Child task",
			Status:   domain.StatusTodo,
		},
		{
			ID:     3,
			Title:  "Done task",
			Status: domain.StatusDone,
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	// Verify all tasks are present
	assert.Contains(t, output, "Parent task")
	assert.Contains(t, output, "Child task")
	assert.Contains(t, output, "Done task")
	// Verify statuses
	assert.Contains(t, output, "in_progress")
	assert.Contains(t, output, "todo")
	assert.Contains(t, output, "done")
}

func TestNewEditCommand_UpdateTitle(t *testing.T) {
	// Setup mock repository
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Original title",
		Status: domain.StatusTodo,
	}

	// Create container with mock
	container := newTestContainer(repo)

	// Create command
	cmd := newEditCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1", "--title", "Updated title"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Updated task #1")
	assert.Contains(t, buf.String(), "Updated title")

	// Verify task was updated
	task := repo.tasks[1]
	assert.Equal(t, "Updated title", task.Title)
}

func TestNewEditCommand_UpdateDescription(t *testing.T) {
	// Setup mock repository
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Test task",
		Description: "Old description",
		Status:      domain.StatusTodo,
	}

	// Create container with mock
	container := newTestContainer(repo)

	// Create command
	cmd := newEditCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1", "--desc", "New description"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Updated task #1")

	// Verify description was updated
	task := repo.tasks[1]
	assert.Equal(t, "New description", task.Description)
}

func TestNewEditCommand_AddLabels(t *testing.T) {
	// Setup mock repository
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Labels: []string{"existing"},
		Status: domain.StatusTodo,
	}

	// Create container with mock
	container := newTestContainer(repo)

	// Create command
	cmd := newEditCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1", "--add-label", "new", "--add-label", "urgent"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Updated task #1")

	// Verify labels were added
	task := repo.tasks[1]
	assert.Contains(t, task.Labels, "existing")
	assert.Contains(t, task.Labels, "new")
	assert.Contains(t, task.Labels, "urgent")
}

func TestNewEditCommand_RemoveLabels(t *testing.T) {
	// Setup mock repository
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Labels: []string{"keep", "remove-me"},
		Status: domain.StatusTodo,
	}

	// Create container with mock
	container := newTestContainer(repo)

	// Create command
	cmd := newEditCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1", "--rm-label", "remove-me"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Updated task #1")

	// Verify label was removed
	task := repo.tasks[1]
	assert.Contains(t, task.Labels, "keep")
	assert.NotContains(t, task.Labels, "remove-me")
}

func TestNewEditCommand_MultipleUpdates(t *testing.T) {
	// Setup mock repository
	repo := newMockTaskRepository()
	repo.tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Original",
		Description: "Old desc",
		Labels:      []string{"old"},
		Status:      domain.StatusTodo,
	}

	// Create container with mock
	container := newTestContainer(repo)

	// Create command
	cmd := newEditCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1", "--title", "New Title", "--desc", "New desc", "--add-label", "new", "--rm-label", "old"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)

	// Verify all fields were updated
	task := repo.tasks[1]
	assert.Equal(t, "New Title", task.Title)
	assert.Equal(t, "New desc", task.Description)
	assert.Contains(t, task.Labels, "new")
	assert.NotContains(t, task.Labels, "old")
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"seconds", 30 * time.Second, "30s"},
		{"just under minute", 59 * time.Second, "59s"},
		{"one minute", 1 * time.Minute, "1m"},
		{"minutes", 5 * time.Minute, "5m"},
		{"just under hour", 59 * time.Minute, "59m"},
		{"one hour", 1 * time.Hour, "1h"},
		{"hours", 5 * time.Hour, "5h"},
		{"just under day", 23 * time.Hour, "23h"},
		{"one day", 24 * time.Hour, "1d"},
		{"days", 3 * 24 * time.Hour, "3d"},
		{"week", 7 * 24 * time.Hour, "7d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
