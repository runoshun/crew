package cli

import (
	"bufio"
	"errors"
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
	var review bool

	cmd := &cobra.Command{
		Use:   "attach <id>",
		Short: "Attach to a running session",
		Long: `Attach to a running tmux session for a task.

This replaces the current process with the tmux session.
Use Ctrl+G to detach from the session (configured in .git/crew/tmux.conf).

By default, attaches to the work session (crew-<id>).
Use --review to attach to the review session (crew-<id>-review).

Preconditions:
  - Task must exist
  - Session must be running

Examples:
  # Attach to work session for task #1
  crew attach 1

  # Attach to review session for task #1
  crew attach 1 --review`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Execute use case
			uc := c.AttachSessionUseCase()
			_, err = uc.Execute(cmd.Context(), usecase.AttachSessionInput{
				TaskID: taskID,
				Review: review,
			})
			if err != nil {
				return err
			}

			// This line should never be reached (attach replaces the process)
			return nil
		},
	}

	cmd.Flags().BoolVar(&review, "review", false, "Attach to review session instead of work session")

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

// ErrAutoFixMaxRetries is returned when auto_fix reaches the maximum retry limit.
var ErrAutoFixMaxRetries = errors.New("auto_fix reached maximum retries")

// newCompleteCommand creates the complete command for marking a task as complete.
func newCompleteCommand(c *app.Container) *cobra.Command {
	var opts struct {
		comment string
	}

	cmd := &cobra.Command{
		Use:   "complete [id]",
		Short: "Mark task as complete",
		Long: `Mark a task as complete (in_progress/needs_input → reviewing).

If no ID is provided, the task ID is auto-detected from the current branch name.

Preconditions:
  - Task status must be 'in_progress' or 'needs_input'
  - No uncommitted changes in the worktree

If [complete].command is configured, it will be executed before transitioning
the status. If the command fails, the completion is aborted.

If skip_review is enabled (via task setting or config), the task transitions
directly to 'reviewed'. Otherwise, it transitions to 'reviewing' and a review
session is started.

Auto-fix mode:
  When [complete].auto_fix is enabled, the review runs synchronously.
  - If LGTM: outputs "LGTM" and exits successfully
  - If not LGTM: outputs review feedback for the worker to address
  - Use [complete].auto_fix_max_retries to limit retry attempts (default: 3)

Examples:
  # Complete task by ID
  crew complete 1

  # Complete with a comment
  crew complete 1 --comment "Implementation complete"

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
				TaskID:  taskID,
				Comment: opts.comment,
			})
			if err != nil {
				return err
			}

			// Handle auto_fix mode
			if out.AutoFixEnabled {
				return handleAutoFixReview(cmd, c, out)
			}

			if out.ShouldStartReview {
				// Start review automatically (background)
				reviewUC := c.ReviewTaskUseCase(cmd.OutOrStdout(), cmd.ErrOrStderr())
				reviewOut, reviewErr := reviewUC.Execute(cmd.Context(), usecase.ReviewTaskInput{
					TaskID: taskID,
					Wait:   false, // Background execution
				})

				if reviewErr != nil {
					// Review failed to start
					// Note: Task is already in reviewing status. We log the error but don't fail the command.
					// The user can try 'crew review' later or manually fix the state.
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Task #%d completed, but failed to start review: %v\n", out.Task.ID, reviewErr)
					// We might consider reverting status here, but CompleteTask has committed it.
					// Leaving it as is for now as per minimal changes strategy.
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Completed task #%d and started review (session: %s): %s\n", out.Task.ID, reviewOut.SessionName, out.Task.Title)
				}
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Completed task #%d: %s\n", out.Task.ID, out.Task.Title)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.comment, "comment", "m", "", "Add a completion comment")

	return cmd
}

// handleAutoFixReview runs synchronous review for auto_fix mode.
// It outputs "LGTM" if the review passes, or the review feedback if not.
// On non-LGTM, it reverts task status to in_progress and increments retry count.
func handleAutoFixReview(cmd *cobra.Command, c *app.Container, out *usecase.CompleteTaskOutput) error {
	task := out.Task
	retryCount := task.AutoFixRetryCount

	// Check retry limit (before running review)
	if retryCount >= out.AutoFixMaxRetries {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "auto_fix: reached maximum retries (%d)\n", out.AutoFixMaxRetries)
		return ErrAutoFixMaxRetries
	}

	// Run synchronous review
	reviewUC := c.ReviewTaskUseCase(cmd.OutOrStdout(), cmd.ErrOrStderr())
	reviewOut, reviewErr := reviewUC.Execute(cmd.Context(), usecase.ReviewTaskInput{
		TaskID: task.ID,
		Wait:   true, // Synchronous execution for auto_fix
	})

	if reviewErr != nil {
		// Revert to in_progress on review failure
		task.Status = domain.StatusInProgress
		_ = c.Tasks.Save(task)
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Task #%d completed, but review failed: %v\n", task.ID, reviewErr)
		return reviewErr
	}

	// Check if LGTM
	reviewResult := strings.TrimSpace(reviewOut.Review)
	if isLGTM(reviewResult) {
		// LGTM - reset retry count and output success message
		task.AutoFixRetryCount = 0
		_ = c.Tasks.Save(task)
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "LGTM")
		return nil
	}

	// Not LGTM - revert to in_progress, increment retry count, and output feedback
	task.Status = domain.StatusInProgress
	task.AutoFixRetryCount = retryCount + 1
	if saveErr := c.Tasks.Save(task); saveErr != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to save task: %v\n", saveErr)
	}

	// Output review feedback for worker to address
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Review feedback (retry %d/%d):\n%s\n",
		retryCount+1, out.AutoFixMaxRetries, reviewResult)
	return nil
}

// isLGTM checks if the review result indicates LGTM (approved).
func isLGTM(reviewResult string) bool {
	return strings.HasPrefix(reviewResult, domain.ReviewLGTMPrefix)
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

// newReviewSessionEndedCommand creates the _review-session-ended internal command.
// This is called by the review script's EXIT trap to handle review session termination.
func newReviewSessionEndedCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "_review-session-ended <id> <exit-code> <output-file>",
		Short:  "Handle review session termination (internal command)",
		Hidden: true, // Internal command, not shown in help
		Args:   cobra.ExactArgs(3),
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

			// Read output from file
			outputFile := args[2]
			output := ""
			if data, readErr := os.ReadFile(outputFile); readErr == nil {
				output = string(data)
			}

			// Execute use case
			uc := c.ReviewSessionEndedUseCase(cmd.ErrOrStderr())
			_, err = uc.Execute(cmd.Context(), usecase.ReviewSessionEndedInput{
				TaskID:   taskID,
				ExitCode: exitCode,
				Output:   output,
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
		review bool
	}

	cmd := &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop a session",
		Long: `Stop a running session for a task.

This terminates the tmux session and cleans up generated scripts.
When stopping a work session, it clears agent info and updates the
status to 'stopped'.

The worktree is NOT deleted (use 'close' to also delete the worktree).

Examples:
  # Stop session for task #1
  crew stop 1

  # Stop review session for task #1
  crew stop 1 --review`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Execute use case
			uc := c.StopTaskUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.StopTaskInput{
				TaskID: taskID,
				Review: opts.review,
			})
			if err != nil {
				return err
			}

			if out.SessionName == "" {
				if opts.review {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No review session running for task #%d\n", taskID)
					return nil
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No running session for task #%d\n", taskID)
				return nil
			}

			label := "session"
			if opts.review || out.SessionName == domain.ReviewSessionName(taskID) {
				label = "review session"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Stopped %s %s\n", label, out.SessionName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.review, "review", "r", false, "Stop the reviewer session instead of the work session")

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
			_, err = uc.Execute(cmd.Context(), usecase.MergeTaskInput{
				TaskID:     taskID,
				BaseBranch: opts.base,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.base, "base", "", "Base branch to merge into (default: task's base branch or default branch)")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
