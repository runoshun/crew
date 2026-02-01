package usecase

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestReviewTask(
	repo *testutil.MockTaskRepository,
	sessions *testutil.MockSessionManager,
	worktrees *testutil.MockWorktreeManager,
	configLoader *testutil.MockConfigLoader,
	executor *testutil.MockCommandExecutor,
	clock *testutil.MockClock,
	repoRoot string,
	stderr *bytes.Buffer,
) *ReviewTask {
	_ = sessions
	return NewReviewTask(repo, worktrees, configLoader, executor, clock, repoRoot, stderr)
}

func TestReviewTask_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	executor := testutil.NewMockCommandExecutor()
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestReviewTask_Execute_GetError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	executor := testutil.NewMockCommandExecutor()
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestReviewTask_Execute_InvalidStatus(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo, // Not in_progress or done
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	executor := testutil.NewMockCommandExecutor()
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)
}

func TestReviewTask_Execute_ConfigLoadError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusDone,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	configLoader.LoadErr = assert.AnError
	executor := testutil.NewMockCommandExecutor()
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestReviewTask_Execute_AgentNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusDone,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	executor := testutil.NewMockCommandExecutor()
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute with non-existent agent
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
		Agent:  "non-existent-agent",
	})

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAgentNotFound)
}

func TestReviewTask_Execute_WorktreeResolveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusDone,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolveErr = assert.AnError
	configLoader := testutil.NewMockConfigLoader()
	executor := testutil.NewMockCommandExecutor()
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resolve worktree")
}

func TestReviewTask_Execute_UsesDefaultReviewer(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusDone,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"
	configLoader := testutil.NewMockConfigLoader()

	// Verify default reviewer is set
	cfg, _ := configLoader.Load()
	require.NotEmpty(t, cfg.AgentsConfig.DefaultReviewer)
	require.Contains(t, cfg.Agents, cfg.AgentsConfig.DefaultReviewer)

	executor := testutil.NewMockCommandExecutor()
	now := time.Now()
	clock := &testutil.MockClock{NowTime: now}

	executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\n" + domain.ReviewLGTMPrefix + " Looks good!\n")

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute without specifying agent - should use DefaultReviewer
	out, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
		Agent:  "", // Empty - should use DefaultReviewer
	})

	// Assert that execution used DefaultReviewer (command was built successfully)
	assert.NoError(t, err)
	assert.True(t, executor.ExecuteWithContextCalled, "ExecuteWithContext should have been called")
	assert.NotNil(t, executor.ExecutedCmd, "Command should have been executed")
	assert.Equal(t, domain.StatusDone, out.Task.Status)
	assert.Equal(t, domain.ReviewLGTMPrefix+" Looks good!", out.Review)
}

func TestReviewTask_Execute_EmptyDefaultReviewer(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusDone,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	// Create config with empty DefaultReviewer
	cfg := &domain.Config{
		Agents: map[string]domain.Agent{
			"test-reviewer": {
				CommandTemplate: "test {{.Args}} --prompt {{.Prompt}}",
				Role:            domain.RoleReviewer,
			},
		},
		AgentsConfig: domain.AgentsConfig{
			DefaultReviewer: "", // Empty default reviewer
		},
	}
	configLoader := &testutil.MockConfigLoader{Config: cfg}
	executor := testutil.NewMockCommandExecutor()
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute without specifying agent - should fail with empty DefaultReviewer
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
		Agent:  "", // Empty - should try to use DefaultReviewer (which is also empty)
	})

	// Assert that execution failed with agent not found error
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAgentNotFound)
}

func TestReviewTask_Execute_TaskWithIssue(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusDone,
		Issue:  123,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree-gh-123"

	_ = testutil.NewMockConfigLoader()
	_ = repo
	_ = worktrees
	_ = sessions

	// Verify the branch name includes issue
	expectedBranch := domain.BranchName(1, 123)
	assert.Equal(t, "crew-1-gh-123", expectedBranch)
}

func TestReviewTask_Execute_ReviewerAddsComment(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusDone,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = t.TempDir()

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config.Agents["test-reviewer"] = domain.Agent{
		Role:            domain.RoleReviewer,
		CommandTemplate: "echo review",
	}

	executor := testutil.NewMockCommandExecutor()
	now := time.Now()
	clock := &testutil.MockClock{NowTime: now}

	reviewText := domain.ReviewLGTMPrefix + " All checks passed!"
	executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\n" + reviewText + "\n")

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	out, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
		Agent:  "test-reviewer",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, reviewText, out.Review)
	assert.Equal(t, 1, out.Task.ReviewCount)
	require.NotNil(t, out.Task.LastReviewIsLGTM)
	assert.True(t, *out.Task.LastReviewIsLGTM)
	// ReviewTask should record the review as a task comment programmatically
	assert.Len(t, repo.Comments[1], 1)
	assert.Equal(t, "reviewer", repo.Comments[1][0].Author)
	assert.Equal(t, reviewText, repo.Comments[1][0].Text)
}

func TestReviewTask_Execute_NoReviewComment(t *testing.T) {
	// Setup - reviewer does NOT output marker
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusDone,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = t.TempDir()

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config.Agents["test-reviewer"] = domain.Agent{
		Role:            domain.RoleReviewer,
		CommandTemplate: "echo review",
	}

	executor := testutil.NewMockCommandExecutor()
	executor.ExecuteOutput = []byte("some output without marker\n")
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
		Agent:  "test-reviewer",
	})

	// Assert - should fail because marker is missing
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNoReviewComment)
	assert.Empty(t, repo.Comments[1])
}

func TestReviewTask_Execute_WithDisabledAgent(t *testing.T) {
	// Setup
	task := &domain.Task{
		ID:         1,
		Title:      "Test",
		Status:     domain.StatusDone,
		BaseBranch: "main",
	}
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = task
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()

	// Mock config with disabled agent
	cfg := &domain.Config{
		Agents: map[string]domain.Agent{
			"claude-reviewer": {
				CommandTemplate: "claude {{.Args}} --prompt {{.Prompt}}",
				Role:            domain.RoleReviewer,
			},
		},
		AgentsConfig: domain.AgentsConfig{
			DisabledAgents: []string{"claude-reviewer"},
		},
	}
	loader := &testutil.MockConfigLoader{Config: cfg}
	executor := testutil.NewMockCommandExecutor()
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, loader, executor, clock, "/test", &stderr)

	// Execute with disabled agent
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
		Agent:  "claude-reviewer",
	})

	// Assert - disabled agents should return ErrAgentDisabled
	assert.ErrorIs(t, err, domain.ErrAgentDisabled)
	assert.Contains(t, err.Error(), "claude-reviewer")
	assert.Contains(t, err.Error(), "disabled")
}

func TestReviewTask_Execute_StatusReviewing(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"
	configLoader := testutil.NewMockConfigLoader()
	executor := testutil.NewMockCommandExecutor()
	now := time.Now()
	clock := &testutil.MockClock{NowTime: now}

	executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\n" + domain.ReviewLGTMPrefix + " Looks good!\n")

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	out, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
	})

	// Assert - should succeed
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.StatusInProgress, out.Task.Status)
}

func TestReviewTask_Execute_SaveMetadataFails(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	repo.SaveErr = assert.AnError
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"
	configLoader := testutil.NewMockConfigLoader()
	executor := testutil.NewMockCommandExecutor()
	now := time.Now()
	clock := &testutil.MockClock{NowTime: now}

	executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\n" + domain.ReviewLGTMPrefix + "\nLooks good.\n")

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save review")
}

func TestReviewTask_Execute_Verbose(t *testing.T) {
	t.Run("verbose mode streams stdout and stderr", func(t *testing.T) {
		// Setup
		repo := testutil.NewMockTaskRepository()
		repo.Tasks[1] = &domain.Task{
			ID:     1,
			Title:  "Test task",
			Status: domain.StatusInProgress,
		}
		sessions := testutil.NewMockSessionManager()
		worktrees := testutil.NewMockWorktreeManager()
		worktrees.ResolvePath = "/tmp/worktree"
		configLoader := testutil.NewMockConfigLoader()
		executor := testutil.NewMockCommandExecutor()
		executor.ExecuteOutput = []byte("Review output\n")
		executor.StderrOutput = []byte("stderr output\n")
		now := time.Now()
		clock := &testutil.MockClock{NowTime: now}

		executor.ExecuteOutput = []byte("intermediate\n" + domain.ReviewResultMarker + "\n" + domain.ReviewLGTMPrefix + " Looks good!\n")

		var stderr bytes.Buffer
		uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

		// Execute with Verbose=true
		out, err := uc.Execute(context.Background(), ReviewTaskInput{
			TaskID:  1,
			Verbose: true,
		})

		// Assert
		require.NoError(t, err)
		require.NotNil(t, out)
		// In verbose mode, both stdout and stderr are streamed to stderr
		output := stderr.String()
		assert.Contains(t, output, "intermediate")
		assert.Contains(t, output, "stderr output")
		assert.Contains(t, out.Review, domain.ReviewResultMarker)
		assert.Len(t, repo.Comments[1], 1)
		assert.Equal(t, "reviewer", repo.Comments[1][0].Author)
		assert.Equal(t, domain.ReviewLGTMPrefix+" Looks good!", repo.Comments[1][0].Text)
	})

	t.Run("non-verbose mode does not stream output", func(t *testing.T) {
		// Setup
		repo := testutil.NewMockTaskRepository()
		repo.Tasks[1] = &domain.Task{
			ID:     1,
			Title:  "Test task",
			Status: domain.StatusInProgress,
		}
		sessions := testutil.NewMockSessionManager()
		worktrees := testutil.NewMockWorktreeManager()
		worktrees.ResolvePath = "/tmp/worktree"
		configLoader := testutil.NewMockConfigLoader()
		executor := testutil.NewMockCommandExecutor()
		executor.ExecuteOutput = []byte("Review output\n")
		executor.StderrOutput = []byte("stderr output\n")
		now := time.Now()
		clock := &testutil.MockClock{NowTime: now}

		executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\n" + domain.ReviewLGTMPrefix + " Looks good!\n")

		var stderr bytes.Buffer
		uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

		// Execute with Verbose=false
		out, err := uc.Execute(context.Background(), ReviewTaskInput{
			TaskID:  1,
			Verbose: false,
		})

		// Assert
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Empty(t, stderr.String())
		assert.Equal(t, domain.ReviewLGTMPrefix+" Looks good!", out.Review)
		assert.Len(t, repo.Comments[1], 1)
	})
}

func TestReviewTask_Execute_LGTMDetection(t *testing.T) {
	t.Run("LGTM review", func(t *testing.T) {
		// Setup
		repo := testutil.NewMockTaskRepository()
		repo.Tasks[1] = &domain.Task{
			ID:     1,
			Title:  "Test task",
			Status: domain.StatusInProgress,
		}
		sessions := testutil.NewMockSessionManager()
		worktrees := testutil.NewMockWorktreeManager()
		worktrees.ResolvePath = "/tmp/worktree"
		configLoader := testutil.NewMockConfigLoader()
		executor := testutil.NewMockCommandExecutor()
		now := time.Now()
		clock := &testutil.MockClock{NowTime: now}

		executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\n" + domain.ReviewLGTMPrefix + " All good!\n")

		var stderr bytes.Buffer
		uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

		// Execute
		out, err := uc.Execute(context.Background(), ReviewTaskInput{
			TaskID: 1,
		})

		// Assert
		require.NoError(t, err)
		require.NotNil(t, out.Task.LastReviewIsLGTM)
		assert.True(t, *out.Task.LastReviewIsLGTM)
	})

	t.Run("non-LGTM review", func(t *testing.T) {
		// Setup
		repo := testutil.NewMockTaskRepository()
		repo.Tasks[1] = &domain.Task{
			ID:     1,
			Title:  "Test task",
			Status: domain.StatusInProgress,
		}
		sessions := testutil.NewMockSessionManager()
		worktrees := testutil.NewMockWorktreeManager()
		worktrees.ResolvePath = "/tmp/worktree"
		configLoader := testutil.NewMockConfigLoader()
		executor := testutil.NewMockCommandExecutor()
		now := time.Now()
		clock := &testutil.MockClock{NowTime: now}

		executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\n" + "❌ Needs changes\n")

		var stderr bytes.Buffer
		uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

		// Execute
		out, err := uc.Execute(context.Background(), ReviewTaskInput{
			TaskID: 1,
		})

		// Assert
		require.NoError(t, err)
		require.NotNil(t, out.Task.LastReviewIsLGTM)
		assert.False(t, *out.Task.LastReviewIsLGTM)
	})
}

func TestReviewTask_Execute_MultipleReviewerComments(t *testing.T) {
	// Setup - there are existing comments; review output is recorded programmatically
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	// Pre-existing comments (should not affect marker-based parsing)
	repo.Comments[1] = []domain.Comment{
		{Author: "worker", Text: "Work done", Time: time.Now().Add(-time.Hour)},
		{Author: "reviewer", Text: "First review: needs changes", Time: time.Now().Add(-30 * time.Minute)},
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"
	configLoader := testutil.NewMockConfigLoader()
	executor := testutil.NewMockCommandExecutor()
	now := time.Now()
	clock := &testutil.MockClock{NowTime: now}

	newReviewText := domain.ReviewLGTMPrefix + " Fixed now!"
	executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\n" + newReviewText + "\n")

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	out, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
	})

	// Assert - should return the latest reviewer comment
	require.NoError(t, err)
	assert.Equal(t, newReviewText, out.Review)
	require.NotNil(t, out.Task.LastReviewIsLGTM)
	assert.True(t, *out.Task.LastReviewIsLGTM)
}
