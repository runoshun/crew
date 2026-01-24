// Package tmux provides tmux session management.
package tmux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ExecFunc is the function signature for syscall.Exec.
// It is used to allow testing of the Attach method.
type ExecFunc func(argv0 string, argv []string, envv []string) error

// Client manages tmux sessions for git-crew.
// Fields are ordered to minimize memory padding.
type Client struct {
	execFunc   ExecFunc // Function to use for exec (default: syscall.Exec)
	socketPath string   // Path to the tmux socket
	configPath string   // Path to tmux configuration
	crewDir    string   // Path to .git/crew directory
}

// NewClient creates a new tmux client.
// socketPath is the path to the tmux socket (typically .git/crew/tmux.sock).
// crewDir is the path to .git/crew directory.
func NewClient(socketPath, crewDir string) *Client {
	return &Client{
		socketPath: socketPath,
		configPath: filepath.Join(crewDir, "tmux.conf"),
		crewDir:    crewDir,
		execFunc:   syscall.Exec,
	}
}

// SetExecFunc sets the exec function for testing purposes.
// This allows tests to verify the arguments passed to syscall.Exec
// without actually replacing the process.
func (c *Client) SetExecFunc(fn ExecFunc) {
	c.execFunc = fn
}

// Ensure Client implements domain.SessionManager interface.
var _ domain.SessionManager = (*Client)(nil)

// Start creates and starts a new tmux session.
func (c *Client) Start(ctx context.Context, opts domain.StartSessionOptions) error {
	// Check if session already exists
	running, err := c.IsRunning(opts.Name)
	if err != nil {
		return fmt.Errorf("check session: %w", err)
	}
	if running {
		return domain.ErrSessionRunning
	}

	// Write session header to log file before starting
	logPath := domain.SessionLogPath(c.crewDir, opts.Name)
	if err := c.writeSessionHeader(logPath, opts); err != nil {
		return fmt.Errorf("write session header: %w", err)
	}

	// Build tmux command
	// tmux -S <socket> -f <config> new-session -d -s <name> -c <dir> <command>
	args := []string{
		"-S", c.socketPath,
		"-f", c.configPath,
		"new-session",
		"-d",            // Detached
		"-s", opts.Name, // Session name
		"-c", opts.Dir, // Working directory
	}

	// Add command if provided
	if opts.Command != "" {
		args = append(args, opts.Command)
	}

	cmd := exec.CommandContext(ctx, "tmux", args...)
	cmd.Dir = opts.Dir

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start session: %w: %s", err, string(out))
	}

	// Configure session logging with pipe-pane
	if err := c.configureSessionLogging(opts.Name, logPath); err != nil {
		return fmt.Errorf("configure session logging: %w", err)
	}

	// Configure status bar
	if err := c.configureStatusBar(opts.Name, opts.TaskID, opts.TaskTitle, opts.TaskAgent, opts.IsReview); err != nil {
		return fmt.Errorf("configure status bar: %w", err)
	}

	return nil
}

// Stop terminates a tmux session.
// It first sends SIGTERM to the process group of each pane to terminate all processes
// (including nested children like AI agents), then kills the session itself.
// This ensures that child processes are properly terminated and don't become orphaned.
func (c *Client) Stop(sessionName string) error {
	// Check if session exists
	running, err := c.IsRunning(sessionName)
	if err != nil {
		return fmt.Errorf("check session: %w", err)
	}
	if !running {
		return nil // Session already stopped, nothing to do
	}

	// Get all pane PIDs in the session
	// tmux -S <socket> list-panes -t <name> -F '#{pane_pid}'
	// Session names follow our naming convention (crew-N) and are safe to pass to tmux.
	listCmd := exec.Command("tmux", //nolint:gosec // sessionName follows crew-N naming convention
		"-S", c.socketPath,
		"list-panes",
		"-t", sessionName,
		"-F", "#{pane_pid}",
	)
	out, err := listCmd.Output()
	if err == nil && len(out) > 0 {
		// Send SIGTERM to the process group of each pane
		// Using negative PID sends the signal to the entire process group,
		// which terminates all child processes including nested ones (like AI agents)
		pids := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, pidStr := range pids {
			if pidStr == "" {
				continue
			}
			// Parse PID
			var pid int
			if _, parseErr := fmt.Sscanf(pidStr, "%d", &pid); parseErr != nil || pid <= 0 {
				continue
			}
			// Send SIGTERM to the process group (negative PID)
			// syscall.Kill with negative pid sends signal to all processes in the group
			// We ignore ESRCH (no such process) as the process may have already exited
			_ = syscall.Kill(-pid, syscall.SIGTERM)

			// Wait for processes to terminate (max 5 seconds)
			// If the process doesn't exit after the first SIGTERM, send a second SIGTERM
			// after a short delay to ensure it terminates (experimentally required).
			for i := 0; i < 50; i++ {
				time.Sleep(100 * time.Millisecond)
				// Send SIGTERM again after ~500ms (i == 5)
				if i == 5 {
					_ = syscall.Kill(-pid, syscall.SIGTERM)
				}
				// Check if process still exists
				// syscall.Kill with 0 signal checks for existence
				if err := syscall.Kill(pid, 0); err != nil {
					break // Process terminated
				}
			}
		}
	}
	// If list-panes fails, we still try to kill the session

	// tmux -S <socket> kill-session -t <name>
	// Session names follow our naming convention (crew-N) and are safe to pass to tmux.
	cmd := exec.Command("tmux", "-S", c.socketPath, "kill-session", "-t", sessionName) //nolint:gosec // sessionName follows crew-N naming convention
	if out, err := cmd.CombinedOutput(); err != nil {
		// Check if the session still exists - if it's already gone, that's fine
		// (child process termination may have caused the session to exit)
		stillRunning, checkErr := c.IsRunning(sessionName)
		if checkErr != nil || stillRunning {
			// Session still running or check failed - report the original error
			return fmt.Errorf("stop session: %w: %s", err, string(out))
		}
		// Session no longer exists, which is what we wanted
	}

	return nil
}

// Attach attaches to a running tmux session.
// This replaces the current process with tmux.
func (c *Client) Attach(sessionName string) error {
	// Check if session exists
	running, err := c.IsRunning(sessionName)
	if err != nil {
		return fmt.Errorf("check session: %w", err)
	}
	if !running {
		return domain.ErrNoSession
	}

	// tmux -S <socket> -f <config> attach -t <name>
	args := []string{
		"-S", c.socketPath,
		"-f", c.configPath,
		"attach",
		"-t", sessionName,
	}

	// Use syscall.Exec to replace the current process
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("find tmux: %w", err)
	}

	// Prepend "tmux" as argv[0]
	execArgs := append([]string{"tmux"}, args...)

	if err := c.execFunc(tmuxPath, execArgs, os.Environ()); err != nil {
		return fmt.Errorf("attach session: %w", err)
	}

	// This line should never be reached
	return nil
}

// Peek captures the last N lines from a session.
func (c *Client) Peek(sessionName string, lines int, escape bool) (string, error) {
	// Check if session exists
	running, err := c.IsRunning(sessionName)
	if err != nil {
		return "", fmt.Errorf("check session: %w", err)
	}
	if !running {
		return "", domain.ErrNoSession
	}

	// tmux -S <socket> capture-pane -t <name> -p -S -<lines>
	// -p: print to stdout
	// -e: include escape sequences
	// -S -<lines>: start capture from <lines> lines before the current position
	// Session names follow our naming convention (crew-N) and are safe to pass to tmux.
	args := []string{
		"-S", c.socketPath,
		"capture-pane",
		"-t", sessionName,
		"-p",
		"-S", fmt.Sprintf("-%d", lines),
	}
	if escape {
		args = append(args, "-e")
	}

	cmd := exec.Command("tmux", args...)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("peek session: %w", err)
	}

	return strings.TrimSuffix(string(out), "\n"), nil
}

// Send sends keys to a session.
func (c *Client) Send(sessionName string, keys string) error {
	// Check if session exists
	running, err := c.IsRunning(sessionName)
	if err != nil {
		return fmt.Errorf("check session: %w", err)
	}
	if !running {
		return domain.ErrNoSession
	}

	// tmux -S <socket> send-keys -t <name> <keys>
	// Session names follow our naming convention (crew-N) and are safe to pass to tmux.
	// Keys are user input but tmux send-keys handles them safely.
	cmd := exec.Command("tmux", //nolint:gosec // sessionName follows crew-N naming convention
		"-S", c.socketPath,
		"send-keys",
		"-t", sessionName,
		keys,
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("send keys: %w: %s", err, string(out))
	}

	return nil
}

// IsRunning checks if a session is running.
func (c *Client) IsRunning(sessionName string) (bool, error) {
	// tmux -S <socket> has-session -t <name>
	// Exit code 0 = exists, 1 = doesn't exist
	// Session names follow our naming convention (crew-N) and are safe to pass to tmux.
	cmd := exec.Command("tmux", //nolint:gosec // sessionName follows crew-N naming convention
		"-S", c.socketPath,
		"has-session",
		"-t", sessionName,
	)

	err := cmd.Run()
	if err != nil {
		// Check if it's an exit error
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means session doesn't exist
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		// Other errors might mean tmux isn't running at all (socket doesn't exist)
		// which is fine - the session doesn't exist
		return false, nil
	}

	return true, nil
}

// configureStatusBar configures the status bar for a tmux session.
func (c *Client) configureStatusBar(sessionName string, taskID int, taskTitle, taskAgent string, isReview bool) error {
	// Colors
	blue := "#1e66f5"     // Catppuccin Blue - worker main background
	blueDark := "#1146b4" // Darker blue for C-g keybind

	purple := "#8839ef"     // Catppuccin Mauve - reviewer main background
	purpleDark := "#6023c0" // Darker mauve for C-g keybind

	mainBg := blue
	keyBg := blueDark

	// Use purple for reviewer sessions
	if isReview {
		mainBg = purple
		keyBg = purpleDark
	}

	btnBg := "#ffffff"     // White background for arrow button
	white := "#ffffff"     // White text
	black := "#1e1e2e"     // Black text for button
	lightText := "#cdd6f4" // Light text
	mutedText := "#bac2de" // Muted text

	// Truncate title if too long
	title := taskTitle
	if len(title) > 30 {
		title = title[:27] + "..."
	}

	// Shorten worker name
	worker := shortenWorkerName(taskAgent)

	// Build status-left: [←] C-g detach
	// White bg button, dark blue bg for C-g, light text for detach
	statusLeft := fmt.Sprintf("#[bg=%s,fg=%s,bold] ← #[bg=%s,fg=%s,bold] C-g #[bg=%s,fg=%s,nobold] detach ",
		btnBg, black, keyBg, white, mainBg, lightText)

	// Build status-right: #ID Title | worker
	statusRight := fmt.Sprintf("#[bg=%s,fg=%s,bold]#%d #[fg=%s,nobold]%s #[fg=%s]│ %s ",
		mainBg, white, taskID, lightText, title, mutedText, worker)

	// Build tmux commands
	cmds := [][]string{
		{"set-option", "-t", sessionName, "status", "on"},
		{"set-option", "-t", sessionName, "status-position", "top"},
		// Set status bar background
		{"set-option", "-t", sessionName, "status-style", fmt.Sprintf("bg=%s,fg=%s", mainBg, white)},
		{"set-option", "-t", sessionName, "status-left", statusLeft},
		{"set-option", "-t", sessionName, "status-left-length", "30"},
		{"set-option", "-t", sessionName, "status-right", statusRight},
		{"set-option", "-t", sessionName, "status-right-length", "60"},
		// Hide window list (prevents "0:bash*" from appearing in the center)
		{"set-option", "-t", sessionName, "window-status-format", ""},
		{"set-option", "-t", sessionName, "window-status-current-format", ""},
		{"set-option", "-t", sessionName, "mouse", "on"},
		{"bind-key", "-T", "root", "C-g", "detach-client"},
		{"bind-key", "-T", "root", "MouseDown1StatusLeft", "detach-client"},
	}

	// Execute commands
	for _, args := range cmds {
		cmd := exec.Command("tmux", append([]string{"-S", c.socketPath}, args...)...) //nolint:gosec // sessionName follows crew-N naming convention
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("tmux %v: %w: %s", args, err, string(out))
		}
	}

	return nil
}

// shortenWorkerName shortens agent/worker names for display.
// Examples:
//   - "opencode-medium-anthropic" -> "oc-med-an"
//   - "opencode-small" -> "oc-small"
func shortenWorkerName(name string) string {
	// Split by hyphen
	parts := strings.Split(name, "-")
	if len(parts) == 0 {
		return name
	}

	// Shorten each part
	shortened := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "opencode":
			shortened = append(shortened, "oc")
		case "medium":
			shortened = append(shortened, "med")
		case "anthropic":
			shortened = append(shortened, "an")
		default:
			// Keep short parts as-is, abbreviate longer ones
			if len(part) <= 5 {
				shortened = append(shortened, part)
			} else {
				// Take first 3 characters for longer words
				shortened = append(shortened, part[:3])
			}
		}
	}

	return strings.Join(shortened, "-")
}

// GetPaneProcesses retrieves process information for a tmux session.
// It returns the pane's root process and all its children recursively.
func (c *Client) GetPaneProcesses(sessionName string) ([]domain.ProcessInfo, error) {
	// Check if session exists
	running, err := c.IsRunning(sessionName)
	if err != nil {
		return nil, fmt.Errorf("check session: %w", err)
	}
	if !running {
		return nil, domain.ErrNoSession
	}

	// Get pane PID
	// tmux -S <socket> list-panes -t <name> -F '#{pane_pid}'
	cmd := exec.Command("tmux", //nolint:gosec // sessionName follows crew-N naming convention
		"-S", c.socketPath,
		"list-panes",
		"-t", sessionName,
		"-F", "#{pane_pid}",
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("get pane pid: %w", err)
	}

	pidStr := strings.TrimSpace(string(out))
	if pidStr == "" {
		return nil, fmt.Errorf("no pane found for session %s", sessionName)
	}

	var rootPID int
	if _, parseErr := fmt.Sscanf(pidStr, "%d", &rootPID); parseErr != nil || rootPID <= 0 {
		return nil, fmt.Errorf("invalid pane pid: %s", pidStr)
	}

	// Get all processes with ps command (macOS/Linux compatible)
	// ps -o pid,ppid,state,comm -ax
	psCmd := exec.Command("ps", "-o", "pid,ppid,state,comm", "-ax")
	psOut, err := psCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps command failed: %w", err)
	}

	// Parse ps output
	allProcesses := make(map[int]domain.ProcessInfo)
	lines := strings.Split(string(psOut), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header and empty lines
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		var pid, ppid int
		var state, command string

		if _, scanErr := fmt.Sscanf(fields[0], "%d", &pid); scanErr != nil {
			continue
		}
		if _, scanErr := fmt.Sscanf(fields[1], "%d", &ppid); scanErr != nil {
			continue
		}
		state = fields[2]
		command = strings.Join(fields[3:], " ")

		allProcesses[pid] = domain.ProcessInfo{
			PID:     pid,
			PPID:    ppid,
			State:   state,
			Command: command,
		}
	}

	// Build process tree starting from rootPID
	var result []domain.ProcessInfo
	visited := make(map[int]bool)

	var collectProcesses func(pid int)
	collectProcesses = func(pid int) {
		if visited[pid] {
			return
		}
		visited[pid] = true

		proc, ok := allProcesses[pid]
		if !ok {
			return
		}

		result = append(result, proc)

		// Find children
		for childPID, childProc := range allProcesses {
			if childProc.PPID == pid {
				collectProcesses(childPID)
			}
		}
	}

	collectProcesses(rootPID)

	return result, nil
}

// writeSessionHeader writes the session metadata header to the log file.
// This includes timestamp, session name, command, and working directory.
func (c *Client) writeSessionHeader(logPath string, opts domain.StartSessionOptions) error {
	// Ensure logs directory exists
	logsDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logsDir, 0750); err != nil {
		return fmt.Errorf("create logs directory: %w", err)
	}

	// Format header
	header := fmt.Sprintf(`================================================================================
Session: %s
Started: %s
Directory: %s
Command: %s
================================================================================

`, opts.Name, time.Now().Format(time.RFC3339), opts.Dir, opts.Command)

	// Write header (truncate existing file)
	// G306: Using 0600 as the log file is user-private
	if err := os.WriteFile(logPath, []byte(header), 0600); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	return nil
}

// configureSessionLogging sets up pipe-pane to capture session output to a log file.
func (c *Client) configureSessionLogging(sessionName, logPath string) error {
	// tmux pipe-pane -t <session> 'cat >> <logPath>'
	// This captures all output from the session pane and appends it to the log file.
	// Session names follow our naming convention (crew-N) and are safe to pass to tmux.
	cmd := exec.Command("tmux", //nolint:gosec // sessionName follows crew-N naming convention
		"-S", c.socketPath,
		"pipe-pane",
		"-t", sessionName,
		fmt.Sprintf("cat >> %q", logPath),
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pipe-pane: %w: %s", err, string(out))
	}

	return nil
}
