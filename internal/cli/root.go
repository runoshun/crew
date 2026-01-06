// Package cli provides the command-line interface for git-crew.
package cli

import (
	"errors"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/spf13/cobra"
)

// Command group IDs.
const (
	groupSetup   = "setup"
	groupTask    = "task"
	groupSession = "session"
)

// NewRootCommand creates the root command for git-crew.
// It receives the container for dependency injection and version for display.
func NewRootCommand(c *app.Container, version string) *cobra.Command {
	var fullWorker bool
	var fullManager bool

	root := &cobra.Command{
		Use:   "crew",
		Short: "AI agent task management CLI",
		Long: `git-crew is a CLI tool for managing AI coding agent tasks.
It combines git worktree + tmux to achieve a model where 
1 task = 1 worktree = 1 AI session, enabling fully parallel 
and isolated task execution.

Use --help-worker or --help-manager for role-specific detailed help.`,
		Version: version,
		// SilenceUsage prevents usage from being printed on errors
		SilenceUsage: true,
		// SilenceErrors prevents Cobra from printing errors (we handle it in main)
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Skip for some commands
			if cmd.Name() == "_session-ended" || cmd.Name() == "init" {
				return nil
			}

			cfg, err := c.ConfigLoader.Load()
			if err != nil {
				// Ignore error (e.g. not initialized)
				return nil
			}

			for _, w := range cfg.Warnings {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", w)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Handle role-specific help flags
			if fullWorker && fullManager {
				return errors.New("cannot use both --help-worker and --help-manager")
			}

			if fullWorker {
				return showWorkerHelp(cmd.OutOrStdout())
			}
			if fullManager {
				cfg, _ := c.ConfigLoader.Load()
				return showManagerHelp(cmd.OutOrStdout(), cfg)
			}
			// Default: show standard help
			return cmd.Help()
		},
	}

	// Add role-specific help flags
	root.Flags().BoolVar(&fullWorker, "help-worker", false, "Show detailed help for worker agents")
	root.Flags().BoolVar(&fullManager, "help-manager", false, "Show detailed help for manager agents")

	// Define command groups
	root.AddGroup(
		&cobra.Group{ID: groupSetup, Title: "Setup Commands:"},
		&cobra.Group{ID: groupTask, Title: "Task Management:"},
		&cobra.Group{ID: groupSession, Title: "Session Management:"},
	)

	// Setup commands
	initCmd := newInitCommand(c)
	initCmd.GroupID = groupSetup

	configCmd := newConfigCommand(c)
	configCmd.GroupID = groupSetup

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

	closeCmd := newCloseCommand(c)
	closeCmd.GroupID = groupTask

	// Session management commands
	startCmd := newStartCommand(c)
	startCmd.GroupID = groupSession

	stopCmd := newStopCommand(c)
	stopCmd.GroupID = groupSession

	attachCmd := newAttachCommand(c)
	attachCmd.GroupID = groupSession

	sendCmd := newSendCommand(c)
	sendCmd.GroupID = groupSession

	peekCmd := newPeekCommand(c)
	peekCmd.GroupID = groupSession

	execCmd := newExecCommand(c)
	execCmd.GroupID = groupSession

	diffCmd := newDiffCommand(c)
	diffCmd.GroupID = groupSession

	completeCmd := newCompleteCommand(c)
	completeCmd.GroupID = groupSession

	mergeCmd := newMergeCommand(c)
	mergeCmd.GroupID = groupSession

	pruneCmd := newPruneCommand(c)
	pruneCmd.GroupID = groupTask

	// TUI command
	tuiCmd := newTUICommand(c)
	tuiCmd.GroupID = groupTask

	// Manager command
	managerCmd := newManagerCommand(c)
	managerCmd.GroupID = groupSession

	// Snapshot commands
	snapshotCmd := newSnapshotCmd(c)

	// Internal commands (hidden)
	sessionEndedCmd := newSessionEndedCommand(c)

	// Add subcommands
	root.AddCommand(
		initCmd,
		configCmd,
		newCmd,
		listCmd,
		showCmd,
		editCmd,
		rmCmd,
		cpCmd,
		commentCmd,
		closeCmd,
		startCmd,
		stopCmd,
		attachCmd,
		sendCmd,
		peekCmd,
		execCmd,
		diffCmd,
		completeCmd,
		mergeCmd,
		pruneCmd,
		tuiCmd,
		managerCmd,
		snapshotCmd,
		newSyncCmd(c),
		sessionEndedCmd,
	)

	return root
}
