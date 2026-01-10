package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo}, // default
		{"", slog.LevelInfo},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseLevel(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestLogger_Info(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	logger := New(crewDir, slog.LevelInfo)
	defer func() { _ = logger.Close() }()

	// Execute
	logger.Info(1, "task", "test message")

	// Verify global log
	globalLogPath := domain.GlobalLogPath(crewDir)
	content, err := os.ReadFile(globalLogPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "[INFO]")
	assert.Contains(t, string(content), "[task-1]")
	assert.Contains(t, string(content), "[task]")
	assert.Contains(t, string(content), "test message")

	// Verify task log
	taskLogPath := domain.TaskLogPath(crewDir, 1)
	taskContent, err := os.ReadFile(taskLogPath)
	require.NoError(t, err)
	assert.Contains(t, string(taskContent), "[INFO]")
	assert.Contains(t, string(taskContent), "[task-1]")
	assert.Contains(t, string(taskContent), "test message")
}

func TestLogger_GlobalLogOnly(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	logger := New(crewDir, slog.LevelInfo)
	defer func() { _ = logger.Close() }()

	// Execute with taskID = 0 (global only)
	logger.Info(0, "system", "global message")

	// Verify global log
	globalLogPath := domain.GlobalLogPath(crewDir)
	content, err := os.ReadFile(globalLogPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "[global]")
	assert.Contains(t, string(content), "global message")

	// Verify no task-0 log file was created
	taskLogPath := domain.TaskLogPath(crewDir, 0)
	_, err = os.Stat(taskLogPath)
	assert.True(t, os.IsNotExist(err))
}

func TestLogger_LevelFiltering(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	logger := New(crewDir, slog.LevelWarn) // Only warn and above
	defer func() { _ = logger.Close() }()

	// Execute
	logger.Debug(1, "task", "debug message")
	logger.Info(1, "task", "info message")
	logger.Warn(1, "task", "warn message")
	logger.Error(1, "task", "error message")

	// Verify global log (debug and info should be filtered)
	globalLogPath := domain.GlobalLogPath(crewDir)
	content, err := os.ReadFile(globalLogPath)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "debug message")
	assert.NotContains(t, string(content), "info message")
	assert.Contains(t, string(content), "warn message")
	assert.Contains(t, string(content), "error message")
}

func TestLogger_DisabledWhenEmptyCrewDir(t *testing.T) {
	// Setup with empty crewDir
	logger := New("", slog.LevelInfo)
	defer func() { _ = logger.Close() }()

	// Execute - should not panic
	logger.Info(1, "task", "test message")
	logger.Debug(1, "task", "debug message")
	logger.Warn(1, "task", "warn message")
	logger.Error(1, "task", "error message")

	// No assertion needed - just verify no panic
}

func TestLogger_LogFormat(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	logger := New(crewDir, slog.LevelInfo)
	defer func() { _ = logger.Close() }()

	// Execute
	logger.Info(42, "usecase", `task created: "my task"`)

	// Verify format
	globalLogPath := domain.GlobalLogPath(crewDir)
	content, err := os.ReadFile(globalLogPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Len(t, lines, 1)

	// Verify format: [timestamp] [INFO] [task-42] [usecase] message
	line := lines[0]
	assert.Contains(t, line, "[INFO]")
	assert.Contains(t, line, "[task-42]")
	assert.Contains(t, line, "[usecase]")
	assert.Contains(t, line, `task created: "my task"`)
}

func TestLogger_MultipleTaskFiles(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	logger := New(crewDir, slog.LevelInfo)
	defer func() { _ = logger.Close() }()

	// Log to multiple tasks
	logger.Info(1, "task", "message for task 1")
	logger.Info(2, "task", "message for task 2")
	logger.Info(1, "task", "another message for task 1")

	// Verify global log has all messages
	globalLogPath := domain.GlobalLogPath(crewDir)
	globalContent, err := os.ReadFile(globalLogPath)
	require.NoError(t, err)
	assert.Contains(t, string(globalContent), "message for task 1")
	assert.Contains(t, string(globalContent), "message for task 2")
	assert.Contains(t, string(globalContent), "another message for task 1")

	// Verify task 1 log
	task1LogPath := domain.TaskLogPath(crewDir, 1)
	task1Content, err := os.ReadFile(task1LogPath)
	require.NoError(t, err)
	assert.Contains(t, string(task1Content), "message for task 1")
	assert.Contains(t, string(task1Content), "another message for task 1")
	assert.NotContains(t, string(task1Content), "message for task 2")

	// Verify task 2 log
	task2LogPath := domain.TaskLogPath(crewDir, 2)
	task2Content, err := os.ReadFile(task2LogPath)
	require.NoError(t, err)
	assert.Contains(t, string(task2Content), "message for task 2")
	assert.NotContains(t, string(task2Content), "message for task 1")
}

func TestLogger_Close(t *testing.T) {
	// Setup
	crewDir := t.TempDir()
	logger := New(crewDir, slog.LevelInfo)

	// Write some logs
	logger.Info(1, "task", "test message")

	// Close
	err := logger.Close()
	assert.NoError(t, err)

	// Verify files exist
	globalLogPath := domain.GlobalLogPath(crewDir)
	assert.FileExists(t, globalLogPath)

	taskLogPath := domain.TaskLogPath(crewDir, 1)
	assert.FileExists(t, taskLogPath)
}

func TestLogger_CreateLogsDir(t *testing.T) {
	// Setup - crewDir exists but logs subdir doesn't
	crewDir := t.TempDir()
	logsDir := filepath.Join(crewDir, "logs")

	// Verify logs dir doesn't exist initially
	_, err := os.Stat(logsDir)
	assert.True(t, os.IsNotExist(err))

	// Create logger and write log
	logger := New(crewDir, slog.LevelInfo)
	defer func() { _ = logger.Close() }()
	logger.Info(1, "task", "test message")

	// Verify logs dir was created
	stat, err := os.Stat(logsDir)
	require.NoError(t, err)
	assert.True(t, stat.IsDir())
}
