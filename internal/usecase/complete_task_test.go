package usecase

import (
	"bytes"
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool {
	return &b
}

// newTestCompleteTask creates a CompleteTask use case for testing.
// skip_review tests need actual review to start, so we use a simpler approach:
// - For tests that don't involve review, we skip the review part by setting skipReview=true on the task
// - For tests that test the skip_review flag, we check the output status directly
func newTestCompleteTask(
	repo *testutil.MockTaskRepository,
	worktrees *testutil.MockWorktreeManager,
	git *testutil.MockGit,
	configLoader *testutil.MockConfigLoader,
	clock *testutil.MockClock,
	executor *testutil.MockCommandExecutor,
) *CompleteTask {
	sessions := testutil.NewMockSessionManager()
	var stdout, stderr bytes.Buffer
	return NewCompleteTask(repo, worktrees, sessions, git, configLoader, clock, nil, executor, "/tmp/crew", "/tmp/repo", &stdout, &stderr)
}

func TestCompleteTask_Execute_Success(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
	}{
		{"from in_progress", domain.StatusInProgress},
		{"from needs_input", domain.StatusNeedsInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repo := testutil.NewMockTaskRepository()
			repo.Tasks[1] = &domain.Task{
				ID:         1,
				Title:      "Task to complete",
				Status:     tt.status,
				SkipReview: boolPtr(true), // Skip review for basic test
			}

			worktrees := testutil.NewMockWorktreeManager()
			worktrees.ResolvePath = "/tmp/worktree"

			git := &testutil.MockGit{
				HasUncommittedChangesV: false, // No uncommitted changes
			}

			configLoader := testutil.NewMockConfigLoader()
			clock := &testutil.MockClock{}
			executor := testutil.NewMockCommandExecutor()

			uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

			// Execute
			out, err := uc.Execute(context.Background(), CompleteTaskInput{
				TaskID: 1,
			})

			// Assert
			require.NoError(t, err)
			require.NotNil(t, out)
			assert.Equal(t, domain.StatusReviewed, out.Task.Status) // skip_review=true goes to reviewed
			assert.False(t, out.ReviewStarted)

			// Verify task is updated in repository
			savedTask := repo.Tasks[1]
			assert.Equal(t, domain.StatusReviewed, savedTask.Status)
		})
	}
}

func TestCompleteTask_Execute_WithCompleteCommand(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to complete",
		Status:     domain.StatusInProgress,
		SkipReview: boolPtr(true), // Skip review for this test
	}

	// Use actual directory for test
	testDir := t.TempDir()

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = testDir

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config = &domain.Config{
		Complete: domain.CompleteConfig{
			Command: "echo 'Running CI'",
		},
	}
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, executor.ExecuteCalled, "complete.command should be executed")
	assert.Equal(t, "sh", executor.ExecutedCmd.Program)
	assert.Equal(t, []string{"-c", "echo 'Running CI'"}, executor.ExecutedCmd.Args)
	assert.Equal(t, testDir, executor.ExecutedCmd.Dir)
	assert.Equal(t, domain.StatusReviewed, out.Task.Status) // skip_review=true
}

func TestCompleteTask_Execute_CompleteCommandFails(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to complete",
		Status: domain.StatusInProgress,
	}

	// Use actual directory for test
	testDir := t.TempDir()

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = testDir

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config = &domain.Config{
		Complete: domain.CompleteConfig{
			Command: "false", // Command that fails
		},
	}
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()
	executor.ExecuteErr = assert.AnError
	executor.ExecuteOutput = []byte("command failed")

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	_, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "[complete].command failed")

	// Verify task status was NOT changed
	savedTask := repo.Tasks[1]
	assert.Equal(t, domain.StatusInProgress, savedTask.Status)
}

func TestCompleteTask_Execute_UncommittedChanges(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to complete",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: true, // Has uncommitted changes
	}

	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	_, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrUncommittedChanges)

	// Verify task status was NOT changed
	savedTask := repo.Tasks[1]
	assert.Equal(t, domain.StatusInProgress, savedTask.Status)
}

func TestCompleteTask_Execute_NotInProgress(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
	}{
		{"from todo", domain.StatusTodo},
		{"from in_review", domain.StatusForReview},
		{"from error", domain.StatusError},
		{"from done", domain.StatusClosed},
		{"from closed", domain.StatusClosed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repo := testutil.NewMockTaskRepository()
			repo.Tasks[1] = &domain.Task{
				ID:     1,
				Title:  "Task",
				Status: tt.status,
			}

			worktrees := testutil.NewMockWorktreeManager()
			git := &testutil.MockGit{}
			configLoader := testutil.NewMockConfigLoader()
			clock := &testutil.MockClock{}
			executor := testutil.NewMockCommandExecutor()

			uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

			// Execute
			_, err := uc.Execute(context.Background(), CompleteTaskInput{
				TaskID: 1,
			})

			// Assert
			assert.ErrorIs(t, err, domain.ErrInvalidTransition)
		})
	}
}

func TestCompleteTask_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	_, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestCompleteTask_Execute_GetError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError

	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	_, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestCompleteTask_Execute_WorktreeResolveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to complete",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolveErr = assert.AnError

	git := &testutil.MockGit{}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	_, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resolve worktree")
}

func TestCompleteTask_Execute_HasUncommittedChangesError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to complete",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedErr: assert.AnError,
	}

	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	_, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check uncommitted changes")
}

func TestCompleteTask_Execute_ConfigLoadError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to complete",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	configLoader.LoadErr = assert.AnError
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	_, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestCompleteTask_Execute_SaveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to complete",
		Status:     domain.StatusInProgress,
		SkipReview: boolPtr(true),
	}
	repo.SaveErr = assert.AnError

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	_, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")
}

func TestCompleteTask_Execute_WithComment(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to complete",
		Status:     domain.StatusInProgress,
		SkipReview: boolPtr(true),
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID:  1,
		Comment: "Implementation done",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusReviewed, out.Task.Status)

	// Verify comment is added
	comments, err := repo.GetComments(1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "Implementation done", comments[0].Text)
}

func TestCompleteTask_Execute_SkipReview_TaskLevel(t *testing.T) {
	// Test: task.SkipReview = true -> goes directly to reviewed
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with skip_review",
		Status:     domain.StatusInProgress,
		SkipReview: boolPtr(true),
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusReviewed, out.Task.Status)
	assert.False(t, out.ReviewStarted)
}

func TestCompleteTask_Execute_SkipReview_ConfigLevel(t *testing.T) {
	// Test: task.SkipReview = nil (not set), config.Tasks.SkipReview = true -> goes directly to reviewed
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task without skip_review flag",
		Status:     domain.StatusInProgress,
		SkipReview: nil, // Not set on task
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config = &domain.Config{
		Tasks: domain.TasksConfig{
			SkipReview: true, // Set at config level
		},
	}
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusReviewed, out.Task.Status)
	assert.False(t, out.ReviewStarted)
}

func TestCompleteTask_Execute_SkipReview_TaskTrueOverridesConfigFalse(t *testing.T) {
	// Test: task.SkipReview = true overrides config.Tasks.SkipReview = false
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with skip_review explicitly set to true",
		Status:     domain.StatusInProgress,
		SkipReview: boolPtr(true), // Task level override
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config = &domain.Config{
		Tasks: domain.TasksConfig{
			SkipReview: false, // Config says don't skip
		},
	}
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert - task.SkipReview=true should take precedence
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusReviewed, out.Task.Status)
	assert.False(t, out.ReviewStarted)
}

func TestCompleteTask_Execute_SkipReview_TaskFalseOverridesConfigTrue(t *testing.T) {
	// Test: task.SkipReview = false (--no-skip-review) overrides config.Tasks.SkipReview = true
	// This is the key fix: explicit false should prevent skipping even when config says skip
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with skip_review explicitly set to false",
		Status:     domain.StatusInProgress,
		SkipReview: boolPtr(false), // Task explicitly requires review (--no-skip-review)
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config = &domain.Config{
		Tasks: domain.TasksConfig{
			SkipReview: true, // Config says skip, but task overrides
		},
	}
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert - task.SkipReview=false should take precedence over config
	// Task goes to for_review (review will fail to start in mock, but status should be for_review)
	require.NoError(t, err)
	require.NotNil(t, out)
	// Since review fails in mock, status stays for_review
	assert.Equal(t, domain.StatusForReview, out.Task.Status)
	assert.False(t, out.ReviewStarted)
}
