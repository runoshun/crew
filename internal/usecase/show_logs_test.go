package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowLogs_Execute_Success(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}

	crewDir := t.TempDir()
	logsDir := filepath.Join(crewDir, "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	logContent := "line1\nline2\nline3\nline4\nline5\n"
	logPath := domain.SessionLogPath(crewDir, "crew-1")
	require.NoError(t, os.WriteFile(logPath, []byte(logContent), 0644))

	uc := NewShowLogs(repo, crewDir)

	// Execute
	out, err := uc.Execute(context.Background(), ShowLogsInput{
		TaskID: 1,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, logPath, out.LogPath)
	assert.Equal(t, logContent, out.Content)
}

func TestShowLogs_Execute_ReviewSession(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}

	crewDir := t.TempDir()
	logsDir := filepath.Join(crewDir, "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	logContent := "review log content\n"
	logPath := domain.SessionLogPath(crewDir, "crew-1-review")
	require.NoError(t, os.WriteFile(logPath, []byte(logContent), 0644))

	uc := NewShowLogs(repo, crewDir)

	// Execute
	out, err := uc.Execute(context.Background(), ShowLogsInput{
		TaskID: 1,
		Review: true,
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, logPath, out.LogPath)
	assert.Equal(t, logContent, out.Content)
}

func TestShowLogs_Execute_LastLines(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}

	crewDir := t.TempDir()
	logsDir := filepath.Join(crewDir, "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	logContent := "line1\nline2\nline3\nline4\nline5\n"
	logPath := domain.SessionLogPath(crewDir, "crew-1")
	require.NoError(t, os.WriteFile(logPath, []byte(logContent), 0644))

	uc := NewShowLogs(repo, crewDir)

	// Execute
	out, err := uc.Execute(context.Background(), ShowLogsInput{
		TaskID: 1,
		Lines:  3,
	})

	// Assert
	require.NoError(t, err)
	// Last 3 lines: "line4", "line5", "" (trailing newline creates empty last line)
	assert.Equal(t, "line4\nline5\n", out.Content)
}

func TestShowLogs_Execute_TaskNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	crewDir := t.TempDir()

	uc := NewShowLogs(repo, crewDir)

	// Execute
	_, err := uc.Execute(context.Background(), ShowLogsInput{
		TaskID: 999,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestShowLogs_Execute_NoLogFile(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusTodo,
	}

	crewDir := t.TempDir()

	uc := NewShowLogs(repo, crewDir)

	// Execute
	_, err := uc.Execute(context.Background(), ShowLogsInput{
		TaskID: 1,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrNoSession)
}

func TestShowLogs_Execute_GetTaskError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	crewDir := t.TempDir()

	uc := NewShowLogs(repo, crewDir)

	// Execute
	_, err := uc.Execute(context.Background(), ShowLogsInput{
		TaskID: 1,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}
