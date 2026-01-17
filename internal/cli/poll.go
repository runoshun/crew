package cli

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newPollCommand creates the poll command for monitoring task status changes.
func newPollCommand(c *app.Container) *cobra.Command {
	var opts struct {
		Command  string
		Expect   string
		Interval int
		Timeout  int
	}

	cmd := &cobra.Command{
		Use:   "poll <TASK_ID>",
		Short: "Poll task status and exit after one change detection (default timeout: 5m)",
		Long: `Monitor a task's status and execute a command once when the status changes.

The poll command checks the task status at regular intervals and executes
a command template as soon as a status change is detected. It exits immediately
after executing the command for the first detected change.

Polling stops automatically when the timeout is reached (default: 300s).

Expected Status Check (Required):
  Use --expect to specify expected status(es). 
  - If the current status already differs from the expected status(es) on startup,
    the command is executed immediately and the poll command exits.
  - If it matches, the command waits for the first status change.

Command Template:
  The command template can use the following variables:
    {{.TaskID}}    - Task ID
    {{.OldStatus}} - Previous status (or expected status if --expect is used)
    {{.NewStatus}} - New status

Terminal States:
  Reaching a terminal state is also treated as a status change:
    - closed - Task closed (merged or abandoned)
    - error  - Task session terminated with error

Examples:
  # Poll with expected status and notify on change (5m timeout)
  git crew poll 175 --expect todo --command 'notify-send "Task {{.TaskID}} started!"'

  # Multiple expected statuses (exit if status becomes something else)
  git crew poll 199 --expect in_progress,needs_input --command 'say "Task {{.TaskID}} changed to {{.NewStatus}}"'

  # Poll with custom interval and timeout
  git crew poll 175 --expect todo --interval 5 --timeout 60

  # Use as a trigger for next action
  git crew poll 175 --expect in_progress --command 'crew complete 175'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			var taskID int
			if _, err := fmt.Sscanf(args[0], "%d", &taskID); err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}

			// Parse expected statuses
			var expectedStatuses []domain.Status
			if opts.Expect != "" {
				parts := strings.Split(opts.Expect, ",")
				for _, part := range parts {
					status := domain.Status(strings.TrimSpace(part))
					if !status.IsValid() {
						return fmt.Errorf("invalid status: %s", part)
					}
					expectedStatuses = append(expectedStatuses, status)
				}
			}

			// Set default command if not specified
			if opts.Command == "" {
				opts.Command = `echo "{{.TaskID}}: {{.OldStatus}} → {{.NewStatus}}"`
			}

			// Setup signal handling for graceful shutdown
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			// Execute use case
			uc := c.PollTaskUseCase(cmd.OutOrStdout(), cmd.ErrOrStderr())
			_, err := uc.Execute(ctx, usecase.PollTaskInput{
				TaskID:           taskID,
				ExpectedStatuses: expectedStatuses,
				Interval:         opts.Interval,
				Timeout:          opts.Timeout,
				CommandTemplate:  opts.Command,
			})

			return err
		},
	}

	// Flags
	cmd.Flags().StringVarP(&opts.Expect, "expect", "e", "", "Expected status(es) - comma-separated (e.g., 'in_progress' or 'in_progress,needs_input')")
	cmd.Flags().IntVarP(&opts.Interval, "interval", "i", 10, "Polling interval in seconds")
	cmd.Flags().IntVarP(&opts.Timeout, "timeout", "t", 300, "Timeout in seconds")
	cmd.Flags().StringVarP(&opts.Command, "command", "c", "", "Command template to execute on status change")

	// Mark --expect as required (ignore error as flag is guaranteed to exist)
	_ = cmd.MarkFlagRequired("expect")

	return cmd
}
