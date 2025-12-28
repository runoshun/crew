// Package cli provides the command-line interface for git-crew.
package cli

import (
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command for git-crew.
// It receives the container for dependency injection and version for display.
func NewRootCommand(c *app.Container, version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "crew",
		Short: "AI agent task management CLI",
		Long: `git-crew is a CLI tool for managing AI coding agent tasks.
It combines git worktree + tmux to achieve a model where 
1 task = 1 worktree = 1 AI session, enabling fully parallel 
and isolated task execution.`,
		Version: version,
		// SilenceUsage prevents usage from being printed on errors
		SilenceUsage: true,
	}

	// Add subcommands
	root.AddCommand(
		newInitCommand(c),
		newNewCommand(c),
		newListCommand(c),
		newShowCommand(c),
		newEditCommand(c),
		// Commands below will be added as they are implemented:
		// newStartCommand(c),
	)

	return root
}
