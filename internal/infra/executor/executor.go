// Package executor provides command execution functionality.
package executor

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Client implements domain.CommandExecutor interface.
type Client struct{}

// NewClient creates a new command executor client.
func NewClient() *Client {
	return &Client{}
}

// Ensure Client implements domain.CommandExecutor interface.
var _ domain.CommandExecutor = (*Client)(nil)

// Execute runs the command and returns its combined output.
func (c *Client) Execute(cmd *domain.ExecCommand) ([]byte, error) {
	// #nosec G204 - cmd.Program and cmd.Args come from trusted UseCase code
	execCmd := exec.Command(cmd.Program, cmd.Args...)
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	}
	return execCmd.CombinedOutput()
}

// ExecuteInteractive runs a command with stdin/stdout/stderr connected to the terminal.
func (c *Client) ExecuteInteractive(cmd *domain.ExecCommand) error {
	// #nosec G204 - cmd.Program and cmd.Args come from trusted UseCase code
	execCmd := exec.Command(cmd.Program, cmd.Args...)
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	}
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

// ExecuteWithContext runs a command with context and custom stdout/stderr writers.
func (c *Client) ExecuteWithContext(ctx context.Context, cmd *domain.ExecCommand, stdout, stderr io.Writer) error {
	// #nosec G204 - cmd.Program and cmd.Args come from trusted UseCase code
	execCmd := exec.CommandContext(ctx, cmd.Program, cmd.Args...)
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	}
	execCmd.Stdout = stdout
	execCmd.Stderr = stderr
	return execCmd.Run()
}
