package shared

import (
	"context"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTaskRepo is a test double for domain.TaskRepository.
type mockTaskRepo struct {
	tasks    map[int]*domain.Task
	comments map[int][]domain.Comment
	saveErr  error
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{
		tasks:    make(map[int]*domain.Task),
		comments: make(map[int][]domain.Comment),
	}
}

func (m *mockTaskRepo) Get(id int) (*domain.Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, nil
	}
	// Return a copy to avoid mutation
	copy := *t
	return &copy, nil
}

func (m *mockTaskRepo) GetComments(taskID int) ([]domain.Comment, error) {
	return m.comments[taskID], nil
}

func (m *mockTaskRepo) SaveTaskWithComments(task *domain.Task, comments []domain.Comment) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.tasks[task.ID] = task
	m.comments[task.ID] = comments
	return nil
}

// Unused methods
func (m *mockTaskRepo) Save(_ *domain.Task) error                             { return nil }
func (m *mockTaskRepo) List(_ domain.TaskFilter) ([]*domain.Task, error)      { return nil, nil }
func (m *mockTaskRepo) GetChildren(_ int) ([]*domain.Task, error)             { return nil, nil }
func (m *mockTaskRepo) Delete(_ int) error                                    { return nil }
func (m *mockTaskRepo) NextID() (int, error)                                  { return 0, nil }
func (m *mockTaskRepo) AddComment(_ int, _ domain.Comment) error              { return nil }
func (m *mockTaskRepo) UpdateComment(_ int, _ int, _ domain.Comment) error    { return nil }
func (m *mockTaskRepo) SaveSnapshot(_ string) error                           { return nil }
func (m *mockTaskRepo) RestoreSnapshot(_ string) error                        { return nil }
func (m *mockTaskRepo) ListSnapshots(_ string) ([]domain.SnapshotInfo, error) { return nil, nil }
func (m *mockTaskRepo) SyncSnapshot() error                                   { return nil }
func (m *mockTaskRepo) PruneSnapshots(_ int) error                            { return nil }
func (m *mockTaskRepo) Push() error                                           { return nil }
func (m *mockTaskRepo) Fetch(_ string) error                                  { return nil }
func (m *mockTaskRepo) ListNamespaces() ([]string, error)                     { return nil, nil }

// mockSessionManager is a test double for domain.SessionManager.
type mockSessionManager struct {
	isRunning bool
	sentKeys  string
}

func (m *mockSessionManager) IsRunning(_ string) (bool, error) {
	return m.isRunning, nil
}

func (m *mockSessionManager) Send(_ string, keys string) error {
	m.sentKeys += keys
	return nil
}

// Unused methods
func (m *mockSessionManager) Start(_ context.Context, _ domain.StartSessionOptions) error {
	return nil
}
func (m *mockSessionManager) Stop(_ string) error                          { return nil }
func (m *mockSessionManager) Attach(_ string) error                        { return nil }
func (m *mockSessionManager) Peek(_ string, _ int, _ bool) (string, error) { return "", nil }
func (m *mockSessionManager) GetPaneProcesses(_ string) ([]domain.ProcessInfo, error) {
	return nil, nil
}

// mockGit is a test double for domain.Git.
type mockGit struct {
	conflictFiles []string
	conflictErr   error
}

func (m *mockGit) GetMergeConflictFiles(_, _ string) ([]string, error) {
	return m.conflictFiles, m.conflictErr
}

// Unused methods
func (m *mockGit) CurrentBranch() (string, error)               { return "", nil }
func (m *mockGit) BranchExists(_ string) (bool, error)          { return true, nil }
func (m *mockGit) HasUncommittedChanges(_ string) (bool, error) { return false, nil }
func (m *mockGit) HasMergeConflict(_, _ string) (bool, error)   { return false, nil }
func (m *mockGit) Merge(_ string, _ bool) error                 { return nil }
func (m *mockGit) DeleteBranch(_ string, _ bool) error          { return nil }
func (m *mockGit) ListBranches() ([]string, error)              { return nil, nil }
func (m *mockGit) GetDefaultBranch() (string, error)            { return "main", nil }

// mockClock is a test double for domain.Clock.
type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

func TestConflictHandler_CheckAndHandle_NoConflict(t *testing.T) {
	// Setup
	tasks := newMockTaskRepo()
	tasks.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := &mockSessionManager{}
	git := &mockGit{conflictFiles: nil} // No conflicts
	clock := &mockClock{now: time.Now()}

	handler := NewConflictHandler(tasks, sessions, git, clock)

	// Execute
	err := handler.CheckAndHandle(ConflictCheckInput{
		TaskID:     1,
		Branch:     "crew-1",
		BaseBranch: "main",
	})

	// Assert
	assert.NoError(t, err)
	// Task status should not change
	task, _ := tasks.Get(1)
	assert.Equal(t, domain.StatusInProgress, task.Status)
}

func TestConflictHandler_CheckAndHandle_WithConflict(t *testing.T) {
	// Setup
	tasks := newMockTaskRepo()
	tasks.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusReviewing,
	}
	sessions := &mockSessionManager{isRunning: true}
	git := &mockGit{conflictFiles: []string{"file1.txt", "file2.txt"}}
	clock := &mockClock{now: time.Now()}

	handler := NewConflictHandler(tasks, sessions, git, clock)

	// Execute
	err := handler.CheckAndHandle(ConflictCheckInput{
		TaskID:     1,
		Branch:     "crew-1",
		BaseBranch: "main",
	})

	// Assert
	require.ErrorIs(t, err, domain.ErrMergeConflict)

	// Task status should change to in_progress
	savedTask := tasks.tasks[1]
	assert.Equal(t, domain.StatusInProgress, savedTask.Status)

	// Comment should be added
	comments := tasks.comments[1]
	require.Len(t, comments, 1)
	assert.Contains(t, comments[0].Text, "Merge conflict detected")
	assert.Contains(t, comments[0].Text, "file1.txt")
	assert.Contains(t, comments[0].Text, "file2.txt")

	// Session should receive notification
	assert.Contains(t, sessions.sentKeys, "Merge conflict detected")
}

func TestConflictHandler_CheckAndHandle_SessionNotRunning(t *testing.T) {
	// Setup
	tasks := newMockTaskRepo()
	tasks.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusReviewing,
	}
	sessions := &mockSessionManager{isRunning: false}
	git := &mockGit{conflictFiles: []string{"conflict.txt"}}
	clock := &mockClock{now: time.Now()}

	handler := NewConflictHandler(tasks, sessions, git, clock)

	// Execute
	err := handler.CheckAndHandle(ConflictCheckInput{
		TaskID:     1,
		Branch:     "crew-1",
		BaseBranch: "main",
	})

	// Assert
	require.ErrorIs(t, err, domain.ErrMergeConflict)

	// Task status should still change
	savedTask := tasks.tasks[1]
	assert.Equal(t, domain.StatusInProgress, savedTask.Status)

	// Comment should still be added
	comments := tasks.comments[1]
	require.Len(t, comments, 1)

	// No keys sent since session is not running
	assert.Empty(t, sessions.sentKeys)
}

func TestConflictHandler_CheckAndHandle_TaskNotFound(t *testing.T) {
	// Setup
	tasks := newMockTaskRepo() // Empty
	sessions := &mockSessionManager{}
	git := &mockGit{conflictFiles: []string{"conflict.txt"}}
	clock := &mockClock{now: time.Now()}

	handler := NewConflictHandler(tasks, sessions, git, clock)

	// Execute
	err := handler.CheckAndHandle(ConflictCheckInput{
		TaskID:     999,
		Branch:     "crew-999",
		BaseBranch: "main",
	})

	// Assert
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestBuildConflictMessage(t *testing.T) {
	files := []string{"file1.txt", "dir/file2.txt"}
	msg := buildConflictMessage(files)

	assert.Contains(t, msg, "Merge conflict detected")
	assert.Contains(t, msg, "file1.txt")
	assert.Contains(t, msg, "dir/file2.txt")
	assert.Contains(t, msg, "git merge main")
	assert.Contains(t, msg, "crew complete")
}
