//go:build integration

package gitstore

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// testRepoPath creates a temporary git repository for integration testing.
// Returns the path to the repository.
func testRepoPath(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize git repository using git command
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@example.com")
	run(t, dir, "git", "config", "user.name", "Test User")

	// Create initial commit (required for some git operations)
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# Test\n"), 0o644))
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "Initial commit")

	return dir
}

// run executes a command and fails the test if it errors.
func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "command failed: %s %v\noutput: %s", name, args, out)
	return string(out)
}

// =============================================================================
// New Function Tests
// =============================================================================

func TestIntegration_New(t *testing.T) {
	dir := testRepoPath(t)

	store, err := New(dir, "crew-test")
	require.NoError(t, err)
	require.NotNil(t, store)
}

func TestIntegration_New_NotGitRepo(t *testing.T) {
	dir := t.TempDir() // Not a git repo

	store, err := New(dir, "crew-test")
	assert.Error(t, err)
	assert.Nil(t, store)
}

func TestIntegration_New_InvalidPath(t *testing.T) {
	store, err := New("/nonexistent/path", "crew-test")
	assert.Error(t, err)
	assert.Nil(t, store)
}

// =============================================================================
// Persistence Tests
// =============================================================================

func TestIntegration_Persistence(t *testing.T) {
	dir := testRepoPath(t)

	// Create store and save data
	store1, err := New(dir, "crew-test")
	require.NoError(t, err)
	_, err := store1.Initialize(); require.NoError(t, err)

	task := &domain.Task{
		ID:          1,
		Title:       "Persistent Task",
		Description: "This should persist",
		Status:      domain.StatusInProgress,
		Labels:      []string{"test", "integration"},
	}
	require.NoError(t, store1.Save(task))

	// Add comment
	comment := domain.Comment{Text: "Test comment", Time: time.Now()}
	require.NoError(t, store1.AddComment(1, comment))

	// Create new store instance (simulating process restart)
	store2, err := New(dir, "crew-test")
	require.NoError(t, err)

	// Verify data persists
	got, err := store2.Get(1)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Persistent Task", got.Title)
	assert.Equal(t, "This should persist", got.Description)
	assert.Equal(t, domain.StatusInProgress, got.Status)
	assert.Equal(t, []string{"test", "integration"}, got.Labels)

	// Verify comments persist
	comments, err := store2.GetComments(1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "Test comment", comments[0].Text)

	// Verify initialized state persists
	assert.True(t, store2.IsInitialized())
}

func TestIntegration_NextID_Persistence(t *testing.T) {
	dir := testRepoPath(t)

	// Get some IDs
	store1, err := New(dir, "crew-test")
	require.NoError(t, err)

	id1, err := store1.NextID()
	require.NoError(t, err)
	assert.Equal(t, 1, id1)

	id2, err := store1.NextID()
	require.NoError(t, err)
	assert.Equal(t, 2, id2)

	// Create new store instance
	store2, err := New(dir, "crew-test")
	require.NoError(t, err)

	// Next ID should continue from where we left off
	id3, err := store2.NextID()
	require.NoError(t, err)
	assert.Equal(t, 3, id3)
}

// =============================================================================
// Git Interoperability Tests
// =============================================================================

func TestIntegration_GitRefs_Visible(t *testing.T) {
	dir := testRepoPath(t)

	store, err := New(dir, "crew-test")
	require.NoError(t, err)
	_, err := store.Initialize(); require.NoError(t, err)

	// Save a task
	task := &domain.Task{ID: 1, Title: "Git visible task"}
	require.NoError(t, store.Save(task))

	// Verify refs are visible via git command
	out := run(t, dir, "git", "for-each-ref", "--format=%(refname)", "refs/crew-test/")
	assert.Contains(t, out, "refs/crew-test/tasks/1")
	assert.Contains(t, out, "refs/crew-test/meta")
	assert.Contains(t, out, "refs/crew-test/initialized")
}

func TestIntegration_GitCatFile(t *testing.T) {
	dir := testRepoPath(t)

	store, err := New(dir, "crew-test")
	require.NoError(t, err)

	// Save a task
	task := &domain.Task{ID: 1, Title: "Cat file test", Description: "Description"}
	require.NoError(t, store.Save(task))

	// Get blob content via git cat-file
	out := run(t, dir, "git", "cat-file", "-p", "refs/crew-test/tasks/1")
	assert.Contains(t, out, "title: Cat file test")
	assert.Contains(t, out, "description: Description")
}

func TestIntegration_NamespaceIsolation_GitLevel(t *testing.T) {
	dir := testRepoPath(t)

	store1, err := New(dir, "crew-user1")
	require.NoError(t, err)
	store2, err := New(dir, "crew-user2")
	require.NoError(t, err)

	// Save to both stores
	require.NoError(t, store1.Save(&domain.Task{ID: 1, Title: "User1 Task"}))
	require.NoError(t, store2.Save(&domain.Task{ID: 1, Title: "User2 Task"}))

	// Verify separate refs exist
	out := run(t, dir, "git", "for-each-ref", "--format=%(refname)", "refs/")

	assert.Contains(t, out, "refs/crew-user1/tasks/1")
	assert.Contains(t, out, "refs/crew-user2/tasks/1")

	// Verify content is different
	out1 := run(t, dir, "git", "cat-file", "-p", "refs/crew-user1/tasks/1")
	out2 := run(t, dir, "git", "cat-file", "-p", "refs/crew-user2/tasks/1")

	assert.Contains(t, out1, "User1 Task")
	assert.Contains(t, out2, "User2 Task")
}

// =============================================================================
// Snapshot Persistence Tests
// =============================================================================

func TestIntegration_Snapshot_Persistence(t *testing.T) {
	dir := testRepoPath(t)

	// Create store and tasks
	store1, err := New(dir, "crew-test")
	require.NoError(t, err)
	_, err := store1.Initialize(); require.NoError(t, err)

	task1 := &domain.Task{ID: 1, Title: "Task 1", Status: domain.StatusTodo}
	task2 := &domain.Task{ID: 2, Title: "Task 2", Status: domain.StatusInProgress}
	require.NoError(t, store1.Save(task1))
	require.NoError(t, store1.Save(task2))

	// Save snapshot
	mainSHA := "abc123def456"
	require.NoError(t, store1.SaveSnapshot(mainSHA))

	// Modify tasks
	task1.Status = domain.StatusDone
	require.NoError(t, store1.Save(task1))
	require.NoError(t, store1.Delete(2))

	// Create new store instance
	store2, err := New(dir, "crew-test")
	require.NoError(t, err)

	// Verify snapshots persist
	snapshots, err := store2.ListSnapshots(mainSHA)
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	assert.Equal(t, mainSHA, snapshots[0].MainSHA)
	assert.Equal(t, 1, snapshots[0].Seq)

	// Restore from snapshot
	require.NoError(t, store2.RestoreSnapshot(snapshots[0].Ref))

	// Verify restored state
	tasks, err := store2.List(domain.TaskFilter{})
	require.NoError(t, err)
	require.Len(t, tasks, 2)

	restored1, err := store2.Get(1)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusTodo, restored1.Status)

	restored2, err := store2.Get(2)
	require.NoError(t, err)
	assert.Equal(t, "Task 2", restored2.Title)
}

func TestIntegration_Snapshot_GitVisible(t *testing.T) {
	dir := testRepoPath(t)

	store, err := New(dir, "crew-test")
	require.NoError(t, err)
	_, err := store.Initialize(); require.NoError(t, err)

	task := &domain.Task{ID: 1, Title: "Snapshot task"}
	require.NoError(t, store.Save(task))
	require.NoError(t, store.SaveSnapshot("deadbeef"))

	// Verify snapshot ref is visible via git
	out := run(t, dir, "git", "for-each-ref", "--format=%(refname)", "refs/crew-test/snapshots/")
	assert.Contains(t, out, "refs/crew-test/snapshots/deadbeef_001")

	// Verify current ref points to snapshot
	out = run(t, dir, "git", "symbolic-ref", "refs/crew-test/current")
	assert.Contains(t, out, "refs/crew-test/snapshots/deadbeef_001")
}

// =============================================================================
// SyncSnapshot Tests
// =============================================================================

func TestIntegration_SyncSnapshot(t *testing.T) {
	dir := testRepoPath(t)

	// Get current HEAD SHA
	headSHA := strings.TrimSpace(run(t, dir, "git", "rev-parse", "HEAD"))

	store, err := New(dir, "crew-test")
	require.NoError(t, err)
	_, err := store.Initialize(); require.NoError(t, err)

	// Create initial task and snapshot
	task := &domain.Task{ID: 1, Title: "Original", Status: domain.StatusTodo}
	require.NoError(t, store.Save(task))
	require.NoError(t, store.SaveSnapshot(headSHA))

	// Modify task
	task.Status = domain.StatusDone
	require.NoError(t, store.Save(task))

	// Verify modification
	got, err := store.Get(1)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusDone, got.Status)

	// Sync should restore from snapshot (since HEAD hasn't changed)
	require.NoError(t, store.SyncSnapshot())

	// Verify restored
	got, err = store.Get(1)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusTodo, got.Status)
}

// =============================================================================
// PruneSnapshots Tests
// =============================================================================

func TestIntegration_PruneSnapshots(t *testing.T) {
	dir := testRepoPath(t)

	store, err := New(dir, "crew-test")
	require.NoError(t, err)
	_, err := store.Initialize(); require.NoError(t, err)

	task := &domain.Task{ID: 1, Title: "Task", Status: domain.StatusTodo}
	require.NoError(t, store.Save(task))

	mainSHA := "prune123"

	// Create multiple snapshots
	statuses := []domain.Status{domain.StatusTodo, domain.StatusInProgress, domain.StatusDone}
	for i := 0; i < 5; i++ {
		task.Status = statuses[i%3] // vary status
		require.NoError(t, store.Save(task))
		require.NoError(t, store.SaveSnapshot(mainSHA))
	}

	// Verify 5 snapshots exist
	snapshots, err := store.ListSnapshots(mainSHA)
	require.NoError(t, err)
	require.Len(t, snapshots, 5)

	// Prune keeping only 2
	require.NoError(t, store.PruneSnapshots(2))

	// Verify only 2 remain
	snapshots, err = store.ListSnapshots(mainSHA)
	require.NoError(t, err)
	require.Len(t, snapshots, 2)

	// Verify newest ones are kept (seq 4 and 5)
	assert.Equal(t, 4, snapshots[0].Seq)
	assert.Equal(t, 5, snapshots[1].Seq)

	// Verify via git that old refs are removed
	out := run(t, dir, "git", "for-each-ref", "--format=%(refname)", "refs/crew-test/snapshots/")
	assert.NotContains(t, out, "prune123_001")
	assert.NotContains(t, out, "prune123_002")
	assert.NotContains(t, out, "prune123_003")
	assert.Contains(t, out, "refs/crew-test/snapshots/prune123_004")
	assert.Contains(t, out, "refs/crew-test/snapshots/prune123_005")
}

// =============================================================================
// Full Workflow Test
// =============================================================================

func TestIntegration_FullWorkflow(t *testing.T) {
	dir := testRepoPath(t)

	// Step 1: Initialize store
	store, err := New(dir, "crew-test")
	require.NoError(t, err)
	_, err := store.Initialize(); require.NoError(t, err)

	// Step 2: Create tasks
	id1, err := store.NextID()
	require.NoError(t, err)
	task1 := &domain.Task{ID: id1, Title: "Parent task", Status: domain.StatusTodo}
	require.NoError(t, store.Save(task1))

	id2, err := store.NextID()
	require.NoError(t, err)
	task2 := &domain.Task{ID: id2, Title: "Child task", ParentID: &id1, Status: domain.StatusTodo}
	require.NoError(t, store.Save(task2))

	// Step 3: Add comments
	require.NoError(t, store.AddComment(id1, domain.Comment{Text: "Starting work", Time: time.Now()}))

	// Step 4: Save snapshot
	headSHA := strings.TrimSpace(run(t, dir, "git", "rev-parse", "HEAD"))
	require.NoError(t, store.SaveSnapshot(headSHA))

	// Step 5: Modify tasks
	task1.Status = domain.StatusDone
	require.NoError(t, store.Save(task1))
	require.NoError(t, store.Delete(id2))

	// Step 6: Simulate process restart
	store2, err := New(dir, "crew-test")
	require.NoError(t, err)

	// Verify modified state
	tasks, err := store2.List(domain.TaskFilter{})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, domain.StatusDone, tasks[0].Status)

	// Step 7: Restore from snapshot
	snapshots, err := store2.ListSnapshots(headSHA)
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	require.NoError(t, store2.RestoreSnapshot(snapshots[0].Ref))

	// Step 8: Verify restored state
	tasks, err = store2.List(domain.TaskFilter{})
	require.NoError(t, err)
	require.Len(t, tasks, 2)

	parent, err := store2.Get(id1)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusTodo, parent.Status)

	child, err := store2.Get(id2)
	require.NoError(t, err)
	assert.Equal(t, "Child task", child.Title)

	// Comments should still exist
	comments, err := store2.GetComments(id1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "Starting work", comments[0].Text)

	// Step 9: Verify git refs
	out := run(t, dir, "git", "for-each-ref", "--format=%(refname)", "refs/crew-test/")
	assert.Contains(t, out, "refs/crew-test/tasks/1")
	assert.Contains(t, out, "refs/crew-test/tasks/2")
	assert.Contains(t, out, "refs/crew-test/comments/1")
	assert.Contains(t, out, "refs/crew-test/meta")
	assert.Contains(t, out, "refs/crew-test/initialized")
	assert.Contains(t, out, "refs/crew-test/snapshots/")
	assert.Contains(t, out, "refs/crew-test/current")
}
