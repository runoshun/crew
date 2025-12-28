// Package cli provides the command-line interface for git-crew.
package cli

import (
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/spf13/cobra"
)

// Command group IDs.
const (
	groupSetup = "setup"
	groupTask  = "task"
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

	// Define command groups
	root.AddGroup(
		&cobra.Group{ID: groupSetup, Title: "Setup Commands:"},
		&cobra.Group{ID: groupTask, Title: "Task Management:"},
	)

	// Setup commands
	initCmd := newInitCommand(c)
	initCmd.GroupID = groupSetup

	// Task management commands
	newCmd := newNewCommand(c)
	newCmd.GroupID = groupTask

	listCmd := newListCommand(c)
	listCmd.GroupID = groupTask

	showCmd := newShowCommand(c)
	showCmd.GroupID = groupTask

	editCmd := newEditCommand(c)
	editCmd.GroupID = groupTask

	rmCmd := newRmCommand(c)
	rmCmd.GroupID = groupTask

	cpCmd := newCpCommand(c)
	cpCmd.GroupID = groupTask

	commentCmd := newCommentCommand(c)
	commentCmd.GroupID = groupTask

	// Add subcommands
	root.AddCommand(
		initCmd,
		newCmd,
		listCmd,
		showCmd,
		editCmd,
		rmCmd,
		cpCmd,
		commentCmd,
		// Commands below will be added as they are implemented:
		// newStartCommand(c),
	)

	return root
}
