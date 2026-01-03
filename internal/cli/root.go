// Package cli provides the command-line interface for git-crew.
package cli

import (
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
		// SilenceErrors prevents Cobra from printing errors (we handle it in main)
		SilenceErrors: true,
	}

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

	genCmd := newGenCommand(c)
	genCmd.GroupID = groupSetup

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

	// Snapshot commands
	snapshotCmd := newSnapshotCmd(c)

	// Internal commands (hidden)
	sessionEndedCmd := newSessionEndedCommand(c)

	// Add subcommands
	root.AddCommand(
		initCmd,
		configCmd,
		genCmd,
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
		snapshotCmd,
		newSyncCmd(c),
		sessionEndedCmd,
	)

	return root
}
