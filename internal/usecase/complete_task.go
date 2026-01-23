package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// CompleteTaskInput contains the parameters for completing a task.
type CompleteTaskInput struct {
	Comment string // Optional completion comment
	TaskID  int    // Task ID to complete
}

// CompleteTaskOutput contains the result of completing a task.
// Fields are ordered to minimize memory padding.
type CompleteTaskOutput struct {
	Task              *domain.Task // The completed task
	AutoFixReview     string       // Review output when auto_fix runs synchronously
	AutoFixMaxRetries int          // Maximum retry count for auto_fix
	AutoFixRetryCount int          // Current retry count after auto_fix review
	ShouldStartReview bool         // True if review should be started by CLI (for background review)
	AutoFixEnabled    bool         // True if auto_fix mode is enabled
	AutoFixIsLGTM     bool         // True if auto_fix review passed
}

// CompleteTask is the use case for marking a task as complete.
// Status transitions depend on configuration:
//   - skip_review: directly to reviewed
//   - auto_fix: runs synchronous review and sets status to reviewed on LGTM
//   - normal: transitions to reviewing, signals CLI to start background review
//
// Fields are ordered to minimize memory padding.
type CompleteTask struct {
	tasks     domain.TaskRepository
	sessions  domain.SessionManager
	worktrees domain.WorktreeManager
	git       domain.Git
	config    domain.ConfigLoader
	clock     domain.Clock
	logger    domain.Logger
	executor  domain.CommandExecutor
	stderr    io.Writer
	crewDir   string
	repoRoot  string
}

// NewCompleteTask creates a new CompleteTask use case.
func NewCompleteTask(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	worktrees domain.WorktreeManager,
	git domain.Git,
	config domain.ConfigLoader,
	clock domain.Clock,
	logger domain.Logger,
	executor domain.CommandExecutor,
	stderr io.Writer,
	crewDir string,
	repoRoot string,
) *CompleteTask {
	return &CompleteTask{
		tasks:     tasks,
		sessions:  sessions,
		worktrees: worktrees,
		git:       git,
		config:    config,
		clock:     clock,
		logger:    logger,
		executor:  executor,
		stderr:    stderr,
		crewDir:   crewDir,
		repoRoot:  repoRoot,
	}
}

// ErrAutoFixMaxRetries is returned when auto_fix reaches the maximum retry limit.
var ErrAutoFixMaxRetries = errors.New("auto_fix reached maximum retries")

// Execute marks a task as complete.
// Preconditions:
//   - Status is in_progress or needs_input
//   - No uncommitted changes in worktree
//
// Processing:
//   - If [complete].command is configured, execute it (abort on failure)
//   - If skip_review: set status to reviewed
//   - If not skip_review: set status to reviewing (indicating review start)
func (uc *CompleteTask) Execute(ctx context.Context, in CompleteTaskInput) (*CompleteTaskOutput, error) {
	// Get the task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
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

	// Resolve base branch for conflict check
	baseBranch, err := resolveBaseBranch(task, uc.git)
	if err != nil {
		return nil, err
	}

	// Check for merge conflicts with base branch
	conflictHandler := shared.NewConflictHandler(uc.tasks, uc.sessions, uc.git, uc.clock)
	if conflictErr := conflictHandler.CheckAndHandle(shared.ConflictCheckInput{
		TaskID:     task.ID,
		Branch:     branch,
		BaseBranch: baseBranch,
	}); conflictErr != nil {
		return nil, conflictErr
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
	// Priority: task.SkipReview (if explicitly set) > config.Tasks.SkipReview > false
	var skipReview bool
	if task.SkipReview != nil {
		// Task has explicit setting, use it (respects --no-skip-review)
		skipReview = *task.SkipReview
	} else if cfg != nil {
		// Fall back to config setting
		skipReview = cfg.Tasks.SkipReview
	}
	// else: default false

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
			Task:              task,
			ShouldStartReview: false,
		}, nil
	}

	// Determine auto_fix mode
	autoFixEnabled := cfg != nil && cfg.Complete.AutoFix
	autoFixMaxRetries := domain.DefaultAutoFixMaxRetries
	if cfg != nil && cfg.Complete.AutoFixMaxRetries > 0 {
		autoFixMaxRetries = cfg.Complete.AutoFixMaxRetries
	}

	if autoFixEnabled {
		retryCount := task.AutoFixRetryCount
		if retryCount >= autoFixMaxRetries {
			return nil, fmt.Errorf("auto_fix: reached maximum retries (%d): %w", autoFixMaxRetries, ErrAutoFixMaxRetries)
		}

		reviewCmd, err := shared.PrepareReviewCommand(shared.ReviewCommandDeps{
			ConfigLoader: uc.config,
			Worktrees:    uc.worktrees,
			RepoRoot:     uc.repoRoot,
		}, shared.ReviewCommandInput{
			Task: task,
		})
		if err != nil {
			return nil, err
		}

		reviewOut, err := shared.ExecuteReview(ctx, shared.ReviewDeps{
			Tasks:    uc.tasks,
			Executor: uc.executor,
			Clock:    uc.clock,
			Stderr:   uc.stderr,
		}, shared.ReviewInput{
			Task:            task,
			WorktreePath:    reviewCmd.WorktreePath,
			Result:          reviewCmd.Result,
			SkipStatusCheck: true,
		})
		if err != nil {
			return nil, fmt.Errorf("task #%d completed, but review failed: %w", task.ID, err)
		}

		if reviewOut.IsLGTM {
			task.Status = domain.StatusReviewed
			task.AutoFixRetryCount = 0
		} else {
			task.AutoFixRetryCount = retryCount + 1
		}

		if saveErr := uc.tasks.Save(task); saveErr != nil {
			return nil, fmt.Errorf("save task: %w", saveErr)
		}

		if uc.logger != nil {
			if reviewOut.IsLGTM {
				uc.logger.Info(task.ID, "task", "completed (auto_fix LGTM, status: reviewed)")
			} else {
				uc.logger.Info(task.ID, "task", "completed (auto_fix review feedback, status: in_progress)")
			}
		}

		return &CompleteTaskOutput{
			Task:              task,
			ShouldStartReview: false,
			AutoFixEnabled:    true,
			AutoFixMaxRetries: autoFixMaxRetries,
			AutoFixReview:     reviewOut.Review,
			AutoFixIsLGTM:     reviewOut.IsLGTM,
			AutoFixRetryCount: task.AutoFixRetryCount,
		}, nil
	}

	// In normal mode, transition to reviewing to signal that review should start
	task.Status = domain.StatusReviewing

	// Save task
	if saveErr := uc.tasks.Save(task); saveErr != nil {
		return nil, fmt.Errorf("save task: %w", saveErr)
	}

	// Log task completion
	if uc.logger != nil {
		uc.logger.Info(task.ID, "task", "completed (status: reviewing, review should start)")
	}

	return &CompleteTaskOutput{
		Task:              task,
		ShouldStartReview: true,
		AutoFixEnabled:    false,
		AutoFixMaxRetries: autoFixMaxRetries,
	}, nil
}
