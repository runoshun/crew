package domain

import (
	"fmt"
	"path/filepath"
)

// BranchName returns the branch name for a task.
// Format: crew-<id> or crew-<id>-gh-<issue> if issue is linked.
func BranchName(taskID int, issue int) string {
	if issue > 0 {
		return fmt.Sprintf("crew-%d-gh-%d", taskID, issue)
	}
	return fmt.Sprintf("crew-%d", taskID)
}

// SessionName returns the tmux session name for a task.
// Format: crew-<id>
func SessionName(taskID int) string {
	return fmt.Sprintf("crew-%d", taskID)
}

// ScriptPath returns the path to the task script.
func ScriptPath(crewDir string, taskID int) string {
	return filepath.Join(crewDir, "scripts", fmt.Sprintf("task-%d.sh", taskID))
}

// PromptPath returns the path to the task prompt file.
func PromptPath(crewDir string, taskID int) string {
	return filepath.Join(crewDir, "scripts", fmt.Sprintf("task-%d-prompt.txt", taskID))
}

// ReviewScriptPath returns the path to the review script.
func ReviewScriptPath(crewDir string, taskID int) string {
	return filepath.Join(crewDir, "scripts", fmt.Sprintf("review-%d.sh", taskID))
}

// ReviewPromptPath returns the path to the review prompt file.
func ReviewPromptPath(crewDir string, taskID int) string {
	return filepath.Join(crewDir, "scripts", fmt.Sprintf("review-%d-prompt.txt", taskID))
}

// TaskLogPath returns the path to the task log file.
func TaskLogPath(crewDir string, taskID int) string {
	return filepath.Join(crewDir, "logs", fmt.Sprintf("task-%d.log", taskID))
}

// GlobalLogPath returns the path to the global log file.
func GlobalLogPath(crewDir string) string {
	return filepath.Join(crewDir, "logs", "crew.log")
}

// TasksStorePath returns the path to the tasks.json file.
func TasksStorePath(crewDir string) string {
	return filepath.Join(crewDir, "tasks.json")
}

// TmuxSocketPath returns the path to the tmux socket.
func TmuxSocketPath(crewDir string) string {
	return filepath.Join(crewDir, "tmux.sock")
}

// TmuxConfigPath returns the path to the tmux configuration.
func TmuxConfigPath(crewDir string) string {
	return filepath.Join(crewDir, "tmux.conf")
}

// ConfigPath returns the path to the repository config file.
func ConfigPath(crewDir string) string {
	return filepath.Join(crewDir, "config.toml")
}
