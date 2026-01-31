package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ReviewTaskInput contains the parameters for reviewing a task.
// Fields are ordered to minimize memory padding.
type ReviewTaskInput struct {
	Agent   string // Agent name (optional, uses default reviewer if empty)
	Model   string // Model name override (optional, uses agent's default if empty)
	Message string // Additional instructions for the reviewer (optional)
	TaskID  int    // Task ID to review
	Verbose bool   // When true, stream reviewer output to stderr in real-time
}

// ReviewTaskOutput contains the result of reviewing a task.
// Fields are ordered to minimize memory padding.
type ReviewTaskOutput struct {
	Task   *domain.Task // The reviewed task
	Review string       // Review result from the agent
}

// ReviewTask is the use case for reviewing a task with an AI agent.
// Fields are ordered to minimize memory padding.
type ReviewTask struct {
	tasks        domain.TaskRepository
	worktrees    domain.WorktreeManager
	configLoader domain.ConfigLoader
	executor     domain.CommandExecutor
	clock        domain.Clock
	stderr       io.Writer
	repoRoot     string
}

// NewReviewTask creates a new ReviewTask use case.
func NewReviewTask(
	tasks domain.TaskRepository,
	worktrees domain.WorktreeManager,
	configLoader domain.ConfigLoader,
	executor domain.CommandExecutor,
	clock domain.Clock,
	repoRoot string,
	stderr io.Writer,
) *ReviewTask {
	return &ReviewTask{
		tasks:        tasks,
		worktrees:    worktrees,
		configLoader: configLoader,
		executor:     executor,
		clock:        clock,
		repoRoot:     repoRoot,
		stderr:       stderr,
	}
}

// Execute reviews a task using the specified AI agent.
func (uc *ReviewTask) Execute(ctx context.Context, in ReviewTaskInput) (*ReviewTaskOutput, error) {
	// Get task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	// Validate status - must be in_progress or done
	// done is allowed for re-review; review does not change status
	if task.Status != domain.StatusInProgress && task.Status != domain.StatusDone {
		return nil, fmt.Errorf("cannot review task in %s status (must be in_progress or done): %w", task.Status, domain.ErrInvalidTransition)
	}

	reviewCmd, err := shared.PrepareReviewCommand(shared.ReviewCommandDeps{
		ConfigLoader: uc.configLoader,
		Worktrees:    uc.worktrees,
		RepoRoot:     uc.repoRoot,
	}, shared.ReviewCommandInput{
		Task:    task,
		Agent:   in.Agent,
		Model:   in.Model,
		Message: in.Message,
	})
	if err != nil {
		return nil, err
	}

	return uc.executeSync(ctx, task, reviewCmd, in.Verbose)
}

// executeSync runs the review synchronously and returns the result.
func (uc *ReviewTask) executeSync(ctx context.Context, task *domain.Task, reviewCmd *shared.ReviewCommandOutput, verbose bool) (*ReviewTaskOutput, error) {
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
		Verbose:         verbose,
	})
	if err != nil {
		return nil, err
	}
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save review metadata: %w", err)
	}

	return &ReviewTaskOutput{
		Review: reviewOut.Review,
		Task:   task,
	}, nil
}
