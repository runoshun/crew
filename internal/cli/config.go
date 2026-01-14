package cli

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newConfigCommand creates the config command.
func newConfigCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `Manage git-crew configuration files and settings.`,
		// No RunE: shows subcommand list when called without arguments
	}

	// Add subcommands
	cmd.AddCommand(newConfigShowCommand(c))
	cmd.AddCommand(newConfigTemplateCommand(c))
	cmd.AddCommand(newConfigInitCommand(c))

	return cmd
}

// newConfigShowCommand creates the config show subcommand.
func newConfigShowCommand(c *app.Container) *cobra.Command {
	var ignoreGlobal, ignoreOverride, ignoreRepo, ignoreRootRepo bool

	cmd := &cobra.Command{
		Use:   "show",
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
			if err := formatEffectiveConfig(w, out.EffectiveConfig); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&ignoreGlobal, "ignore-global", false, "Ignore global configuration")
	cmd.Flags().BoolVar(&ignoreOverride, "ignore-override", false, "Ignore override configuration (config.override.toml)")
	cmd.Flags().BoolVar(&ignoreRepo, "ignore-repo", false, "Ignore repository configuration (.git/crew/config.toml)")
	cmd.Flags().BoolVar(&ignoreRootRepo, "ignore-root-repo", false, "Ignore root repository configuration (.crew.toml)")

	return cmd
}

// formatEffectiveConfig formats the effective config in TOML format.
// Uses reflection to automatically handle domain.Config structure changes.
func formatEffectiveConfig(w io.Writer, cfg *domain.Config) error {
	output := make(map[string]any)

	// 1. Merge Agents and AgentsConfig under [agents]
	agentsMap := make(map[string]any)

	// Extract AgentsConfig fields dynamically from toml tags
	acVal := reflect.ValueOf(cfg.AgentsConfig)
	acType := acVal.Type()
	for i := 0; i < acVal.NumField(); i++ {
		field := acType.Field(i)
		tag := field.Tag.Get("toml")
		if tag == "" || tag == "-" {
			continue
		}
		tagName := strings.Split(tag, ",")[0]
		val := acVal.Field(i).Interface()
		// omitempty: skip zero values
		if !reflect.ValueOf(val).IsZero() {
			agentsMap[tagName] = val
		}
	}

	// Add Agents as inline entries
	for name, agent := range cfg.Agents {
		agentsMap[name] = agent
	}
	output["agents"] = agentsMap

	// 2. Extract other Config fields automatically from toml tags
	cfgVal := reflect.ValueOf(cfg).Elem()
	cfgType := cfgVal.Type()
	for i := 0; i < cfgVal.NumField(); i++ {
		field := cfgType.Field(i)
		tag := field.Tag.Get("toml")
		if tag == "" || tag == "-" {
			continue
		}
		tagName := strings.Split(tag, ",")[0]
		// Skip special fields already handled
		if tagName == "agents" || field.Name == "Agents" || field.Name == "AgentsConfig" || field.Name == "Warnings" {
			continue
		}
		fieldVal := cfgVal.Field(i)
		// omitempty: skip zero values for bool fields
		if field.Type.Kind() == reflect.Bool && !fieldVal.Bool() {
			if strings.Contains(tag, "omitempty") {
				continue
			}
		}
		output[tagName] = fieldVal.Interface()
	}

	// Encode to TOML
	if err := toml.NewEncoder(w).Encode(output); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	return nil
}

// newConfigTemplateCommand creates the config template subcommand.
func newConfigTemplateCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Output configuration template",
		Long: `Output a configuration file template to stdout.

This command is useful for:
- Piping template output for custom processing
- Comparing against existing configuration files
- Generating initial configuration without creating files

Note: This command outputs a base template with builtin agents registered.
It does not depend on existing configuration files and will work even if they are broken.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Load config with all file sources ignored to get only builtin agents
			// This ensures template output works even if config files are broken
			cfg, err := c.ConfigLoader.LoadWithOptions(domain.LoadConfigOptions{
				IgnoreGlobal:   true,
				IgnoreOverride: true,
				IgnoreRootRepo: true,
				IgnoreRepo:     true,
			})
			if err != nil {
				return err
			}

			uc := c.ShowConfigTemplateUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.ShowConfigTemplateInput{
				Config: cfg,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), out.Template)
			return nil
		},
	}

	return cmd
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
