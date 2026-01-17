package jsonstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

func TestStore_Initialize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")

	store := New(path)

	// Initialize should create the file
	if _, err := store.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// File should exist
	if _, err := os.Stat(path); err != nil {
		t.Errorf("store file not created: %v", err)
	}

	// Initialize again should be idempotent
	if _, err := store.Initialize(); err != nil {
		t.Fatalf("Initialize() second call error = %v", err)
	}
}

func TestStore_NextID(t *testing.T) {
	store := newTestStore(t)

	id1, err := store.NextID()
	if err != nil {
		t.Fatalf("NextID() error = %v", err)
	}
	if id1 != 1 {
		t.Errorf("NextID() = %d, want 1", id1)
	}

	id2, err := store.NextID()
	if err != nil {
		t.Fatalf("NextID() error = %v", err)
	}
	if id2 != 2 {
		t.Errorf("NextID() = %d, want 2", id2)
	}
}

func TestStore_SaveAndGet(t *testing.T) {
	store := newTestStore(t)

	now := time.Now().Truncate(time.Second) // JSON loses nanoseconds
	task := &domain.Task{
		ID:          1,
		Title:       "Test Task",
		Description: "Test Description",
		Status:      domain.StatusTodo,
		Created:     now,
		BaseBranch:  "main",
		Labels:      []string{"test", "feature"},
	}

	// Save
	if err := store.Save(task); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Get
	got, err := store.Get(1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got == nil {
		t.Fatal("Get() returned nil")
	}

	// Verify fields
	if got.ID != task.ID {
		t.Errorf("ID = %d, want %d", got.ID, task.ID)
	}
	if got.Title != task.Title {
		t.Errorf("Title = %q, want %q", got.Title, task.Title)
	}
	if got.Description != task.Description {
		t.Errorf("Description = %q, want %q", got.Description, task.Description)
	}
	if got.Status != task.Status {
		t.Errorf("Status = %q, want %q", got.Status, task.Status)
	}
	if !got.Created.Equal(task.Created) {
		t.Errorf("Created = %v, want %v", got.Created, task.Created)
	}
	if got.BaseBranch != task.BaseBranch {
		t.Errorf("BaseBranch = %q, want %q", got.BaseBranch, task.BaseBranch)
	}
	if len(got.Labels) != len(task.Labels) {
		t.Errorf("Labels = %v, want %v", got.Labels, task.Labels)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	store := newTestStore(t)

	got, err := store.Get(999)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != nil {
		t.Errorf("Get() = %v, want nil for non-existent task", got)
	}
}

func TestStore_SaveWithParentID(t *testing.T) {
	store := newTestStore(t)

	parentID := 1
	task := &domain.Task{
		ID:         2,
		ParentID:   &parentID,
		Title:      "Sub Task",
		Status:     domain.StatusTodo,
		Created:    time.Now(),
		BaseBranch: "main",
	}

	if err := store.Save(task); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Get(2)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ParentID == nil {
		t.Error("ParentID = nil, want non-nil")
	} else if *got.ParentID != parentID {
		t.Errorf("ParentID = %d, want %d", *got.ParentID, parentID)
	}
}

func TestStore_List(t *testing.T) {
	store := newTestStore(t)

	// Create test tasks
	tasks := []*domain.Task{
		{ID: 1, Title: "Task 1", Status: domain.StatusTodo, Created: time.Now(), BaseBranch: "main", Labels: []string{"bug"}},
		{ID: 2, Title: "Task 2", Status: domain.StatusInProgress, Created: time.Now(), BaseBranch: "main", Labels: []string{"feature"}},
		{ID: 3, Title: "Task 3", Status: domain.StatusClosed, Created: time.Now(), BaseBranch: "main", Labels: []string{"bug", "feature"}},
	}

	for _, task := range tasks {
		if err := store.Save(task); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// List all
	all, err := store.List(domain.TaskFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("List() returned %d tasks, want 3", len(all))
	}

	// Verify sorted by ID
	for i, task := range all {
		if task.ID != i+1 {
			t.Errorf("List()[%d].ID = %d, want %d", i, task.ID, i+1)
		}
	}
}

func TestStore_ListWithLabelFilter(t *testing.T) {
	store := newTestStore(t)

	tasks := []*domain.Task{
		{ID: 1, Title: "Task 1", Status: domain.StatusTodo, Created: time.Now(), BaseBranch: "main", Labels: []string{"bug"}},
		{ID: 2, Title: "Task 2", Status: domain.StatusTodo, Created: time.Now(), BaseBranch: "main", Labels: []string{"feature"}},
		{ID: 3, Title: "Task 3", Status: domain.StatusTodo, Created: time.Now(), BaseBranch: "main", Labels: []string{"bug", "feature"}},
	}

	for _, task := range tasks {
		if err := store.Save(task); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Filter by single label
	bugTasks, err := store.List(domain.TaskFilter{Labels: []string{"bug"}})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(bugTasks) != 2 {
		t.Errorf("List(bug) returned %d tasks, want 2", len(bugTasks))
	}

	// Filter by multiple labels (AND)
	both, err := store.List(domain.TaskFilter{Labels: []string{"bug", "feature"}})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(both) != 1 {
		t.Errorf("List(bug, feature) returned %d tasks, want 1", len(both))
	}
	if len(both) > 0 && both[0].ID != 3 {
		t.Errorf("List(bug, feature)[0].ID = %d, want 3", both[0].ID)
	}
}

func TestStore_ListWithParentFilter(t *testing.T) {
	store := newTestStore(t)

	parentID := 1
	tasks := []*domain.Task{
		{ID: 1, Title: "Parent", Status: domain.StatusTodo, Created: time.Now(), BaseBranch: "main"},
		{ID: 2, Title: "Child 1", ParentID: &parentID, Status: domain.StatusTodo, Created: time.Now(), BaseBranch: "main"},
		{ID: 3, Title: "Child 2", ParentID: &parentID, Status: domain.StatusTodo, Created: time.Now(), BaseBranch: "main"},
		{ID: 4, Title: "Orphan", Status: domain.StatusTodo, Created: time.Now(), BaseBranch: "main"},
	}

	for _, task := range tasks {
		if err := store.Save(task); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Filter by parent
	children, err := store.List(domain.TaskFilter{ParentID: &parentID})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(children) != 2 {
		t.Errorf("List(parent=1) returned %d tasks, want 2", len(children))
	}
}

func TestStore_GetChildren(t *testing.T) {
	store := newTestStore(t)

	parentID := 1
	tasks := []*domain.Task{
		{ID: 1, Title: "Parent", Status: domain.StatusTodo, Created: time.Now(), BaseBranch: "main"},
		{ID: 2, Title: "Child 1", ParentID: &parentID, Status: domain.StatusTodo, Created: time.Now(), BaseBranch: "main"},
		{ID: 3, Title: "Child 2", ParentID: &parentID, Status: domain.StatusTodo, Created: time.Now(), BaseBranch: "main"},
	}

	for _, task := range tasks {
		if err := store.Save(task); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	children, err := store.GetChildren(1)
	if err != nil {
		t.Fatalf("GetChildren() error = %v", err)
	}
	if len(children) != 2 {
		t.Errorf("GetChildren() returned %d tasks, want 2", len(children))
	}
}

func TestStore_Delete(t *testing.T) {
	store := newTestStore(t)

	task := &domain.Task{
		ID:         1,
		Title:      "To Delete",
		Status:     domain.StatusTodo,
		Created:    time.Now(),
		BaseBranch: "main",
	}

	if err := store.Save(task); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Add a comment
	comment := domain.Comment{Text: "Test comment", Time: time.Now()}
	if err := store.AddComment(1, comment); err != nil {
		t.Fatalf("AddComment() error = %v", err)
	}

	// Delete
	if err := store.Delete(1); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	got, err := store.Get(1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != nil {
		t.Error("Get() returned task after Delete()")
	}

	// Verify comments also deleted
	comments, err := store.GetComments(1)
	if err != nil {
		t.Fatalf("GetComments() error = %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("GetComments() returned %d comments after Delete(), want 0", len(comments))
	}
}

func TestStore_Comments(t *testing.T) {
	store := newTestStore(t)

	task := &domain.Task{
		ID:         1,
		Title:      "Task with comments",
		Status:     domain.StatusTodo,
		Created:    time.Now(),
		BaseBranch: "main",
	}

	if err := store.Save(task); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	now := time.Now().Truncate(time.Second)
	comments := []domain.Comment{
		{Text: "First comment", Time: now},
		{Text: "Second comment", Time: now.Add(time.Hour)},
	}

	for _, c := range comments {
		if err := store.AddComment(1, c); err != nil {
			t.Fatalf("AddComment() error = %v", err)
		}
	}

	got, err := store.GetComments(1)
	if err != nil {
		t.Fatalf("GetComments() error = %v", err)
	}
	if len(got) != 2 {
		t.Errorf("GetComments() returned %d comments, want 2", len(got))
	}
	if len(got) > 0 && got[0].Text != "First comment" {
		t.Errorf("GetComments()[0].Text = %q, want %q", got[0].Text, "First comment")
	}
}

func TestStore_GetCommentsEmpty(t *testing.T) {
	store := newTestStore(t)

	comments, err := store.GetComments(999)
	if err != nil {
		t.Fatalf("GetComments() error = %v", err)
	}
	if comments == nil {
		t.Error("GetComments() returned nil, want empty slice")
	}
	if len(comments) != 0 {
		t.Errorf("GetComments() returned %d comments, want 0", len(comments))
	}
}

func TestStore_UpdateTask(t *testing.T) {
	store := newTestStore(t)

	task := &domain.Task{
		ID:         1,
		Title:      "Original Title",
		Status:     domain.StatusTodo,
		Created:    time.Now(),
		BaseBranch: "main",
	}

	if err := store.Save(task); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Update
	task.Title = "Updated Title"
	task.Status = domain.StatusInProgress
	task.Agent = "claude"

	if err := store.Save(task); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	got, err := store.Get(1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", got.Title, "Updated Title")
	}
	if got.Status != domain.StatusInProgress {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusInProgress)
	}
	if got.Agent != "claude" {
		t.Errorf("Agent = %q, want %q", got.Agent, "claude")
	}
}

// newTestStore creates a new store with a temporary file for testing.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")
	store := New(path)
	if _, err := store.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	return store
}
