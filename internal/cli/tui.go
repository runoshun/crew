package cli

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/tui"
)

// newTUICommand creates the tui command for launching the interactive TUI.
func newTUICommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI",
		Long:  `Launch the interactive terminal user interface for managing tasks.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			model := tui.New(c)
			p := tea.NewProgram(model, tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}
	return cmd
}
