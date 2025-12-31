// Package jsonstore provides a JSON file-based implementation of TaskRepository.
package jsonstore

import (
	"time"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// storeData represents the JSON file structure.
// Fields are ordered to minimize memory padding.
type storeData struct {
	Tasks    map[string]*taskData `json:"tasks"`
	Comments map[string][]comment `json:"comments"`
	Meta     meta                 `json:"meta"`
}

// meta contains store metadata.
type meta struct {
	NextTaskID int `json:"nextTaskID"`
}

// taskData is the JSON representation of a task (without ID, which is the map key).
type taskData = domain.Task

// comment is the JSON representation of a comment.
type comment = domain.Comment

// Store implements domain.TaskRepository using a JSON file.
type Store struct {
	path     string
	lockPath string
}

// New creates a new Store for the given file path.
// The file does not need to exist; it will be created on first write.
func New(path string) *Store {
	return &Store{
		path:     path,
		lockPath: path + ".lock",
	}
}

// Get retrieves a task by ID.
func (s *Store) Get(id int) (*domain.Task, error) {
	var task *domain.Task
	err := s.withLock(func(data *storeData) error {
		key := strconv.Itoa(id)
		if t, ok := data.Tasks[key]; ok {
			task = t
			task.ID = id
		}
		return nil
	})
	return task, err
}

// List retrieves tasks matching the filter.
func (s *Store) List(filter domain.TaskFilter) ([]*domain.Task, error) {
	var tasks []*domain.Task
	err := s.withLock(func(data *storeData) error {
		for key, t := range data.Tasks {
			id, _ := strconv.Atoi(key)
			t.ID = id

			// Apply ParentID filter
			if filter.ParentID != nil {
				if t.ParentID == nil || *t.ParentID != *filter.ParentID {
					continue
				}
			}

			// Apply Labels filter (AND condition)
			if len(filter.Labels) > 0 {
				if !containsAll(t.Labels, filter.Labels) {
					continue
				}
			}

			tasks = append(tasks, t)
		}
		return nil
	})

	// Sort by ID for consistent ordering
	slices.SortFunc(tasks, func(a, b *domain.Task) int {
		return a.ID - b.ID
	})

	return tasks, err
}

// GetChildren retrieves direct children of a task.
func (s *Store) GetChildren(parentID int) ([]*domain.Task, error) {
	return s.List(domain.TaskFilter{ParentID: &parentID})
}

// Save creates or updates a task.
func (s *Store) Save(task *domain.Task) error {
	return s.withLockWrite(func(data *storeData) error {
		key := strconv.Itoa(task.ID)
		data.Tasks[key] = task
		return nil
	})
}

// Delete removes a task by ID.
func (s *Store) Delete(id int) error {
	return s.withLockWrite(func(data *storeData) error {
		key := strconv.Itoa(id)
		delete(data.Tasks, key)
		delete(data.Comments, key)
		return nil
	})
}

// NextID returns the next available task ID.
func (s *Store) NextID() (int, error) {
	var id int
	err := s.withLockWrite(func(data *storeData) error {
		id = data.Meta.NextTaskID
		data.Meta.NextTaskID++
		return nil
	})
	return id, err
}

// GetComments retrieves comments for a task.
func (s *Store) GetComments(taskID int) ([]domain.Comment, error) {
	var comments []domain.Comment
	err := s.withLock(func(data *storeData) error {
		key := strconv.Itoa(taskID)
		if c, ok := data.Comments[key]; ok {
			comments = c
		} else {
			comments = []domain.Comment{} // Return empty slice, not nil
		}
		return nil
	})
	return comments, err
}

// AddComment adds a comment to a task.
func (s *Store) AddComment(taskID int, comment domain.Comment) error {
	return s.withLockWrite(func(data *storeData) error {
		key := strconv.Itoa(taskID)
		data.Comments[key] = append(data.Comments[key], comment)
		return nil
	})
}

// IsInitialized checks if the store file exists.
func (s *Store) IsInitialized() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

// Initialize creates an empty store file if it doesn't exist.
func (s *Store) Initialize() error {
	// Ensure parent directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Check if file already exists
	if _, err := os.Stat(s.path); err == nil {
		return nil // Already exists
	}

	// Create empty store
	data := &storeData{
		Meta:     meta{NextTaskID: 1},
		Tasks:    make(map[string]*taskData),
		Comments: make(map[string][]comment),
	}

	return s.write(data)
}

// withLock executes fn with a shared (read) lock.
func (s *Store) withLock(fn func(*storeData) error) error {
	lock, err := s.acquireLock(syscall.LOCK_SH)
	if err != nil {
		return err
	}
	defer s.releaseLock(lock)

	data, err := s.read()
	if err != nil {
		return err
	}

	return fn(data)
}

// withLockWrite executes fn with an exclusive (write) lock and writes the result.
func (s *Store) withLockWrite(fn func(*storeData) error) error {
	lock, err := s.acquireLock(syscall.LOCK_EX)
	if err != nil {
		return err
	}
	defer s.releaseLock(lock)

	data, err := s.read()
	if err != nil {
		return err
	}

	if err := fn(data); err != nil {
		return err
	}

	return s.write(data)
}

func (s *Store) acquireLock(lockType int) (*os.File, error) {
	// Ensure lock file directory exists
	dir := filepath.Dir(s.lockPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	lock, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	if err := syscall.Flock(int(lock.Fd()), lockType); err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("acquire lock: %w", err)
	}

	return lock, nil
}

func (s *Store) releaseLock(lock *os.File) {
	_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	_ = lock.Close()
}

func (s *Store) read() (*storeData, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrNotInitialized
		}
		return nil, fmt.Errorf("read store file: %w", err)
	}

	var data storeData
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("parse store file: %w", err)
	}

	// Ensure maps are initialized
	if data.Tasks == nil {
		data.Tasks = make(map[string]*taskData)
	}
	if data.Comments == nil {
		data.Comments = make(map[string][]comment)
	}

	return &data, nil
}

func (s *Store) write(data *storeData) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal store data: %w", err)
	}

	// Write to temp file first, then rename for atomicity
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0o600); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath) // Clean up
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// containsAll checks if slice contains all elements in required.
func containsAll(slice, required []string) bool {
	for _, r := range required {
		found := false
		for _, item := range slice {
			if item == r {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Ensure Store implements TaskRepository.
var _ domain.TaskRepository = (*Store)(nil)

// === Snapshot operations (no-op for JSON store) ===

// SaveSnapshot is a no-op for JSON store.
func (s *Store) SaveSnapshot(mainSHA string) error {
	return nil
}

// RestoreSnapshot is a no-op for JSON store.
func (s *Store) RestoreSnapshot(snapshotRef string) error {
	return nil
}

// ListSnapshots returns empty for JSON store.
func (s *Store) ListSnapshots(mainSHA string) ([]domain.SnapshotInfo, error) {
	return nil, nil
}

// SyncSnapshot is a no-op for JSON store.
func (s *Store) SyncSnapshot() error {
	return nil
}

// PruneSnapshots is a no-op for JSON store.
func (s *Store) PruneSnapshots(olderThan time.Duration) error {
	return nil
}
