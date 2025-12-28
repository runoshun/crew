package cli

import (
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newConfigCommand creates the config command.
func newConfigCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Display configuration file contents",
		Long: `Display configuration file contents.

Shows the path and contents of both global and repository configuration files.
Files that don't exist are marked as "(not found)".`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc := c.ShowConfigUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.ShowConfigInput{})
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()

			// Display global config
			_, _ = fmt.Fprintf(w, "# Global config: %s\n", out.GlobalConfig.Path)
			if out.GlobalConfig.Exists {
				_, _ = fmt.Fprintln(w, out.GlobalConfig.Content)
			} else {
				_, _ = fmt.Fprintln(w, "(not found)")
			}

			_, _ = fmt.Fprintln(w)

			// Display repository config
			_, _ = fmt.Fprintf(w, "# Repository config: %s\n", out.RepoConfig.Path)
			if out.RepoConfig.Exists {
				_, _ = fmt.Fprintln(w, out.RepoConfig.Content)
			} else {
				_, _ = fmt.Fprintln(w, "(not found)")
			}

			return nil
		},
	}

	// Add init subcommand
	cmd.AddCommand(newConfigInitCommand(c))

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
			uc := c.InitConfigUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.InitConfigInput{
				Global: global,
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
