package cli

import (
	"bytes"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestContainer creates an app.Container with mock dependencies.
func newTestContainer(repo *testutil.MockTaskRepository) *app.Container {
	logger := testutil.NewMockLogger()
	executor := testutil.NewMockCommandExecutor()
	container := app.NewWithDeps(
		app.Config{},
		repo,
		&testutil.MockStoreInitializer{},
		&testutil.MockClock{NowTime: time.Now()},
		logger,
		executor,
	)
	container.Git = &testutil.MockGit{}
	container.Worktrees = testutil.NewMockWorktreeManager()
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
	// Mock git to return feature-branch as current branch (now default for new tasks)
	container.Git = &testutil.MockGit{CurrentBranchName: testutil.StringPtr("feature-branch")}

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
	assert.Contains(t, output, "NAMESPACE")
	assert.Contains(t, output, "TITLE")
}

func TestNewListCommand_WithTasks(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:        1,
		Namespace: "test",
		Title:     "First task",
		Status:    domain.StatusTodo,
	}
	repo.Tasks[2] = &domain.Task{
		ID:        2,
		Namespace: "test",
		Title:     "Second task",
		Status:    domain.StatusInProgress,
		Agent:     "claude",
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
	assert.Contains(t, output, "To Do")
	assert.Contains(t, output, "In Progress")
	assert.Contains(t, output, "claude")
	assert.Contains(t, output, "test")
}

// =============================================================================
// Show Command Tests
// =============================================================================

func TestNewShowCommand_ByID(t *testing.T) {
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
	cmd.SetArgs([]string{"1", "--json"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "\"title\": \"Test task\"")
	assert.Contains(t, output, "\"description\": \"Task description\"")
}

func TestNewShowCommand_WithComments(t *testing.T) {
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
	cmd.SetArgs([]string{"1", "--json"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "\"text\": \"First comment\"")
}

func TestNewShowCommand_JSON(t *testing.T) {
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
	cmd.SetArgs([]string{"1", "--json"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "\"id\": 1")
	assert.Contains(t, output, "\"title\": \"Test task\"")
	assert.Contains(t, output, "\"status\": \"todo\"")
	assert.Contains(t, output, "\"branch\": \"crew-1\"")
	assert.Contains(t, output, "\"labels\": []")
	assert.Contains(t, output, "\"comments\": []")
}

func TestNewShowCommand_JSON_WithAllFields(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	parentID := 42
	started := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	lastReviewAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	lastReviewIsLGTM := true
	repo.Tasks[1] = &domain.Task{
		ID:               1,
		Title:            "Full task",
		Description:      "Detailed description",
		Status:           domain.StatusInProgress,
		Agent:            "test-agent",
		Created:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Started:          started,
		ParentID:         &parentID,
		Labels:           []string{"bug", "critical"},
		Issue:            123,
		ReviewCount:      2,
		LastReviewAt:     lastReviewAt,
		LastReviewIsLGTM: &lastReviewIsLGTM,
		BaseBranch:       "main",
	}
	repo.Comments[1] = []domain.Comment{
		{
			Time:     time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
			Text:     "Test comment",
			Type:     domain.CommentTypeReport,
			Tags:     []string{"docs"},
			Metadata: map[string]string{"source": "cli"},
		},
	}
	container := newTestContainer(repo)

	// Create command
	cmd := newShowCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1", "--json"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "\"id\": 1")
	assert.Contains(t, output, "\"title\": \"Full task\"")
	assert.Contains(t, output, "\"status\": \"in_progress\"")
	assert.Contains(t, output, "\"agent\": \"test-agent\"")
	assert.Contains(t, output, "\"branch\": \"crew-1-gh-123\"")
	assert.Contains(t, output, "\"parent_id\": 42")
	assert.Contains(t, output, "\"labels\": [\n    \"bug\",\n    \"critical\"\n  ]")
	assert.Contains(t, output, "\"issue\": 123")
	assert.Contains(t, output, "\"started\": \"2024-01-01T10:00:00Z\"")
	assert.Contains(t, output, "\"reviewCount\": 2")
	assert.Contains(t, output, "\"lastReviewAt\": \"2024-01-01T12:00:00Z\"")
	assert.Contains(t, output, "\"lastReviewIsLGTM\": true")
	assert.Contains(t, output, "\"text\": \"Test comment\"")
	assert.Contains(t, output, "\"type\": \"report\"")
	assert.Contains(t, output, "\"tags\": [")
	assert.Contains(t, output, "\"docs\"")
	assert.Contains(t, output, "\"metadata\": {")
}

// =============================================================================
// Print Functions Tests
// =============================================================================

func TestPrintTaskList_Empty(t *testing.T) {
	var buf bytes.Buffer
	clock := &testutil.MockClock{NowTime: time.Now()}

	printTaskList(&buf, []*domain.Task{}, clock)

	// Should only have header
	expected := "ID   NAMESPACE   PARENT   STATUS   AGENT   LABELS   TITLE\n"
	assert.Equal(t, expected, buf.String())
}

func TestPrintTaskList_SingleTask(t *testing.T) {
	var buf bytes.Buffer
	clock := &testutil.MockClock{NowTime: time.Now()}

	tasks := []*domain.Task{
		{
			ID:        1,
			Namespace: "test",
			Title:     "Test task",
			Status:    domain.StatusTodo,
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAMESPACE")
	assert.Contains(t, output, "PARENT")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "AGENT")
	assert.Contains(t, output, "LABELS")
	assert.Contains(t, output, "TITLE")
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "-") // PARENT is nil
	assert.Contains(t, output, "To Do")
	assert.Contains(t, output, "test")
	assert.Contains(t, output, "Test task")
}

func TestPrintTaskList_WithParent(t *testing.T) {
	var buf bytes.Buffer
	clock := &testutil.MockClock{NowTime: time.Now()}

	parentID := 1
	tasks := []*domain.Task{
		{
			ID:        2,
			Namespace: "test",
			ParentID:  &parentID,
			Title:     "Child task",
			Status:    domain.StatusTodo,
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
			ID:        1,
			Namespace: "test",
			Title:     "Task with agent",
			Status:    domain.StatusInProgress,
			Agent:     "claude",
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
			ID:        1,
			Namespace: "test",
			Title:     "Task with labels",
			Status:    domain.StatusTodo,
			Labels:    []string{"bug", "urgent"},
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
			ID:        1,
			Namespace: "test",
			Title:     "In progress task",
			Status:    domain.StatusInProgress,
			Started:   started,
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "In Progress (1h)")
}

func TestPrintTaskList_MultipleTasks(t *testing.T) {
	var buf bytes.Buffer
	clock := &testutil.MockClock{NowTime: time.Now()}

	parentID := 1
	tasks := []*domain.Task{
		{
			ID:        1,
			Namespace: "test",
			Title:     "Parent task",
			Status:    domain.StatusInProgress,
			Agent:     "claude",
			Labels:    []string{"feature"},
		},
		{
			ID:        2,
			Namespace: "test",
			ParentID:  &parentID,
			Title:     "Child task",
			Status:    domain.StatusTodo,
		},
		{
			ID:        3,
			Namespace: "test",
			Title:     "Done task",
			Status:    domain.StatusClosed,
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	// Verify all tasks are present
	assert.Contains(t, output, "Parent task")
	assert.Contains(t, output, "Child task")
	assert.Contains(t, output, "Done task")
	// Verify statuses
	assert.Contains(t, output, "In Progress")
	assert.Contains(t, output, "To Do")
	assert.Contains(t, output, "Closed (Abandoned)")
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

func TestNewSubstateCommand_Success(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1}
	container := newTestContainer(repo)

	cmd := newSubstateCommand(container)
	cmd.SetArgs([]string{"1", "awaiting_permission"})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Equal(t, domain.SubstateAwaitingPermission, repo.Tasks[1].ExecutionSubstate)
}

func TestNewSubstateCommand_InvalidSubstate(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1}
	container := newTestContainer(repo)

	cmd := newSubstateCommand(container)
	cmd.SetArgs([]string{"1", "bad"})

	err := cmd.Execute()

	assert.ErrorIs(t, err, domain.ErrInvalidExecutionSubstate)
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

func TestNewCpCommand_AllCopiesCommentsAndWorktree(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	repo.Comments[1] = []domain.Comment{
		{Text: "First", Time: time.Now(), Author: "worker"},
	}
	repo.NextIDN = 2
	container := newTestContainer(repo)

	// Create command
	cmd := newCpCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"1", "--all"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Copied task #1 to #2")
	assert.Equal(t, repo.Comments[1], repo.Comments[2])
	worktrees, ok := container.Worktrees.(*testutil.MockWorktreeManager)
	if assert.True(t, ok) {
		assert.True(t, worktrees.CreateCalled)
		assert.Equal(t, domain.BranchName(2, 0), worktrees.CreateBranch)
	}
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

func TestNewCommentCommand_WithTypeAndTags(t *testing.T) {
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
	cmd.SetArgs([]string{"1", "Typed comment", "--type", "report", "--tag", "docs", "--tags", "testing,refactoring"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.NoError(t, err)
	comments := repo.Comments[1]
	require.Len(t, comments, 1)
	assert.Equal(t, domain.CommentTypeReport, comments[0].Type)
	assert.Equal(t, []string{"docs", "refactoring", "testing"}, comments[0].Tags)
}

func TestNewCommentCommand_InvalidType(t *testing.T) {
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
	cmd.SetArgs([]string{"1", "Comment", "--type", "invalid"})

	// Execute
	err := cmd.Execute()

	// Assert
	assert.ErrorIs(t, err, domain.ErrInvalidCommentType)
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
		expected string
		duration time.Duration
	}{
		{"zero", "0s", 0},
		{"seconds", "30s", 30 * time.Second},
		{"just under minute", "59s", 59 * time.Second},
		{"one minute", "1m", 1 * time.Minute},
		{"minutes", "5m", 5 * time.Minute},
		{"just under hour", "59m", 59 * time.Minute},
		{"one hour", "1h", 1 * time.Hour},
		{"hours", "5h", 5 * time.Hour},
		{"just under day", "23h", 23 * time.Hour},
		{"one day", "1d", 24 * time.Hour},
		{"days", "3d", 3 * 24 * time.Hour},
		{"week", "7d", 7 * 24 * time.Hour},
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
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-42")}

	id, err := resolveTaskID([]string{}, git)

	assert.NoError(t, err)
	assert.Equal(t, 42, id)
}

func TestResolveTaskID_FromCrewBranchWithIssue(t *testing.T) {
	// Auto-detect from crew branch with issue suffix
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-7-gh-123")}

	id, err := resolveTaskID([]string{}, git)

	assert.NoError(t, err)
	assert.Equal(t, 7, id)
}

func TestResolveTaskID_NonCrewBranch(t *testing.T) {
	// On a non-crew branch (e.g., main)
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}

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
