package cli

import (
	"fmt"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

func newPruneCommand(c *app.Container) *cobra.Command {
	var (
		all    bool
		dryRun bool
		yes    bool
	)

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Cleanup branches and worktrees for completed tasks",
		Long: `Prune removes branches and worktrees for closed tasks.
It also cleans up orphan crew branches and worktrees.

Note: Tasks themselves are NOT deleted, only their branches and worktrees.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			uc := c.PruneTasksUseCase()

			// First, run as dry-run to show what will be deleted
			// unless -y is specified, in which case we just do it if not dry-run

			// Actually, the common pattern is:
			// If --dry-run: show what would be done, do nothing.
			// If not --dry-run and not --yes: show what would be done, ask confirmation.
			// If not --dry-run and --yes: just do it.

			// So we always need to know what will be deleted first.
			// Let's use DryRun=true for the preview phase if we need confirmation.

			previewInput := usecase.PruneTasksInput{
				All:    all,
				DryRun: true,
			}

			preview, err := uc.Execute(cmd.Context(), previewInput)
			if err != nil {
				return err
			}

			if len(preview.DeletedBranches) == 0 && len(preview.DeletedWorktrees) == 0 {
				fmt.Println("Nothing to prune.")
				return nil
			}

			// Display what will be pruned
			if len(preview.DeletedBranches) > 0 {
				fmt.Println("Branches to be deleted:")
				for _, b := range preview.DeletedBranches {
					fmt.Printf("  - %s\n", b)
				}
			}
			if len(preview.DeletedWorktrees) > 0 {
				fmt.Println("Worktrees to be deleted:")
				for _, w := range preview.DeletedWorktrees {
					fmt.Printf("  - %s\n", w)
				}
			}
			fmt.Println()

			if dryRun {
				fmt.Println("Dry run: no changes made.")
				return nil
			}

			if !yes {
				fmt.Print("Are you sure you want to delete these resources? [y/N] ")
				var response string
				if _, scanErr := fmt.Scanln(&response); scanErr != nil {
					// If scan fails (e.g. EOF), assume no
					fmt.Println("\nAborted.")
					return nil
				}
				if strings.ToLower(response) != "y" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			// Execute actual pruning
			input := usecase.PruneTasksInput{
				All:    all,
				DryRun: false,
			}

			// We could use the output from Execute, but we already showed the preview.
			// The only difference might be if something changed in between (unlikely in this context)
			// or errors during deletion.

			out, err := uc.Execute(cmd.Context(), input)
			if err != nil {
				// We might have partial success
				return err
			}

			// Summarize results
			deletedCount := len(out.DeletedBranches) + len(out.DeletedWorktrees)
			fmt.Printf("Successfully pruned %d items.\n", deletedCount)

			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "(deprecated) No longer needed, all closed tasks are pruned")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Display only, no deletion")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
