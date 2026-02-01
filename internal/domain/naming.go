package domain

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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

// ReviewSessionName returns the tmux session name for a task review.
// Format: crew-<id>-review
func ReviewSessionName(taskID int) string {
	return fmt.Sprintf("crew-%d-review", taskID)
}

// ManagerSessionName returns the tmux session name for the manager.
// Format: crew-manager (fixed, only one manager session)
func ManagerSessionName() string {
	return "crew-manager"
}

// ScriptPath returns the path to the task script.
func ScriptPath(crewDir string, taskID int) string {
	return filepath.Join(crewDir, "scripts", fmt.Sprintf("task-%d.sh", taskID))
}

// PromptPath returns the path to the task prompt file.
func PromptPath(crewDir string, taskID int) string {
	return filepath.Join(crewDir, "scripts", fmt.Sprintf("task-%d-prompt.txt", taskID))
}

// TaskLogPath returns the path to the task log file.
func TaskLogPath(crewDir string, taskID int) string {
	return filepath.Join(crewDir, "logs", fmt.Sprintf("task-%d.log", taskID))
}

// GlobalLogPath returns the path to the global log file.
func GlobalLogPath(crewDir string) string {
	return filepath.Join(crewDir, "logs", "crew.log")
}

// SessionLogPath returns the path to the session log file.
// The session log captures stderr from the tmux session.
func SessionLogPath(crewDir string, sessionName string) string {
	return filepath.Join(crewDir, "logs", sessionName+".log")
}

// TasksStorePath returns the path to the tasks directory.
func TasksStorePath(crewDir string) string {
	return filepath.Join(crewDir, "tasks")
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
// worktreeDir should be the base directory for worktrees (e.g., .crew/worktrees).
func WorktreePath(worktreeDir string, taskID int) string {
	return filepath.Join(worktreeDir, fmt.Sprintf("%d", taskID))
}

// ManagerScriptPath returns the path to the manager script.
func ManagerScriptPath(crewDir string) string {
	return filepath.Join(crewDir, "scripts", "manager.sh")
}

// WorkspacesFileName is the name of the workspaces file.
const WorkspacesFileName = "workspaces.toml"

// WorkspacesFilePath returns the path to the workspaces.toml file.
// globalCrewDir is typically ~/.config/crew (resolved by caller).
func WorkspacesFilePath(globalCrewDir string) string {
	return filepath.Join(globalCrewDir, WorkspacesFileName)
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

// NamespaceFromEmail derives a namespace from the local part of an email address.
// Returns empty string if the email is invalid or cannot be sanitized.
func NamespaceFromEmail(email string) string {
	if email == "" {
		return ""
	}
	idx := strings.Index(email, "@")
	if idx <= 0 {
		return ""
	}
	local := email[:idx]
	return SanitizeNamespace(local)
}

// SanitizeNamespace converts a raw namespace string to a safe format.
// Keeps lowercase letters and digits, converts other characters to single hyphens.
func SanitizeNamespace(input string) string {
	if input == "" {
		return ""
	}
	var b strings.Builder
	prevDash := false
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if ch >= 'A' && ch <= 'Z' {
			ch = ch - 'A' + 'a'
		}
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			b.WriteByte(ch)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	result := strings.Trim(b.String(), "-")
	return result
}
