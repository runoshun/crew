package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/runoshun/git-crew/v2/internal/app"
)

// newTUICommand creates the tui command for launching the interactive TUI.
// This command now launches the unified TUI (same as running `crew` without arguments).
func newTUICommand(_ *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI",
		Long:  `Launch the interactive terminal user interface for managing tasks.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			cwd, _ := os.Getwd()
			return launchUnifiedTUI(cwd)
		},
	}
	return cmd
}
