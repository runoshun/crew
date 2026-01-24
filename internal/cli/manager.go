package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newManagerCommand creates the manager command for launching a manager agent.
func newManagerCommand(c *app.Container) *cobra.Command {
	var opts struct {
		model   string
		agent   string
		session bool
	}

	cmd := &cobra.Command{
		Use:   "manager [prompt]",
		Short: "Launch a manager agent",
		Long: `Launch a manager agent for task orchestration.

Managers are read-only agents that can create and monitor tasks,
but delegate actual code implementation to worker agents.

The manager agent is launched in the current directory (not a worktree)
and has access to all crew commands for task management.

If no agent is specified, the default manager is used.

Examples:
  # Launch the default manager
  crew manager

  # Launch with an additional prompt
  crew manager "Review task 215"

  # Launch a specific manager agent
  crew manager --agent my-manager

  # Launch a specific manager with additional prompt
  crew manager "Review task 215" --agent my-manager

  # Launch with a specific model
  crew manager --model opus

  # Launch in a tmux session (background)
  crew manager --session`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			additionalPrompt := ""
			if len(args) > 0 {
				additionalPrompt = args[0]
			}

			// Execute use case
			uc := c.StartManagerUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.StartManagerInput{
				Name:             opts.agent,
				Model:            opts.model,
				AdditionalPrompt: additionalPrompt,
				Session:          opts.session,
			})
			if err != nil {
				return err
			}

			// If session mode, print session name and return
			if opts.session {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started manager session: %s\n", out.SessionName)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Use 'crew attach --manager' to attach to the session.\n")
				return nil
			}

			// Execute the manager command (foreground mode)
			// We use a script file to properly handle the PROMPT variable
			return executeManagerScript(out, c.Config.CrewDir, c.Executor)
		},
	}

	cmd.Flags().StringVarP(&opts.model, "model", "m", "", "Model to use (overrides manager default)")
	cmd.Flags().StringVarP(&opts.agent, "agent", "a", "", "Manager agent to use (defaults to configured default)")
	cmd.Flags().BoolVarP(&opts.session, "session", "s", false, "Start in a tmux session (crew-manager)")

	return cmd
}

// executeManagerScript writes a script file and executes it.
// This ensures proper handling of the PROMPT shell variable.
func executeManagerScript(out *usecase.StartManagerOutput, crewDir string, executor domain.CommandExecutor) error {
	// Create scripts directory if it doesn't exist
	scriptsDir := filepath.Join(crewDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o750); err != nil {
		return fmt.Errorf("create scripts directory: %w", err)
	}

	// Write script file
	scriptPath := domain.ManagerScriptPath(crewDir)
	script := out.BuildScript()
	if script == "" {
		return fmt.Errorf("failed to build script")
	}

	// G306: We intentionally use 0700 because this is an executable script
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil { //nolint:gosec // executable script requires execute permission
		return fmt.Errorf("write script file: %w", err)
	}

	// Execute the script using CommandExecutor
	execCmd := domain.NewCommand("bash", []string{scriptPath}, "")
	return executor.ExecuteInteractive(execCmd)
}
