package shared

import (
	"errors"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTaskMockRepo is a test double for domain.TaskRepository focused on Get.
type getTaskMockRepo struct {
	tasks  map[int]*domain.Task
	getErr error
}

func newGetTaskMockRepo() *getTaskMockRepo {
	return &getTaskMockRepo{
		tasks: make(map[int]*domain.Task),
	}
}

func (m *getTaskMockRepo) Get(id int) (*domain.Task, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	t, ok := m.tasks[id]
	if !ok {
		return nil, nil
	}
	// Return a copy to avoid mutation
	copy := *t
	return &copy, nil
}

// Unused methods - satisfy interface
func (m *getTaskMockRepo) Save(_ *domain.Task) error                          { return nil }
func (m *getTaskMockRepo) List(_ domain.TaskFilter) ([]*domain.Task, error)   { return nil, nil }
func (m *getTaskMockRepo) GetChildren(_ int) ([]*domain.Task, error)          { return nil, nil }
func (m *getTaskMockRepo) Delete(_ int) error                                 { return nil }
func (m *getTaskMockRepo) NextID() (int, error)                               { return 0, nil }
func (m *getTaskMockRepo) AddComment(_ int, _ domain.Comment) error           { return nil }
func (m *getTaskMockRepo) GetComments(_ int) ([]domain.Comment, error)        { return nil, nil }
func (m *getTaskMockRepo) UpdateComment(_ int, _ int, _ domain.Comment) error { return nil }
func (m *getTaskMockRepo) SaveTaskWithComments(_ *domain.Task, _ []domain.Comment) error {
	return nil
}
func (m *getTaskMockRepo) SaveSnapshot(_ string) error                           { return nil }
func (m *getTaskMockRepo) RestoreSnapshot(_ string) error                        { return nil }
func (m *getTaskMockRepo) ListSnapshots(_ string) ([]domain.SnapshotInfo, error) { return nil, nil }
func (m *getTaskMockRepo) SyncSnapshot() error                                   { return nil }
func (m *getTaskMockRepo) PruneSnapshots(_ int) error                            { return nil }
func (m *getTaskMockRepo) Push() error                                           { return nil }
func (m *getTaskMockRepo) Fetch(_ string) error                                  { return nil }
func (m *getTaskMockRepo) ListNamespaces() ([]string, error)                     { return nil, nil }

func TestGetTask_Success(t *testing.T) {
	repo := newGetTaskMockRepo()
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}

	task, err := GetTask(repo, 1)

	require.NoError(t, err)
	assert.Equal(t, 1, task.ID)
	assert.Equal(t, "Test task", task.Title)
}

func TestGetTask_NotFound(t *testing.T) {
	repo := newGetTaskMockRepo() // Empty

	task, err := GetTask(repo, 999)

	assert.Nil(t, task)
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestGetTask_RepositoryError(t *testing.T) {
	repo := newGetTaskMockRepo()
	repo.getErr = errors.New("database connection failed")

	task, err := GetTask(repo, 1)

	assert.Nil(t, task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
	assert.Contains(t, err.Error(), "database connection failed")
}
