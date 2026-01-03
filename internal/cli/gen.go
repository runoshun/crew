package cli

import (
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newGenCommand creates the gen command.
func newGenCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate files (skills, etc.)",
		Long:  `Generate various files for git-crew usage.`,
	}

	// Add subcommands
	cmd.AddCommand(newGenSkillCommand(c))

	return cmd
}

// newGenSkillCommand creates the gen skill subcommand.
func newGenSkillCommand(c *app.Container) *cobra.Command {
	var opts struct {
		claude   bool
		opencode bool
		codex    bool
	}

	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Generate crew-manager skill file",
		Long: `Generate crew-manager skill file for AI agents.

This command creates a skill file that helps manager agents orchestrate
the complete workflow: task creation → agent startup → monitoring → review → merge.

The skill file will be generated at:
- .claude/skills/crew-manager/SKILL.md
- .codex/skills/crew-manager/SKILL.md
- .opencode/skill/crew-manager/SKILL.md

Use flags to specify output destinations:
  --claude    Generate only for .claude/
  --opencode  Generate only for .opencode/
  --codex     Generate only for .codex/
  (no flags)  Generate for all destinations (default)`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc := c.GenSkillUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.GenSkillInput{
				Claude:   opts.claude,
				OpenCode: opts.opencode,
				Codex:    opts.codex,
			})
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Generated crew-manager skill at:")
			for _, path := range out.CreatedPaths {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", path)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.claude, "claude", false, "Generate only for .claude/ directory")
	cmd.Flags().BoolVar(&opts.opencode, "opencode", false, "Generate only for .opencode/ directory")
	cmd.Flags().BoolVar(&opts.codex, "codex", false, "Generate only for .codex/ directory")

	return cmd
}
