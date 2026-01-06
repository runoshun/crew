package usecase

import (
	"context"
	"os/exec"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
				ID:     1,
				Title:  "Task to complete",
				Status: tt.status,
			}

			worktrees := testutil.NewMockWorktreeManager()
			worktrees.ResolvePath = "/tmp/worktree"

			git := &testutil.MockGit{
				HasUncommittedChangesV: false, // No uncommitted changes
			}

			configLoader := testutil.NewMockConfigLoader()
			clock := &testutil.MockClock{}

			uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)

			// Execute
			out, err := uc.Execute(context.Background(), CompleteTaskInput{
				TaskID: 1,
			})

			// Assert
			require.NoError(t, err)
			require.NotNil(t, out)
			assert.Equal(t, domain.StatusInReview, out.Task.Status)

			// Verify task is updated in repository
			savedTask := repo.Tasks[1]
			assert.Equal(t, domain.StatusInReview, savedTask.Status)
		})
	}
}

func TestCompleteTask_Execute_WithCompleteCommand(t *testing.T) {
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
			Command: "echo 'Running CI'",
		},
	}
	clock := &testutil.MockClock{}

	uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)

	// Track if command was executed
	cmdExecuted := false
	uc.SetExecCmd(func(name string, args ...string) *exec.Cmd {
		cmdExecuted = true
		// Return a command that succeeds
		return exec.Command("true")
	})

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, cmdExecuted, "complete.command should be executed")
	assert.Equal(t, domain.StatusInReview, out.Task.Status)
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

	uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)
	// Use actual exec so "false" command fails
	// (default execCmd is exec.Command)

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

	uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)

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
		{"from in_review", domain.StatusInReview},
		{"from error", domain.StatusError},
		{"from done", domain.StatusDone},
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

			uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)

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

	uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)

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

	uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)

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

	uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)

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

	uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)

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

	uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)

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
		ID:     1,
		Title:  "Task to complete",
		Status: domain.StatusInProgress,
	}
	repo.SaveErr = assert.AnError

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	git := &testutil.MockGit{
		HasUncommittedChangesV: false,
	}

	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{}

	uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)

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
	clock := &testutil.MockClock{}

	uc := NewCompleteTask(repo, worktrees, git, configLoader, clock)

	// Execute
	out, err := uc.Execute(context.Background(), CompleteTaskInput{
		TaskID:  1,
		Comment: "Implementation done",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusInReview, out.Task.Status)

	// Verify comment is added
	comments, err := repo.GetComments(1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "Implementation done", comments[0].Text)
}
