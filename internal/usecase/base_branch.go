package usecase

import (
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ResolveBaseBranch resolves the base branch for a task.
//
// Resolution priority:
//  1. task.BaseBranch (if not empty)
//  2. git.GetDefaultBranch()
//
// This helper centralizes the BaseBranch resolution logic used across
// NewTask, StartTask, ShowDiff, and MergeTask usecases.
func ResolveBaseBranch(task *domain.Task, git domain.Git) (string, error) {
	if task.BaseBranch != "" {
		return task.BaseBranch, nil
	}

	defaultBranch, err := git.GetDefaultBranch()
	if err != nil {
		return "", fmt.Errorf("get default branch: %w", err)
	}

	return defaultBranch, nil
}
