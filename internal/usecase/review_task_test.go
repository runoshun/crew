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

// addReviewerCommentOnExecute sets up the executor to simulate adding a reviewer comment.
// The callback is called during ExecuteWithContext to simulate the reviewer agent running `crew comment`.
func addReviewerCommentOnExecute(executor *testutil.MockCommandExecutor, addComment func()) {
	executor.OnExecuteWithContext = addComment
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

	// Setup reviewer to add a comment during execution
	addReviewerCommentOnExecute(executor, func() {
		repo.Comments[1] = append(repo.Comments[1], domain.Comment{
			Author: "reviewer",
			Text:   domain.ReviewLGTMPrefix + " Looks good!",
			Time:   now,
		})
	})

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

	// Simulate reviewer adding a comment during execution
	reviewText := domain.ReviewLGTMPrefix + " All checks passed!"
	addReviewerCommentOnExecute(executor, func() {
		repo.Comments[1] = append(repo.Comments[1], domain.Comment{
			Author: "reviewer",
			Text:   reviewText,
			Time:   now,
		})
	})

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
}

func TestReviewTask_Execute_NoReviewComment(t *testing.T) {
	// Setup - reviewer does NOT add a comment
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
	// No OnExecuteWithContext - reviewer doesn't add a comment
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
		Agent:  "test-reviewer",
	})

	// Assert - should fail because reviewer didn't add a comment
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNoReviewComment)
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

	// Setup reviewer to add a comment during execution
	addReviewerCommentOnExecute(executor, func() {
		repo.Comments[1] = append(repo.Comments[1], domain.Comment{
			Author: "reviewer",
			Text:   domain.ReviewLGTMPrefix + " Looks good!",
			Time:   now,
		})
	})

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

	// Setup reviewer to add a comment during execution
	addReviewerCommentOnExecute(executor, func() {
		repo.Comments[1] = append(repo.Comments[1], domain.Comment{
			Author: "reviewer",
			Text:   domain.ReviewLGTMPrefix + "\nLooks good.",
			Time:   now,
		})
	})

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save review metadata")
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

		// Setup reviewer to add a comment during execution
		addReviewerCommentOnExecute(executor, func() {
			repo.Comments[1] = append(repo.Comments[1], domain.Comment{
				Author: "reviewer",
				Text:   domain.ReviewLGTMPrefix + " Looks good!",
				Time:   now,
			})
		})

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
		assert.Contains(t, output, "Review output")
		assert.Contains(t, output, "stderr output")
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

		// Setup reviewer to add a comment during execution
		addReviewerCommentOnExecute(executor, func() {
			repo.Comments[1] = append(repo.Comments[1], domain.Comment{
				Author: "reviewer",
				Text:   domain.ReviewLGTMPrefix + " Looks good!",
				Time:   now,
			})
		})

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

		// Setup reviewer to add an LGTM comment
		addReviewerCommentOnExecute(executor, func() {
			repo.Comments[1] = append(repo.Comments[1], domain.Comment{
				Author: "reviewer",
				Text:   domain.ReviewLGTMPrefix + " All good!",
				Time:   now,
			})
		})

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

		// Setup reviewer to add a non-LGTM comment
		addReviewerCommentOnExecute(executor, func() {
			repo.Comments[1] = append(repo.Comments[1], domain.Comment{
				Author: "reviewer",
				Text:   "❌ Needs changes",
				Time:   now,
			})
		})

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
	// Setup - there are existing comments, and reviewer adds a new one
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}
	// Pre-existing comments
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

	// Setup reviewer to add a new LGTM comment
	newReviewText := domain.ReviewLGTMPrefix + " Fixed now!"
	addReviewerCommentOnExecute(executor, func() {
		repo.Comments[1] = append(repo.Comments[1], domain.Comment{
			Author: "reviewer",
			Text:   newReviewText,
			Time:   now,
		})
	})

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
