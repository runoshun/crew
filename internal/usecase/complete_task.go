package usecase

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// CompleteTaskInput contains the parameters for completing a task.
type CompleteTaskInput struct {
	TaskID int // Task ID to complete
}

// CompleteTaskOutput contains the result of completing a task.
type CompleteTaskOutput struct {
	Task *domain.Task // The completed task
}

// CompleteTask is the use case for marking a task as complete.
// It transitions the task from in_progress to in_review.
type CompleteTask struct {
	tasks     domain.TaskRepository
	worktrees domain.WorktreeManager
	git       domain.Git
	config    domain.ConfigLoader
	execCmd   func(name string, args ...string) *exec.Cmd
}

// NewCompleteTask creates a new CompleteTask use case.
func NewCompleteTask(
	tasks domain.TaskRepository,
	worktrees domain.WorktreeManager,
	git domain.Git,
	config domain.ConfigLoader,
) *CompleteTask {
	return &CompleteTask{
		tasks:     tasks,
		worktrees: worktrees,
		git:       git,
		config:    config,
		execCmd:   exec.Command,
	}
}

// SetExecCmd sets a custom exec.Cmd factory for testing.
func (uc *CompleteTask) SetExecCmd(fn func(name string, args ...string) *exec.Cmd) {
	uc.execCmd = fn
}

// Execute marks a task as complete.
// Preconditions:
//   - Status is in_progress
//   - No uncommitted changes in worktree
//
// Processing:
//   - If [complete].command is configured, execute it (abort on failure)
//   - Update status to in_review
func (uc *CompleteTask) Execute(_ context.Context, in CompleteTaskInput) (*CompleteTaskOutput, error) {
	// Get the task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Validate status - must be in_progress
	if task.Status != domain.StatusInProgress {
		return nil, fmt.Errorf("cannot complete task in %s status (must be in_progress): %w", task.Status, domain.ErrInvalidTransition)
	}

	// Resolve worktree path
	branch := domain.BranchName(task.ID, task.Issue)
	worktreePath, err := uc.worktrees.Resolve(branch)
	if err != nil {
		return nil, fmt.Errorf("resolve worktree: %w", err)
	}

	// Check for uncommitted changes
	hasUncommitted, err := uc.git.HasUncommittedChanges(worktreePath)
	if err != nil {
		return nil, fmt.Errorf("check uncommitted changes: %w", err)
	}
	if hasUncommitted {
		return nil, domain.ErrUncommittedChanges
	}

	// Load config and execute complete.command if configured
	cfg, err := uc.config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if cfg != nil && cfg.Complete.Command != "" {
		// Execute the complete command
		cmd := uc.execCmd("sh", "-c", cfg.Complete.Command)
		cmd.Dir = worktreePath
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("[complete].command failed: %s: %w", string(output), err)
		}
	}

	// Update status to in_review
	task.Status = domain.StatusInReview

	// Save task
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &CompleteTaskOutput{Task: task}, nil
}
