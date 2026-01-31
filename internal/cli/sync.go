package cli

import (
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/spf13/cobra"
)

func newSyncCmd(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync tasks with remote (deprecated)",
		Long:  "Deprecated: tasks are stored locally and are not synced with remotes",
	}

	cmd.AddCommand(newPushCmd(c))
	cmd.AddCommand(newFetchCmd(c))
	cmd.AddCommand(newNamespacesCmd(c))

	return cmd
}

func newPushCmd(c *app.Container) *cobra.Command {
	_ = c
	return &cobra.Command{
		Use:   "push",
		Short: "Push task refs to remote (deprecated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "sync push is deprecated and currently a no-op"); err != nil {
				return fmt.Errorf("write warning: %w", err)
			}
			return nil
		},
	}
}

func newFetchCmd(c *app.Container) *cobra.Command {
	_ = c
	cmd := &cobra.Command{
		Use:   "fetch [namespace]",
		Short: "Fetch task refs from remote (deprecated)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "sync fetch is deprecated and currently a no-op"); err != nil {
				return fmt.Errorf("write warning: %w", err)
			}
			return nil
		},
	}
	return cmd
}

func newNamespacesCmd(c *app.Container) *cobra.Command {
	_ = c
	return &cobra.Command{
		Use:   "namespaces",
		Short: "List available namespaces on remote (deprecated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "sync namespaces is deprecated and currently a no-op"); err != nil {
				return fmt.Errorf("write warning: %w", err)
			}
			return nil
		},
	}
}
