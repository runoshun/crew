package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartTask_Execute_Success(t *testing.T) {
	// Setup temp directory for script generation
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
	worktrees.CreatePath = "/path/to/worktree"
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

	// Execute
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)
	assert.Equal(t, "/path/to/worktree", out.WorktreePath)

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
	assert.Equal(t, "/path/to/worktree", sessions.StartOpts.Dir)

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
	// Verify prompt is embedded in script
	assert.Contains(t, script, "Test task")
	assert.Contains(t, script, "git crew complete")
}

func TestStartTask_Execute_FromInReview(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusInReview,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

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

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Test task",
		Status:     domain.StatusError,
		BaseBranch: "main",
	}
	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 999,
		Agent:  "claude",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestStartTask_Execute_InvalidStatus(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
	}{
		{"done", domain.StatusDone},
		{"closed", domain.StatusClosed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crewDir := t.TempDir()

			repo := testutil.NewMockTaskRepository()
			repo.Tasks[1] = &domain.Task{
				ID:     1,
				Title:  "Test task",
				Status: tt.status,
			}
			sessions := testutil.NewMockSessionManager()
			worktrees := testutil.NewMockWorktreeManager()
			configLoader := testutil.NewMockConfigLoader()
			clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

			uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

			// Execute
			_, err := uc.Execute(context.Background(), StartTaskInput{
				TaskID: 1,
				Agent:  "claude",
			})

			// Assert
			assert.ErrorIs(t, err, domain.ErrInvalidTransition)
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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrSessionRunning)
}

func TestStartTask_Execute_NoAgent(t *testing.T) {
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
	// No default_agent in config
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

	// Execute without agent
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrNoAgent)
}

func TestStartTask_Execute_WithDefaultAgent(t *testing.T) {
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
	configLoader.Config.WorkersConfig.Default = "opencode" // default from [workers] section
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

	// Execute without specifying agent
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "", // No agent specified
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify task uses default agent
	task := repo.Tasks[1]
	assert.Equal(t, "opencode", task.Agent)

	// Verify script uses default agent
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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

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
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

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
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

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
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

	// Execute
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)
	// Session name is still crew-1 (not crew-1-gh-123)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify prompt in script includes issue info
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	assert.Contains(t, string(scriptContent), "#123")
}

func TestStartTask_Execute_WithDescription(t *testing.T) {
	crewDir := t.TempDir()

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
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)

	// Verify prompt in script includes description
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	assert.Contains(t, string(scriptContent), "This is a detailed description of the task.")
}

func TestStartTask_ScriptGeneration(t *testing.T) {
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
	worktrees.CreatePath = "/path/to/worktree"
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

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
	assert.Contains(t, script, "opencode \"$PROMPT\"")

	// Verify script is executable (mode 0700)
	info, err := os.Stat(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm(), "script should have 0700 permissions")
}

func TestStartTask_CleanupScript(t *testing.T) {
	crewDir := t.TempDir()
	scriptsDir := filepath.Join(crewDir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0755))

	// Create dummy file
	scriptPath := domain.ScriptPath(crewDir, 1)
	require.NoError(t, os.WriteFile(scriptPath, []byte("test"), 0755))

	// Create use case
	uc := NewStartTask(nil, nil, nil, nil, nil, crewDir)

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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir)

	// Execute without agent (will try to load default from config)
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "", // No agent specified, will try to load from config
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}
