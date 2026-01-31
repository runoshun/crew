package usecase

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ShowLogsInput contains the parameters for showing session logs.
type ShowLogsInput struct {
	TaskID int // Task ID to show logs for
	Lines  int // Number of lines to display from the end (0 = all)
}

// ShowLogsOutput contains the result of showing session logs.
type ShowLogsOutput struct {
	LogPath string // Path to the log file
	Content string // Log file content
}

// ShowLogs is the use case for viewing session logs.
type ShowLogs struct {
	tasks   domain.TaskRepository
	crewDir string
}

// NewShowLogs creates a new ShowLogs use case.
func NewShowLogs(
	tasks domain.TaskRepository,
	crewDir string,
) *ShowLogs {
	return &ShowLogs{
		tasks:   tasks,
		crewDir: crewDir,
	}
}

// Execute reads and returns the session log content.
func (uc *ShowLogs) Execute(_ context.Context, in ShowLogsInput) (*ShowLogsOutput, error) {
	// Get task to verify it exists
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	// Get session name
	sessionName := domain.SessionName(task.ID)

	// Get log file path
	logPath := domain.SessionLogPath(uc.crewDir, sessionName)

	// Read log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no log file found for session %s: %w", sessionName, domain.ErrNoSession)
		}
		return nil, fmt.Errorf("read log file: %w", err)
	}

	// If lines is specified, get only the last N lines
	result := string(content)
	if in.Lines > 0 {
		lines := strings.Split(result, "\n")
		if len(lines) > in.Lines {
			lines = lines[len(lines)-in.Lines:]
		}
		result = strings.Join(lines, "\n")
	}

	return &ShowLogsOutput{
		LogPath: logPath,
		Content: result,
	}, nil
}
