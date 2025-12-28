package cli

import (
	"fmt"
	"strconv"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newStartCommand creates the start command for starting a task.
func newStartCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <id> [agent]",
		Short: "Start an AI agent session",
		Long: `Start an AI agent session for a task.

This creates a worktree and branch (if they don't exist),
starts a tmux session with the specified agent, and updates
the task status to 'in_progress'.

The agent argument specifies the command to run in the session.
In the MVP version, this is the full command (e.g., "claude", "bash").

Examples:
  # Start task #1 with claude
  git crew start 1 claude

  # Start task with a different agent
  git crew start 1 bash`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Get agent from args
			agent := ""
			if len(args) > 1 {
				agent = args[1]
			}

			// Execute use case
			uc := c.StartTaskUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.StartTaskInput{
				TaskID: taskID,
				Agent:  agent,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started task #%d (session: %s, worktree: %s)\n",
				taskID, out.SessionName, out.WorktreePath)
			return nil
		},
	}

	return cmd
}

// newAttachCommand creates the attach command for attaching to a session.
func newAttachCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attach <id>",
		Short: "Attach to a running session",
		Long: `Attach to a running tmux session for a task.

This replaces the current process with the tmux session.
Use Ctrl+G to detach from the session (configured in .git/crew/tmux.conf).

Preconditions:
  - Task must exist
  - Session must be running

Examples:
  # Attach to session for task #1
  git crew attach 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Execute use case
			uc := c.AttachSessionUseCase()
			_, err = uc.Execute(cmd.Context(), usecase.AttachSessionInput{
				TaskID: taskID,
			})
			if err != nil {
				return err
			}

			// This line should never be reached (attach replaces the process)
			return nil
		},
	}

	return cmd
}

// newSessionEndedCommand creates the _session-ended internal command.
// This is called by the task script's EXIT trap to handle session termination.
func newSessionEndedCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "_session-ended <id> <exit-code>",
		Short:  "Handle session termination (internal command)",
		Hidden: true, // Internal command, not shown in help
		Args:   cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Parse exit code
			exitCode, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid exit code: %w", err)
			}

			// Execute use case
			uc := c.SessionEndedUseCase()
			_, err = uc.Execute(cmd.Context(), usecase.SessionEndedInput{
				TaskID:   taskID,
				ExitCode: exitCode,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
