package usecase

import (
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// resolveBaseBranch resolves the base branch for a task.
//
// Resolution priority:
//  1. task.BaseBranch (if not empty)
//  2. git.GetDefaultBranch()
//
// This helper centralizes the BaseBranch resolution logic used across
// StartTask, ShowDiff, and MergeTask usecases.
func resolveBaseBranch(task *domain.Task, git domain.Git) (string, error) {
	if task.BaseBranch != "" {
		return task.BaseBranch, nil
	}

	defaultBranch, err := git.GetDefaultBranch()
	if err != nil {
		return "", fmt.Errorf("get default branch: %w", err)
	}

	return defaultBranch, nil
}

// resolveNewTaskBaseBranch resolves the base branch for a new task.
//
// Resolution priority:
//  1. baseBranch parameter (if not empty)
//  2. config.Tasks.NewTaskBase:
//     - "default": use git.GetDefaultBranch()
//     - "" or "current" (default): use git.CurrentBranch()
//
// This helper centralizes the BaseBranch resolution logic for new task creation.
func resolveNewTaskBaseBranch(baseBranch string, git domain.Git, config *domain.Config) (string, error) {
	if baseBranch != "" {
		return baseBranch, nil
	}

	// Check config for new task base branch setting
	// Default is "current" (use current branch)
	if config != nil && config.Tasks.NewTaskBase == "default" {
		defaultBranch, err := git.GetDefaultBranch()
		if err != nil {
			return "", fmt.Errorf("get default branch: %w", err)
		}
		return defaultBranch, nil
	}

	// Use current branch (default behavior)
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}

	return currentBranch, nil
}
