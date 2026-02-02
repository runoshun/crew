package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

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

	cmd.AddCommand(newACPStartCommand(c))
	cmd.AddCommand(newACPRunCommand(c))
	cmd.AddCommand(newACPSendCommand(c))
	cmd.AddCommand(newACPPermissionCommand(c))
	cmd.AddCommand(newACPCancelCommand(c))
	cmd.AddCommand(newACPStopCommand(c))
	cmd.AddCommand(newACPAttachCommand(c))
	cmd.AddCommand(newACPPeekCommand(c))
	cmd.AddCommand(newACPLogCommand(c))
	cmd.AddCommand(newACPConsoleCommand(c))
	return cmd
}

// newACPStartCommand creates the ACP start command.
func newACPStartCommand(c *app.Container) *cobra.Command {
	var opts struct {
		agent string
		model string
	}

	cmd := &cobra.Command{
		Use:   "start <task-id>",
		Short: "Start an ACP session in tmux",
		Long: `Start an ACP session in tmux for a task.

This creates a worktree (if needed) and launches "crew acp run" in a tmux session.

Examples:
  crew acp start 1 --agent my-acp-agent
  crew acp start 1 --agent my-acp-agent --model gpt-4o`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			uc := c.ACPStartUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.ACPStartInput{
				TaskID: taskID,
				Agent:  opts.agent,
				Model:  opts.model,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started ACP task #%d (session: %s, worktree: %s)\n", taskID, out.SessionName, out.WorktreePath)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.agent, "agent", "", "Agent name (default: config agents.default_worker)")
	cmd.Flags().StringVarP(&opts.model, "model", "m", "", "Model override")

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
		Use:   "send [task-id] [text]",
		Short: "Send a prompt to an ACP session",
		Long: `Send a prompt to an ACP session.

You can pass task ID and text as positional arguments or use --task/--text flags (do not mix).
When using positional arguments, remaining tokens are joined as the text payload.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if cmd.Flags().Changed("task") || cmd.Flags().Changed("text") {
				return fmt.Errorf("cannot mix positional arguments with --task/--text")
			}
			if len(args) < 2 {
				return fmt.Errorf("text is required")
			}
			if _, err := parseTaskID(args[0]); err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				var err error
				opts.taskID, err = parseTaskID(args[0])
				if err != nil {
					return fmt.Errorf("invalid task ID: %w", err)
				}
			}
			if len(args) > 1 {
				opts.text = strings.Join(args[1:], " ")
			}
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

	return cmd
}

// newACPPermissionCommand creates the ACP permission command.
func newACPPermissionCommand(c *app.Container) *cobra.Command {
	var opts struct {
		optionID string
		taskID   int
	}

	cmd := &cobra.Command{
		Use:   "permission [task-id] [option-id|#index]",
		Short: "Respond to a permission request",
		Long: `Respond to a permission request.

You can pass task ID and option as positional arguments or use --task/--option flags (do not mix).
Positional option values are treated as a single token; use --option to avoid shell splitting.
Prefix the option with "#" to select by index from the latest permission request.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if cmd.Flags().Changed("task") || cmd.Flags().Changed("option") {
				return fmt.Errorf("cannot mix positional arguments with --task/--option")
			}
			if len(args) < 2 {
				return fmt.Errorf("option is required")
			}
			if len(args) > 2 {
				return fmt.Errorf("too many arguments")
			}
			if _, err := parseTaskID(args[0]); err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				var err error
				opts.taskID, err = parseTaskID(args[0])
				if err != nil {
					return fmt.Errorf("invalid task ID: %w", err)
				}
			}
			if len(args) > 1 {
				opts.optionID = args[1]
			}
			if opts.taskID <= 0 {
				return fmt.Errorf("task is required")
			}
			if opts.optionID == "" {
				return fmt.Errorf("option is required")
			}

			optionID := strings.TrimSpace(opts.optionID)
			if strings.HasPrefix(optionID, "#") {
				logUC := c.ACPLogUseCase()
				out, err := logUC.Execute(cmd.Context(), usecase.ACPLogInput{TaskID: opts.taskID})
				if err != nil {
					return err
				}
				optionID, err = resolvePermissionOptionID(optionID, out.Events, cmd.ErrOrStderr())
				if err != nil {
					return err
				}
			}

			uc := c.ACPControlUseCase()
			_, err := uc.Execute(cmd.Context(), usecase.ACPControlInput{
				TaskID:      opts.taskID,
				CommandType: domain.ACPCommandPermission,
				OptionID:    optionID,
			})
			return err
		},
	}

	cmd.Flags().IntVar(&opts.taskID, "task", 0, "Task ID")
	cmd.Flags().StringVar(&opts.optionID, "option", "", "Permission option ID or #index")

	return cmd
}

// newACPCancelCommand creates the ACP cancel command.
func newACPCancelCommand(c *app.Container) *cobra.Command {
	var opts struct {
		taskID int
	}

	cmd := &cobra.Command{
		Use:   "cancel [task-id]",
		Short: "Cancel the current ACP session",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if cmd.Flags().Changed("task") {
				return fmt.Errorf("cannot mix positional arguments with --task")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}
			if _, err := parseTaskID(args[0]); err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
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

			uc := c.ACPControlUseCase()
			_, err := uc.Execute(cmd.Context(), usecase.ACPControlInput{
				TaskID:      opts.taskID,
				CommandType: domain.ACPCommandCancel,
			})
			return err
		},
	}

	cmd.Flags().IntVar(&opts.taskID, "task", 0, "Task ID")

	return cmd
}

// newACPStopCommand creates the ACP stop command.
func newACPStopCommand(c *app.Container) *cobra.Command {
	var opts struct {
		taskID int
	}

	cmd := &cobra.Command{
		Use:   "stop [task-id]",
		Short: "Stop the ACP session cleanly",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if cmd.Flags().Changed("task") {
				return fmt.Errorf("cannot mix positional arguments with --task")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}
			if _, err := parseTaskID(args[0]); err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
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

			uc := c.ACPControlUseCase()
			_, err := uc.Execute(cmd.Context(), usecase.ACPControlInput{
				TaskID:      opts.taskID,
				CommandType: domain.ACPCommandStop,
			})
			return err
		},
	}

	cmd.Flags().IntVar(&opts.taskID, "task", 0, "Task ID")

	return cmd
}

// newACPAttachCommand creates the ACP attach command.
func newACPAttachCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attach <task-id>",
		Short: "Attach to a running ACP session",
		Long: `Attach to a running tmux session for an ACP task.

This replaces the current process with the tmux session.
Use your configured tmux detach key to leave the session.

Examples:
  crew acp attach 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			uc := c.ACPAttachUseCase()
			_, err = uc.Execute(cmd.Context(), usecase.ACPAttachInput{TaskID: taskID})
			if err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}

// newACPPeekCommand creates the ACP peek command.
func newACPPeekCommand(c *app.Container) *cobra.Command {
	var opts struct {
		lines  int
		escape bool
	}

	cmd := &cobra.Command{
		Use:   "peek <task-id>",
		Short: "View ACP session output non-interactively",
		Long: `View ACP session output non-interactively using tmux capture-pane.

This captures and displays the last N lines from a running ACP session
without attaching to it.

Examples:
  crew acp peek 1
  crew acp peek 1 --lines 50
  crew acp peek 1 --escape`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			uc := c.ACPPeekUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.ACPPeekInput{
				TaskID: taskID,
				Lines:  opts.lines,
				Escape: opts.escape,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), out.Output)
			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.lines, "lines", "n", usecase.DefaultPeekLines, "Number of lines to display")
	cmd.Flags().BoolVarP(&opts.escape, "escape", "e", false, "Include ANSI escape sequences (colors)")

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
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if cmd.Flags().Changed("task") {
				return fmt.Errorf("cannot mix positional arguments with --task")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}
			if _, err := parseTaskID(args[0]); err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			return nil
		},
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

	return cmd
}

func resolvePermissionOptionID(option string, events []domain.ACPEvent, warn io.Writer) (string, error) {
	option = strings.TrimSpace(option)
	if !strings.HasPrefix(option, "#") {
		return option, nil
	}
	idxText := strings.TrimPrefix(option, "#")
	if idxText == "" {
		return "", fmt.Errorf("option index must be >= 1")
	}
	idx, err := strconv.Atoi(idxText)
	if err != nil {
		return "", fmt.Errorf("option index must be numeric")
	}
	if idx <= 0 {
		return "", fmt.Errorf("option index must be >= 1")
	}

	var invalidPayloadErr error
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.Type != domain.ACPEventRequestPermission {
			continue
		}
		if len(event.Payload) == 0 {
			if invalidPayloadErr == nil {
				invalidPayloadErr = fmt.Errorf("permission request payload is empty")
			}
			continue
		}
		var req acpsdk.RequestPermissionRequest
		if err := json.Unmarshal(event.Payload, &req); err != nil {
			if invalidPayloadErr == nil {
				invalidPayloadErr = fmt.Errorf("decode permission request: %w", err)
			}
			continue
		}
		if invalidPayloadErr != nil && warn != nil {
			_, _ = fmt.Fprintf(warn, "warning: latest permission request payload is invalid; using previous request: %v\n", invalidPayloadErr)
		}
		if idx > len(req.Options) {
			return "", fmt.Errorf("permission option index out of range: %d", idx)
		}
		return string(req.Options[idx-1].OptionId), nil
	}

	if invalidPayloadErr != nil {
		return "", invalidPayloadErr
	}
	return "", fmt.Errorf("no permission requests found")
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

	var msgBuffer strings.Builder
	var lastMsgTime time.Time

	flushMessageBuffer := func() {
		if msgBuffer.Len() == 0 {
			return
		}
		text := msgBuffer.String()
		// Truncate if too long
		if len(text) > 200 {
			text = text[:200] + "..."
		}
		// Replace newlines for single-line display
		text = strings.ReplaceAll(text, "\n", "\\n")
		_, _ = fmt.Fprintf(out, "[%s] AGENT_MESSAGE: %s\n", lastMsgTime.Format("15:04:05"), text)
		msgBuffer.Reset()
	}

	for _, event := range events {
		// Accumulate agent message chunks
		if event.Type == domain.ACPEventAgentMessageChunk {
			text := extractAgentMessageText(event.Payload)
			if text != "" {
				if msgBuffer.Len() == 0 {
					lastMsgTime = event.Timestamp
				}
				msgBuffer.WriteString(text)
			}
			continue
		}

		// Flush accumulated messages before printing other events
		flushMessageBuffer()

		ts := event.Timestamp.Format("15:04:05")
		eventType := formatEventType(event.Type)
		detail := extractEventDetail(event)
		if detail != "" {
			_, _ = fmt.Fprintf(out, "[%s] %s: %s\n", ts, eventType, detail)
		} else {
			_, _ = fmt.Fprintf(out, "[%s] %s\n", ts, eventType)
		}
	}

	// Flush any remaining messages
	flushMessageBuffer()
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
	case domain.ACPEventSessionUpdate,
		domain.ACPEventPromptSent,
		domain.ACPEventSessionEnd,
		domain.ACPEventAgentMessageChunk,
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

// extractAgentMessageText extracts only the text from an agent message chunk payload.
// Unlike extractAgentMessageDetail, it does not truncate or escape the text.
func extractAgentMessageText(payload json.RawMessage) string {
	var notification acpsdk.SessionNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		return ""
	}
	if notification.Update.AgentMessageChunk == nil {
		return ""
	}
	chunk := notification.Update.AgentMessageChunk
	if chunk.Content.Text != nil {
		return chunk.Content.Text.Text
	}
	return ""
}

// newACPConsoleCommand creates the ACP console command.
func newACPConsoleCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "console <task-id>",
		Short: "Interactive ACP console",
		Long: `Open an interactive console for an ACP session.

The console provides:
- Real-time log viewing
- Prompt input
- Permission request handling

Keys:
  Enter    Send prompt
  1-9      Select permission option
  Esc      Cancel current operation
  Ctrl+D   Stop session
  Ctrl+C   Quit console

Examples:
  crew acp console 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			return c.RunACPConsole(cmd.Context(), taskID)
		},
	}

	return cmd
}
