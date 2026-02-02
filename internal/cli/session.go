package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newDiffCommand creates the diff command for showing task changes.
func newDiffCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <id> [args...]",
		Short: "Display task change diff",
		Long: `Display the diff for a task's changes.

The diff command executes the configured diff.command (or a default git diff)
from the task's worktree directory.

Any additional arguments after the task ID are passed to the diff command
through the {{.Args}} template variable.

Examples:
  # Show diff for task #1
  crew diff 1

  # Show diff with --stat
  crew diff 1 --stat

  # Show diff for specific file
  crew diff 1 -- path/to/file.go`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Get additional args (everything after the task ID)
			var diffArgs []string
			if len(args) > 1 {
				diffArgs = args[1:]
			}

			// Execute use case
			uc := c.ShowDiffUseCase(cmd.OutOrStdout(), cmd.ErrOrStderr())
			_, err = uc.Execute(cmd.Context(), usecase.ShowDiffInput{
				TaskID: taskID,
				Args:   diffArgs,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// newStartCommand creates the start command for starting a task.
func newStartCommand(c *app.Container) *cobra.Command {
	var opts struct {
		model        string
		prompts      []string
		continueFlag bool
		skipReview   bool
	}

	cmd := &cobra.Command{
		Use:   "start <id> [agent]",
		Short: "Start an AI agent session",
		Long: `Start an AI agent session for a task.

This creates a worktree and branch (if they don't exist),
starts a tmux session with the specified agent, and updates
the task status to 'in_progress'.

The agent argument specifies the command to run in the session.
In the MVP version, this is the full command (e.g., "claude", "bash").

Examples:
  # Start task #1 with claude
  crew start 1 claude

  # Start task with a different agent
  crew start 1 bash

  # Start task with a specific model
  crew start 1 claude --model sonnet
  crew start 1 opencode -m gpt-4o

  # Continue a stopped task
  crew start 1 --continue
  crew start 1 -c

  # Start task with skip_review enabled (skip review on completion)
  crew start 1 claude --skip-review

  # Start task with additional prompt
  crew start 1 claude --prompt "Focus on performance optimization"

  # Start task with multiple additional prompts
  crew start 1 claude -p "Use TDD approach" -p "Write comprehensive tests"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Get agent from args
			agent := ""
			if len(args) > 1 {
				agent = args[1]
			}

			// Execute use case
			uc := c.StartTaskUseCase()
			input := usecase.StartTaskInput{
				TaskID:            taskID,
				Agent:             agent,
				Model:             opts.model,
				Continue:          opts.continueFlag,
				AdditionalPrompts: opts.prompts,
			}
			// Set skip_review only if flag was explicitly provided
			if cmd.Flags().Changed("skip-review") {
				input.SkipReview = &opts.skipReview
			}
			out, err := uc.Execute(cmd.Context(), input)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started task #%d (session: %s, worktree: %s)\n",
				taskID, out.SessionName, out.WorktreePath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.model, "model", "m", "", "Model to use (overrides agent default)")
	cmd.Flags().BoolVarP(&opts.continueFlag, "continue", "c", false, "Continue from previous session")
	cmd.Flags().BoolVar(&opts.skipReview, "skip-review", false, "Set skip_review for this task (skip review on completion)")
	cmd.Flags().StringArrayVarP(&opts.prompts, "prompt", "p", nil, "Additional prompt to append (can be specified multiple times)")

	return cmd
}

// newAttachCommand creates the attach command for attaching to a session.
func newAttachCommand(c *app.Container) *cobra.Command {
	var opts struct {
		manager bool
	}

	cmd := &cobra.Command{
		Use:   "attach [id]",
		Short: "Attach to a running session",
		Long: `Attach to a running tmux session for a task or manager.

This replaces the current process with the tmux session.
Use Ctrl+G to detach from the session (configured in .crew/tmux.conf).

By default, attaches to the work session (crew-<id>).
Use --manager to attach to the manager session (crew-manager).

Preconditions:
  - Task must exist (unless --manager is used)
  - Session must be running

Examples:
  # Attach to work session for task #1
  crew attach 1

  # Attach to manager session
  crew attach --manager`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle manager session
			if opts.manager {
				sessionName := domain.ManagerSessionName()
				running, err := c.Sessions.IsRunning(sessionName)
				if err != nil {
					return fmt.Errorf("check session: %w", err)
				}
				if !running {
					return fmt.Errorf("manager session %q: %w", sessionName, domain.ErrNoSession)
				}
				return c.Sessions.Attach(sessionName)
			}

			// Require task ID for non-manager operations
			if len(args) == 0 {
				return fmt.Errorf("task ID is required (or use --manager for manager session)")
			}

			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Execute use case
			uc := c.AttachSessionUseCase()
			_, err = uc.Execute(cmd.Context(), usecase.AttachSessionInput{
				TaskID: taskID,
			})
			if err != nil {
				return err
			}

			// This line should never be reached (attach replaces the process)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.manager, "manager", false, "Attach to manager session (crew-manager)")

	return cmd
}

// newPeekCommand creates the peek command for viewing session output.
func newPeekCommand(c *app.Container) *cobra.Command {
	var opts struct {
		lines  int
		escape bool
	}

	cmd := &cobra.Command{
		Use:   "peek <id>",
		Short: "View session output non-interactively",
		Long: `View session output non-interactively using tmux capture-pane.

This captures and displays the last N lines from a running session
without attaching to it.

Preconditions:
  - Task must exist
  - Session must be running

Examples:
  # View last 30 lines (default) of task #1 session
  crew peek 1

  # View last 50 lines
  crew peek 1 --lines 50
  crew peek 1 -n 50

  # View with ANSI escape sequences (colors)
  crew peek 1 --escape
  crew peek 1 -e`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Execute use case
			uc := c.PeekSessionUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.PeekSessionInput{
				TaskID: taskID,
				Lines:  opts.lines,
				Escape: opts.escape,
			})
			if err != nil {
				return err
			}

			// Print output
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), out.Output)
			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.lines, "lines", "n", 0, "Number of lines to display (default: 30)")
	cmd.Flags().BoolVarP(&opts.escape, "escape", "e", false, "Include ANSI escape sequences (colors)")

	return cmd
}

// newCompleteCommand creates the complete command for marking a task as complete.
func newCompleteCommand(c *app.Container) *cobra.Command {
	var opts struct {
		comment     string
		reviewer    string
		forceReview bool
		verbose     bool
	}

	cmd := &cobra.Command{
		Use:   "complete [id]",
		Short: "Mark task as complete",
		Long: `Mark a task as complete.

If no ID is provided, the task ID is auto-detected from the current branch name.

Preconditions:
	  - Task status must be 'in_progress'
	  - No uncommitted changes in the worktree

If [complete].command is configured, it will be executed before transitioning
the status. If the command fails, the completion is aborted.

		Review requirement:
		  - skip_review enabled: bypasses review requirement
		  - otherwise: runs review until the result matches [complete].review_success_regex (default: "✅ LGTM")
		    or [complete].max_reviews attempts are reached (default: 1)
		  - --force-review runs review even if skip_review is enabled
		  - review count increases when a review result is recorded

Examples:
	  # Complete task by ID
	  crew complete 1

  # Complete with a comment
  crew complete 1 --comment "Implementation complete"

	  # Complete with reviewer override
	  crew complete 1 --reviewer claude-reviewer

	  # Force review even if not required
	  crew complete 1 --force-review

  # Auto-detect task from current branch (when working in a worktree)
  crew complete`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve task ID
			taskID, err := resolveTaskID(args, c.Git)
			if err != nil {
				return err
			}

			// Execute use case
			uc := c.CompleteTaskUseCase(cmd.OutOrStdout(), cmd.ErrOrStderr())
			out, err := uc.Execute(cmd.Context(), usecase.CompleteTaskInput{
				TaskID:      taskID,
				Comment:     opts.comment,
				ForceReview: opts.forceReview,
				ReviewAgent: opts.reviewer,
				Verbose:     opts.verbose,
			})
			if err != nil {
				// Print conflict message to stdout if present
				if out != nil && out.ConflictMessage != "" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), out.ConflictMessage)
				}
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Completed task #%d: %s\n", out.Task.ID, out.Task.Title)

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.comment, "comment", "m", "", "Add a completion comment")
	cmd.Flags().BoolVar(&opts.forceReview, "force-review", false, "Run review even when not required")
	cmd.Flags().StringVarP(&opts.reviewer, "reviewer", "r", "", "Reviewer agent override")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Show reviewer output in real-time")

	return cmd
}

// newSessionEndedCommand creates the _session-ended internal command.
// This is called by the task script's EXIT trap to handle session termination.
func newSessionEndedCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "_session-ended <id> <exit-code>",
		Short:  "Handle session termination (internal command)",
		Hidden: true, // Internal command, not shown in help
		Args:   cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Parse exit code
			exitCode, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid exit code: %w", err)
			}

			// Execute use case
			uc := c.SessionEndedUseCase()
			_, err = uc.Execute(cmd.Context(), usecase.SessionEndedInput{
				TaskID:   taskID,
				ExitCode: exitCode,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// newSendCommand creates the send command for sending keys to a session.
func newSendCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send <id> <keys>",
		Short: "Send key input to a session",
		Long: `Send key input to a running tmux session for a task.

The keys argument can be:
  - Special keys: Tab, Escape, Enter
  - Any text to be typed into the session

Preconditions:
  - Task must exist
  - Session must be running

Examples:
  # Send Enter key to task #1
  crew send 1 Enter

  # Send Tab key for completion
  crew send 1 Tab

  # Send Escape key
  crew send 1 Escape

  # Send text
  crew send 1 "hello world"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Get keys from args
			keys := args[1]

			// Execute use case
			uc := c.SendKeysUseCase()
			_, err = uc.Execute(cmd.Context(), usecase.SendKeysInput{
				TaskID: taskID,
				Keys:   keys,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Sent keys to task #%d\n", taskID)
			return nil
		},
	}

	return cmd
}

// newStopCommand creates the stop command for stopping a task session.
func newStopCommand(c *app.Container) *cobra.Command {
	var opts struct {
		manager bool
	}

	cmd := &cobra.Command{
		Use:   "stop [id]",
		Short: "Stop a session",
		Long: `Stop a running session for a task or manager.

This terminates the tmux session and cleans up generated scripts.
When stopping a work session, it clears agent info and updates the
status to 'error'.

The worktree is NOT deleted (use 'close' to also delete the worktree).

Examples:
  # Stop session for task #1
  crew stop 1

  # Stop manager session
  crew stop --manager`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle manager session
			if opts.manager {
				sessionName := domain.ManagerSessionName()
				running, err := c.Sessions.IsRunning(sessionName)
				if err != nil {
					return fmt.Errorf("check session: %w", err)
				}
				if !running {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No manager session running")
					return nil
				}
				if err := c.Sessions.Stop(sessionName); err != nil {
					return fmt.Errorf("stop manager session: %w", err)
				}
				// Clean up manager script
				scriptPath := domain.ManagerScriptPath(c.Config.CrewDir)
				_ = os.Remove(scriptPath)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Stopped manager session %s\n", sessionName)
				return nil
			}

			// Require task ID for non-manager operations
			if len(args) == 0 {
				return fmt.Errorf("task ID is required (or use --manager for manager session)")
			}

			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Execute use case
			uc := c.StopTaskUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.StopTaskInput{
				TaskID: taskID,
			})
			if err != nil {
				return err
			}

			if out.SessionName == "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No running session for task #%d\n", taskID)
				return nil
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Stopped session %s\n", out.SessionName)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.manager, "manager", false, "Stop the manager session (crew-manager)")

	return cmd
}

// newExecCommand creates the exec command for executing commands in a task's worktree.
func newExecCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <id> -- <command...>",
		Short: "Execute a command in task's worktree",
		Long: `Execute a command in the task's worktree directory.

This runs the specified command from the task's worktree with stdout/stderr
piped to the terminal. The command's exit code is preserved.

The task's status is NOT modified. This is a read/execute-only operation.

Preconditions:
  - Task must exist
  - Worktree must exist

Examples:
  # Run tests in task #1's worktree
  crew exec 1 -- npm test

  # Check git status in task's worktree
  crew exec 1 -- git status

  # Run a build command
  crew exec 1 -- make build`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Get command (everything after "--")
			// Cobra should have already handled "--" separator, so args[1:] should be the command
			var command []string
			if len(args) > 1 {
				command = args[1:]
			} else {
				return fmt.Errorf("command is required after task ID")
			}

			// Execute use case
			uc := c.ExecCommandUseCase()
			_, err = uc.Execute(cmd.Context(), usecase.ExecCommandInput{
				TaskID:  taskID,
				Command: command,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// newLogsCommand creates the logs command for viewing session logs.
func newLogsCommand(c *app.Container) *cobra.Command {
	var opts struct {
		lines int
	}

	cmd := &cobra.Command{
		Use:   "logs <id>",
		Short: "View session logs",
		Long: `View the log file for a task's session.

The session log contains:
  - Session start timestamp and metadata
  - Session command output (stdout/stderr)
  - All terminal output captured during the session

Logs persist after the session ends, allowing debugging of
startup errors and reviewing session history.

Examples:
  # View logs for task #1
  crew logs 1

  # View last 50 lines of logs
  crew logs 1 --lines 50
  crew logs 1 -n 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Execute use case
			uc := c.ShowLogsUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.ShowLogsInput{
				TaskID: taskID,
				Lines:  opts.lines,
			})
			if err != nil {
				return err
			}

			// Print log content
			_, _ = fmt.Fprint(cmd.OutOrStdout(), out.Content)
			return nil
		},
	}

	cmd.Flags().IntVarP(&opts.lines, "lines", "n", 0, "Number of lines to display from the end (0 = all)")

	return cmd
}

// newMergeCommand creates the merge command for merging a task branch into a base branch.
func newMergeCommand(c *app.Container) *cobra.Command {
	var opts struct {
		base string
		yes  bool
	}

	cmd := &cobra.Command{
		Use:   "merge <id>",
		Short: "Merge task branch into base branch",
		Long: `Merge a task branch into the base branch and mark the task as closed.

Preconditions:
  - Current branch must be the base branch
  - Base branch's working tree must be clean
  - No merge conflict

Base branch selection:
  - If --base is not specified, uses task's base branch (or default branch if task has no base branch)
  - If --base is specified, uses the specified branch (allows merging to different branch)

Processing:
  1. If session is running, stop it
  2. Execute git merge --no-ff
  3. Delete the worktree
  4. Update task status to 'closed' (with close_reason: merged)

Examples:
  # Merge task #1 into its base branch (or default branch if not set)
  crew merge 1

  # Merge task #1 into feature/workspace branch (override task's base branch)
  crew merge 1 --base feature/workspace

  # Skip confirmation prompt
  crew merge 1 --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Get task info for confirmation
			showUC := c.ShowTaskUseCase()
			showOut, err := showUC.Execute(cmd.Context(), usecase.ShowTaskInput{
				TaskID: taskID,
			})
			if err != nil {
				return err
			}

			// Determine target base branch for confirmation message
			// Match the logic in MergeTask.Execute
			targetBaseBranch := opts.base
			if targetBaseBranch == "" {
				targetBaseBranch = showOut.Task.BaseBranch
				if targetBaseBranch == "" {
					// Note: At runtime, this would call GetDefaultBranch()
					// For confirmation message, we use "default branch" as placeholder
					targetBaseBranch = "default branch"
				}
			}

			// Get branch name for confirmation message
			branch := domain.BranchName(showOut.Task.ID, showOut.Task.Issue)

			// Confirmation prompt (skip with --yes)
			if !opts.yes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Merge task #%d (%s) to %s? [y/N] ", taskID, branch, targetBaseBranch)
				reader := bufio.NewReader(os.Stdin)
				response, readErr := reader.ReadString('\n')
				if readErr != nil {
					return fmt.Errorf("read response: %w", readErr)
				}
				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			// Execute use case
			uc := c.MergeTaskUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.MergeTaskInput{
				TaskID:     taskID,
				BaseBranch: opts.base,
			})
			if err != nil {
				// Print conflict message to stdout if present
				if out != nil && out.ConflictMessage != "" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), out.ConflictMessage)
				}
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.base, "base", "", "Base branch to merge into (default: task's base branch or default branch)")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
