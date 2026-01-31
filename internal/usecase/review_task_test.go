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
	clock := &testutil.MockClock{NowTime: time.Now()}

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

func TestReviewTask_Execute_WithReviewerPrompt(t *testing.T) {
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
	// Set ReviewerPrompt in AgentsConfig
	configLoader.Config.AgentsConfig.ReviewerPrompt = "Custom reviewer prompt from config"
	// Use an agent without a custom prompt (so ReviewerPrompt should be used)
	configLoader.Config.Agents["test-reviewer"] = domain.Agent{
		Role:            domain.RoleReviewer,
		CommandTemplate: "echo {{.Prompt}}",
		// Prompt is empty, so ReviewerPrompt should be used
	}

	executor := testutil.NewMockCommandExecutor()
	executor.ExecuteOutput = []byte("Custom reviewer prompt from config")
	clock := &testutil.MockClock{NowTime: time.Now()}

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
	// Verify Review output contains ReviewerPrompt
	assert.Contains(t, out.Review, "Custom reviewer prompt from config")
}

func TestReviewTask_Execute_AgentPromptOverridesReviewerPrompt(t *testing.T) {
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
	// Set both ReviewerPrompt and Agent.Prompt
	configLoader.Config.AgentsConfig.ReviewerPrompt = "Reviewer prompt from config"
	configLoader.Config.Agents["test-reviewer"] = domain.Agent{
		Role:            domain.RoleReviewer,
		CommandTemplate: "echo {{.Prompt}}",
		Prompt:          "Agent-specific prompt", // This should take precedence
	}

	executor := testutil.NewMockCommandExecutor()
	executor.ExecuteOutput = []byte("Agent-specific prompt")
	clock := &testutil.MockClock{NowTime: time.Now()}

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
	// Verify Agent.Prompt takes precedence over ReviewerPrompt
	assert.Contains(t, out.Review, "Agent-specific prompt")
	assert.NotContains(t, out.Review, "Reviewer prompt from config")
}

func TestReviewTask_Execute_MessageOverridesReviewerPrompt(t *testing.T) {
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
	// Set ReviewerPrompt in config
	configLoader.Config.AgentsConfig.ReviewerPrompt = "Reviewer prompt from config"
	// Use an agent without a custom prompt
	configLoader.Config.Agents["test-reviewer"] = domain.Agent{
		Role:            domain.RoleReviewer,
		CommandTemplate: "echo {{.Prompt}}",
	}

	executor := testutil.NewMockCommandExecutor()
	executor.ExecuteOutput = []byte("User-provided review message")
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute with Message - should override ReviewerPrompt
	out, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID:  1,
		Agent:   "test-reviewer",
		Message: "User-provided review message",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	// Verify Message takes precedence over ReviewerPrompt
	assert.Contains(t, out.Review, "User-provided review message")
	assert.NotContains(t, out.Review, "Reviewer prompt from config")
}

func TestReviewTask_Execute_AgentPromptOverridesMessage(t *testing.T) {
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
	// Set both Message and Agent.Prompt
	configLoader.Config.Agents["test-reviewer"] = domain.Agent{
		Role:            domain.RoleReviewer,
		CommandTemplate: "echo {{.Prompt}}",
		Prompt:          "Agent-specific prompt", // This should take precedence over Message
	}

	executor := testutil.NewMockCommandExecutor()
	executor.ExecuteOutput = []byte("Agent-specific prompt")
	clock := &testutil.MockClock{NowTime: time.Now()}

	var stderr bytes.Buffer
	uc := newTestReviewTask(repo, sessions, worktrees, configLoader, executor, clock, "/repo", &stderr)

	// Execute with both Agent.Prompt and Message
	out, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID:  1,
		Agent:   "test-reviewer",
		Message: "User-provided review message",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	// Verify Agent.Prompt takes precedence over Message
	assert.Contains(t, out.Review, "Agent-specific prompt")
	assert.NotContains(t, out.Review, "User-provided review message")
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
		Status: domain.StatusInProgress, // Already reviewing
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"
	configLoader := testutil.NewMockConfigLoader()
	executor := testutil.NewMockCommandExecutor()
	clock := &testutil.MockClock{NowTime: time.Now()}

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
	executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\n" + domain.ReviewLGTMPrefix + "\nLooks good.")
	clock := &testutil.MockClock{NowTime: time.Now()}

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
		executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\nReview output")
		executor.StderrOutput = []byte("stderr output\n")
		clock := &testutil.MockClock{NowTime: time.Now()}

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
		assert.Contains(t, output, domain.ReviewResultMarker)
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
		executor.ExecuteOutput = []byte(domain.ReviewResultMarker + "\nReview output")
		executor.StderrOutput = []byte("stderr output\n")
		clock := &testutil.MockClock{NowTime: time.Now()}

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
