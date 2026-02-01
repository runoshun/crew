package usecase

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

func ensureACPWorktree(
	task *domain.Task,
	cfg *domain.Config,
	agent domain.Agent,
	worktrees domain.WorktreeManager,
	git domain.Git,
	runner domain.ScriptRunner,
	repoRoot string,
) (string, bool, error) {
	branch := domain.BranchName(task.ID, task.Issue)
	exists, err := worktrees.Exists(branch)
	if err != nil {
		return "", false, fmt.Errorf("check worktree: %w", err)
	}
	if exists {
		wtPath, resolveErr := worktrees.Resolve(branch)
		if resolveErr != nil {
			return "", false, fmt.Errorf("resolve worktree: %w", resolveErr)
		}
		return wtPath, false, nil
	}

	baseBranch, err := resolveBaseBranch(task, git)
	if err != nil {
		return "", false, err
	}

	wtPath, err := worktrees.Create(branch, baseBranch)
	if err != nil {
		return "", false, fmt.Errorf("create worktree: %w", err)
	}

	if setupErr := worktrees.SetupWorktree(wtPath, &cfg.Worktree); setupErr != nil {
		_ = worktrees.Remove(branch)
		return "", false, fmt.Errorf("setup worktree: %w", setupErr)
	}

	if setupErr := setupACPAgent(task, wtPath, agent, runner, repoRoot); setupErr != nil {
		_ = worktrees.Remove(branch)
		return "", false, fmt.Errorf("setup agent: %w", setupErr)
	}

	return wtPath, true, nil
}

func setupACPAgent(
	task *domain.Task,
	wtPath string,
	agent domain.Agent,
	runner domain.ScriptRunner,
	repoRoot string,
) error {
	if agent.SetupScript == "" {
		return nil
	}

	gitDir := filepath.Join(repoRoot, ".git")
	data := struct {
		GitDir   string
		RepoRoot string
		Worktree string
		TaskID   int
	}{
		GitDir:   gitDir,
		RepoRoot: repoRoot,
		Worktree: wtPath,
		TaskID:   task.ID,
	}

	tmpl, err := template.New("acp-setup").Parse(agent.SetupScript)
	if err != nil {
		return fmt.Errorf("parse setup script template: %w", err)
	}

	var script strings.Builder
	if err := tmpl.Execute(&script, data); err != nil {
		return fmt.Errorf("expand setup script template: %w", err)
	}

	if err := runner.Run(wtPath, script.String()); err != nil {
		return fmt.Errorf("run setup script: %w", err)
	}

	return nil
}
