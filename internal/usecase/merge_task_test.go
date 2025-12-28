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
		ID:     1,
		Title:  "Task to merge",
		Status: domain.StatusInReview,
		Issue:  0,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

	// Execute
	out, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusDone, out.Task.Status)
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
		ID:     1,
		Title:  "Task with issue",
		Status: domain.StatusInReview,
		Issue:  123,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

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

func TestMergeTask_Execute_ForceStopSession(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task with running session",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

	// Execute with force
	out, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
		Force:  true,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, sessions.StopCalled)
	assert.Equal(t, domain.StatusDone, out.Task.Status)
}

func TestMergeTask_Execute_NoWorktree(t *testing.T) {
	// Setup - worktree doesn't exist (already removed or never created)
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task without worktree",
		Status: domain.StatusInReview,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = false // worktree doesn't exist
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

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

	uc := NewMergeTask(repo, sessions, worktrees, git)

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
		ID:     1,
		Title:  "Task to merge",
		Status: domain.StatusInReview,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{
		CurrentBranchName: "feature-branch", // Not on main
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

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
		ID:     1,
		Title:  "Task to merge",
		Status: domain.StatusInReview,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{
		CurrentBranchName:      "main",
		HasUncommittedChangesV: true, // Has uncommitted changes
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

	// Execute
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrUncommittedChanges)
}

func TestMergeTask_Execute_SessionRunningNoForce(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Task with running session",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

	// Execute without force
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
		Force:  false,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrSessionRunning)
	assert.False(t, sessions.StopCalled)
}

func TestMergeTask_Execute_GetTaskError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{}

	uc := NewMergeTask(repo, sessions, worktrees, git)

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
		ID:     1,
		Title:  "Task to merge",
		Status: domain.StatusInReview,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
		MergeErr:          assert.AnError,
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

	// Execute
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "merge branch")
}

func TestMergeTask_Execute_DeleteBranchError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to merge",
		Status: domain.StatusInReview,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
		DeleteBranchErr:   assert.AnError,
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

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
		ID:     1,
		Title:  "Task to merge",
		Status: domain.StatusInReview,
	}
	repo.SaveErr = assert.AnError
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

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
		ID:      1,
		Title:   "Task with running session",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.StopErr = assert.AnError
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

	// Execute with force
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
		Force:  true,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stop session")
}

func TestMergeTask_Execute_RemoveWorktreeError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task to merge",
		Status: domain.StatusInReview,
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	worktrees.RemoveErr = assert.AnError
	git := &testutil.MockGit{
		CurrentBranchName: "main",
	}

	uc := NewMergeTask(repo, sessions, worktrees, git)

	// Execute
	_, err := uc.Execute(context.Background(), MergeTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remove worktree")
}
