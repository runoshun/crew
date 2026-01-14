package cli

import (
	"github.com/spf13/cobra"

	"github.com/runoshun/git-crew/v2/internal/app"
)

// newTUICommand creates the tui command for launching the interactive TUI.
func newTUICommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI",
		Long:  `Launch the interactive terminal user interface for managing tasks.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchTUI(c)
		},
	}
	return cmd
}
