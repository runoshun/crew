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

	return nil
}

// Stop terminates a tmux session.
// It first kills all processes running in the session's panes, then kills the session itself.
// This ensures that child processes (like AI agents) are properly terminated
// and don't become orphaned when the session is closed.
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
		// Kill child processes of each pane
		// We use SIGTERM to allow graceful shutdown
		pids := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, pid := range pids {
			if pid == "" {
				continue
			}
			// pkill -TERM -P <pid> sends SIGTERM to all child processes
			// We ignore errors here because:
			// - The process might have already exited
			// - There might be no child processes
			killCmd := exec.Command("pkill", "-TERM", "-P", pid)
			_ = killCmd.Run()
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
func (c *Client) Peek(sessionName string, lines int) (string, error) {
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
	// -S -<lines>: start capture from <lines> lines before the current position
	// Session names follow our naming convention (crew-N) and are safe to pass to tmux.
	cmd := exec.Command("tmux", //nolint:gosec // sessionName follows crew-N naming convention
		"-S", c.socketPath,
		"capture-pane",
		"-t", sessionName,
		"-p",
		"-S", fmt.Sprintf("-%d", lines),
	)

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
