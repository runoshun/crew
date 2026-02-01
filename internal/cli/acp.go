package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newACPCommand creates the ACP command group.
func newACPCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acp",
		Short: "ACP commands",
	}

	cmd.AddCommand(newACPRunCommand(c))
	cmd.AddCommand(newACPSendCommand(c))
	cmd.AddCommand(newACPPermissionCommand(c))
	cmd.AddCommand(newACPCancelCommand(c))
	cmd.AddCommand(newACPStopCommand(c))
	cmd.AddCommand(newACPLogCommand(c))
	return cmd
}

// newACPRunCommand creates the ACP run command.
func newACPRunCommand(c *app.Container) *cobra.Command {
	var opts struct {
		agent  string
		model  string
		taskID int
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run an ACP session in the current terminal",
		Long: `Run an ACP session that connects to a wrapper agent via stdio.

This command initializes ACP, creates a session, and then waits for IPC commands
such as prompt/permission/cancel/stop.

Examples:
  crew acp run --task 1 --agent my-acp-agent
  crew acp run --task 1 --agent my-acp-agent --model gpt-4o`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.taskID <= 0 {
				return fmt.Errorf("task is required")
			}
			if opts.agent == "" {
				return fmt.Errorf("agent is required")
			}

			uc := c.ACPRunUseCase(cmd.OutOrStdout(), cmd.ErrOrStderr())
			_, err := uc.Execute(cmd.Context(), usecase.ACPRunInput{
				TaskID: opts.taskID,
				Agent:  opts.agent,
				Model:  opts.model,
			})
			return err
		},
	}

	cmd.Flags().IntVar(&opts.taskID, "task", 0, "Task ID")
	cmd.Flags().StringVar(&opts.agent, "agent", "", "Agent name")
	cmd.Flags().StringVarP(&opts.model, "model", "m", "", "Model override")
	_ = cmd.MarkFlagRequired("task")
	_ = cmd.MarkFlagRequired("agent")

	return cmd
}

// newACPSendCommand creates the ACP send command.
func newACPSendCommand(c *app.Container) *cobra.Command {
	var opts struct {
		text   string
		taskID int
	}

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a prompt to an ACP session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.taskID <= 0 {
				return fmt.Errorf("task is required")
			}
			if opts.text == "" {
				return fmt.Errorf("text is required")
			}

			uc := c.ACPControlUseCase()
			_, err := uc.Execute(cmd.Context(), usecase.ACPControlInput{
				TaskID:      opts.taskID,
				CommandType: domain.ACPCommandPrompt,
				Text:        opts.text,
			})
			return err
		},
	}

	cmd.Flags().IntVar(&opts.taskID, "task", 0, "Task ID")
	cmd.Flags().StringVar(&opts.text, "text", "", "Prompt text to send")
	_ = cmd.MarkFlagRequired("task")
	_ = cmd.MarkFlagRequired("text")

	return cmd
}

// newACPPermissionCommand creates the ACP permission command.
func newACPPermissionCommand(c *app.Container) *cobra.Command {
	var opts struct {
		optionID string
		taskID   int
	}

	cmd := &cobra.Command{
		Use:   "permission",
		Short: "Respond to a permission request",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.taskID <= 0 {
				return fmt.Errorf("task is required")
			}
			if opts.optionID == "" {
				return fmt.Errorf("option is required")
			}

			uc := c.ACPControlUseCase()
			_, err := uc.Execute(cmd.Context(), usecase.ACPControlInput{
				TaskID:      opts.taskID,
				CommandType: domain.ACPCommandPermission,
				OptionID:    opts.optionID,
			})
			return err
		},
	}

	cmd.Flags().IntVar(&opts.taskID, "task", 0, "Task ID")
	cmd.Flags().StringVar(&opts.optionID, "option", "", "Permission option ID")
	_ = cmd.MarkFlagRequired("task")
	_ = cmd.MarkFlagRequired("option")

	return cmd
}

// newACPCancelCommand creates the ACP cancel command.
func newACPCancelCommand(c *app.Container) *cobra.Command {
	var opts struct {
		taskID int
	}

	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel the current ACP session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.taskID <= 0 {
				return fmt.Errorf("task is required")
			}

			uc := c.ACPControlUseCase()
			_, err := uc.Execute(cmd.Context(), usecase.ACPControlInput{
				TaskID:      opts.taskID,
				CommandType: domain.ACPCommandCancel,
			})
			return err
		},
	}

	cmd.Flags().IntVar(&opts.taskID, "task", 0, "Task ID")
	_ = cmd.MarkFlagRequired("task")

	return cmd
}

// newACPStopCommand creates the ACP stop command.
func newACPStopCommand(c *app.Container) *cobra.Command {
	var opts struct {
		taskID int
	}

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the ACP session cleanly",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.taskID <= 0 {
				return fmt.Errorf("task is required")
			}

			uc := c.ACPControlUseCase()
			_, err := uc.Execute(cmd.Context(), usecase.ACPControlInput{
				TaskID:      opts.taskID,
				CommandType: domain.ACPCommandStop,
			})
			return err
		},
	}

	cmd.Flags().IntVar(&opts.taskID, "task", 0, "Task ID")
	_ = cmd.MarkFlagRequired("task")

	return cmd
}

// newACPLogCommand creates the ACP log command.
func newACPLogCommand(c *app.Container) *cobra.Command {
	var opts struct {
		taskID int
		raw    bool
	}

	cmd := &cobra.Command{
		Use:   "log [task-id]",
		Short: "View ACP session event log",
		Long: `View the event log for an ACP session.

Events are stored in .crew/acp/<namespace>/<task-id>/events.jsonl and include
session updates, tool calls, permission requests, and more.

Examples:
  crew acp log 1
  crew acp log 1 --raw
  crew acp log --task 1`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Accept task ID as positional argument or flag
			if len(args) > 0 {
				var err error
				opts.taskID, err = parseTaskID(args[0])
				if err != nil {
					return fmt.Errorf("invalid task ID: %w", err)
				}
			}
			if opts.taskID <= 0 {
				return fmt.Errorf("task is required")
			}

			uc := c.ACPLogUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.ACPLogInput{
				TaskID: opts.taskID,
			})
			if err != nil {
				return err
			}

			if len(out.Events) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No events found.")
				return nil
			}

			if opts.raw {
				return printRawEvents(cmd, out.Events)
			}
			return printFormattedEvents(cmd, out.Events)
		},
	}

	cmd.Flags().IntVar(&opts.taskID, "task", 0, "Task ID")
	cmd.Flags().BoolVar(&opts.raw, "raw", false, "Output raw JSON events")
	_ = cmd.MarkFlagRequired("task")

	return cmd
}

func printRawEvents(cmd *cobra.Command, events []domain.ACPEvent) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	for _, event := range events {
		if err := enc.Encode(event); err != nil {
			return err
		}
	}
	return nil
}

func printFormattedEvents(cmd *cobra.Command, events []domain.ACPEvent) error {
	out := cmd.OutOrStdout()
	for _, event := range events {
		ts := event.Timestamp.Format("15:04:05")
		eventType := formatEventType(event.Type)
		detail := extractEventDetail(event)
		if detail != "" {
			_, _ = fmt.Fprintf(out, "[%s] %s: %s\n", ts, eventType, detail)
		} else {
			_, _ = fmt.Fprintf(out, "[%s] %s\n", ts, eventType)
		}
	}
	return nil
}

func formatEventType(t domain.ACPEventType) string {
	switch t {
	case domain.ACPEventSessionUpdate:
		return "SESSION_UPDATE"
	case domain.ACPEventRequestPermission:
		return "PERMISSION_REQUEST"
	case domain.ACPEventPermissionResponse:
		return "PERMISSION_RESPONSE"
	case domain.ACPEventPromptSent:
		return "PROMPT_SENT"
	case domain.ACPEventSessionEnd:
		return "SESSION_END"
	case domain.ACPEventAgentMessageChunk:
		return "AGENT_MESSAGE"
	case domain.ACPEventAgentThoughtChunk:
		return "AGENT_THOUGHT"
	case domain.ACPEventToolCall:
		return "TOOL_CALL"
	case domain.ACPEventToolCallUpdate:
		return "TOOL_CALL_UPDATE"
	case domain.ACPEventUserMessageChunk:
		return "USER_MESSAGE"
	case domain.ACPEventPlan:
		return "PLAN"
	case domain.ACPEventCurrentModeUpdate:
		return "MODE_UPDATE"
	case domain.ACPEventAvailableCommands:
		return "AVAILABLE_COMMANDS"
	}
	return string(t)
}

func extractEventDetail(event domain.ACPEvent) string {
	if len(event.Payload) == 0 {
		return ""
	}

	switch event.Type {
	case domain.ACPEventRequestPermission:
		return extractPermissionRequestDetail(event.Payload)
	case domain.ACPEventPermissionResponse:
		return extractPermissionResponseDetail(event.Payload)
	case domain.ACPEventToolCall:
		return extractToolCallDetail(event.Payload)
	case domain.ACPEventToolCallUpdate:
		return extractToolCallUpdateDetail(event.Payload)
	case domain.ACPEventAgentMessageChunk:
		return extractAgentMessageDetail(event.Payload)
	case domain.ACPEventSessionUpdate,
		domain.ACPEventPromptSent,
		domain.ACPEventSessionEnd,
		domain.ACPEventAgentThoughtChunk,
		domain.ACPEventUserMessageChunk,
		domain.ACPEventPlan,
		domain.ACPEventCurrentModeUpdate,
		domain.ACPEventAvailableCommands:
		return ""
	}
	return ""
}

func extractPermissionRequestDetail(payload json.RawMessage) string {
	var req acpsdk.RequestPermissionRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return ""
	}

	title := ""
	if req.ToolCall.Title != nil {
		title = *req.ToolCall.Title
	}

	var options []string
	for _, opt := range req.Options {
		options = append(options, fmt.Sprintf("%s[%s]", opt.Name, opt.OptionId))
	}
	return fmt.Sprintf("%s | options: %s", title, strings.Join(options, ", "))
}

func extractPermissionResponseDetail(payload json.RawMessage) string {
	var resp acpsdk.RequestPermissionResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		return ""
	}

	if resp.Outcome.Selected != nil {
		return fmt.Sprintf("selected: %s", resp.Outcome.Selected.OptionId)
	}
	if resp.Outcome.Cancelled != nil {
		return "cancelled"
	}
	return ""
}

func extractToolCallDetail(payload json.RawMessage) string {
	var notification acpsdk.SessionNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		return ""
	}
	if notification.Update.ToolCall == nil {
		return ""
	}
	tc := notification.Update.ToolCall
	return fmt.Sprintf("id=%s %s", tc.ToolCallId, tc.Title)
}

func extractToolCallUpdateDetail(payload json.RawMessage) string {
	var notification acpsdk.SessionNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		return ""
	}
	if notification.Update.ToolCallUpdate == nil {
		return ""
	}
	update := notification.Update.ToolCallUpdate
	status := ""
	if update.Status != nil {
		status = string(*update.Status)
	}
	return fmt.Sprintf("id=%s status=%s", update.ToolCallId, status)
}

func extractAgentMessageDetail(payload json.RawMessage) string {
	var notification acpsdk.SessionNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		return ""
	}
	if notification.Update.AgentMessageChunk == nil {
		return ""
	}
	chunk := notification.Update.AgentMessageChunk
	if chunk.Content.Text != nil {
		text := chunk.Content.Text.Text
		if len(text) > 80 {
			text = text[:80] + "..."
		}
		// Replace newlines for single-line display
		text = strings.ReplaceAll(text, "\n", "\\n")
		return text
	}
	return ""
}
