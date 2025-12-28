//go:build integration

package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRepo creates a temporary git repository for integration testing.
// Returns the path to the repository and a cleanup function.
func testRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize git repository
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

// crew runs the crew CLI command in the given directory.
func crew(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	// Build the binary if not already built
	binPath := buildCrew(t)

	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		output += stderr.String()
	}

	return output, err
}

// crewMust runs the crew CLI command and fails if it errors.
func crewMust(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := crew(t, dir, args...)
	require.NoError(t, err, "crew %v failed: %s", args, out)
	return out
}

var (
	crewBinPath string
	buildOnce   sync.Once
	buildErr    error
)

// buildCrew builds the crew binary once and caches the path.
func buildCrew(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		// Get the module root directory
		wd, err := os.Getwd()
		if err != nil {
			buildErr = err
			return
		}

		// Find module root by looking for go.mod
		moduleRoot := wd
		for {
			if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); err == nil {
				break
			}
			parent := filepath.Dir(moduleRoot)
			if parent == moduleRoot {
				buildErr = os.ErrNotExist
				return
			}
			moduleRoot = parent
		}

		// Build to a fixed temp directory (not per-test)
		tmpDir := os.TempDir()
		binPath := filepath.Join(tmpDir, "crew-integration-test")
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/crew")
		cmd.Dir = moduleRoot
		out, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = err
			t.Logf("build failed: %s", out)
			return
		}

		crewBinPath = binPath
	})

	require.NoError(t, buildErr, "failed to build crew binary")
	return crewBinPath
}

// =============================================================================
// Init Command Integration Tests
// =============================================================================

func TestIntegration_Init(t *testing.T) {
	dir := testRepo(t)

	// Run init
	out := crewMust(t, dir, "init")
	assert.Contains(t, out, "Initialized")

	// Verify .git/crew directory exists
	crewDir := filepath.Join(dir, ".git", "crew")
	info, err := os.Stat(crewDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify tasks.json exists
	tasksFile := filepath.Join(crewDir, "tasks.json")
	_, err = os.Stat(tasksFile)
	require.NoError(t, err)
}

func TestIntegration_Init_AlreadyInitialized(t *testing.T) {
	dir := testRepo(t)

	// First init
	crewMust(t, dir, "init")

	// Second init should fail
	_, err := crew(t, dir, "init")
	assert.Error(t, err)
}

// =============================================================================
// New Task Integration Tests
// =============================================================================

func TestIntegration_New(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	// Create a task
	out := crewMust(t, dir, "new", "--title", "Test task")
	assert.Contains(t, out, "Created task #1")

	// Verify with show
	out = crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Test task")
	assert.Contains(t, out, "todo")
}

func TestIntegration_New_WithAllOptions(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	// Create task with all options
	out := crewMust(t, dir, "new",
		"--title", "Full task",
		"--desc", "Task description",
		"--label", "bug",
		"--label", "urgent",
		"--issue", "42",
	)
	assert.Contains(t, out, "Created task #1")

	// Verify with show
	out = crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Full task")
	assert.Contains(t, out, "Task description")
	assert.Contains(t, out, "bug")
	assert.Contains(t, out, "urgent")
	assert.Contains(t, out, "#42")
}

func TestIntegration_New_WithParent(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	// Create parent task
	crewMust(t, dir, "new", "--title", "Parent task")

	// Create child task
	out := crewMust(t, dir, "new", "--title", "Child task", "--parent", "1")
	assert.Contains(t, out, "Created task #2")

	// Verify parent shows child in sub-tasks
	out = crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Sub-tasks:")
	assert.Contains(t, out, "Child task")

	// Verify child shows parent
	out = crewMust(t, dir, "show", "2")
	assert.Contains(t, out, "Parent: #1")
}

// =============================================================================
// List Tasks Integration Tests
// =============================================================================

func TestIntegration_List(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	// Create tasks
	crewMust(t, dir, "new", "--title", "Task one")
	crewMust(t, dir, "new", "--title", "Task two")
	crewMust(t, dir, "new", "--title", "Task three")

	// List all tasks
	out := crewMust(t, dir, "list")
	assert.Contains(t, out, "Task one")
	assert.Contains(t, out, "Task two")
	assert.Contains(t, out, "Task three")
}

func TestIntegration_List_WithParentFilter(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	// Create parent and children
	crewMust(t, dir, "new", "--title", "Parent")
	crewMust(t, dir, "new", "--title", "Child 1", "--parent", "1")
	crewMust(t, dir, "new", "--title", "Child 2", "--parent", "1")
	crewMust(t, dir, "new", "--title", "Standalone")

	// List only children of parent
	out := crewMust(t, dir, "list", "--parent", "1")
	assert.Contains(t, out, "Child 1")
	assert.Contains(t, out, "Child 2")
	assert.NotContains(t, out, "Parent")
	assert.NotContains(t, out, "Standalone")
}

func TestIntegration_List_WithLabelFilter(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	// Create tasks with labels
	crewMust(t, dir, "new", "--title", "Bug task", "--label", "bug")
	crewMust(t, dir, "new", "--title", "Feature task", "--label", "feature")
	crewMust(t, dir, "new", "--title", "Both", "--label", "bug", "--label", "feature")

	// Filter by bug label
	out := crewMust(t, dir, "list", "--label", "bug")
	assert.Contains(t, out, "Bug task")
	assert.Contains(t, out, "Both")
	assert.NotContains(t, out, "Feature task")
}

// =============================================================================
// Show Task Integration Tests
// =============================================================================

func TestIntegration_Show(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	crewMust(t, dir, "new", "--title", "Show test", "--desc", "Test description")

	out := crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Task 1: Show test")
	assert.Contains(t, out, "Test description")
	assert.Contains(t, out, "Status: todo")
	assert.Contains(t, out, "Parent: none")
	assert.Contains(t, out, "Branch: crew-1")
}

func TestIntegration_Show_NotFound(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	_, err := crew(t, dir, "show", "999")
	assert.Error(t, err)
}

// =============================================================================
// Edit Task Integration Tests
// =============================================================================

func TestIntegration_Edit(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	crewMust(t, dir, "new", "--title", "Original title")

	// Edit title
	out := crewMust(t, dir, "edit", "1", "--title", "Updated title")
	assert.Contains(t, out, "Updated task #1")

	// Verify change
	out = crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Updated title")
	assert.NotContains(t, out, "Original title")
}

func TestIntegration_Edit_Labels(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	crewMust(t, dir, "new", "--title", "Label test", "--label", "old")

	// Add and remove labels
	crewMust(t, dir, "edit", "1", "--add-label", "new", "--rm-label", "old")

	// Verify
	out := crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "new")
	assert.NotContains(t, out, "old")
}

// =============================================================================
// Delete Task Integration Tests
// =============================================================================

func TestIntegration_Rm(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	crewMust(t, dir, "new", "--title", "To delete")

	// Delete
	out := crewMust(t, dir, "rm", "1")
	assert.Contains(t, out, "Deleted task #1")

	// Verify deleted
	_, err := crew(t, dir, "show", "1")
	assert.Error(t, err)
}

func TestIntegration_Rm_NotFound(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	_, err := crew(t, dir, "rm", "999")
	assert.Error(t, err)
}

// =============================================================================
// Copy Task Integration Tests
// =============================================================================

func TestIntegration_Cp(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	crewMust(t, dir, "new", "--title", "Original", "--desc", "Description", "--label", "test")

	// Copy
	out := crewMust(t, dir, "cp", "1")
	assert.Contains(t, out, "Copied task #1 to #2")

	// Verify copy
	out = crewMust(t, dir, "show", "2")
	assert.Contains(t, out, "Original (copy)")
	assert.Contains(t, out, "Description")
	assert.Contains(t, out, "test")
}

func TestIntegration_Cp_WithCustomTitle(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	crewMust(t, dir, "new", "--title", "Original")

	// Copy with custom title
	out := crewMust(t, dir, "cp", "1", "--title", "Custom copy")
	assert.Contains(t, out, "Copied task #1 to #2")

	// Verify
	out = crewMust(t, dir, "show", "2")
	assert.Contains(t, out, "Custom copy")
	assert.NotContains(t, out, "Original")
}

// =============================================================================
// Comment Integration Tests
// =============================================================================

func TestIntegration_Comment(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	crewMust(t, dir, "new", "--title", "Comment test")

	// Add comment
	out := crewMust(t, dir, "comment", "1", "This is a test comment")
	assert.Contains(t, out, "Added comment to task #1")

	// Verify comment in show
	out = crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Comments:")
	assert.Contains(t, out, "This is a test comment")
}

func TestIntegration_Comment_Multiple(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	crewMust(t, dir, "new", "--title", "Multi comment")

	// Add multiple comments
	crewMust(t, dir, "comment", "1", "First comment")
	crewMust(t, dir, "comment", "1", "Second comment")

	// Verify both comments
	out := crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "First comment")
	assert.Contains(t, out, "Second comment")
}

// =============================================================================
// End-to-End Workflow Tests
// =============================================================================

func TestIntegration_FullWorkflow(t *testing.T) {
	dir := testRepo(t)

	// Initialize
	crewMust(t, dir, "init")

	// Create parent task
	crewMust(t, dir, "new", "--title", "Feature: Auth", "--label", "feature")

	// Create sub-tasks
	crewMust(t, dir, "new", "--title", "Implement login", "--parent", "1")
	crewMust(t, dir, "new", "--title", "Implement logout", "--parent", "1")

	// Add comment to parent
	crewMust(t, dir, "comment", "1", "Starting auth feature")

	// Edit sub-task
	crewMust(t, dir, "edit", "2", "--add-label", "in-progress")

	// List all
	out := crewMust(t, dir, "list")
	assert.Contains(t, out, "Feature: Auth")
	assert.Contains(t, out, "Implement login")
	assert.Contains(t, out, "Implement logout")

	// Show parent with sub-tasks and comments
	out = crewMust(t, dir, "show", "1")
	assert.Contains(t, out, "Feature: Auth")
	assert.Contains(t, out, "Sub-tasks:")
	assert.Contains(t, out, "Implement login")
	assert.Contains(t, out, "Comments:")
	assert.Contains(t, out, "Starting auth feature")

	// Copy a task
	crewMust(t, dir, "cp", "2", "--title", "Implement SSO")

	// Delete original sub-task
	crewMust(t, dir, "rm", "2")

	// Verify task is deleted
	_, err := crew(t, dir, "show", "2")
	assert.Error(t, err)

	// Verify copied task exists
	out = crewMust(t, dir, "show", "4")
	assert.Contains(t, out, "Implement SSO")
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestIntegration_NotInitialized(t *testing.T) {
	dir := testRepo(t)

	// Commands should fail without init
	_, err := crew(t, dir, "new", "--title", "Test")
	assert.Error(t, err)

	_, err = crew(t, dir, "list")
	assert.Error(t, err)

	_, err = crew(t, dir, "show", "1")
	assert.Error(t, err)
}

func TestIntegration_NotGitRepo(t *testing.T) {
	dir := t.TempDir() // Not a git repo

	_, err := crew(t, dir, "init")
	assert.Error(t, err)
}

func TestIntegration_InvalidTaskID(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	// Various invalid IDs
	for _, id := range []string{"invalid", "-1", "0", "abc"} {
		_, err := crew(t, dir, "show", id)
		assert.Error(t, err, "expected error for ID: %s", id)
	}
}

func TestIntegration_EmptyTitle(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	_, err := crew(t, dir, "new", "--title", "")
	assert.Error(t, err)
}

func TestIntegration_ParentNotFound(t *testing.T) {
	dir := testRepo(t)
	crewMust(t, dir, "init")

	_, err := crew(t, dir, "new", "--title", "Child", "--parent", "999")
	assert.Error(t, err)
}

func TestIntegration_Help(t *testing.T) {
	dir := testRepo(t)

	// Help should work without init
	out := crewMust(t, dir, "--help")
	assert.Contains(t, out, "git-crew")
	assert.Contains(t, out, "Setup Commands:")
	assert.Contains(t, out, "Task Management:")

	// Subcommand help
	out = crewMust(t, dir, "new", "--help")
	assert.Contains(t, out, "Create a new task")
	assert.Contains(t, out, "--title")
}

func TestIntegration_Version(t *testing.T) {
	dir := testRepo(t)

	out := crewMust(t, dir, "--version")
	// Should contain version info (even if "dev" or similar)
	assert.True(t, len(strings.TrimSpace(out)) > 0)
}
