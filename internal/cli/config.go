package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newConfigCommand creates the config command.
func newConfigCommand(c *app.Container) *cobra.Command {
	var ignoreGlobal, ignoreRepo bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Display effective configuration",
		Long: `Display effective configuration after merging all sources.

Shows which config files were loaded and the final merged configuration.
Use --ignore-global or --ignore-repo to exclude specific sources for debugging.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc := c.ShowConfigUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.ShowConfigInput{
				IgnoreGlobal: ignoreGlobal,
				IgnoreRepo:   ignoreRepo,
			})
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()

			// Display loaded files section
			_, _ = fmt.Fprintln(w, "[Loaded from]")
			if !ignoreGlobal {
				if out.GlobalConfig.Exists {
					_, _ = fmt.Fprintf(w, "- %s\n", out.GlobalConfig.Path)
				} else {
					_, _ = fmt.Fprintf(w, "- %s (not found)\n", out.GlobalConfig.Path)
				}
			}
			if !ignoreRepo {
				if out.RepoConfig.Exists {
					_, _ = fmt.Fprintf(w, "- %s\n", out.RepoConfig.Path)
				} else {
					_, _ = fmt.Fprintf(w, "- %s (not found)\n", out.RepoConfig.Path)
				}
			}

			_, _ = fmt.Fprintln(w)

			// Display effective config in TOML format
			_, _ = fmt.Fprintln(w, "[Effective Config]")
			formatEffectiveConfig(w, out.EffectiveConfig)

			return nil
		},
	}

	cmd.Flags().BoolVar(&ignoreGlobal, "ignore-global", false, "Ignore global configuration")
	cmd.Flags().BoolVar(&ignoreRepo, "ignore-repo", false, "Ignore repository configuration")

	// Add init subcommand
	cmd.AddCommand(newConfigInitCommand(c))

	return cmd
}

// formatEffectiveConfig formats the effective config in TOML-like format.
func formatEffectiveConfig(w io.Writer, cfg *domain.Config) {
	// [agents] section - defaults
	_, _ = fmt.Fprintln(w, "[agents]")
	if cfg.AgentsConfig.DefaultWorker != "" {
		_, _ = fmt.Fprintf(w, "default_worker = %q\n", cfg.AgentsConfig.DefaultWorker)
	}
	if cfg.AgentsConfig.DefaultManager != "" {
		_, _ = fmt.Fprintf(w, "default_manager = %q\n", cfg.AgentsConfig.DefaultManager)
	}
	_, _ = fmt.Fprintln(w)

	// [agents.<name>] sections - sorted for consistent output
	agentNames := make([]string, 0, len(cfg.Agents))
	for name := range cfg.Agents {
		agentNames = append(agentNames, name)
	}
	sort.Strings(agentNames)

	for _, name := range agentNames {
		agent := cfg.Agents[name]
		_, _ = fmt.Fprintf(w, "[agents.%s]\n", name)
		if agent.Inherit != "" {
			_, _ = fmt.Fprintf(w, "inherit = %q\n", agent.Inherit)
		}
		if agent.CommandTemplate != "" {
			_, _ = fmt.Fprintf(w, "command_template = %q\n", agent.CommandTemplate)
		}
		if agent.Role != "" {
			_, _ = fmt.Fprintf(w, "role = %q\n", string(agent.Role))
		}
		if agent.Args != "" {
			_, _ = fmt.Fprintf(w, "args = %q\n", agent.Args)
		}
		if agent.DefaultModel != "" {
			_, _ = fmt.Fprintf(w, "default_model = %q\n", agent.DefaultModel)
		}
		if agent.SystemPrompt != "" {
			_, _ = fmt.Fprintf(w, "system_prompt = %s\n", formatMultilineString(agent.SystemPrompt))
		}
		if agent.Prompt != "" {
			_, _ = fmt.Fprintf(w, "prompt = %s\n", formatMultilineString(agent.Prompt))
		}
		if agent.Hidden {
			_, _ = fmt.Fprintf(w, "hidden = true\n")
		}
		_, _ = fmt.Fprintln(w)
	}

	// [complete] section
	if cfg.Complete.Command != "" {
		_, _ = fmt.Fprintln(w, "[complete]")
		_, _ = fmt.Fprintf(w, "command = %q\n", cfg.Complete.Command)
		_, _ = fmt.Fprintln(w)
	}

	// [diff] section
	if cfg.Diff.Command != "" || cfg.Diff.TUICommand != "" {
		_, _ = fmt.Fprintln(w, "[diff]")
		if cfg.Diff.Command != "" {
			_, _ = fmt.Fprintf(w, "command = %q\n", cfg.Diff.Command)
		}
		if cfg.Diff.TUICommand != "" {
			_, _ = fmt.Fprintf(w, "tui_command = %q\n", cfg.Diff.TUICommand)
		}
		_, _ = fmt.Fprintln(w)
	}

	// [log] section
	if cfg.Log.Level != "" {
		_, _ = fmt.Fprintln(w, "[log]")
		_, _ = fmt.Fprintf(w, "level = %q\n", cfg.Log.Level)
	}
}

// formatMultilineString formats a string for TOML output.
// Uses triple quotes for multiline strings.
func formatMultilineString(s string) string {
	if strings.Contains(s, "\n") {
		// Use TOML multiline basic string
		return fmt.Sprintf(`"""%s"""`, "\n"+s)
	}
	return fmt.Sprintf("%q", s)
}

// newConfigInitCommand creates the config init subcommand.
func newConfigInitCommand(c *app.Container) *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate configuration file template",
		Long: `Generate a configuration file template.

By default, creates the repository configuration file at .git/crew/config.toml.
With --global, creates the global configuration file at ~/.config/crew/config.toml.

Error conditions:
- Target file already exists: error`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Load config to get registered builtins for template generation
			cfg, err := c.ConfigLoader.Load()
			if err != nil {
				return err
			}

			uc := c.InitConfigUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.InitConfigInput{
				Global: global,
				Config: cfg,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created config file: %s\n", out.Path)
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Generate global configuration")

	return cmd
}
