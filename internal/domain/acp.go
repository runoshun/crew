package domain

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"
)

// ACPCommandType represents the type of IPC command for ACP runner control.
type ACPCommandType string

// ACP IPC command types.
const (
	ACPCommandPrompt     ACPCommandType = "prompt"
	ACPCommandPermission ACPCommandType = "permission"
	ACPCommandCancel     ACPCommandType = "cancel"
	ACPCommandStop       ACPCommandType = "stop"
)

// ACPCommand represents a command sent to the ACP runner via IPC.
// Fields are ordered to minimize memory padding.
type ACPCommand struct {
	CreatedAt time.Time      `json:"created_at"`
	ID        string         `json:"id"`
	Type      ACPCommandType `json:"type"`
	Text      string         `json:"text,omitempty"`
	OptionID  string         `json:"option_id,omitempty"`
}

// Validate ensures the command has required fields for its type.
func (c ACPCommand) Validate() error {
	switch c.Type {
	case ACPCommandPrompt:
		if c.Text == "" {
			return fmt.Errorf("%w: prompt text is required", ErrInvalidACPCommand)
		}
	case ACPCommandPermission:
		if c.OptionID == "" {
			return fmt.Errorf("%w: permission option_id is required", ErrInvalidACPCommand)
		}
	case ACPCommandCancel, ACPCommandStop:
		// No extra fields required.
	default:
		return fmt.Errorf("%w: unknown type %q", ErrInvalidACPCommand, c.Type)
	}
	return nil
}

// ACPIPC defines a command-based IPC interface for ACP runner control.
type ACPIPC interface {
	// Next blocks until a command is available or context is canceled.
	Next(ctx context.Context) (ACPCommand, error)
	// Send enqueues a command for the runner.
	Send(ctx context.Context, cmd ACPCommand) error
}

// ACPIPCFactory creates ACPIPC instances scoped to a task.
type ACPIPCFactory interface {
	// ForTask returns an IPC instance bound to a namespace and task ID.
	ForTask(namespace string, taskID int) ACPIPC
}

// ACPDir returns the base directory for ACP data for a task.
// Format: <crewDir>/acp/<namespace>/<task-id>
func ACPDir(crewDir, namespace string, taskID int) string {
	return filepath.Join(crewDir, "acp", namespace, strconv.Itoa(taskID))
}
