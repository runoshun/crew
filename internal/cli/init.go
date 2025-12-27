package cli

import (
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newInitCommand creates the init command.
func newInitCommand(c *app.Container) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize repository for git-crew",
		Long: `Initialize a repository for git-crew.

This command creates the .git/crew/ directory with:
- tmux.conf: minimal tmux configuration
- tasks.json: empty task store
- scripts/: directory for task scripts
- logs/: directory for log files

Preconditions:
- Current directory must be inside a git repository

Error conditions:
- Already initialized: "crew already initialized"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get use case from container
			uc := c.InitRepoUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.InitRepoInput{
				CrewDir:   c.Config.CrewDir,
				StorePath: c.Config.StorePath,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Initialized git-crew in %s\n", out.CrewDir)
			return nil
		},
	}
}
