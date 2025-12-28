// Package testutil provides shared test utilities and mock implementations.
package testutil

import (
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// MockClock is a test double for domain.Clock.
type MockClock struct {
	NowTime time.Time
}

// Now returns the configured time.
func (m *MockClock) Now() time.Time {
	return m.NowTime
}

// MockTaskRepository is a test double for domain.TaskRepository.
// Fields are ordered to minimize memory padding.
type MockTaskRepository struct {
	Tasks    map[int]*domain.Task
	Comments map[int][]domain.Comment
	SaveErr  error
	GetErr   error
	NextIDN  int
}

// NewMockTaskRepository creates a new MockTaskRepository with initialized maps.
func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{
		Tasks:    make(map[int]*domain.Task),
		NextIDN:  1,
		Comments: make(map[int][]domain.Comment),
	}
}

// Get retrieves a task by ID.
func (m *MockTaskRepository) Get(id int) (*domain.Task, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	task, ok := m.Tasks[id]
	if !ok {
		return nil, nil
	}
	return task, nil
}

// List returns all tasks (filtering not implemented in mock).
func (m *MockTaskRepository) List(_ domain.TaskFilter) ([]*domain.Task, error) {
	tasks := make([]*domain.Task, 0, len(m.Tasks))
	for _, t := range m.Tasks {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// GetChildren returns children of a parent task.
func (m *MockTaskRepository) GetChildren(parentID int) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for _, t := range m.Tasks {
		if t.ParentID != nil && *t.ParentID == parentID {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

// Save saves a task.
func (m *MockTaskRepository) Save(task *domain.Task) error {
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.Tasks[task.ID] = task
	return nil
}

// Delete removes a task by ID.
func (m *MockTaskRepository) Delete(id int) error {
	delete(m.Tasks, id)
	return nil
}

// NextID returns the next available task ID.
func (m *MockTaskRepository) NextID() (int, error) {
	id := m.NextIDN
	m.NextIDN++
	return id, nil
}

// GetComments returns comments for a task.
func (m *MockTaskRepository) GetComments(taskID int) ([]domain.Comment, error) {
	return m.Comments[taskID], nil
}

// AddComment adds a comment to a task.
func (m *MockTaskRepository) AddComment(taskID int, comment domain.Comment) error {
	m.Comments[taskID] = append(m.Comments[taskID], comment)
	return nil
}

// MockStoreInitializer is a test double for domain.StoreInitializer.
type MockStoreInitializer struct{}

// Initialize is a no-op for testing.
func (m *MockStoreInitializer) Initialize() error {
	return nil
}

// MockTaskRepositoryWithNextIDError extends MockTaskRepository to return error on NextID.
type MockTaskRepositoryWithNextIDError struct {
	*MockTaskRepository
	NextIDErr error
}

// NextID returns an error if configured.
func (m *MockTaskRepositoryWithNextIDError) NextID() (int, error) {
	if m.NextIDErr != nil {
		return 0, m.NextIDErr
	}
	return m.MockTaskRepository.NextID()
}

// MockTaskRepositoryWithListError extends MockTaskRepository to return error on List.
type MockTaskRepositoryWithListError struct {
	*MockTaskRepository
	ListErr error
}

// List returns an error if configured.
func (m *MockTaskRepositoryWithListError) List(_ domain.TaskFilter) ([]*domain.Task, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	return m.MockTaskRepository.List(domain.TaskFilter{})
}

// MockTaskRepositoryWithChildrenError extends MockTaskRepository to return error on GetChildren.
type MockTaskRepositoryWithChildrenError struct {
	*MockTaskRepository
	ChildrenErr error
}

// GetChildren returns an error if configured.
func (m *MockTaskRepositoryWithChildrenError) GetChildren(_ int) ([]*domain.Task, error) {
	if m.ChildrenErr != nil {
		return nil, m.ChildrenErr
	}
	return m.MockTaskRepository.GetChildren(0)
}

// MockTaskRepositoryWithCommentsError extends MockTaskRepository to return error on GetComments.
type MockTaskRepositoryWithCommentsError struct {
	*MockTaskRepository
	CommentsErr error
}

// GetComments returns an error if configured.
func (m *MockTaskRepositoryWithCommentsError) GetComments(_ int) ([]domain.Comment, error) {
	if m.CommentsErr != nil {
		return nil, m.CommentsErr
	}
	return m.MockTaskRepository.GetComments(0)
}

// MockTaskRepositoryWithDeleteError extends MockTaskRepository to return error on Delete.
type MockTaskRepositoryWithDeleteError struct {
	*MockTaskRepository
	DeleteErr error
}

// Delete returns an error if configured.
func (m *MockTaskRepositoryWithDeleteError) Delete(_ int) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	return m.MockTaskRepository.Delete(0)
}

// MockTaskRepositoryWithAddCommentError extends MockTaskRepository to return error on AddComment.
type MockTaskRepositoryWithAddCommentError struct {
	*MockTaskRepository
	AddCommentErr error
}

// AddComment returns an error if configured.
func (m *MockTaskRepositoryWithAddCommentError) AddComment(_ int, _ domain.Comment) error {
	if m.AddCommentErr != nil {
		return m.AddCommentErr
	}
	return nil
}

// MockGit is a test double for domain.Git.
// Fields are ordered to minimize memory padding.
type MockGit struct {
	CurrentBranchErr  error
	CurrentBranchName string
}

// Ensure MockGit implements domain.Git interface.
var _ domain.Git = (*MockGit)(nil)

// CurrentBranch returns the configured branch name or error.
func (m *MockGit) CurrentBranch() (string, error) {
	if m.CurrentBranchErr != nil {
		return "", m.CurrentBranchErr
	}
	return m.CurrentBranchName, nil
}

// BranchExists is not implemented yet.
func (m *MockGit) BranchExists(_ string) (bool, error) {
	panic("not implemented")
}

// HasUncommittedChanges is not implemented yet.
func (m *MockGit) HasUncommittedChanges(_ string) (bool, error) {
	panic("not implemented")
}

// HasMergeConflict is not implemented yet.
func (m *MockGit) HasMergeConflict(_, _ string) (bool, error) {
	panic("not implemented")
}

// Merge is not implemented yet.
func (m *MockGit) Merge(_ string, _ bool) error {
	panic("not implemented")
}

// DeleteBranch is not implemented yet.
func (m *MockGit) DeleteBranch(_ string) error {
	panic("not implemented")
}
