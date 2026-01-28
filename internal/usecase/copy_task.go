package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// CopyTaskInput contains the parameters for copying a task.
// Fields are ordered to minimize memory padding.
type CopyTaskInput struct {
	Title    *string // New title (optional, defaults to "<original> (copy)")
	SourceID int     // Source task ID to copy
	CopyAll  bool    // If true, copy comments and code state (branch/worktree)
}

// CopyTaskOutput contains the result of copying a task.
type CopyTaskOutput struct {
	TaskID int // The ID of the new task
}

// CopyTask is the use case for copying a task.
type CopyTask struct {
	tasks     domain.TaskRepository
	clock     domain.Clock
	worktrees domain.WorktreeManager
	git       domain.Git
}

// NewCopyTask creates a new CopyTask use case.
func NewCopyTask(tasks domain.TaskRepository, clock domain.Clock, worktrees domain.WorktreeManager, git domain.Git) *CopyTask {
	return &CopyTask{
		tasks:     tasks,
		clock:     clock,
		worktrees: worktrees,
		git:       git,
	}
}

// Execute copies a task with the given input.
// The new task copies: title (with " (copy)" suffix), description, labels.
// The new task does NOT copy: issue, PR, comments (unless CopyAll is true).
// The base branch is inherited from the source task.
func (uc *CopyTask) Execute(_ context.Context, in CopyTaskInput) (*CopyTaskOutput, error) {
	// Get source task
	source, err := uc.tasks.Get(in.SourceID)
	if err != nil {
		return nil, fmt.Errorf("get source task: %w", err)
	}
	if source == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Get next task ID
	id, err := uc.tasks.NextID()
	if err != nil {
		return nil, fmt.Errorf("generate task ID: %w", err)
	}

	// Determine title
	title := source.Title + " (copy)"
	if in.Title != nil {
		title = *in.Title
	}

	// Inherit base branch from source (copy uses same base)
	baseBranch := source.BaseBranch

	// Copy labels (create new slice to avoid sharing)
	var labels []string
	if len(source.Labels) > 0 {
		labels = make([]string, len(source.Labels))
		copy(labels, source.Labels)
	}

	// Create new task
	now := uc.clock.Now()
	task := &domain.Task{
		ID:          id,
		ParentID:    source.ParentID, // Inherit parent
		Title:       title,
		Description: source.Description,
		Status:      domain.StatusTodo,
		Created:     now,
		BaseBranch:  baseBranch,
		Labels:      labels,
		// NOT copied: Issue, PR, Agent, Session, Started
	}

	if in.CopyAll {
		if uc.worktrees == nil || uc.git == nil {
			return nil, domain.ErrCopyAllRequiresManagers
		}

		comments, err := uc.tasks.GetComments(in.SourceID)
		if err != nil {
			return nil, fmt.Errorf("get comments: %w", err)
		}
		copiedComments := make([]domain.Comment, len(comments))
		copy(copiedComments, comments)

		sourceBranch := domain.BranchName(source.ID, source.Issue)
		baseRef := sourceBranch
		branchExists, err := uc.git.BranchExists(sourceBranch)
		if err != nil {
			return nil, fmt.Errorf("check source branch: %w", err)
		}
		if !branchExists {
			baseRef, err = resolveBaseBranch(source, uc.git)
			if err != nil {
				return nil, err
			}
			baseBranch = baseRef
			task.BaseBranch = baseBranch
		}

		newBranch := domain.BranchName(id, 0)
		if _, err := uc.worktrees.Create(newBranch, baseRef); err != nil {
			return nil, fmt.Errorf("create worktree: %w", err)
		}

		if err := uc.tasks.SaveTaskWithComments(task, copiedComments); err != nil {
			var cleanupErrs []error
			if removeErr := uc.worktrees.Remove(newBranch); removeErr != nil {
				cleanupErrs = append(cleanupErrs, fmt.Errorf("remove worktree: %w", removeErr))
			}
			if deleteErr := uc.git.DeleteBranch(newBranch, true); deleteErr != nil {
				cleanupErrs = append(cleanupErrs, fmt.Errorf("delete branch: %w", deleteErr))
			}
			if len(cleanupErrs) > 0 {
				cleanupErr := errors.Join(cleanupErrs...)
				return nil, errors.Join(
					fmt.Errorf("save task: %w", err),
					fmt.Errorf("cleanup failed: %w", cleanupErr),
				)
			}
			return nil, fmt.Errorf("save task: %w", err)
		}

		return &CopyTaskOutput{TaskID: id}, nil
	}

	// Save task
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &CopyTaskOutput{TaskID: id}, nil
}
