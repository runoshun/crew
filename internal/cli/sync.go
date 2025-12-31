package cli

import (
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/spf13/cobra"
)

func newSyncCmd(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync tasks with remote",
		Long:  "Push and fetch task refs to/from remote repository",
	}

	cmd.AddCommand(newPushCmd(c))
	cmd.AddCommand(newFetchCmd(c))
	cmd.AddCommand(newNamespacesCmd(c))

	return cmd
}

func newPushCmd(c *app.Container) *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Push task refs to remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.Tasks.Push(); err != nil {
				return fmt.Errorf("push failed: %w", err)
			}

			fmt.Println("Tasks pushed to remote")
			return nil
		},
	}
}

func newFetchCmd(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch [namespace]",
		Short: "Fetch task refs from remote",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace := ""
			if len(args) > 0 {
				namespace = args[0]
			}

			if err := c.Tasks.Fetch(namespace); err != nil {
				return fmt.Errorf("fetch failed: %w", err)
			}

			if namespace == "" {
				fmt.Println("Tasks fetched from remote")
			} else {
				fmt.Printf("Tasks fetched from remote (namespace: %s)\n", namespace)
			}
			return nil
		},
	}
	return cmd
}

func newNamespacesCmd(c *app.Container) *cobra.Command {
	return &cobra.Command{
		Use:   "namespaces",
		Short: "List available namespaces on remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			namespaces, err := c.Tasks.ListNamespaces()
			if err != nil {
				return fmt.Errorf("list namespaces failed: %w", err)
			}

			if len(namespaces) == 0 {
				fmt.Println("No namespaces found on remote")
				return nil
			}

			for _, ns := range namespaces {
				fmt.Println(ns)
			}
			return nil
		},
	}
}
