package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// CompleteTaskInput contains the parameters for completing a task.
type CompleteTaskInput struct {
	Comment string // Optional completion comment
	TaskID  int    // Task ID to complete
}

// CompleteTaskOutput contains the result of completing a task.
// Fields are ordered to minimize memory padding.
type CompleteTaskOutput struct {
	Task          *domain.Task // The completed task
	ReviewSession string       // Review session name (if ReviewStarted is true)
	ReviewStarted bool         // True if review was automatically started
}

// CompleteTask is the use case for marking a task as complete.
// It transitions the task from in_progress to for_review (or reviewed if skip_review).
// If not skipping review, it automatically starts the review process.
// Fields are ordered to minimize memory padding.
type CompleteTask struct {
	tasks     domain.TaskRepository
	worktrees domain.WorktreeManager
	sessions  domain.SessionManager
	git       domain.Git
	config    domain.ConfigLoader
	clock     domain.Clock
	logger    domain.Logger
	executor  domain.CommandExecutor
	stdout    io.Writer
	stderr    io.Writer
	crewDir   string
	repoRoot  string
}

// NewCompleteTask creates a new CompleteTask use case.
func NewCompleteTask(
	tasks domain.TaskRepository,
	worktrees domain.WorktreeManager,
	sessions domain.SessionManager,
	git domain.Git,
	config domain.ConfigLoader,
	clock domain.Clock,
	logger domain.Logger,
	executor domain.CommandExecutor,
	crewDir string,
	repoRoot string,
	stdout, stderr io.Writer,
) *CompleteTask {
	return &CompleteTask{
		tasks:     tasks,
		worktrees: worktrees,
		sessions:  sessions,
		git:       git,
		config:    config,
		clock:     clock,
		logger:    logger,
		executor:  executor,
		crewDir:   crewDir,
		repoRoot:  repoRoot,
		stdout:    stdout,
		stderr:    stderr,
	}
}

// Execute marks a task as complete.
// Preconditions:
//   - Status is in_progress or needs_input
//   - No uncommitted changes in worktree
//
// Processing:
//   - If [complete].command is configured, execute it (abort on failure)
//   - If skip_review: set status to reviewed
//   - If not skip_review: set status to for_review, then start review automatically
func (uc *CompleteTask) Execute(ctx context.Context, in CompleteTaskInput) (*CompleteTaskOutput, error) {
	// Get the task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Validate status - must be in_progress or needs_input
	if task.Status != domain.StatusInProgress && task.Status != domain.StatusNeedsInput {
		return nil, fmt.Errorf("cannot complete task in %s status (must be in_progress or needs_input): %w", task.Status, domain.ErrInvalidTransition)
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
		// Execute the complete command using CommandExecutor
		cmd := domain.NewShellCommand(cfg.Complete.Command, worktreePath)
		output, execErr := uc.executor.Execute(cmd)
		if execErr != nil {
			return nil, fmt.Errorf("[complete].command failed: %s: %w", string(output), execErr)
		}
	}

	// Determine if we should skip review
	// Priority: task.SkipReview > config.Tasks.SkipReview > false
	skipReview := task.SkipReview
	if !skipReview && cfg != nil {
		skipReview = cfg.Tasks.SkipReview
	}

	// Add comment if provided
	if in.Comment != "" {
		comment := domain.Comment{
			Text: in.Comment,
			Time: uc.clock.Now(),
		}
		if commentErr := uc.tasks.AddComment(task.ID, comment); commentErr != nil {
			return nil, fmt.Errorf("add comment: %w", commentErr)
		}
	}

	if skipReview {
		// Skip review: go directly to reviewed status
		task.Status = domain.StatusReviewed

		// Save task
		if saveErr := uc.tasks.Save(task); saveErr != nil {
			return nil, fmt.Errorf("save task: %w", saveErr)
		}

		// Log task completion
		if uc.logger != nil {
			uc.logger.Info(task.ID, "task", "completed (status: reviewed, skip_review: true)")
		}

		return &CompleteTaskOutput{
			Task:          task,
			ReviewStarted: false,
		}, nil
	}

	// Not skipping review: transition to for_review first, then start review
	task.Status = domain.StatusForReview

	// Save task with for_review status
	if saveErr := uc.tasks.Save(task); saveErr != nil {
		return nil, fmt.Errorf("save task: %w", saveErr)
	}

	// Start review automatically using ReviewTask
	reviewUC := NewReviewTask(
		uc.tasks,
		uc.sessions,
		uc.worktrees,
		uc.config,
		uc.executor,
		uc.clock,
		uc.logger,
		uc.crewDir,
		uc.repoRoot,
		uc.stdout,
		uc.stderr,
	)

	reviewOut, reviewErr := reviewUC.Execute(ctx, ReviewTaskInput{
		TaskID: task.ID,
		Wait:   false, // Background execution
	})
	if reviewErr != nil {
		// Review failed to start, but completion was successful
		// Log the error but don't fail the completion
		if uc.logger != nil {
			uc.logger.Info(task.ID, "task", fmt.Sprintf("completed (status: for_review), review start failed: %v", reviewErr))
		}

		// Re-fetch task to get latest state (ReviewTask may have modified it)
		task, _ = uc.tasks.Get(in.TaskID)

		return &CompleteTaskOutput{
			Task:          task,
			ReviewStarted: false,
		}, nil
	}

	// Log task completion with review started
	if uc.logger != nil {
		uc.logger.Info(task.ID, "task", fmt.Sprintf("completed and review started (session: %s)", reviewOut.SessionName))
	}

	return &CompleteTaskOutput{
		Task:          reviewOut.Task,
		ReviewStarted: true,
		ReviewSession: reviewOut.SessionName,
	}, nil
}
