package usecase

import (
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
// For tests that don't involve review requirements, we either:
// - set SkipReview=true, or
// - set ReviewCount to satisfy min_reviews
func newTestCompleteTask(
	repo *testutil.MockTaskRepository,
	sessions *testutil.MockSessionManager,
	worktrees *testutil.MockWorktreeManager,
	git *testutil.MockGit,
	configLoader *testutil.MockConfigLoader,
	clock *testutil.MockClock,
	executor *testutil.MockCommandExecutor,
) *CompleteTask {
	return NewCompleteTask(repo, sessions, worktrees, git, configLoader, clock, nil, executor, nil, "/tmp/crew", "/tmp/repo")
}

func TestCompleteTask_Execute_Success(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
	}{
		{"from in_progress", domain.StatusInProgress},
		{"from needs_input", domain.StatusInProgress},
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

			uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

			// Execute
			out, err := uc.Execute(context.Background(), CompleteTaskInput{
				TaskID: 1,
			})

			// Assert
			require.NoError(t, err)
			require.NotNil(t, out)
			assert.Equal(t, domain.StatusDone, out.Task.Status) // skip_review=true goes to done
			assert.False(t, out.ShouldStartReview)

			// Verify task is updated in repository
			savedTask := repo.Tasks[1]
			assert.Equal(t, domain.StatusDone, savedTask.Status)
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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

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
	assert.Equal(t, domain.StatusDone, out.Task.Status) // skip_review=true
}

func TestCompleteTask_Execute_CompleteCommandFails(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Task to complete",
		Status:      domain.StatusInProgress,
		ReviewCount: 1,
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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

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
		{"from in_review", domain.StatusDone},
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

			uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID:  1,
		Comment: "Implementation done",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusDone, out.Task.Status)

	// Verify comment is added
	comments, err := repo.GetComments(1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "Implementation done", comments[0].Text)
}

func TestCompleteTask_Execute_SkipReview_TaskLevel(t *testing.T) {
	// Test: task.SkipReview = true -> goes directly to done
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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusDone, out.Task.Status)
	assert.False(t, out.ShouldStartReview)
}

func TestCompleteTask_Execute_SkipReview_ConfigLevel(t *testing.T) {
	// Test: task.SkipReview = nil (not set), config.Tasks.SkipReview = true -> goes directly to done
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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusDone, out.Task.Status)
	assert.False(t, out.ShouldStartReview)
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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert - task.SkipReview=true should take precedence
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusDone, out.Task.Status)
	assert.False(t, out.ShouldStartReview)
}

func TestCompleteTask_Execute_SkipReview_TaskFalseOverridesConfigTrue(t *testing.T) {
	// Test: task.SkipReview = false (--no-skip-review) overrides config.Tasks.SkipReview = true
	// This is the key fix: explicit false should prevent skipping even when config says skip
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Task with skip_review explicitly set to false",
		Status:      domain.StatusInProgress,
		SkipReview:  boolPtr(false), // Task explicitly requires review (--no-skip-review)
		ReviewCount: 1,
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

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert - task.SkipReview=false should take precedence over config
	// Task goes to done (ready for review in new model)
	require.NoError(t, err)
	require.NotNil(t, out)
	// Status should be done (review requirement satisfied)
	assert.Equal(t, domain.StatusDone, out.Task.Status)
	assert.False(t, out.ShouldStartReview)
}

func TestCompleteTask_Execute_ReviewCountRequirement(t *testing.T) {
	// Insufficient reviews should fail without changing status or adding comments.
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Task requiring reviews",
		Status:      domain.StatusInProgress,
		SkipReview:  boolPtr(false),
		ReviewCount: 0,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config.Complete.MinReviews = 2
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, executor)

	_, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID:  1,
		Comment: "Ready to complete",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "review required: have 0, need 2")
	assert.Contains(t, err.Error(), "crew review 1")
	assert.Equal(t, domain.StatusInProgress, repo.Tasks[1].Status)
	assert.Empty(t, repo.Comments[1])

	// Sufficient reviews should allow completion.
	repo.Tasks[1].ReviewCount = 2

	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusDone, out.Task.Status)
	assert.False(t, out.ShouldStartReview)
}

func TestCompleteTask_Execute_MergeConflict(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with conflict",
		Status:     domain.StatusInProgress,
		SkipReview: boolPtr(true),
		BaseBranch: "main",
	}

	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
		MergeConflictFiles:     &[]string{"conflict.txt"},
		DefaultBranchName:      testutil.StringPtr("main"),
	}

	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, sessions, worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	require.ErrorIs(t, err, domain.ErrMergeConflict)
	require.NotNil(t, out)

	// Conflict message should be in output (not in comment)
	assert.Contains(t, out.ConflictMessage, "conflict.txt")
	assert.Contains(t, out.ConflictMessage, "git merge main")
	assert.Contains(t, out.ConflictMessage, "crew complete")
	assert.NotContains(t, out.ConflictMessage, "git fetch")

	// Task status should be in_progress
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusInProgress, task.Status)

	// No comment should be added (message is returned for stdout)
	comments := repo.Comments[1]
	assert.Empty(t, comments)
}

func TestCompleteTask_Execute_NoMergeConflict(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task without conflict",
		Status:     domain.StatusInProgress,
		SkipReview: boolPtr(true),
		BaseBranch: "main",
	}

	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
		MergeConflictFiles:     nil, // No conflicts
		DefaultBranchName:      testutil.StringPtr("main"),
	}

	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()

	uc := newTestCompleteTask(repo, sessions, worktrees, git, configLoader, clock, executor)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert - should succeed
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusDone, out.Task.Status)
}

func TestCompleteTask_Execute_DeprecatedReviewSettingsWarn(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with deprecated review settings",
		Status:     domain.StatusInProgress,
		SkipReview: boolPtr(true),
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config.Complete.ReviewMode = domain.ReviewModeAutoFix
	configLoader.Config.Complete.ReviewModeSet = true
	configLoader.Config.Complete.AutoFixSet = true

	clock := &testutil.MockClock{}
	executor := testutil.NewMockCommandExecutor()
	logger := testutil.NewMockLogger()

	uc := NewCompleteTask(repo, testutil.NewMockSessionManager(), worktrees, git, configLoader, clock, logger, executor, nil, "/tmp/crew", "/tmp/repo")

	_, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	require.NoError(t, err)

	var reviewModeWarned bool
	var autoFixWarned bool
	for _, entry := range logger.Entries {
		if entry.Level != "WARN" || entry.TaskID != 1 || entry.Category != "task" {
			continue
		}
		if entry.Msg == "complete.review_mode is deprecated and ignored" {
			reviewModeWarned = true
		}
		if entry.Msg == "complete.auto_fix is deprecated and ignored" {
			autoFixWarned = true
		}
	}
	assert.True(t, reviewModeWarned)
	assert.True(t, autoFixWarned)
}
