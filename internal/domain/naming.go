package domain

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
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

// WorktreePath returns the path to a worktree for a task.
func WorktreePath(crewDir string, taskID int) string {
	return filepath.Join(crewDir, "worktrees", fmt.Sprintf("%d", taskID))
}

// branchPattern matches crew branch names: crew-<id> or crew-<id>-gh-<issue>
var branchPattern = regexp.MustCompile(`^crew-(\d+)(?:-gh-\d+)?$`)

// ParseBranchTaskID extracts the task ID from a branch name.
// Returns the task ID and true if the branch follows the crew naming convention,
// or 0 and false if not.
func ParseBranchTaskID(branch string) (int, bool) {
	matches := branchPattern.FindStringSubmatch(branch)
	if matches == nil {
		return 0, false
	}
	id, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, false
	}
	return id, true
}
