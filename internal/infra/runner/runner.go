// Package runner provides script execution functionality.
package runner

import (
	"fmt"
	"os/exec"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Client implements domain.ScriptRunner interface.
type Client struct{}

// NewClient creates a new script runner client.
func NewClient() *Client {
	return &Client{}
}

// Ensure Client implements domain.ScriptRunner interface.
var _ domain.ScriptRunner = (*Client)(nil)

// Run executes a script in a directory.
func (c *Client) Run(dir, script string) error {
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = dir

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("execute script: %w: %s", err, string(out))
	}

	return nil
}
