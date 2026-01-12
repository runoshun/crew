// Package executor provides command execution functionality.
package executor

import (
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
