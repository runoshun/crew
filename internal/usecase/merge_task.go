package usecase

import (
	"context"
	"fmt"
	"os"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// MergeTaskInput contains the parameters for merging a task.
type MergeTaskInput struct {
	BaseBranch string // Target branch to merge into (defaults to task.BaseBranch or GetDefaultBranch())
	TaskID     int    // Task ID to merge
}

// MergeTaskOutput contains the result of merging a task.
type MergeTaskOutput struct {
	Task *domain.Task // The merged task
}

// MergeTask is the use case for merging a task branch into main.
type MergeTask struct {
	tasks     domain.TaskRepository
	sessions  domain.SessionManager
	worktrees domain.WorktreeManager
	git       domain.Git
	crewDir   string
}

// NewMergeTask creates a new MergeTask use case.
func NewMergeTask(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	worktrees domain.WorktreeManager,
	git domain.Git,
	crewDir string,
) *MergeTask {
	return &MergeTask{
		tasks:     tasks,
		sessions:  sessions,
		worktrees: worktrees,
		git:       git,
		crewDir:   crewDir,
	}
}

// Execute merges a task branch into the base branch.
// Preconditions:
// - Current branch is the base branch
// - Base branch's working tree is clean
//
// BaseBranch behavior:
// - If BaseBranch is specified (non-empty), uses it regardless of task.BaseBranch
// - If BaseBranch is empty, uses task.BaseBranch
// - If both are empty, uses GetDefaultBranch()
//
// Processing:
// 1. If session is running, stop it
// 2. Execute git merge --no-ff
// 3. Delete worktree
// 4. Delete branch
// 5. Update status to done
func (uc *MergeTask) Execute(_ context.Context, in MergeTaskInput) (*MergeTaskOutput, error) {
	// Get the task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Determine target base branch
	// Priority: in.BaseBranch > task.BaseBranch > GetDefaultBranch()
	targetBaseBranch := in.BaseBranch
	if targetBaseBranch == "" {
		var resolveErr error
		targetBaseBranch, resolveErr = ResolveBaseBranch(task, uc.git)
		if resolveErr != nil {
			return nil, resolveErr
		}
	}

	// Check current branch is the target base branch
	currentBranch, err := uc.git.CurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}
	if currentBranch != targetBaseBranch {
		// For backward compatibility, keep ErrNotOnMainBranch for "main" branch
		if targetBaseBranch == "main" {
			return nil, domain.ErrNotOnMainBranch
		}
		return nil, domain.ErrNotOnBaseBranch
	}

	// Check base branch's working tree is clean
	// Use repo root for checking base branch's working tree (empty string will be handled by the caller)
	hasChanges, err := uc.git.HasUncommittedChanges("")
	if err != nil {
		return nil, fmt.Errorf("check uncommitted changes: %w", err)
	}
	if hasChanges {
		return nil, domain.ErrUncommittedChanges
	}

	// Get branch name
	branch := domain.BranchName(task.ID, task.Issue)

	// Check if session is running
	sessionName := domain.SessionName(task.ID)
	running, err := uc.sessions.IsRunning(sessionName)
	if err != nil {
		return nil, fmt.Errorf("check session running: %w", err)
	}

	if running {
		// Always stop session before merge
		if stopErr := uc.sessions.Stop(sessionName); stopErr != nil {
			return nil, fmt.Errorf("stop session: %w", stopErr)
		}
		// Clean up script files (mirroring StopTask behavior)
		uc.cleanupScriptFiles(task.ID)
	}

	// Execute git merge --no-ff first (before deleting worktree)
	// This way, if merge fails due to conflict, worktree is preserved for resolution
	if mergeErr := uc.git.Merge(branch, true); mergeErr != nil {
		return nil, fmt.Errorf("merge branch: %w", mergeErr)
	}

	// Delete worktree if it exists (after successful merge)
	exists, err := uc.worktrees.Exists(branch)
	if err != nil {
		return nil, fmt.Errorf("check worktree exists: %w", err)
	}
	if exists {
		if err := uc.worktrees.Remove(branch); err != nil {
			return nil, fmt.Errorf("remove worktree: %w", err)
		}
	}

	// Delete the branch after merge
	if err := uc.git.DeleteBranch(branch, false); err != nil {
		return nil, fmt.Errorf("delete branch: %w", err)
	}

	// Update status to done
	task.Status = domain.StatusDone
	task.Agent = ""
	task.Session = ""

	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &MergeTaskOutput{Task: task}, nil
}

// cleanupScriptFiles removes the generated script files.
func (uc *MergeTask) cleanupScriptFiles(taskID int) {
	scriptPath := domain.ScriptPath(uc.crewDir, taskID)
	_ = os.Remove(scriptPath)
	promptPath := domain.PromptPath(uc.crewDir, taskID)
	_ = os.Remove(promptPath)
}
