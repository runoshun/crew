package cli

import (
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/app"
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
