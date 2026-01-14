package cli

import (
	"fmt"
	"io"

	"github.com/pelletier/go-toml/v2"
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newConfigCommand creates the config command.
func newConfigCommand(c *app.Container) *cobra.Command {
	var ignoreGlobal, ignoreOverride, ignoreRepo, ignoreRootRepo bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Display effective configuration",
		Long: `Display effective configuration after merging all sources.

Shows which config files were loaded and the final merged configuration.
Use --ignore-global, --ignore-override, --ignore-repo or --ignore-root-repo to exclude specific sources for debugging.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc := c.ShowConfigUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.ShowConfigInput{
				IgnoreGlobal:   ignoreGlobal,
				IgnoreOverride: ignoreOverride,
				IgnoreRepo:     ignoreRepo,
				IgnoreRootRepo: ignoreRootRepo,
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
			if !ignoreOverride {
				if out.OverrideConfig.Exists {
					_, _ = fmt.Fprintf(w, "- %s\n", out.OverrideConfig.Path)
				} else {
					_, _ = fmt.Fprintf(w, "- %s (not found)\n", out.OverrideConfig.Path)
				}
			}
			if !ignoreRootRepo {
				if out.RootRepoConfig.Exists {
					_, _ = fmt.Fprintf(w, "- %s\n", out.RootRepoConfig.Path)
				} else {
					_, _ = fmt.Fprintf(w, "- %s (not found)\n", out.RootRepoConfig.Path)
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
	cmd.Flags().BoolVar(&ignoreOverride, "ignore-override", false, "Ignore override configuration (config.override.toml)")
	cmd.Flags().BoolVar(&ignoreRepo, "ignore-repo", false, "Ignore repository configuration (.git/crew/config.toml)")
	cmd.Flags().BoolVar(&ignoreRootRepo, "ignore-root-repo", false, "Ignore root repository configuration (.crew.toml)")

	// Add init subcommand
	cmd.AddCommand(newConfigInitCommand(c))

	return cmd
}

// formatEffectiveConfig formats the effective config in TOML format.
func formatEffectiveConfig(w io.Writer, cfg *domain.Config) {
	// go-toml/v2 does not support MarshalTOML interface, so we need to manually
	// construct the structure that merges Agents and AgentsConfig under [agents].
	//nolint:govet // fieldalignment: readability prioritized over memory optimization in local struct
	type agentsSection struct {
		Agents          map[string]domain.Agent `toml:",inline"`
		DisabledAgents  []string                `toml:"disabled_agents,omitempty"`
		DefaultWorker   string                  `toml:"worker_default,omitempty"`
		DefaultManager  string                  `toml:"manager_default,omitempty"`
		DefaultReviewer string                  `toml:"reviewer_default,omitempty"`
		WorkerPrompt    string                  `toml:"worker_prompt,omitempty"`
		ManagerPrompt   string                  `toml:"manager_prompt,omitempty"`
		ReviewerPrompt  string                  `toml:"reviewer_prompt,omitempty"`
	}

	type effectiveConfig struct {
		Agents         agentsSection         `toml:"agents"`
		Complete       domain.CompleteConfig `toml:"complete"`
		Diff           domain.DiffConfig     `toml:"diff"`
		Log            domain.LogConfig      `toml:"log"`
		Tasks          domain.TasksConfig    `toml:"tasks"`
		TUI            domain.TUIConfig      `toml:"tui"`
		Worktree       domain.WorktreeConfig `toml:"worktree"`
		OnboardingDone bool                  `toml:"onboarding_done,omitempty"`
	}

	output := effectiveConfig{
		Agents: agentsSection{
			Agents:          cfg.Agents,
			DisabledAgents:  cfg.AgentsConfig.DisabledAgents,
			DefaultWorker:   cfg.AgentsConfig.DefaultWorker,
			DefaultManager:  cfg.AgentsConfig.DefaultManager,
			DefaultReviewer: cfg.AgentsConfig.DefaultReviewer,
			WorkerPrompt:    cfg.AgentsConfig.WorkerPrompt,
			ManagerPrompt:   cfg.AgentsConfig.ManagerPrompt,
			ReviewerPrompt:  cfg.AgentsConfig.ReviewerPrompt,
		},
		Complete:       cfg.Complete,
		Diff:           cfg.Diff,
		Log:            cfg.Log,
		Tasks:          cfg.Tasks,
		TUI:            cfg.TUI,
		Worktree:       cfg.Worktree,
		OnboardingDone: cfg.OnboardingDone,
	}

	data, err := toml.Marshal(output)
	if err != nil {
		_, _ = fmt.Fprintf(w, "Error marshaling config: %v\n", err)
		return
	}
	if _, err := w.Write(data); err != nil {
		_, _ = fmt.Fprintf(w, "Error writing config: %v\n", err)
	}
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
