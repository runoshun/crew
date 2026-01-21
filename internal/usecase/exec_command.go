package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ExecCommandInput contains the parameters for executing a command in a task's worktree.
// Fields are ordered to minimize memory padding.
type ExecCommandInput struct {
	Command []string // Command and arguments to execute (required)
	TaskID  int      // Task ID (required)
}

// ExecCommandOutput contains the result of executing a command.
type ExecCommandOutput struct {
	WorktreePath string // Path to the worktree where the command was executed
}

// ExecCommand is the use case for executing a command in a task's worktree.
type ExecCommand struct {
	tasks     domain.TaskRepository
	worktrees domain.WorktreeManager
}

// NewExecCommand creates a new ExecCommand use case.
func NewExecCommand(
	tasks domain.TaskRepository,
	worktrees domain.WorktreeManager,
) *ExecCommand {
	return &ExecCommand{
		tasks:     tasks,
		worktrees: worktrees,
	}
}

// Execute runs a command in the task's worktree directory.
func (uc *ExecCommand) Execute(_ context.Context, in ExecCommandInput) (*ExecCommandOutput, error) {
	// Validate input
	if len(in.Command) == 0 {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// Get task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	// Get worktree path
	branch := domain.BranchName(task.ID, task.Issue)
	wtPath, err := uc.worktrees.Resolve(branch)
	if err != nil {
		return nil, fmt.Errorf("resolve worktree: %w", err)
	}

	// Execute command in worktree directory
	//nolint:gosec // G204: Command is intentionally user-provided for exec functionality
	cmd := exec.Command(in.Command[0], in.Command[1:]...)
	cmd.Dir = wtPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("execute command: %w", err)
	}

	return &ExecCommandOutput{
		WorktreePath: wtPath,
	}, nil
}
