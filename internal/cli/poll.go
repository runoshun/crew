package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newPollCommand creates the poll command for monitoring task status changes.
func newPollCommand(c *app.Container) *cobra.Command {
	var opts struct {
		Command  string
		Interval int
		Timeout  int
	}

	cmd := &cobra.Command{
		Use:   "poll <TASK_ID>",
		Short: "Poll task status and execute command on change",
		Long: `Monitor a task's status and execute a command when the status changes.

The poll command checks the task status at regular intervals and executes
a command template when a status change is detected. It automatically exits
when the task reaches a terminal state (done, closed, error) or when the
timeout is reached.

Command Template:
  The command template can use the following variables:
    {{.TaskID}}    - Task ID
    {{.OldStatus}} - Previous status
    {{.NewStatus}} - New status

Terminal States:
  The following states are considered terminal and will stop polling:
    - done   - Task merged successfully
    - closed - Task closed without merging
    - error  - Task session terminated with error

Examples:
  # Basic polling (default interval: 10s, no timeout)
  git crew poll 175

  # Poll with custom interval and timeout
  git crew poll 175 --interval 5 --timeout 300

  # Execute notification on status change
  git crew poll 175 --command 'notify-send "Task {{.TaskID}}: {{.NewStatus}}"'

  # Run in background
  git crew poll 175 --command 'echo "{{.TaskID}}: {{.OldStatus}} → {{.NewStatus}}"' &`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			var taskID int
			if _, err := fmt.Sscanf(args[0], "%d", &taskID); err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}

			// Set default command if not specified
			if opts.Command == "" {
				opts.Command = `echo "{{.TaskID}}: {{.OldStatus}} → {{.NewStatus}}"`
			}

			// Setup signal handling for graceful shutdown
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			// Execute use case
			uc := c.PollTaskUseCase()
			_, err := uc.Execute(ctx, usecase.PollTaskInput{
				TaskID:          taskID,
				Interval:        opts.Interval,
				Timeout:         opts.Timeout,
				CommandTemplate: opts.Command,
			})

			return err
		},
	}

	// Flags
	cmd.Flags().IntVarP(&opts.Interval, "interval", "i", 10, "Polling interval in seconds")
	cmd.Flags().IntVarP(&opts.Timeout, "timeout", "t", 0, "Timeout in seconds (0 = no timeout)")
	cmd.Flags().StringVarP(&opts.Command, "command", "c", "", "Command template to execute on status change")

	return cmd
}
