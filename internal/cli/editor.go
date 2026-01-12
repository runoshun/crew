package cli

import (
	"fmt"
	"os"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// getEditor returns the user's preferred editor from environment variables.
// It checks EDITOR, then VISUAL, and defaults to vim if neither is set.
func getEditor() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vim"
	}
	return editor
}

// openEditor opens the specified file in the user's editor.
// It returns an error if the editor cannot be started or exits with a non-zero status.
func openEditor(filePath string, executor domain.CommandExecutor) error {
	editor := getEditor()

	cmd := domain.NewCommand(editor, []string{filePath}, "")
	if err := executor.ExecuteInteractive(cmd); err != nil {
		return fmt.Errorf("failed to run editor %s: %w", editor, err)
	}

	return nil
}
