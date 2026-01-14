package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestWorktree creates a temp directory with .git/info for setup scripts
func setupTestWorktree(t *testing.T) string {
	t.Helper()
	worktreeDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(worktreeDir, ".git", "info"), 0o755))
	return worktreeDir
}

func TestStartTask_Execute_Success(t *testing.T) {
	// Setup temp directory for script generation
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)
	assert.Equal(t, worktreeDir, out.WorktreePath)

	// Verify task updated
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusInProgress, task.Status)
	assert.Equal(t, "claude", task.Agent)
	assert.Equal(t, "crew-1", task.Session)
	assert.Equal(t, clock.NowTime, task.Started)

	// Verify mocks called
	assert.True(t, worktrees.CreateCalled)
	assert.True(t, sessions.StartCalled)
	assert.Equal(t, "crew-1", sessions.StartOpts.Name)
	assert.Equal(t, worktreeDir, sessions.StartOpts.Dir)

	// Verify script was generated (command should be the script path)
	expectedScriptPath := domain.ScriptPath(crewDir, 1)
	assert.Equal(t, expectedScriptPath, sessions.StartOpts.Command)

	// Verify script file exists
	assert.FileExists(t, domain.ScriptPath(crewDir, 1))

	// Verify script content (includes embedded prompt)
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	script := string(scriptContent)
	assert.Contains(t, script, "#!/bin/bash")
	assert.Contains(t, script, "_session-ended 1")
	assert.Contains(t, script, "claude")
	// Verify prompt is embedded in script (default prompt references task ID and crew commands)
	assert.Contains(t, script, "Task #1")
	assert.Contains(t, script, "crew --help-worker")
}

func TestStartTask_Execute_WithAgentSetup(t *testing.T) {
	// Setup temp directories
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	// Create .git/info directory for exclude patterns
	gitInfoDir := filepath.Join(repoRoot, ".git", "info")
	require.NoError(t, os.MkdirAll(gitInfoDir, 0o755))

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	// Configure agent with setup script (includes exclude patterns)
	configLoader.Config.Agents["custom"] = domain.Agent{
		Role:            domain.RoleWorker,
		CommandTemplate: "custom-cmd {{.Prompt}}",
		SetupScript: `# Add exclude patterns
echo ".test-exclude/" >> "$GIT_INFO_EXCLUDE"
echo 'setup done'`,
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	mockRunner := testutil.NewMockScriptRunner()
	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, mockRunner, crewDir, repoRoot)

	// Execute
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "custom",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify task uses custom agent
	task := repo.Tasks[1]
	assert.Equal(t, "custom", task.Agent)
}

func TestStartTask_Execute_FromInReview(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusInReview,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusInProgress, task.Status)
}

func TestStartTask_Execute_FromError(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusError,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusInProgress, task.Status)
}

func TestStartTask_Execute_TaskNotFound(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	repoRoot := t.TempDir()
	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 999,
		Agent:  "claude",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestStartTask_Execute_AllowAllStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
	}{
		{"done", domain.StatusDone},
		{"closed", domain.StatusClosed},
		{"in_progress", domain.StatusInProgress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crewDir := t.TempDir()
			repoRoot := t.TempDir()
			worktreeDir := setupTestWorktree(t)

			repo := testutil.NewMockTaskRepository()
			repo.Tasks[1] = &domain.Task{
				ID:     1,
				Title:  "Test task",
				Status: tt.status,
			}
			sessions := testutil.NewMockSessionManager()
			worktrees := testutil.NewMockWorktreeManager()
			worktrees.CreatePath = worktreeDir
			configLoader := testutil.NewMockConfigLoader()
			clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

			uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

			// Execute
			_, err := uc.Execute(context.Background(), StartTaskInput{
				TaskID: 1,
				Agent:  "claude",
			})

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, domain.StatusInProgress, repo.Tasks[1].Status)
		})
	}
}

func TestStartTask_Execute_SessionAlreadyRunning(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true // Session already running
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	repoRoot := t.TempDir()
	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrSessionRunning)
}

func TestStartTask_Execute_WithDefaultAgent(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	// Set default worker to "opencode"
	configLoader.Config.AgentsConfig.DefaultWorker = "opencode"
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute without specifying agent
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "", // No agent specified - falls back to default
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify task uses default worker name
	task := repo.Tasks[1]
	assert.Equal(t, "opencode", task.Agent)

	// Verify script uses the underlying agent command (opencode) from the referenced agent
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	assert.Contains(t, string(scriptContent), "opencode")
}

func TestStartTask_Execute_WorktreeCreateError(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreateErr = assert.AnError
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	repoRoot := t.TempDir()
	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create worktree")
}

func TestStartTask_Execute_SessionStartError(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	sessions.StartErr = assert.AnError
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start session")

	// Verify worktree was rolled back
	assert.True(t, worktrees.RemoveCalled)

	// Verify script file was cleaned up
	assert.NoFileExists(t, domain.ScriptPath(crewDir, 1))
}

func TestStartTask_Execute_SaveError(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	repo.SaveErr = assert.AnError
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")

	// Verify session and worktree were rolled back
	assert.True(t, sessions.StopCalled)
	assert.True(t, worktrees.RemoveCalled)

	// Verify script file was cleaned up
	assert.NoFileExists(t, domain.ScriptPath(crewDir, 1))
}

func TestStartTask_Execute_WithIssue(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		Issue:      123,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	// Use custom prompt that includes issue
	configLoader.Config.Agents["claude"] = domain.Agent{
		Role:            domain.RoleWorker,
		CommandTemplate: configLoader.Config.Agents["claude"].CommandTemplate,
		Prompt:          "Task #{{.TaskID}}{{if .Issue}} (Issue #{{.Issue}}){{end}}",
		DefaultModel:    configLoader.Config.Agents["claude"].DefaultModel,
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)
	// Session name is still crew-1 (not crew-1-gh-123)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify prompt template expanded with issue info
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	assert.Contains(t, string(scriptContent), "Issue #123")
}

func TestStartTask_Execute_WithDescription(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:          1,
		Title:       "Test task",
		Description: "This is a detailed description of the task.",
		Status:      domain.StatusTodo,
		BaseBranch:  "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	// Use custom prompt that includes description
	configLoader.Config.Agents["claude"] = domain.Agent{
		Role:            domain.RoleWorker,
		CommandTemplate: configLoader.Config.Agents["claude"].CommandTemplate,
		Prompt:          "Task: {{.Title}}\n{{.Description}}",
		DefaultModel:    configLoader.Config.Agents["claude"].DefaultModel,
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)

	// Verify prompt template expanded with description
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	assert.Contains(t, string(scriptContent), "This is a detailed description of the task.")
}

func TestStartTask_ScriptGeneration(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "opencode",
	})

	require.NoError(t, err)

	// Verify script structure
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	script := string(scriptContent)

	// Check required script elements
	assert.Contains(t, script, "#!/bin/bash")
	assert.Contains(t, script, "set -o pipefail")
	assert.Contains(t, script, "END_OF_PROMPT") // Heredoc marker
	assert.Contains(t, script, "SESSION_ENDED()")
	assert.Contains(t, script, "_session-ended 1")
	assert.Contains(t, script, "trap SESSION_ENDED EXIT")
	assert.Contains(t, script, "trap 'exit 130' INT")
	assert.Contains(t, script, "trap 'exit 143' TERM")
	// Verify opencode command is rendered with args from builtin config
	assert.Contains(t, script, "opencode")
	assert.Contains(t, script, "--prompt")
	assert.Contains(t, script, "\"$PROMPT\"")

	// Verify script is executable (mode 0700)
	info, err := os.Stat(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm(), "script should have 0700 permissions")
}

func TestStartTask_CleanupScript(t *testing.T) {
	crewDir := t.TempDir()
	scriptsDir := filepath.Join(crewDir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0o755))

	// Create dummy file
	scriptPath := domain.ScriptPath(crewDir, 1)
	require.NoError(t, os.WriteFile(scriptPath, []byte("test"), 0o755))

	// Create use case
	uc := NewStartTask(nil, nil, nil, nil, nil, nil, nil, nil, crewDir, "")

	// Cleanup
	uc.cleanupScript(1)

	// Verify file is removed
	assert.NoFileExists(t, scriptPath)
}

func TestStartTask_Execute_ConfigLoadError(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	configLoader.LoadErr = assert.AnError // Config load error
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	repoRoot := t.TempDir()
	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute without agent (will try to load default from config)
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "", // No agent specified, will try to load from config
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestStartTask_Execute_WithUnknownAgent(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	repoRoot := t.TempDir()
	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute with unknown agent name should fail
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "my-custom-agent",
	})

	// Assert - unknown agents should return ErrAgentNotFound
	assert.ErrorIs(t, err, domain.ErrAgentNotFound)
}

func TestStartTask_Execute_WithConfiguredAgent(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	// Configure a custom agent in config
	configLoader.Config.Agents["my-agent"] = domain.Agent{
		Role:            domain.RoleWorker,
		CommandTemplate: "my-agent-bin {{.Args}} {{.Prompt}}",
		Args:            "--custom-flag",
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	repoRoot := t.TempDir()
	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute with configured agent
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "my-agent",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify script uses configured agent command and args
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	script := string(scriptContent)
	assert.Contains(t, script, "my-agent-bin")
	assert.Contains(t, script, "--custom-flag")
}

func TestStartTask_Execute_WithAgentPrompt(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	// Configure agent-specific prompt
	configLoader.Config.Agents["claude"] = domain.Agent{
		Role:            domain.RoleWorker,
		CommandTemplate: configLoader.Config.Agents["claude"].CommandTemplate,
		Prompt:          "Custom agent prompt for claude",
		DefaultModel:    configLoader.Config.Agents["claude"].DefaultModel,
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)

	// Verify script contains agent-specific prompt
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	assert.Contains(t, string(scriptContent), "Custom agent prompt for claude")
}

func TestStartTask_Execute_WithModelOverride(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config.Agents["claude"] = domain.Agent{
		Role:            domain.RoleWorker,
		CommandTemplate: "claude --model {{.Model}} {{.Args}} {{.Prompt}}",
		DefaultModel:    "config-model",
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute with model override
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
		Model:  "sonnet",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify script uses overridden model
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	script := string(scriptContent)
	// CLI flag should take precedence over config model
	assert.Contains(t, script, "--model sonnet")
	assert.NotContains(t, script, "--model config-model")
}

func TestStartTask_Execute_WithConfigModel(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config.Agents["claude"] = domain.Agent{
		Role:            domain.RoleWorker,
		CommandTemplate: "claude --model {{.Model}} {{.Args}} {{.Prompt}}",
		DefaultModel:    "config-model",
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute without specifying model flag
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify script uses config-defined model
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	assert.Contains(t, string(scriptContent), "--model config-model")
}

func TestStartTask_Execute_WithDefaultModel(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute without model (should use agent's default)
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
		Model:  "", // No model specified
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify script uses default model from builtin agents
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	script := string(scriptContent)
	// Default model for claude is "opus" (from builtin agents)
	assert.Contains(t, script, "--model opus")
}

func TestStartTask_Execute_OpencodeWithModelOverride(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute with opencode and model override
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "opencode",
		Model:  "gpt-4o",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify script uses overridden model
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	script := string(scriptContent)
	// The model should be expanded: -m {{.Model}} -> -m gpt-4o
	assert.Contains(t, script, "-m gpt-4o")
}

func TestStartTask_Execute_WithSetupScript(t *testing.T) {
	// Setup temp directories
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	// Configure agent with setup script that uses template variables
	configLoader.Config.Agents["custom"] = domain.Agent{
		Role:            domain.RoleWorker,
		CommandTemplate: "custom-cmd {{.Prompt}}",
		SetupScript:     `echo ".new-pattern/" >> {{.Worktree}}/.git/info/exclude`,
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	mockRunner := testutil.NewMockScriptRunner()
	uc := NewStartTask(repo, sessions, worktrees, configLoader, &testutil.MockGit{}, clock, nil, mockRunner, crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "custom",
	})

	// Assert
	require.NoError(t, err)

	// Verify setup script was called with expanded template
	assert.True(t, mockRunner.RunCalled)
	assert.Equal(t, worktreeDir, mockRunner.RunDir)
	// The script should have template variables expanded
	expectedScript := fmt.Sprintf(`echo ".new-pattern/" >> %s/.git/info/exclude`, worktreeDir)
	assert.Equal(t, expectedScript, mockRunner.RunScript)
}

func TestStartTask_Execute_WithDisabledAgent(t *testing.T) {
	// Setup
	task := &domain.Task{
		ID:         1,
		Title:      "Test",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	mockTasks := testutil.NewMockTaskRepository()
	mockTasks.Tasks[1] = task
	mockSessions := testutil.NewMockSessionManager()
	mockWorktrees := testutil.NewMockWorktreeManager()
	mockGit := &testutil.MockGit{}

	// Mock config with disabled agent
	cfg := &domain.Config{
		Agents: map[string]domain.Agent{
			"claude": {
				CommandTemplate: "claude {{.Args}} --prompt {{.Prompt}}",
				Role:            domain.RoleWorker,
			},
		},
		AgentsConfig: domain.AgentsConfig{
			DisabledAgents: []string{"claude"},
		},
	}
	mockLoader := &testutil.MockConfigLoader{Config: cfg}

	uc := NewStartTask(mockTasks, mockSessions, mockWorktrees, mockLoader, mockGit, domain.RealClock{}, nil, testutil.NewMockScriptRunner(), "/test/.git/crew", "/test")

	// Execute with disabled agent
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert - disabled agents should return ErrAgentDisabled
	assert.ErrorIs(t, err, domain.ErrAgentDisabled)
	assert.Contains(t, err.Error(), "claude")
	assert.Contains(t, err.Error(), "disabled")
}

func TestStartTask_Execute_WithWorkerPrompt(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	// Set WorkerPrompt in AgentsConfig
	configLoader.Config.AgentsConfig.WorkerPrompt = "Custom worker prompt from config"
	// Use an agent without a custom prompt (so WorkerPrompt should be used)
	configLoader.Config.Agents["test-agent"] = domain.Agent{
		Role:            domain.RoleWorker,
		CommandTemplate: "test-cmd {{.Prompt}}",
		// Prompt is empty, so WorkerPrompt should be used
	}
	git := &testutil.MockGit{}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, git, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "test-agent",
	})

	// Assert
	require.NoError(t, err)

	// Verify script contains WorkerPrompt
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	assert.Contains(t, string(scriptContent), "Custom worker prompt from config")
}

func TestStartTask_Execute_AgentPromptOverridesWorkerPrompt(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	configLoader := testutil.NewMockConfigLoader()
	// Set both WorkerPrompt and Agent.Prompt
	configLoader.Config.AgentsConfig.WorkerPrompt = "Worker prompt from config"
	configLoader.Config.Agents["test-agent"] = domain.Agent{
		Role:            domain.RoleWorker,
		CommandTemplate: "test-cmd {{.Prompt}}",
		Prompt:          "Agent-specific prompt", // This should take precedence
	}
	git := &testutil.MockGit{}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, git, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "test-agent",
	})

	// Assert
	require.NoError(t, err)

	// Verify Agent.Prompt takes precedence over WorkerPrompt
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	script := string(scriptContent)
	assert.Contains(t, script, "Agent-specific prompt")
	assert.NotContains(t, script, "Worker prompt from config")
}

func TestStartTask_Execute_WorktreeConfigPassed(t *testing.T) {
	// Setup temp directory for script generation
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir

	configLoader := testutil.NewMockConfigLoader()
	// Set WorktreeConfig in config
	configLoader.Config.Worktree = domain.WorktreeConfig{
		SetupCommand: "npm install",
		Copy:         []string{"node_modules", ".env"},
	}

	git := &testutil.MockGit{}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, git, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)

	// Verify SetupWorktree was called with correct config
	assert.True(t, worktrees.SetupCalled, "SetupWorktree should be called")
	assert.Equal(t, worktreeDir, worktrees.SetupWorktreePath)
	require.NotNil(t, worktrees.SetupWorktreeConfig)
	assert.Equal(t, "npm install", worktrees.SetupWorktreeConfig.SetupCommand)
	assert.Equal(t, []string{"node_modules", ".env"}, worktrees.SetupWorktreeConfig.Copy)
}

func TestStartTask_Execute_WorktreeSetupError(t *testing.T) {
	// Setup temp directory for script generation
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := setupTestWorktree(t)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.CreatePath = worktreeDir
	worktrees.SetupErr = assert.AnError // Simulate setup failure

	configLoader := testutil.NewMockConfigLoader()
	git := &testutil.MockGit{}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, git, clock, nil, testutil.NewMockScriptRunner(), crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "setup worktree")

	// Verify worktree was cleaned up after error
	assert.True(t, worktrees.RemoveCalled, "Worktree should be removed after setup error")
}
