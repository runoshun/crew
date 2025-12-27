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
