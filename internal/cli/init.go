package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newInitCommand creates the init command.
func newInitCommand(c *app.Container) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize repository for git-crew",
		Long: `Initialize a repository for git-crew.

This command creates the .crew/ directory with:
- tmux.conf: minimal tmux configuration
- tasks.json: empty task store
- scripts/: directory for task scripts
- logs/: directory for log files

Preconditions:
- Current directory must be inside a git repository

Error conditions:
- Already initialized: "crew already initialized"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get use case from container
			uc := c.InitRepoUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.InitRepoInput{
				CrewDir:   c.Config.CrewDir,
				RepoRoot:  c.Config.RepoRoot,
				StorePath: c.Config.StorePath,
			})
			if err != nil {
				return err
			}

			if out.AlreadyInitialized {
				if out.Repaired {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Repaired task ID sequence")
				} else {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "crew already initialized")
				}
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Initialized git-crew in %s\n", out.CrewDir)
			}

			// Prompt to add .crew/ to .gitignore if needed
			if out.GitignoreNeedsAdd {
				if promptAddToGitignore(cmd) {
					if err := addCrewToGitignore(c.Config.RepoRoot); err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to update .gitignore: %v\n", err)
					} else {
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Added .crew/ to .gitignore")
					}
				}
			}

			return nil
		},
	}
}

// promptAddToGitignore prompts the user to add .crew/ to .gitignore.
// Returns true if the user agrees.
func promptAddToGitignore(cmd *cobra.Command) bool {
	_, _ = fmt.Fprint(cmd.OutOrStdout(), ".crew/ is not in .gitignore. Add it? [Y/n] ")
	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))
	// Default to yes (empty or "y" or "yes")
	return input == "" || input == "y" || input == "yes"
}

// addCrewToGitignore adds .crew/ to the .gitignore file.
func addCrewToGitignore(repoRoot string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	// Read existing content
	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Ensure there's a newline at the end before adding
	newContent := string(content)
	if len(newContent) > 0 && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += ".crew/\n"

	return os.WriteFile(gitignorePath, []byte(newContent), 0o600)
}
