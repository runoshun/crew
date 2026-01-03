package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestExecCommand_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	worktrees := testutil.NewMockWorktreeManager()

	uc := NewExecCommand(repo, worktrees)

	// Execute
	_, err := uc.Execute(context.Background(), ExecCommandInput{
		TaskID:  999,
		Command: []string{"echo", "test"},
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestExecCommand_Execute_EmptyCommand(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	worktrees := testutil.NewMockWorktreeManager()

	uc := NewExecCommand(repo, worktrees)

	// Execute
	_, err := uc.Execute(context.Background(), ExecCommandInput{
		TaskID:  1,
		Command: []string{},
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command cannot be empty")
}

func TestExecCommand_Execute_WorktreeNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolveErr = domain.ErrWorktreeNotFound

	uc := NewExecCommand(repo, worktrees)

	// Execute
	_, err := uc.Execute(context.Background(), ExecCommandInput{
		TaskID:  1,
		Command: []string{"echo", "test"},
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resolve worktree")
}

func TestExecCommand_Execute_GetTaskError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	worktrees := testutil.NewMockWorktreeManager()

	uc := NewExecCommand(repo, worktrees)

	// Execute
	_, err := uc.Execute(context.Background(), ExecCommandInput{
		TaskID:  1,
		Command: []string{"echo", "test"},
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

// Note: We cannot easily test the actual command execution in unit tests
// because it requires running external commands. Integration tests would be
// better suited for testing the actual execution behavior.
// The tests above cover the main error cases and input validation.
