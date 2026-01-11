package cli

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/spf13/cobra"
)

// newListAgentsCommand creates the list-agents command.
func newListAgentsCommand(c *app.Container) *cobra.Command {
	var opts struct {
		All      bool
		Disabled bool
	}

	cmd := &cobra.Command{
		Use:   "list-agents",
		Short: "List available agents",
		Long: `List all configured agents with their roles and descriptions.

By default, only enabled agents are shown. Use --all to show all agents
including disabled ones, or --disabled to show only disabled agents.

Note: --all and --disabled are mutually exclusive.

Examples:
  # List enabled agents
  crew list-agents

  # List all agents including disabled ones
  crew list-agents --all

  # List only disabled agents
  crew list-agents --disabled`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Validate mutually exclusive flags
			if opts.All && opts.Disabled {
				return fmt.Errorf("--all and --disabled are mutually exclusive")
			}

			cfg, err := c.ConfigLoader.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Collect agents to display
			type agentInfo struct {
				Name        string
				Role        domain.Role
				Description string
				Disabled    bool
			}

			var agents []agentInfo
			for name, agent := range cfg.Agents {
				disabled := domain.IsAgentDisabled(name, cfg.AgentsConfig.DisabledAgents)

				// Filter based on flags
				if opts.Disabled && !disabled {
					continue
				}
				if !opts.All && !opts.Disabled && disabled {
					continue
				}

				agents = append(agents, agentInfo{
					Name:        name,
					Role:        agent.Role,
					Description: agent.Description,
					Disabled:    disabled,
				})
			}

			// Sort by name
			sort.Slice(agents, func(i, j int) bool {
				return agents[i].Name < agents[j].Name
			})

			// Output
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "NAME\tROLE\tDESCRIPTION\tSTATUS")

			for _, a := range agents {
				role := string(a.Role)
				if role == "" {
					role = "worker"
				}
				status := ""
				if a.Disabled {
					status = "disabled"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Name, role, a.Description, status)
			}

			return w.Flush()
		},
	}

	cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "Show all agents (including disabled)")
	cmd.Flags().BoolVarP(&opts.Disabled, "disabled", "d", false, "Show only disabled agents")

	return cmd
}
