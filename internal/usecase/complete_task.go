package usecase

import (
	"context"
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
	Task              *domain.Task      // The completed task
	ConflictMessage   string            // Conflict message to display (only set when ErrMergeConflict is returned)
	AutoFixReview     string            // Deprecated: auto_fix output (ignored)
	ReviewMode        domain.ReviewMode // Deprecated: review_mode (ignored)
	AutoFixMaxRetries int               // Deprecated: auto_fix setting (ignored)
	AutoFixRetryCount int               // Deprecated: auto_fix state (ignored)
	ShouldStartReview bool              // Deprecated: review auto-start (always false)
	AutoFixIsLGTM     bool              // Deprecated: auto_fix result (ignored)
}

// CompleteTask is the use case for marking a task as complete.
// Status transitions depend on configuration:
//   - skip_review: directly to done
//   - otherwise: require review count to meet min_reviews, then set done
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

// Execute marks a task as complete.
// Preconditions:
//   - Status is in_progress
//   - No uncommitted changes in worktree
//
// Processing:
//   - Validate review requirement (skip_review/min_reviews)
//   - Check for merge conflicts with base branch
//   - Run [complete].command if configured (abort on failure)
//   - Set status to done and save
func (uc *CompleteTask) Execute(ctx context.Context, in CompleteTaskInput) (*CompleteTaskOutput, error) {
	// Get the task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
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

	// Load config for completion checks
	cfg, err := uc.config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
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

	minReviews := domain.DefaultMinReviews
	if cfg != nil && cfg.Complete.MinReviews > 0 {
		minReviews = cfg.Complete.MinReviews
	}
	if cfg != nil && uc.logger != nil {
		if cfg.Complete.ReviewModeSet {
			uc.logger.Warn(task.ID, "task", "complete.review_mode is deprecated and ignored")
		}
		if cfg.Complete.AutoFixSet {
			uc.logger.Warn(task.ID, "task", "complete.auto_fix is deprecated and ignored")
		}
	}

	if !skipReview {
		if task.ReviewCount < minReviews {
			return nil, fmt.Errorf("review required: have %d, need %d (run \"crew review %d\")", task.ReviewCount, minReviews, task.ID)
		}
	}

	// Resolve base branch for conflict check
	baseBranch, err := resolveBaseBranch(task, uc.git)
	if err != nil {
		return nil, err
	}

	// Check for merge conflicts with base branch
	conflictHandler := shared.NewConflictHandler(uc.tasks, uc.sessions, uc.git, uc.clock)
	conflictOut, conflictErr := conflictHandler.CheckAndHandle(shared.ConflictCheckInput{
		TaskID:     task.ID,
		Branch:     branch,
		BaseBranch: baseBranch,
	})
	if conflictErr != nil {
		return &CompleteTaskOutput{ConflictMessage: conflictOut.Message}, conflictErr
	}

	if cfg != nil && cfg.Complete.Command != "" {
		// Execute the complete command using CommandExecutor
		cmd := domain.NewShellCommand(cfg.Complete.Command, worktreePath)
		output, execErr := uc.executor.Execute(cmd)
		if execErr != nil {
			return nil, fmt.Errorf("[complete].command failed: %s: %w", string(output), execErr)
		}
	}

	// Add comment if provided (only after completion conditions are met)
	if in.Comment != "" {
		comment := domain.Comment{
			Text: in.Comment,
			Time: uc.clock.Now(),
		}
		if commentErr := uc.tasks.AddComment(task.ID, comment); commentErr != nil {
			return nil, fmt.Errorf("add comment: %w", commentErr)
		}
	}

	task.Status = domain.StatusDone

	if saveErr := uc.tasks.Save(task); saveErr != nil {
		return nil, fmt.Errorf("save task: %w", saveErr)
	}

	if uc.logger != nil {
		if skipReview {
			uc.logger.Info(task.ID, "task", "completed (status: done, skip_review: true)")
		} else {
			uc.logger.Info(task.ID, "task", "completed (status: done, review requirement satisfied)")
		}
	}

	return &CompleteTaskOutput{
		Task:              task,
		ShouldStartReview: false,
		ReviewMode:        domain.ReviewModeAuto,
	}, nil
}
