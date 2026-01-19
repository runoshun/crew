package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeTask_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to merge",
		Status:     domain.StatusForReview,
		Issue:      0,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	out, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusClosed, out.Task.Status)
	assert.Empty(t, out.Task.Agent)
	assert.Empty(t, out.Task.Session)

	// Verify calls
	assert.True(t, worktrees.RemoveCalled)
	assert.True(t, git.MergeCalled)
	assert.Equal(t, "crew-1", git.MergeBranch)
	assert.True(t, git.MergeNoFF)
	assert.True(t, git.DeleteBranchCalled)
	assert.Equal(t, "crew-1", git.DeletedBranch)
}

func TestMergeTask_Execute_SuccessWithIssue(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with issue",
		Status:     domain.StatusForReview,
		Issue:      123,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	out, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "crew-1-gh-123", git.MergeBranch)
	assert.Equal(t, "crew-1-gh-123", git.DeletedBranch)
}

func TestMergeTask_Execute_StopsRunningSession(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with running session",
		Status:     domain.StatusInProgress,
		Agent:      "claude",
		Session:    "crew-1",
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute - session should be stopped after merge
	out, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, sessions.StopCalled)
	assert.Equal(t, domain.StatusClosed, out.Task.Status)
}

func TestMergeTask_Execute_NoWorktree(t *testing.T) {
	// Setup - worktree doesn't exist (already removed or never created)
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task without worktree",
		Status:     domain.StatusForReview,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = false // worktree doesn't exist
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	out, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert - should succeed without removing worktree
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, worktrees.RemoveCalled)
	assert.True(t, git.MergeCalled)
}

func TestMergeTask_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestMergeTask_Execute_NotOnMain(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to merge",
		Status:     domain.StatusForReview,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{
		CurrentBranchName: "feature-branch", // Not on main
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrNotOnMainBranch)
}

func TestMergeTask_Execute_UncommittedChanges(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to merge",
		Status:     domain.StatusForReview,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{
		CurrentBranchName:      "main",
		HasUncommittedChangesV: true, // Has uncommitted changes
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrUncommittedChanges)
}

func TestMergeTask_Execute_GetTaskError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestMergeTask_Execute_MergeError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to merge",
		Status:     domain.StatusForReview,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
		MergeErr:          assert.AnError,
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "merge branch")
	assert.False(t, sessions.StopCalled)
}

func TestMergeTask_Execute_DeleteBranchError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to merge",
		Status:     domain.StatusForReview,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
		DeleteBranchErr:   assert.AnError,
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete branch")
}

func TestMergeTask_Execute_SaveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to merge",
		Status:     domain.StatusForReview,
		BaseBranch: "main",
	}
	repo.SaveErr = assert.AnError
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")
}

func TestMergeTask_Execute_StopSessionError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with running session",
		Status:     domain.StatusInProgress,
		Agent:      "claude",
		Session:    "crew-1",
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.StopErr = assert.AnError
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute - stopping session should fail
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stop session")
}

func TestMergeTask_Execute_RemoveWorktreeError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to merge",
		Status:     domain.StatusForReview,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	worktrees.RemoveErr = assert.AnError
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remove worktree")
}

func TestMergeTask_Execute_WithCustomBaseBranch(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to merge to feature branch",
		Status:     domain.StatusForReview,
		BaseBranch: "feature/workspace",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "feature/workspace",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute
	out, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID:     1,
		BaseBranch: "feature/workspace",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusClosed, out.Task.Status)
	assert.True(t, git.MergeCalled)
	assert.Equal(t, "crew-1", git.MergeBranch)
}

func TestMergeTask_Execute_PrioritizesInputBaseBranch(t *testing.T) {
	// Setup - in.BaseBranch should override task.BaseBranch
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with different base branch",
		Status:     domain.StatusForReview,
		BaseBranch: "feature/workspace",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "release",
		DefaultBranchName: "develop",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute - input base branch should take precedence
	out, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID:     1,
		BaseBranch: "release",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusClosed, out.Task.Status)
	assert.True(t, git.MergeCalled)
	assert.False(t, git.GetDefaultBranchCalled)
}

func TestMergeTask_Execute_BaseBranchMismatch(t *testing.T) {
	// Setup - task has different base branch but --base allows override
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with feature base branch",
		Status:     domain.StatusForReview,
		BaseBranch: "feature/workspace",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute - merge to main even though task is based on feature branch (--base override)
	out, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID:     1,
		BaseBranch: "main",
	})

	// Assert - should succeed (override is allowed)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusClosed, out.Task.Status)
	assert.True(t, git.MergeCalled)
}

func TestMergeTask_Execute_NotOnBaseBranch(t *testing.T) {
	// Setup - current branch doesn't match target base branch
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task to merge",
		Status:     domain.StatusForReview,
		BaseBranch: "feature/workspace",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute - trying to merge to feature branch but on main
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID:     1,
		BaseBranch: "feature/workspace",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrNotOnBaseBranch)
}

func TestMergeTask_Execute_UseTaskBaseBranch(t *testing.T) {
	// Setup - task has custom base branch, --base not specified
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with custom base",
		Status:     domain.StatusForReview,
		BaseBranch: "feature/workspace",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "feature/workspace",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute - BaseBranch not specified, should use task's BaseBranch
	out, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
		// BaseBranch not specified
	})

	// Assert - should merge to feature/workspace (task's base branch)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusClosed, out.Task.Status)
	assert.True(t, git.MergeCalled)
	assert.False(t, git.GetDefaultBranchCalled)
}

func TestMergeTask_Execute_EmptyTaskBaseBranch(t *testing.T) {
	// Setup - task has empty base branch, --base not specified
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task without base branch",
		Status:     domain.StatusForReview,
		BaseBranch: "", // empty
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "develop",
		DefaultBranchName: "develop",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute - BaseBranch not specified, task's BaseBranch is empty, should use GetDefaultBranch
	out, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert - should merge to develop (default branch)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusClosed, out.Task.Status)
	assert.True(t, git.MergeCalled)
	assert.True(t, git.GetDefaultBranchCalled)
}

func TestMergeTask_Execute_DefaultBranchMismatch(t *testing.T) {
	// Setup - derived default branch doesn't match current branch
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task without base branch",
		Status:     domain.StatusForReview,
		BaseBranch: "",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{
		CurrentBranchName: "main",
		DefaultBranchName: "develop",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git, t.TempDir())

	// Execute - BaseBranch not specified, default branch is develop but current is main
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrNotOnBaseBranch)
	assert.True(t, git.GetDefaultBranchCalled)
}
