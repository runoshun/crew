package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/tui/workspace"
	"github.com/spf13/cobra"
)

// newWorkspaceCommand creates the workspace command for managing multiple repositories.
func newWorkspaceCommand(_ *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"ws"},
		Short:   "Launch workspace TUI for managing multiple repositories",
		Long: `Launch the workspace TUI that allows you to switch between multiple
git repositories and view task summaries for each.

This command can be run from outside a git repository.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return launchWorkspaceTUI()
		},
	}

	// Add subcommands for non-interactive management
	cmd.AddCommand(newWorkspaceAddCommand())
	cmd.AddCommand(newWorkspaceRemoveCommand())
	cmd.AddCommand(newWorkspaceListCommand())

	return cmd
}

// launchWorkspaceTUI launches the workspace TUI (with workspace panel visible).
// This uses NewUnified but always shows the workspace panel.
func launchWorkspaceTUI() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}
	model := workspace.NewUnified(cwd)
	model.SetShowWorkspace(true) // Force workspace panel visible
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// newWorkspaceAddCommand creates the workspace add subcommand.
func newWorkspaceAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Add a repository to the workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			store, err := workspace.NewStoreFromDefault()
			if err != nil {
				return err
			}
			return store.AddRepo(args[0])
		},
	}
}

// newWorkspaceRemoveCommand creates the workspace remove subcommand.
func newWorkspaceRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <path>",
		Aliases: []string{"rm"},
		Short:   "Remove a repository from the workspace",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			store, err := workspace.NewStoreFromDefault()
			if err != nil {
				return err
			}
			return store.RemoveRepo(args[0])
		},
	}
}

// newWorkspaceListCommand creates the workspace list subcommand.
func newWorkspaceListCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List repositories in the workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := workspace.NewStoreFromDefault()
			if err != nil {
				return err
			}
			file, err := store.Load()
			if err != nil {
				return err
			}
			for _, repo := range file.Repos {
				cmd.Printf("%s\t%s\n", repo.DisplayName(), repo.Path)
			}
			return nil
		},
	}
}
