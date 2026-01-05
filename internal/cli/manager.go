package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newManagerCommand creates the manager command for launching a manager agent.
func newManagerCommand(c *app.Container) *cobra.Command {
	var opts struct {
		model string
	}

	cmd := &cobra.Command{
		Use:   "manager [name]",
		Short: "Launch a manager agent",
		Long: `Launch a manager agent for task orchestration.

Managers are read-only agents that can create and monitor tasks,
but delegate actual code implementation to worker agents.

The manager agent is launched in the current directory (not a worktree)
and has access to all crew commands for task management.

If no name is specified, the default manager is used.

Examples:
  # Launch the default manager
  crew manager

  # Launch a specific manager
  crew manager my-manager

  # Launch with a specific model
  crew manager --model opus`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			managerName := ""
			if len(args) > 0 {
				managerName = args[0]
			}

			// Execute use case
			uc := c.StartManagerUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.StartManagerInput{
				Name:  managerName,
				Model: opts.model,
			})
			if err != nil {
				return err
			}

			// Execute the manager command
			// We use a script file to properly handle the PROMPT variable
			return executeManagerScript(out, c.Config.CrewDir)
		},
	}

	cmd.Flags().StringVarP(&opts.model, "model", "m", "", "Model to use (overrides manager default)")

	return cmd
}

// executeManagerScript writes a script file and executes it.
// This ensures proper handling of the PROMPT shell variable.
func executeManagerScript(out *usecase.StartManagerOutput, crewDir string) error {
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

	// Execute the script
	execCmd := exec.Command("bash", scriptPath)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}
