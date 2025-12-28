package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newNewCommand creates the new command for creating tasks.
func newNewCommand(c *app.Container) *cobra.Command {
	var opts struct {
		Title       string
		Description string
		Labels      []string
		ParentID    int
		Issue       int
	}

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new task",
		Long: `Create a new task for git-crew to manage.

The task is created with status 'todo'. The worktree and branch
are not created until the task is started with 'git crew start'.

Examples:
  # Create a root task
  git crew new --title "Auth refactoring"

  # Create a sub-task under task #1
  git crew new --parent 1 --title "OAuth2.0 implementation"

  # Create a task linked to a GitHub issue
  git crew new --title "Fix login bug" --issue 42

  # Create a task with labels
  git crew new --title "Add feature" --label feature --label urgent`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Build input
			input := usecase.NewTaskInput{
				Title:       opts.Title,
				Description: opts.Description,
				Issue:       opts.Issue,
				Labels:      opts.Labels,
			}

			// Set parent ID if specified
			if opts.ParentID > 0 {
				input.ParentID = &opts.ParentID
			}

			// Execute use case
			uc := c.NewTaskUseCase()
			out, err := uc.Execute(cmd.Context(), input)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created task #%d\n", out.TaskID)
			return nil
		},
	}

	// Required flags
	cmd.Flags().StringVar(&opts.Title, "title", "", "Task title (required)")
	_ = cmd.MarkFlagRequired("title")

	// Optional flags
	cmd.Flags().StringVar(&opts.Description, "desc", "", "Task description")
	cmd.Flags().IntVar(&opts.ParentID, "parent", 0, "Parent task ID (creates a sub-task)")
	cmd.Flags().IntVar(&opts.Issue, "issue", 0, "Linked GitHub issue number")
	cmd.Flags().StringArrayVar(&opts.Labels, "label", nil, "Labels (can specify multiple)")

	return cmd
}

// newListCommand creates the list command for listing tasks.
func newListCommand(c *app.Container) *cobra.Command {
	var opts struct {
		Labels   []string
		ParentID int
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Long: `Display a list of tasks.

Output format is tab-separated with columns:
  ID, PARENT, STATUS, AGENT, LABELS, [ELAPSED], TITLE

ELAPSED is only shown for tasks with status 'in_progress'.

Examples:
  # List all tasks
  git crew list

  # List only sub-tasks of task #1
  git crew list --parent 1

  # List tasks with specific labels
  git crew list --label bug --label urgent`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Build input
			input := usecase.ListTasksInput{
				Labels: opts.Labels,
			}

			// Set parent ID if specified
			if opts.ParentID > 0 {
				input.ParentID = &opts.ParentID
			}

			// Execute use case
			uc := c.ListTasksUseCase()
			out, err := uc.Execute(cmd.Context(), input)
			if err != nil {
				return err
			}

			// Print output
			printTaskList(cmd.OutOrStdout(), out.Tasks, c.Clock)
			return nil
		},
	}

	// Optional flags
	cmd.Flags().IntVar(&opts.ParentID, "parent", 0, "Show only children of this task")
	cmd.Flags().StringArrayVar(&opts.Labels, "label", nil, "Filter by labels (AND condition)")

	return cmd
}

// printTaskList prints tasks in TSV format.
func printTaskList(w io.Writer, tasks []*domain.Task, clock domain.Clock) {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	defer func() { _ = tw.Flush() }()

	// Header
	_, _ = fmt.Fprintln(tw, "ID\tPARENT\tSTATUS\tAGENT\tLABELS\tTITLE")

	// Rows
	for _, task := range tasks {
		parentStr := "-"
		if task.ParentID != nil {
			parentStr = fmt.Sprintf("%d", *task.ParentID)
		}

		agentStr := "-"
		if task.Agent != "" {
			agentStr = task.Agent
		}

		labelsStr := "-"
		if len(task.Labels) > 0 {
			labelsStr = "[" + strings.Join(task.Labels, ",") + "]"
		}

		// Format status with optional elapsed time for in_progress
		statusStr := string(task.Status)
		if task.Status == domain.StatusInProgress && !task.Started.IsZero() {
			elapsed := clock.Now().Sub(task.Started)
			statusStr = fmt.Sprintf("%s (%s)", task.Status, formatDuration(elapsed))
		}

		_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\n",
			task.ID,
			parentStr,
			statusStr,
			agentStr,
			labelsStr,
			task.Title,
		)
	}
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// newShowCommand creates the show command for displaying task details.
func newShowCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [id]",
		Short: "Display task details",
		Long: `Display detailed information about a task.

If no ID is provided, the task ID is auto-detected from the current branch name.
The branch must follow the naming convention: crew-<id> or crew-<id>-gh-<issue>

Output includes:
  - Task ID and title
  - Description
  - Status, parent, branch, labels
  - GitHub issue and PR numbers (if linked)
  - Running agent and session info
  - Sub-tasks (if any)
  - Comments (if any)

Examples:
  # Show task by ID
  git crew show 1

  # Auto-detect task from current branch
  git crew show`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve task ID
			taskID, err := resolveTaskID(args, c.Git)
			if err != nil {
				return err
			}

			// Execute use case
			uc := c.ShowTaskUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.ShowTaskInput{
				TaskID: taskID,
			})
			if err != nil {
				return err
			}

			// Print output
			printTaskDetails(cmd.OutOrStdout(), out)
			return nil
		},
	}

	return cmd
}

// resolveTaskID resolves the task ID from arguments or current branch.
func resolveTaskID(args []string, git domain.Git) (int, error) {
	if len(args) > 0 {
		// Parse from argument
		id, err := parseTaskID(args[0])
		if err != nil {
			return 0, fmt.Errorf("invalid task ID: %w", err)
		}
		return id, nil
	}

	// Auto-detect from branch
	if git == nil {
		return 0, fmt.Errorf("task ID is required (not on a crew branch)")
	}

	branch, err := git.CurrentBranch()
	if err != nil {
		return 0, fmt.Errorf("failed to detect current branch: %w", err)
	}

	id, ok := domain.ParseBranchTaskID(branch)
	if !ok {
		return 0, fmt.Errorf("task ID is required (current branch '%s' is not a crew branch)", branch)
	}

	return id, nil
}

// parseTaskID parses a task ID string to int.
func parseTaskID(s string) (int, error) {
	// Remove leading # if present
	s = strings.TrimPrefix(s, "#")
	var id int
	_, err := fmt.Sscanf(s, "%d", &id)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, fmt.Errorf("task ID must be positive")
	}
	return id, nil
}

// newEditCommand creates the edit command for editing task information.
func newEditCommand(c *app.Container) *cobra.Command {
	var opts struct {
		Title        string
		Description  string
		AddLabels    []string
		RemoveLabels []string
	}

	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit task information",
		Long: `Edit an existing task's title, description, or labels.

At least one of --title, --desc, --add-label, or --rm-label must be specified.

Examples:
  # Change task title
  git crew edit 1 --title "New task title"

  # Update description
  git crew edit 1 --desc "Updated description text"

  # Add labels
  git crew edit 1 --add-label bug --add-label urgent

  # Remove labels
  git crew edit 1 --rm-label old-label

  # Multiple changes at once
  git crew edit 1 --title "New title" --add-label feature --rm-label draft`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Build input
			input := usecase.EditTaskInput{
				TaskID:       taskID,
				AddLabels:    opts.AddLabels,
				RemoveLabels: opts.RemoveLabels,
			}

			// Set optional fields only if provided
			if cmd.Flags().Changed("title") {
				input.Title = &opts.Title
			}
			if cmd.Flags().Changed("desc") {
				input.Description = &opts.Description
			}

			// Execute use case
			uc := c.EditTaskUseCase()
			out, err := uc.Execute(cmd.Context(), input)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated task #%d: %s\n", out.Task.ID, out.Task.Title)
			return nil
		},
	}

	// Optional flags
	cmd.Flags().StringVar(&opts.Title, "title", "", "New task title")
	cmd.Flags().StringVar(&opts.Description, "desc", "", "New task description")
	cmd.Flags().StringArrayVar(&opts.AddLabels, "add-label", nil, "Labels to add (can specify multiple)")
	cmd.Flags().StringArrayVar(&opts.RemoveLabels, "rm-label", nil, "Labels to remove (can specify multiple)")

	return cmd
}

// newRmCommand creates the rm command for deleting tasks.
func newRmCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <id>",
		Short: "Delete a task",
		Long: `Delete a task from git-crew.

This removes the task from the store. In Phase 2, this does not
clean up worktrees or sessions - that will be added in later phases.

Examples:
  # Delete task by ID
  git crew rm 1

  # Delete task using # prefix
  git crew rm "#1"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Execute use case
			uc := c.DeleteTaskUseCase()
			_, err = uc.Execute(cmd.Context(), usecase.DeleteTaskInput{
				TaskID: taskID,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted task #%d\n", taskID)
			return nil
		},
	}

	return cmd
}

// printTaskDetails prints task details in a formatted output.
func printTaskDetails(w io.Writer, out *usecase.ShowTaskOutput) {
	task := out.Task

	// Header
	_, _ = fmt.Fprintf(w, "# Task %d: %s\n\n", task.ID, task.Title)

	// Description
	if task.Description != "" {
		_, _ = fmt.Fprintf(w, "%s\n\n", task.Description)
	}

	// Fields
	_, _ = fmt.Fprintf(w, "Status: %s\n", task.Status)

	if task.ParentID != nil {
		_, _ = fmt.Fprintf(w, "Parent: #%d\n", *task.ParentID)
	} else {
		_, _ = fmt.Fprintln(w, "Parent: none")
	}

	_, _ = fmt.Fprintf(w, "Branch: %s\n", domain.BranchName(task.ID, task.Issue))

	if len(task.Labels) > 0 {
		_, _ = fmt.Fprintf(w, "Labels: [%s]\n", strings.Join(task.Labels, ", "))
	} else {
		_, _ = fmt.Fprintln(w, "Labels: none")
	}

	_, _ = fmt.Fprintf(w, "Created: %s\n", task.Created.Format(time.RFC3339))

	if task.Issue > 0 {
		_, _ = fmt.Fprintf(w, "Issue: #%d\n", task.Issue)
	}

	if task.PR > 0 {
		_, _ = fmt.Fprintf(w, "PR: #%d\n", task.PR)
	}

	if task.Agent != "" {
		_, _ = fmt.Fprintf(w, "Agent: %s (session: %s)\n", task.Agent, task.Session)
	}

	// Sub-tasks
	if len(out.Children) > 0 {
		_, _ = fmt.Fprintln(w, "\nSub-tasks:")
		for _, child := range out.Children {
			_, _ = fmt.Fprintf(w, "  #%d [%s] %s\n", child.ID, child.Status, child.Title)
		}
	}

	// Comments
	if len(out.Comments) > 0 {
		_, _ = fmt.Fprintln(w, "\nComments:")
		for _, comment := range out.Comments {
			_, _ = fmt.Fprintf(w, "  [%s] %s\n", comment.Time.Format(time.RFC3339), comment.Text)
		}
	}
}
