// Package cli provides the command-line interface for git-crew.
package cli

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/tui/workspace"
	"github.com/spf13/cobra"
)

// Command group IDs.
const (
	groupSetup   = "setup"
	groupTask    = "task"
	groupSession = "session"
)

// launchUnifiedTUIFunc is a function variable for launching unified TUI, allowing it to be mocked in tests.
var launchUnifiedTUIFunc = launchUnifiedTUI

// NewRootCommand creates the root command for git-crew.
// It receives the container for dependency injection and version for display.
func NewRootCommand(c *app.Container, version string) *cobra.Command {
	var fullWorker bool
	var fullReviewer bool
	var fullManager bool
	var managerOnboarding bool
	var managerAuto bool
	var reviewerFollowUp bool

	root := &cobra.Command{
		Use:   "crew",
		Short: "AI agent task management CLI",
		Long: `git-crew is a CLI tool for managing AI coding agent tasks.
It combines git worktree + tmux to achieve a model where
1 task = 1 worktree = 1 AI session, enabling fully parallel
and isolated task execution.

Use --help-worker, --help-reviewer, or --help-manager for role-specific detailed help.
Use --help-manager-onboarding to see the onboarding guide for new projects.
Use --help-manager-auto to see the auto mode guide.`,
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

			// Skip if container is nil (e.g. in tests)
			if c == nil {
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
			flagCount := 0
			if fullWorker {
				flagCount++
			}
			if fullReviewer {
				flagCount++
			}
			if fullManager {
				flagCount++
			}
			if managerOnboarding {
				flagCount++
			}
			if managerAuto {
				flagCount++
			}
			if flagCount > 1 {
				return errors.New("cannot use multiple help flags together")
			}
			if reviewerFollowUp && !fullReviewer {
				return errors.New("--follow-up can only be used with --help-reviewer")
			}

			var cfg *domain.Config
			if c != nil {
				cfg, _ = c.ConfigLoader.Load()
			}

			if fullWorker {
				return showWorkerHelp(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg)
			}
			if fullReviewer {
				return showReviewerHelp(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg, reviewerFollowUp)
			}
			if fullManager {
				return showManagerHelp(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg)
			}
			if managerOnboarding {
				return showManagerOnboardingHelp(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg)
			}
			if managerAuto {
				return showManagerAutoHelp(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg)
			}
			// Default: launch unified TUI
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current directory: %w", err)
			}
			return launchUnifiedTUIFunc(cwd)
		},
	}

	// Add role-specific help flags
	root.Flags().BoolVar(&fullWorker, "help-worker", false, "Show detailed help for worker agents")
	root.Flags().BoolVar(&fullReviewer, "help-reviewer", false, "Show detailed help for reviewer agents")
	root.Flags().BoolVar(&reviewerFollowUp, "follow-up", false, "Show follow-up guidance in reviewer help")
	// Backward-compatible alias (typo kept for compatibility)
	root.Flags().BoolVar(&fullReviewer, "help-reviwer", false, "DEPRECATED: use --help-reviewer")
	_ = root.Flags().MarkDeprecated("help-reviwer", "use --help-reviewer")
	_ = root.Flags().MarkHidden("help-reviwer")
	root.Flags().BoolVar(&fullManager, "help-manager", false, "Show detailed help for manager agents")
	root.Flags().BoolVar(&managerOnboarding, "help-manager-onboarding", false, "Show onboarding guide for setting up crew")
	root.Flags().BoolVar(&managerAuto, "help-manager-auto", false, "Show auto mode guide for manager agents")

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

	migrateCmd := newMigrateCommand(c)
	migrateCmd.GroupID = groupSetup

	// Task management commands
	newCmd := newNewCommand(c)
	newCmd.GroupID = groupTask

	listCmd := newListCommand(c)
	listCmd.GroupID = groupTask

	showCmd := newShowCommand(c)
	showCmd.GroupID = groupTask

	editCmd := newEditCommand(c)
	editCmd.GroupID = groupTask

	substateCmd := newSubstateCommand(c)
	substateCmd.GroupID = groupTask

	rmCmd := newRmCommand(c)
	rmCmd.GroupID = groupTask

	cpCmd := newCpCommand(c)
	cpCmd.GroupID = groupTask

	commentCmd := newCommentCommand(c)
	commentCmd.GroupID = groupTask

	commentsCmd := newCommentsCommand(c)
	commentsCmd.GroupID = groupTask

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

	pollCmd := newPollCommand(c)
	pollCmd.GroupID = groupSession

	logsCmd := newLogsCommand(c)
	logsCmd.GroupID = groupSession

	pruneCmd := newPruneCommand(c)
	pruneCmd.GroupID = groupTask

	// TUI command
	tuiCmd := newTUICommand(c)
	tuiCmd.GroupID = groupTask

	// Manager command
	managerCmd := newManagerCommand(c)
	managerCmd.GroupID = groupSession

	// Agent list command
	listAgentsCmd := newListAgentsCommand(c)
	listAgentsCmd.GroupID = groupSetup

	// Workspace command (can run without git repo)
	workspaceCmd := newWorkspaceCommand(c)
	workspaceCmd.GroupID = groupSession

	// Internal commands (hidden)
	sessionEndedCmd := newSessionEndedCommand(c)

	// Add subcommands
	root.AddCommand(
		initCmd,
		configCmd,
		migrateCmd,
		listAgentsCmd,
		newCmd,
		listCmd,
		showCmd,
		editCmd,
		substateCmd,
		rmCmd,
		cpCmd,
		commentCmd,
		commentsCmd,
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
		pollCmd,
		logsCmd,
		pruneCmd,
		tuiCmd,
		managerCmd,
		workspaceCmd,
		sessionEndedCmd,
	)

	return root
}

// launchUnifiedTUI launches the unified TUI that works from any directory.
// It detects if cwd is in a git repository and adjusts the view accordingly.
func launchUnifiedTUI(cwd string) error {
	model := workspace.NewUnified(cwd)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
