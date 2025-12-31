package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/spf13/cobra"
)

func newSnapshotCmd(container *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage task snapshots",
		Long:  "Save, list, and restore task state snapshots.",
	}

	cmd.AddCommand(newSnapshotSaveCmd(container))
	cmd.AddCommand(newSnapshotListCmd(container))
	cmd.AddCommand(newSnapshotRestoreCmd(container))

	return cmd
}

// getGitHEAD returns the current git HEAD SHA.
func getGitHEAD() (string, error) {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func newSnapshotSaveCmd(container *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save current task state as a snapshot",
		Long:  "Save the current task state as a snapshot linked to the current git HEAD.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get current HEAD
			head, err := getGitHEAD()
			if err != nil {
				return fmt.Errorf("get HEAD: %w", err)
			}

			if err := container.Tasks.SaveSnapshot(head); err != nil {
				return fmt.Errorf("save snapshot: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Snapshot saved for %s\n", head[:7])
			return nil
		},
	}

	return cmd
}

func newSnapshotListCmd(container *app.Container) *cobra.Command {
	var allFlag bool

	cmd := &cobra.Command{
		Use:   "list [sha]",
		Short: "List snapshots",
		Long:  "List all snapshots, optionally filtered by git SHA.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var mainSHA string
			if len(args) > 0 {
				mainSHA = args[0]
			} else if !allFlag {
				// Default to current HEAD
				head, err := getGitHEAD()
				if err != nil {
					return fmt.Errorf("get HEAD: %w", err)
				}
				mainSHA = head
			}

			snapshots, err := container.Tasks.ListSnapshots(mainSHA)
			if err != nil {
				return fmt.Errorf("list snapshots: %w", err)
			}

			if len(snapshots) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No snapshots found")
				return nil
			}

			for _, s := range snapshots {
				fmt.Fprintf(cmd.OutOrStdout(), "%s_%03d  (%s)\n", s.MainSHA[:7], s.Seq, s.Ref)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Show all snapshots (not just current HEAD)")

	return cmd
}

func newSnapshotRestoreCmd(container *app.Container) *cobra.Command {
	var noSave bool

	cmd := &cobra.Command{
		Use:   "restore <ref>",
		Short: "Restore tasks from a snapshot",
		Long:  "Restore the task state from a previously saved snapshot.\nAutomatically saves current state before restoring unless --no-save is specified.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			snapshotRef := args[0]

			// Save current state before restoring (unless --no-save)
			if !noSave {
				head, err := getGitHEAD()
				if err != nil {
					return fmt.Errorf("get HEAD: %w", err)
				}
				if err := container.Tasks.SaveSnapshot(head); err != nil {
					return fmt.Errorf("save current state: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Saved current state for %s\n", head[:7])
			}

			if err := container.Tasks.RestoreSnapshot(snapshotRef); err != nil {
				return fmt.Errorf("restore snapshot: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Restored from %s\n", snapshotRef)
			return nil
		},
	}

	cmd.Flags().BoolVar(&noSave, "no-save", false, "Don't save current state before restoring")

	return cmd
}
