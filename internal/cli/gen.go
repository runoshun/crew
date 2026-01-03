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
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Generate crew-manager skill file",
		Long: `Generate crew-manager skill file for AI agents.

This command creates a skill file that helps manager agents orchestrate
the complete workflow: task creation → agent startup → monitoring → review → merge.

The skill file will be generated at:
- .claude/skills/crew-manager/SKILL.md
- .codex/skills/crew-manager/SKILL.md
- .opencode/skills/crew-manager/SKILL.md`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			uc := c.GenSkillUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.GenSkillInput{})
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

	return cmd
}
