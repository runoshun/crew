package cli

import (
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/app"
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
