// Package logging provides file-based logging for git-crew.
// It outputs logs to both a global log file (.git/crew/logs/crew.log)
// and task-specific log files (.git/crew/logs/task-N.log).
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Ensure Logger implements domain.Logger interface.
var _ domain.Logger = (*Logger)(nil)

// Logger wraps slog.Logger with file-based output support.
// Fields are ordered to minimize memory padding.
type Logger struct {
	globalFile *os.File
	taskFiles  map[int]*os.File
	crewDir    string
	mu         sync.Mutex
	level      slog.Level
}

// New creates a new Logger that writes to the crew log directory.
// If crewDir is empty, logging is disabled (returns a no-op logger).
func New(crewDir string, level slog.Level) *Logger {
	return &Logger{
		crewDir:   crewDir,
		level:     level,
		taskFiles: make(map[int]*os.File),
	}
}

// ParseLevel parses a log level string into slog.Level.
func ParseLevel(levelStr string) slog.Level {
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ensureLogsDir creates the logs directory if it doesn't exist.
func (l *Logger) ensureLogsDir() error {
	logsDir := filepath.Join(l.crewDir, "logs")
	return os.MkdirAll(logsDir, 0o750)
}

// ensureGlobalFile opens or returns the global log file.
func (l *Logger) ensureGlobalFile() (*os.File, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.globalFile != nil {
		return l.globalFile, nil
	}

	if err := l.ensureLogsDir(); err != nil {
		return nil, fmt.Errorf("create logs directory: %w", err)
	}

	path := domain.GlobalLogPath(l.crewDir)
	// G302: Log files are append-only and need read access by repository users
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640) //nolint:gosec // Log file readable by owner and group
	if err != nil {
		return nil, fmt.Errorf("open global log file: %w", err)
	}
	l.globalFile = f
	return f, nil
}

// ensureTaskFile opens or returns the task log file.
func (l *Logger) ensureTaskFile(taskID int) (*os.File, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if f, ok := l.taskFiles[taskID]; ok {
		return f, nil
	}

	if err := l.ensureLogsDir(); err != nil {
		return nil, fmt.Errorf("create logs directory: %w", err)
	}

	path := domain.TaskLogPath(l.crewDir, taskID)
	// G302: Log files are append-only and need read access by repository users
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640) //nolint:gosec // Log file readable by owner and group
	if err != nil {
		return nil, fmt.Errorf("open task log file: %w", err)
	}
	l.taskFiles[taskID] = f
	return f, nil
}

// Close closes all open log files.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var lastErr error
	if l.globalFile != nil {
		if err := l.globalFile.Close(); err != nil {
			lastErr = err
		}
		l.globalFile = nil
	}
	for id, f := range l.taskFiles {
		if err := f.Close(); err != nil {
			lastErr = err
		}
		delete(l.taskFiles, id)
	}
	return lastErr
}

// formatLog formats a log entry in the specified format.
// Format: [2025-12-30 09:32:51] [INFO] [task-1] [category] message
func formatLog(t time.Time, level slog.Level, taskID int, category, msg string) string {
	levelStr := levelToString(level)
	taskStr := "global"
	if taskID > 0 {
		taskStr = fmt.Sprintf("task-%d", taskID)
	}
	return fmt.Sprintf("[%s] [%s] [%s] [%s] %s\n",
		t.Format("2006-01-02 15:04:05"),
		levelStr,
		taskStr,
		category,
		msg,
	)
}

func levelToString(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// log writes a log entry to appropriate files based on taskID.
// If taskID is 0, logs only to global log.
// If taskID > 0, logs to both global and task-specific log.
func (l *Logger) log(level slog.Level, taskID int, category, msg string) {
	if l.crewDir == "" {
		return // Logging disabled
	}

	if level < l.level {
		return // Skip if below minimum level
	}

	now := time.Now()
	entry := formatLog(now, level, taskID, category, msg)

	// Write to global log
	if gf, err := l.ensureGlobalFile(); err == nil {
		_, _ = io.WriteString(gf, entry)
	}

	// Write to task log if taskID is specified
	if taskID > 0 {
		if tf, err := l.ensureTaskFile(taskID); err == nil {
			_, _ = io.WriteString(tf, entry)
		}
	}
}

// Info logs an info message.
func (l *Logger) Info(taskID int, category, msg string) {
	l.log(slog.LevelInfo, taskID, category, msg)
}

// Debug logs a debug message.
func (l *Logger) Debug(taskID int, category, msg string) {
	l.log(slog.LevelDebug, taskID, category, msg)
}

// Warn logs a warning message.
func (l *Logger) Warn(taskID int, category, msg string) {
	l.log(slog.LevelWarn, taskID, category, msg)
}

// Error logs an error message.
func (l *Logger) Error(taskID int, category, msg string) {
	l.log(slog.LevelError, taskID, category, msg)
}
