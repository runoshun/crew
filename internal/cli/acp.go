package cli

import (
	"fmt"

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
