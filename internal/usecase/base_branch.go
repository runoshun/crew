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
//  2. git.GetNewTaskBaseBranch()
//
// This helper centralizes the BaseBranch resolution logic for new task creation.
func resolveNewTaskBaseBranch(baseBranch string, git domain.Git) (string, error) {
	if baseBranch != "" {
		return baseBranch, nil
	}

	newTaskBaseBranch, err := git.GetNewTaskBaseBranch()
	if err != nil {
		return "", fmt.Errorf("get new task base branch: %w", err)
	}

	return newTaskBaseBranch, nil
}
