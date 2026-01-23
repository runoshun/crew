package usecase

import (
	"context"
	"fmt"

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
	ShouldStartReview bool         // True if review should be started by CLI (for background review)
	AutoFixEnabled    bool         // True if auto_fix mode is enabled
	AutoFixMaxRetries int          // Maximum retry count for auto_fix
}

// CompleteTask is the use case for marking a task as complete.
// It transitions the task from in_progress to reviewing (or reviewed if skip_review).
// If not skipping review, it signals to the CLI that review should be started.
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
		crewDir:   crewDir,
		repoRoot:  repoRoot,
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

	// In auto_fix mode, keep status as in_progress; CLI will change to reviewed on LGTM
	// In normal mode, transition to reviewing to signal that review should start
	if !autoFixEnabled {
		task.Status = domain.StatusReviewing
	}

	// Save task
	if saveErr := uc.tasks.Save(task); saveErr != nil {
		return nil, fmt.Errorf("save task: %w", saveErr)
	}

	// Log task completion
	if uc.logger != nil {
		if autoFixEnabled {
			uc.logger.Info(task.ID, "task", "completed (auto_fix pending review, status: in_progress)")
		} else {
			uc.logger.Info(task.ID, "task", "completed (status: reviewing, review should start)")
		}
	}

	// If auto_fix is enabled, CLI will run synchronous review instead of background
	return &CompleteTaskOutput{
		Task:              task,
		ShouldStartReview: !autoFixEnabled, // Background review only when auto_fix is off
		AutoFixEnabled:    autoFixEnabled,
		AutoFixMaxRetries: autoFixMaxRetries,
	}, nil
}
