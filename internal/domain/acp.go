package domain

import (
	"context"
	"encoding/json"
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

// ACPEventType represents the type of ACP event.
type ACPEventType string

// ACP event types.
const (
	ACPEventSessionUpdate      ACPEventType = "session_update"
	ACPEventRequestPermission  ACPEventType = "request_permission"
	ACPEventPermissionResponse ACPEventType = "permission_response"
	ACPEventPromptSent         ACPEventType = "prompt_sent"
	ACPEventSessionEnd         ACPEventType = "session_end"
	ACPEventAgentMessageChunk  ACPEventType = "agent_message_chunk"
	ACPEventAgentThoughtChunk  ACPEventType = "agent_thought_chunk"
	ACPEventToolCall           ACPEventType = "tool_call"
	ACPEventToolCallUpdate     ACPEventType = "tool_call_update"
	ACPEventUserMessageChunk   ACPEventType = "user_message_chunk"
	ACPEventPlan               ACPEventType = "plan"
	ACPEventCurrentModeUpdate  ACPEventType = "current_mode_update"
	ACPEventAvailableCommands  ACPEventType = "available_commands"
)

// ACPEvent represents an event in the ACP session lifecycle.
// Events are persisted to events.jsonl for later replay and analysis.
type ACPEvent struct {
	Timestamp time.Time       `json:"ts"`
	Type      ACPEventType    `json:"type"`
	SessionID string          `json:"session_id"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// ACPEventWriter writes ACP events to persistent storage.
type ACPEventWriter interface {
	// Write appends an event to the event log.
	Write(ctx context.Context, event ACPEvent) error
	// Close releases any resources held by the writer.
	Close() error
}

// ACPEventWriterFactory creates event writers scoped to a task.
type ACPEventWriterFactory interface {
	// ForTask returns an event writer bound to a namespace and task ID.
	ForTask(namespace string, taskID int) (ACPEventWriter, error)
}

// ACPEventReader reads ACP events from persistent storage.
type ACPEventReader interface {
	// ReadAll returns all events from the event log.
	ReadAll(ctx context.Context) ([]ACPEvent, error)
}
