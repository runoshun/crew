package usecase

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	repoRoot := t.TempDir()
	worktreeDir := t.TempDir() // Use temp directory for worktree

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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	worktreeDir := t.TempDir()

	// Create .git/info directory for exclude patterns
	gitInfoDir := filepath.Join(repoRoot, ".git", "info")
	require.NoError(t, os.MkdirAll(gitInfoDir, 0755))

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
	// Configure worker with agent reference
	configLoader.Config.Agents["test-agent"] = domain.Agent{
		WorktreeSetupScript: "echo 'setup done'",
		ExcludePatterns:     []string{".test-exclude/"},
	}
	configLoader.Config.Workers["custom"] = domain.Worker{
		Agent:           "test-agent",
		CommandTemplate: "{{.Command}} {{.Prompt}}",
		Command:         "custom-cmd",
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

	// Execute
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "custom",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify exclude patterns were applied
	excludePath := filepath.Join(repoRoot, ".git", "info", "exclude")
	excludeContent, err := os.ReadFile(excludePath)
	require.NoError(t, err)
	assert.Contains(t, string(excludeContent), ".test-exclude/")

	// Verify task uses custom agent
	task := repo.Tasks[1]
	assert.Equal(t, "custom", task.Agent)
}

func TestStartTask_Execute_ExcludePatternsNoDuplicate(t *testing.T) {
	// Setup temp directories
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := t.TempDir()

	// Create .git/info directory with existing exclude pattern
	gitInfoDir := filepath.Join(repoRoot, ".git", "info")
	require.NoError(t, os.MkdirAll(gitInfoDir, 0755))
	excludePath := filepath.Join(gitInfoDir, "exclude")
	require.NoError(t, os.WriteFile(excludePath, []byte(".existing-pattern/\n"), 0644))

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
	// Configure agent with exclude patterns (including one that already exists)
	configLoader.Config.Agents["test-agent"] = domain.Agent{
		ExcludePatterns: []string{".existing-pattern/", ".new-pattern/"},
	}
	configLoader.Config.Workers["custom"] = domain.Worker{
		Agent:           "test-agent",
		CommandTemplate: "{{.Command}} {{.Prompt}}",
		Command:         "custom-cmd",
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "custom",
	})

	// Assert
	require.NoError(t, err)

	// Verify exclude patterns: .existing-pattern/ should NOT be duplicated
	excludeContent, err := os.ReadFile(excludePath)
	require.NoError(t, err)
	content := string(excludeContent)
	assert.Contains(t, content, ".new-pattern/")
	// Count occurrences of .existing-pattern/
	count := strings.Count(content, ".existing-pattern/")
	assert.Equal(t, 1, count, "existing pattern should not be duplicated")
}

func TestStartTask_Execute_FromInReview(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := t.TempDir()

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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	worktreeDir := t.TempDir()

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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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

			repoRoot := t.TempDir()
			uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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

	repoRoot := t.TempDir()
	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	worktreeDir := t.TempDir()

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
	// default worker is already set up by NewMockConfigLoader (workers.default)
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

	// Execute without specifying agent
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "", // No agent specified
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify task uses default worker name
	task := repo.Tasks[1]
	assert.Equal(t, domain.DefaultWorkerName, task.Agent)

	// Verify script uses the underlying agent command (opencode)
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
	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	worktreeDir := t.TempDir()

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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	worktreeDir := t.TempDir()

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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	worktreeDir := t.TempDir()

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
	configLoader.Config.WorkersConfig.Prompt = "Task #{{.TaskID}}{{if .Issue}} (Issue #{{.Issue}}){{end}}"
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	worktreeDir := t.TempDir()

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
	configLoader.Config.WorkersConfig.Prompt = "Task: {{.Title}}\n{{.Description}}"
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	worktreeDir := t.TempDir()

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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	uc := NewStartTask(nil, nil, nil, nil, nil, crewDir, "")

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
	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

	// Execute without agent (will try to load default from config)
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "", // No agent specified, will try to load from config
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestStartTask_Execute_WithCustomAgent(t *testing.T) {
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
	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

	// Execute with unknown agent name (custom agent)
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "my-custom-agent",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify task uses custom agent
	task := repo.Tasks[1]
	assert.Equal(t, "my-custom-agent", task.Agent)

	// Verify script uses custom agent with simple command format
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	script := string(scriptContent)
	assert.Contains(t, script, "my-custom-agent")
	assert.Contains(t, script, `"$PROMPT"`)
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
	configLoader.Config.Workers["my-agent"] = domain.Worker{
		CommandTemplate: "{{.Command}} {{.Args}} {{.Prompt}}",
		Command:         "my-agent-bin",
		Args:            "--custom-flag",
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	repoRoot := t.TempDir()
	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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

func TestStartTask_Execute_WithWorkerPrompt(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := t.TempDir()

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
	// Configure worker-specific prompt
	configLoader.Config.Workers["claude"] = domain.Worker{
		CommandTemplate: "{{.Command}} {{.Prompt}}",
		Command:         "claude",
		Prompt:          "Custom worker prompt for claude",
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

	// Execute
	_, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
	})

	// Assert
	require.NoError(t, err)

	// Verify script contains worker-specific prompt
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	assert.Contains(t, string(scriptContent), "Custom worker prompt for claude")
}

func TestStartTask_Execute_WithModelOverride(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := t.TempDir()

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
	configLoader.Config.Workers["claude"] = domain.Worker{
		CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
		Command:         "claude",
		SystemArgs:      "--model {{.Model}}",
		Model:           "config-model",
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	worktreeDir := t.TempDir()

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
	configLoader.Config.Workers["claude"] = domain.Worker{
		CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
		Command:         "claude",
		SystemArgs:      "--model {{.Model}}",
		Model:           "config-model",
	}
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
	worktreeDir := t.TempDir()

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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

	// Execute without model (should use agent's default)
	out, err := uc.Execute(context.Background(), StartTaskInput{
		TaskID: 1,
		Agent:  "claude",
		Model:  "", // No model specified
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "crew-1", out.SessionName)

	// Verify script uses default model from BuiltinWorkers
	scriptContent, err := os.ReadFile(domain.ScriptPath(crewDir, 1))
	require.NoError(t, err)
	script := string(scriptContent)
	// Default model for claude is "opus" (from BuiltinWorkers)
	assert.Contains(t, script, "--model opus")
}

func TestStartTask_Execute_OpencodeWithModelOverride(t *testing.T) {
	crewDir := t.TempDir()
	repoRoot := t.TempDir()
	worktreeDir := t.TempDir()

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

	uc := NewStartTask(repo, sessions, worktrees, configLoader, clock, crewDir, repoRoot)

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
