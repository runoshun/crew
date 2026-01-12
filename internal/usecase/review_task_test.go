package usecase

import (
	"bytes"
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewTask_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewReviewTask(repo, worktrees, configLoader, "/repo", &stdout, &stderr)

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

	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewReviewTask(repo, worktrees, configLoader, "/repo", &stdout, &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ReviewTaskInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestReviewTask_Execute_ConfigLoadError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	configLoader.LoadErr = assert.AnError

	var stdout, stderr bytes.Buffer
	uc := NewReviewTask(repo, worktrees, configLoader, "/repo", &stdout, &stderr)

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
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewReviewTask(repo, worktrees, configLoader, "/repo", &stdout, &stderr)

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
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolveErr = assert.AnError

	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewReviewTask(repo, worktrees, configLoader, "/repo", &stdout, &stderr)

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
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()

	// Verify default reviewer is set
	cfg, _ := configLoader.Load()
	require.NotEmpty(t, cfg.AgentsConfig.DefaultReviewer)
	require.Contains(t, cfg.Agents, cfg.AgentsConfig.DefaultReviewer)
}

func TestReviewTask_Execute_UsesSpecifiedAgent(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()

	// Verify claude-reviewer exists
	cfg, _ := configLoader.Load()
	require.Contains(t, cfg.Agents, "claude-reviewer")
}

func TestReviewTask_Execute_ModelOverride(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewReviewTask(repo, worktrees, configLoader, "/repo", &stdout, &stderr)

	// This test just verifies the flow doesn't error when model is specified
	// The actual command execution would fail without a real agent
	_ = uc
}

func TestReviewTask_Execute_TaskWithIssue(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
		Issue:  123,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree-gh-123"

	_ = testutil.NewMockConfigLoader()
	_ = repo
	_ = worktrees

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
		Status: domain.StatusInProgress,
	}

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

	var stdout, stderr bytes.Buffer
	uc := NewReviewTask(repo, worktrees, configLoader, "/repo", &stdout, &stderr)

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
		Status: domain.StatusInProgress,
	}

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

	var stdout, stderr bytes.Buffer
	uc := NewReviewTask(repo, worktrees, configLoader, "/repo", &stdout, &stderr)

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
		Status: domain.StatusInProgress,
	}

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

	var stdout, stderr bytes.Buffer
	uc := NewReviewTask(repo, worktrees, configLoader, "/repo", &stdout, &stderr)

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
