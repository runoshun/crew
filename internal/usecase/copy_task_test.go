package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyTask_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Original task",
		Description: "Task description",
		Status:      domain.StatusInProgress,
		Labels:      []string{"bug", "urgent"},
		Issue:       42,
		PR:          10,
		BaseBranch:  "main",
	}
	repo.NextIDN = 2
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock, nil, nil)

	// Execute
	out, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, out.TaskID)

	// Verify new task
	task := repo.Tasks[2]
	require.NotNil(t, task)
	assert.Equal(t, 2, task.ID)
	assert.Equal(t, "Original task (copy)", task.Title)
	assert.Equal(t, "Task description", task.Description)
	assert.Equal(t, domain.StatusTodo, task.Status)
	assert.Equal(t, []string{"bug", "urgent"}, task.Labels)
	assert.Equal(t, clock.NowTime, task.Created)
	// Base branch should be inherited from source
	assert.Equal(t, "main", task.BaseBranch)
	// Issue and PR should NOT be copied
	assert.Equal(t, 0, task.Issue)
	assert.Equal(t, 0, task.PR)
}

func TestCopyTask_Execute_WithCustomTitle(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	repo.NextIDN = 2
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock, nil, nil)

	// Execute with custom title
	customTitle := "Custom new title"
	out, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
		Title:    &customTitle,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, out.TaskID)

	task := repo.Tasks[2]
	assert.Equal(t, "Custom new title", task.Title)
}

func TestCopyTask_Execute_WithParent(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	parentID := 1
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Parent task",
		Status:     domain.StatusInProgress,
		BaseBranch: "main",
	}
	repo.Tasks[2] = &domain.Task{
		ID:         2,
		ParentID:   &parentID,
		Title:      "Child task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	repo.NextIDN = 3
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock, nil, nil)

	// Execute - copy child task
	out, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 2,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 3, out.TaskID)

	// Verify parent is inherited
	task := repo.Tasks[3]
	require.NotNil(t, task.ParentID)
	assert.Equal(t, 1, *task.ParentID)
}

func TestCopyTask_Execute_SourceNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock, nil, nil)

	// Execute with non-existent source
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestCopyTask_Execute_GetError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock, nil, nil)

	// Execute
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get source task")
}

func TestCopyTask_Execute_SaveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	repo.NextIDN = 2
	repo.SaveErr = assert.AnError
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock, nil, nil)

	// Execute
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")
}

func TestCopyTask_Execute_NextIDError(t *testing.T) {
	// Setup
	repo := &testutil.MockTaskRepositoryWithNextIDError{
		MockTaskRepository: testutil.NewMockTaskRepository(),
		NextIDErr:          assert.AnError,
	}
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock, nil, nil)

	// Execute
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generate task ID")
}

func TestCopyTask_Execute_LabelsAreCopied(t *testing.T) {
	// Setup - verify labels are deep copied (not shared)
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		Labels:     []string{"original"},
		BaseBranch: "main",
	}
	repo.NextIDN = 2
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock, nil, nil)

	// Execute
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})
	require.NoError(t, err)

	// Modify original task's labels
	repo.Tasks[1].Labels[0] = "modified"

	// Verify copied task's labels are not affected
	assert.Equal(t, []string{"original"}, repo.Tasks[2].Labels)
}

func TestCopyTask_Execute_EmptyLabels(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		Labels:     nil,
		BaseBranch: "main",
	}
	repo.NextIDN = 2
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock, nil, nil)

	// Execute
	out, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
	})

	// Assert
	require.NoError(t, err)
	task := repo.Tasks[out.TaskID]
	assert.Nil(t, task.Labels)
}

func TestCopyTask_Execute_CopyAll_CopiesCommentsAndCreatesWorktree(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	repo.Comments[1] = []domain.Comment{
		{Text: "First", Time: time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC), Author: "worker"},
		{Text: "Second", Time: time.Date(2024, 1, 2, 1, 0, 0, 0, time.UTC), Author: "reviewer"},
	}
	repo.NextIDN = 2
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)}
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{}
	uc := NewCopyTask(repo, clock, worktrees, git)

	// Execute
	out, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
		CopyAll:  true,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, out.TaskID)
	assert.Equal(t, repo.Comments[1], repo.Comments[2])
	assert.True(t, worktrees.CreateCalled)
	assert.Equal(t, domain.BranchName(2, 0), worktrees.CreateBranch)
	assert.Equal(t, domain.BranchName(1, 0), worktrees.CreateBaseBranch)
}

func TestCopyTask_Execute_CopyAll_RequiresManagers(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Original task",
		Status: domain.StatusTodo,
	}
	repo.NextIDN = 2
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)}
	uc := NewCopyTask(repo, clock, nil, nil)

	// Execute
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
		CopyAll:  true,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrCopyAllRequiresManagers)
}

func TestCopyTask_Execute_CopyAll_UsesDefaultBaseBranchWhenSourceMissing(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Original task",
		Status: domain.StatusTodo,
	}
	repo.NextIDN = 2
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)}
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{
		DefaultBranchName: testutil.StringPtr("develop"),
		BranchExistsMap: map[string]bool{
			domain.BranchName(1, 0): false,
		},
	}
	uc := NewCopyTask(repo, clock, worktrees, git)

	// Execute
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
		CopyAll:  true,
	})

	// Assert
	require.NoError(t, err)
	assert.True(t, git.GetDefaultBranchCalled)
	assert.Equal(t, "develop", worktrees.CreateBaseBranch)
	assert.Equal(t, "develop", repo.Tasks[2].BaseBranch)
}

func TestCopyTask_Execute_CopyAll_SaveErrorCleansUp(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Original task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	repo.NextIDN = 2
	repo.SaveErr = assert.AnError
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)}
	worktrees := testutil.NewMockWorktreeManager()
	git := &testutil.MockGit{}
	uc := NewCopyTask(repo, clock, worktrees, git)

	// Execute
	_, err := uc.Execute(context.Background(), CopyTaskInput{
		SourceID: 1,
		CopyAll:  true,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")
	assert.True(t, worktrees.RemoveCalled)
	assert.True(t, git.DeleteBranchCalled)
}
