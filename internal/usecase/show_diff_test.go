package usecase

import (
	"bytes"
	"context"
	"os/exec"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowDiff_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with diff",
		Status:     domain.StatusInProgress,
		BaseBranch: "main",
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	// Track command execution
	var capturedCommand string
	uc.SetExecCmd(func(name string, args ...string) *exec.Cmd {
		// Capture the command
		if len(args) >= 2 {
			capturedCommand = args[1]
		}
		return exec.Command("echo", "diff output")
	})

	// Execute
	out, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "/tmp/worktree", out.WorktreePath)
	// Verify default command template was used with BaseBranch
	assert.Contains(t, capturedCommand, "git diff main...HEAD")
}

func TestShowDiff_Execute_WithArgs(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task with diff",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	// Track command execution
	var capturedCommand string
	uc.SetExecCmd(func(name string, args ...string) *exec.Cmd {
		if len(args) >= 2 {
			capturedCommand = args[1]
		}
		return exec.Command("echo", "diff output")
	})

	// Execute with extra args
	out, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 1,
		Args:   []string{"--stat", "--color"},
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	// Verify args are expanded in the command
	assert.Contains(t, capturedCommand, "--stat --color")
}

func TestShowDiff_Execute_WithConfiguredCommand(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task with diff",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config = &domain.Config{
		Diff: domain.DiffConfig{
			Command: "git diff origin/main...HEAD{{if .Args}} {{.Args}}{{end}} | delta",
		},
		Agents: make(map[string]domain.Agent),
	}

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	// Track command execution
	var capturedCommand string
	uc.SetExecCmd(func(name string, args ...string) *exec.Cmd {
		if len(args) >= 2 {
			capturedCommand = args[1]
		}
		return exec.Command("echo", "diff output")
	})

	// Execute
	out, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 1,
		Args:   []string{"--stat"},
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	// Verify configured command was used
	assert.Contains(t, capturedCommand, "origin/main")
	assert.Contains(t, capturedCommand, "delta")
	assert.Contains(t, capturedCommand, "--stat")
}

func TestShowDiff_Execute_TaskWithIssue(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task with issue",
		Status: domain.StatusInProgress,
		Issue:  123,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree-gh-123"

	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	uc.SetExecCmd(func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "diff output")
	})

	// Execute
	out, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "/tmp/worktree-gh-123", out.WorktreePath)
}

func TestShowDiff_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestShowDiff_Execute_GetError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError

	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}

func TestShowDiff_Execute_WorktreeResolveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolveErr = assert.AnError

	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resolve worktree")
}

func TestShowDiff_Execute_ConfigLoadError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()
	configLoader.LoadErr = assert.AnError

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestShowDiff_Execute_InvalidTemplate(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Task",
		Status: domain.StatusInProgress,
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config = &domain.Config{
		Diff: domain.DiffConfig{
			Command: "{{.Invalid}", // Invalid template
		},
		Agents: make(map[string]domain.Agent),
	}

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	// Execute
	_, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse diff command template")
}

func TestShowDiff_Execute_NoArgsNoExpansion(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task",
		Status:     domain.StatusInProgress,
		BaseBranch: "main",
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	var capturedCommand string
	uc.SetExecCmd(func(name string, args ...string) *exec.Cmd {
		if len(args) >= 2 {
			capturedCommand = args[1]
		}
		return exec.Command("echo", "diff output")
	})

	// Execute without args
	_, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 1,
		Args:   nil,
	})

	// Assert
	require.NoError(t, err)
	// Verify command doesn't have trailing space from Args expansion
	assert.Equal(t, "git diff main...HEAD", capturedCommand)
}

func TestShowDiff_Execute_CustomBaseBranch(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with custom base",
		Status:     domain.StatusInProgress,
		BaseBranch: "develop",
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	var capturedCommand string
	uc.SetExecCmd(func(name string, args ...string) *exec.Cmd {
		if len(args) >= 2 {
			capturedCommand = args[1]
		}
		return exec.Command("echo", "diff output")
	})

	// Execute
	_, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	// Verify BaseBranch is expanded correctly
	assert.Equal(t, "git diff develop...HEAD", capturedCommand)
}

func TestShowDiff_Execute_EmptyBaseBranchDefaultsToMain(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task without base branch",
		Status:     domain.StatusInProgress,
		BaseBranch: "", // Empty base branch
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	var capturedCommand string
	uc.SetExecCmd(func(name string, args ...string) *exec.Cmd {
		if len(args) >= 2 {
			capturedCommand = args[1]
		}
		return exec.Command("echo", "diff output")
	})

	// Execute
	_, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	// Verify empty BaseBranch defaults to "main"
	assert.Equal(t, "git diff main...HEAD", capturedCommand)
}

func TestShowDiff_Execute_UseTUICommand(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with TUI diff",
		Status:     domain.StatusInProgress,
		BaseBranch: "main",
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config = &domain.Config{
		Diff: domain.DiffConfig{
			Command:    "git diff main...HEAD | delta",
			TUICommand: "git diff --color main...HEAD | less -R",
		},
		Agents: make(map[string]domain.Agent),
	}

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	var capturedCommand string
	uc.SetExecCmd(func(name string, args ...string) *exec.Cmd {
		if len(args) >= 2 {
			capturedCommand = args[1]
		}
		return exec.Command("echo", "diff output")
	})

	// Execute with UseTUICommand = true
	_, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID:        1,
		UseTUICommand: true,
	})

	// Assert
	require.NoError(t, err)
	// Verify TUICommand was used instead of Command
	assert.Contains(t, capturedCommand, "less -R")
	assert.NotContains(t, capturedCommand, "delta")
}

func TestShowDiff_Execute_UseTUICommandFallbackToCommand(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with no TUI command",
		Status:     domain.StatusInProgress,
		BaseBranch: "main",
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config = &domain.Config{
		Diff: domain.DiffConfig{
			Command:    "git diff main...HEAD | delta",
			TUICommand: "", // Empty TUICommand
		},
		Agents: make(map[string]domain.Agent),
	}

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	var capturedCommand string
	uc.SetExecCmd(func(name string, args ...string) *exec.Cmd {
		if len(args) >= 2 {
			capturedCommand = args[1]
		}
		return exec.Command("echo", "diff output")
	})

	// Execute with UseTUICommand = true but TUICommand is empty
	_, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID:        1,
		UseTUICommand: true,
	})

	// Assert
	require.NoError(t, err)
	// Verify Command was used as fallback when TUICommand is empty
	assert.Contains(t, capturedCommand, "delta")
}

func TestShowDiff_Execute_UseTUICommandFalse(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:         1,
		Title:      "Task with CLI diff",
		Status:     domain.StatusInProgress,
		BaseBranch: "main",
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ResolvePath = "/tmp/worktree"

	configLoader := testutil.NewMockConfigLoader()
	configLoader.Config = &domain.Config{
		Diff: domain.DiffConfig{
			Command:    "git diff main...HEAD | delta",
			TUICommand: "git diff --color main...HEAD | less -R",
		},
		Agents: make(map[string]domain.Agent),
	}

	var stdout, stderr bytes.Buffer
	uc := NewShowDiff(repo, worktrees, &testutil.MockGit{}, configLoader, &stdout, &stderr)

	var capturedCommand string
	uc.SetExecCmd(func(name string, args ...string) *exec.Cmd {
		if len(args) >= 2 {
			capturedCommand = args[1]
		}
		return exec.Command("echo", "diff output")
	})

	// Execute with UseTUICommand = false (default)
	_, err := uc.Execute(context.Background(), ShowDiffInput{
		TaskID:        1,
		UseTUICommand: false,
	})

	// Assert
	require.NoError(t, err)
	// Verify Command was used (not TUICommand) when UseTUICommand is false
	assert.Contains(t, capturedCommand, "delta")
	assert.NotContains(t, capturedCommand, "less -R")
}
