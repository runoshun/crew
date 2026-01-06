package cli

import (
	"bytes"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// newTestContainer creates an app.Container with mock dependencies.
func newTestContainer(repo *testutil.MockTaskRepository) *app.Container {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	container := app.NewWithDeps(
		app.Config{},
		repo,
		&testutil.MockStoreInitializer{},
		&testutil.MockClock{NowTime: time.Now()},
		logger,
	)
	container.Git = &testutil.MockGit{}
	return container
}

// =============================================================================
// New Command Tests
// =============================================================================

func TestNewNewCommand_CreateTask(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newNewCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--title", "Test task"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Created task #1")

	// Verify task was created
	task := repo.Tasks[1]
	assert.NotNil(t, task)
	assert.Equal(t, "Test task", task.Title)
	assert.Equal(t, domain.StatusTodo, task.Status)
}

func TestNewNewCommand_WithDescription(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newNewCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--title", "Test task", "--body", "Task description"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	task := repo.Tasks[1]
	assert.Equal(t, "Task description", task.Description)
}

func TestNewNewCommand_WithParent(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Parent task",
		Status: domain.StatusTodo,
	}
	repo.NextIDN = 2
	container := newTestContainer(repo)

	// Create command
	cmd := newNewCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--title", "Child task", "--parent", "1"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Created task #2")

	task := repo.Tasks[2]
	assert.NotNil(t, task.ParentID)
	assert.Equal(t, 1, *task.ParentID)
}

func TestNewNewCommand_WithLabels(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newNewCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--title", "Test task", "--label", "bug", "--label", "urgent"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	task := repo.Tasks[1]
	assert.Contains(t, task.Labels, "bug")
	assert.Contains(t, task.Labels, "urgent")
}

func TestNewNewCommand_WithIssue(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newNewCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--title", "Fix bug", "--issue", "42"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	task := repo.Tasks[1]
	assert.Equal(t, 42, task.Issue)
}

func TestNewNewCommand_WithBaseBranch(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newNewCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--title", "Test task", "--base", "develop"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	task := repo.Tasks[1]
	assert.Equal(t, "develop", task.BaseBranch)
}

func TestNewNewCommand_DefaultBaseBranch(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)
	// Mock git to return current branch
	container.Git = &testutil.MockGit{CurrentBranchName: "feature-branch"}

	// Create command
	cmd := newNewCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--title", "Test task"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	task := repo.Tasks[1]
	assert.Equal(t, "feature-branch", task.BaseBranch)
}

// =============================================================================
// List Command Tests
// =============================================================================

func TestNewListCommand_Empty(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newListCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "TITLE")
}

func TestNewListCommand_WithTasks(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "First task",
		Status: domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:     2,
		Title:  "Second task",
		Status: domain.StatusInProgress,
		Agent:  "claude",
	}
	container := newTestContainer(repo)

	// Create command
	cmd := newListCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "First task")
	assert.Contains(t, output, "Second task")
	assert.Contains(t, output, "todo")
	assert.Contains(t, output, "in_progress")
	assert.Contains(t, output, "claude")
}

// =============================================================================
// Show Command Tests
// =============================================================================

func TestNewShowCommand_ByID(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Test task",
		Description: "Task description",
		Status:      domain.StatusTodo,
		Created:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		BaseBranch:  "main",
	}
	container := newTestContainer(repo)

	// Create command
	cmd := newShowCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Task 1")
	assert.Contains(t, output, "Test task")
	assert.Contains(t, output, "Task description")
	assert.Contains(t, output, "todo")
}

func TestNewShowCommand_WithSubtasks(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	parentID := 1
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Parent task",
		Status:     domain.StatusInProgress,
		Created:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		BaseBranch: "main",
	}
	repo.Tasks[2] = &domain.Task{
		ID:         2,
		ParentID:   &parentID,
		Title:      "Child task 1",
		Status:     domain.StatusTodo,
		Created:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		BaseBranch: "main",
	}
	repo.Tasks[3] = &domain.Task{
		ID:         3,
		ParentID:   &parentID,
		Title:      "Child task 2",
		Status:     domain.StatusDone,
		Created:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		BaseBranch: "main",
	}
	container := newTestContainer(repo)

	// Create command
	cmd := newShowCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Parent task")
	assert.Contains(t, output, "Sub-tasks:")
	assert.Contains(t, output, "Child task 1")
	assert.Contains(t, output, "Child task 2")
}

func TestNewShowCommand_WithComments(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		Created:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		BaseBranch: "main",
	}
	repo.Comments[1] = []domain.Comment{
		{
			Text: "First comment",
			Time: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		},
	}
	container := newTestContainer(repo)

	// Create command
	cmd := newShowCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Comments:")
	assert.Contains(t, output, "First comment")
}

// =============================================================================
// Print Functions Tests
// =============================================================================

func TestPrintTaskList_Empty(t *testing.T) {
	var buf bytes.Buffer
	clock := &testutil.MockClock{NowTime: time.Now()}

	printTaskList(&buf, []*domain.Task{}, clock)

	// Should only have header
	expected := "ID   PARENT   STATUS   AGENT   LABELS   TITLE\n"
	assert.Equal(t, expected, buf.String())
}

func TestPrintTaskList_SingleTask(t *testing.T) {
	var buf bytes.Buffer
	clock := &testutil.MockClock{NowTime: time.Now()}

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
	clock := &testutil.MockClock{NowTime: time.Now()}

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
	clock := &testutil.MockClock{NowTime: time.Now()}

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
	clock := &testutil.MockClock{NowTime: time.Now()}

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
	clock := &testutil.MockClock{NowTime: now}

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
	clock := &testutil.MockClock{NowTime: time.Now()}

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
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
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
	task := repo.Tasks[1]
	assert.Equal(t, "Updated title", task.Title)
}

func TestNewEditCommand_UpdateDescription(t *testing.T) {
	// Setup mock repository
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
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
	cmd.SetArgs([]string{"1", "--body", "New description"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Updated task #1")

	// Verify description was updated
	task := repo.Tasks[1]
	assert.Equal(t, "New description", task.Description)
}

func TestNewEditCommand_AddLabels(t *testing.T) {
	// Setup mock repository
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
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
	task := repo.Tasks[1]
	assert.Contains(t, task.Labels, "existing")
	assert.Contains(t, task.Labels, "new")
	assert.Contains(t, task.Labels, "urgent")
}

func TestNewEditCommand_RemoveLabels(t *testing.T) {
	// Setup mock repository
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
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
	task := repo.Tasks[1]
	assert.Contains(t, task.Labels, "keep")
	assert.NotContains(t, task.Labels, "remove-me")
}

func TestNewEditCommand_MultipleUpdates(t *testing.T) {
	// Setup mock repository
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
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
	cmd.SetArgs([]string{"1", "--title", "New Title", "--body", "New desc", "--add-label", "new", "--rm-label", "old"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)

	// Verify all fields were updated
	task := repo.Tasks[1]
	assert.Equal(t, "New Title", task.Title)
	assert.Equal(t, "New desc", task.Description)
	assert.Contains(t, task.Labels, "new")
	assert.NotContains(t, task.Labels, "old")
}

// =============================================================================
// Rm Command Tests
// =============================================================================

func TestNewRmCommand_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to delete",
		Status: domain.StatusTodo,
	}
	container := newTestContainer(repo)

	// Create command
	cmd := newRmCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Deleted task #1")

	// Verify task was deleted
	_, exists := repo.Tasks[1]
	assert.False(t, exists, "task should be deleted from repository")
}

func TestNewRmCommand_WithHashPrefix(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to delete",
		Status: domain.StatusTodo,
	}
	container := newTestContainer(repo)

	// Create command
	cmd := newRmCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"#1"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Deleted task #1")

	// Verify task was deleted
	_, exists := repo.Tasks[1]
	assert.False(t, exists, "task should be deleted from repository")
}

func TestNewRmCommand_NotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newRmCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"999"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestNewRmCommand_InvalidID(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newRmCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"invalid"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestNewRmCommand_NoArgs(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newRmCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	// Execute
	err := cmd.Execute()

	// Assert - should fail due to missing argument
	assert.Error(t, err)
}

// =============================================================================
// Cp Command Tests
// =============================================================================

func TestNewCpCommand_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Original task",
		Description: "Task description",
		Status:      domain.StatusTodo,
		Labels:      []string{"bug"},
		BaseBranch:  "main",
	}
	repo.NextIDN = 2
	container := newTestContainer(repo)

	// Create command
	cmd := newCpCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Copied task #1 to #2")

	// Verify new task was created
	task := repo.Tasks[2]
	assert.NotNil(t, task)
	assert.Equal(t, "Original task (copy)", task.Title)
	assert.Equal(t, "Task description", task.Description)
	assert.Equal(t, domain.StatusTodo, task.Status)
}

func TestNewCpCommand_WithCustomTitle(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	repo.NextIDN = 2
	container := newTestContainer(repo)

	// Create command
	cmd := newCpCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1", "--title", "Custom new title"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Copied task #1 to #2")

	task := repo.Tasks[2]
	assert.Equal(t, "Custom new title", task.Title)
}

func TestNewCpCommand_SourceNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newCpCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"999"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestNewCpCommand_InvalidID(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newCpCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"invalid"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestNewCpCommand_NoArgs(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newCpCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	// Execute
	err := cmd.Execute()

	// Assert - should fail due to missing argument
	assert.Error(t, err)
}

// =============================================================================
// Comment Command Tests
// =============================================================================

func TestNewCommentCommand_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	container := newTestContainer(repo)

	// Create command
	cmd := newCommentCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1", "This is a test comment"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Added comment to task #1")

	// Verify comment was added
	comments := repo.Comments[1]
	assert.Len(t, comments, 1)
	assert.Equal(t, "This is a test comment", comments[0].Text)
}

func TestNewCommentCommand_WithHashPrefix(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	container := newTestContainer(repo)

	// Create command
	cmd := newCommentCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"#1", "Comment with hash prefix"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Added comment to task #1")

	// Verify comment was added
	comments := repo.Comments[1]
	assert.Len(t, comments, 1)
	assert.Equal(t, "Comment with hash prefix", comments[0].Text)
}

func TestNewCommentCommand_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newCommentCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"999", "Comment on missing task"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestNewCommentCommand_EmptyMessage(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}
	container := newTestContainer(repo)

	// Create command
	cmd := newCommentCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1", ""})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrEmptyMessage)
}

func TestNewCommentCommand_InvalidID(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newCommentCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"invalid", "Some message"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestNewCommentCommand_NoArgs(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newCommentCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	// Execute
	err := cmd.Execute()

	// Assert - should fail due to missing arguments
	assert.Error(t, err)
}

func TestNewCommentCommand_OnlyID(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	container := newTestContainer(repo)

	// Create command
	cmd := newCommentCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1"})

	// Execute
	err := cmd.Execute()

	// Assert - should fail due to missing message argument
	assert.Error(t, err)
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

// =============================================================================
// resolveTaskID Tests
// =============================================================================

func TestResolveTaskID_FromArgs(t *testing.T) {
	// When args are provided, git is not used
	id, err := resolveTaskID([]string{"42"}, nil)

	assert.NoError(t, err)
	assert.Equal(t, 42, id)
}

func TestResolveTaskID_FromArgsWithHash(t *testing.T) {
	// With # prefix
	id, err := resolveTaskID([]string{"#123"}, nil)

	assert.NoError(t, err)
	assert.Equal(t, 123, id)
}

func TestResolveTaskID_InvalidArg(t *testing.T) {
	_, err := resolveTaskID([]string{"invalid"}, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestResolveTaskID_NegativeArg(t *testing.T) {
	_, err := resolveTaskID([]string{"-5"}, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestResolveTaskID_ZeroArg(t *testing.T) {
	_, err := resolveTaskID([]string{"0"}, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestResolveTaskID_NoArgsNoGit(t *testing.T) {
	// No args and no git client
	_, err := resolveTaskID([]string{}, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task ID is required")
}

func TestResolveTaskID_FromCrewBranch(t *testing.T) {
	// Auto-detect from crew branch
	git := &testutil.MockGit{CurrentBranchName: "crew-42"}

	id, err := resolveTaskID([]string{}, git)

	assert.NoError(t, err)
	assert.Equal(t, 42, id)
}

func TestResolveTaskID_FromCrewBranchWithIssue(t *testing.T) {
	// Auto-detect from crew branch with issue suffix
	git := &testutil.MockGit{CurrentBranchName: "crew-7-gh-123"}

	id, err := resolveTaskID([]string{}, git)

	assert.NoError(t, err)
	assert.Equal(t, 7, id)
}

func TestResolveTaskID_NonCrewBranch(t *testing.T) {
	// On a non-crew branch (e.g., main)
	git := &testutil.MockGit{CurrentBranchName: "main"}

	_, err := resolveTaskID([]string{}, git)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task ID is required")
	assert.Contains(t, err.Error(), "main")
	assert.Contains(t, err.Error(), "not a crew branch")
}

func TestResolveTaskID_GitError(t *testing.T) {
	// Git returns an error
	git := &testutil.MockGit{CurrentBranchErr: assert.AnError}

	_, err := resolveTaskID([]string{}, git)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to detect current branch")
}
