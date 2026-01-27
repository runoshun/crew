// Package usecase contains the application use cases.
package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// InitRepoInput contains the input parameters for InitRepo.
type InitRepoInput struct {
	CrewDir   string // Path to .crew directory
	RepoRoot  string // Path to repository root
	StorePath string // Path to tasks.json
}

// InitRepoOutput contains the output from InitRepo.
type InitRepoOutput struct {
	CrewDir            string // Path to created crew directory
	AlreadyInitialized bool   // True if was already initialized (repair only)
	Repaired           bool   // True if meta was repaired (NextTaskID updated)
	GitignoreNeedsAdd  bool   // True if .crew/ is not in .gitignore and needs to be added
}

// InitRepo initializes a repository for git-crew.
type InitRepo struct {
	storeInit domain.StoreInitializer
}

// NewInitRepo creates a new InitRepo use case.
func NewInitRepo(storeInit domain.StoreInitializer) *InitRepo {
	return &InitRepo{storeInit: storeInit}
}

// Execute initializes a repository for git-crew.
// It creates the .crew/ directory, tmux.conf, and empty tasks.json.
// If already initialized, it still runs Initialize() to repair any inconsistencies.
func (uc *InitRepo) Execute(_ context.Context, in InitRepoInput) (*InitRepoOutput, error) {
	alreadyInitialized := uc.storeInit.IsInitialized()

	if !alreadyInitialized {
		// Create crew directory
		if err := os.MkdirAll(in.CrewDir, 0o750); err != nil {
			return nil, fmt.Errorf("create crew directory: %w", err)
		}

		// Create scripts directory
		scriptsDir := filepath.Join(in.CrewDir, "scripts")
		if err := os.MkdirAll(scriptsDir, 0o750); err != nil {
			return nil, fmt.Errorf("create scripts directory: %w", err)
		}

		// Create logs directory
		logsDir := filepath.Join(in.CrewDir, "logs")
		if err := os.MkdirAll(logsDir, 0o750); err != nil {
			return nil, fmt.Errorf("create logs directory: %w", err)
		}

		// Create tmux.conf with minimal configuration
		tmuxConfPath := filepath.Join(in.CrewDir, "tmux.conf")
		tmuxConf := `# git-crew tmux configuration
unbind-key -a              # Unbind all keys
bind-key -n C-g detach-client  # Ctrl+G to detach
set -g status off          # Hide status bar
set -g escape-time 0       # No escape delay
`
		if err := os.WriteFile(tmuxConfPath, []byte(tmuxConf), 0o600); err != nil {
			return nil, fmt.Errorf("create tmux.conf: %w", err)
		}
	}

	// Initialize task store (creates empty tasks.json or repairs inconsistencies)
	repaired, err := uc.storeInit.Initialize()
	if err != nil {
		return nil, fmt.Errorf("initialize task store: %w", err)
	}

	// Check if .crew/ needs to be added to .gitignore
	gitignoreNeedsAdd := false
	if !alreadyInitialized && in.RepoRoot != "" {
		gitignoreNeedsAdd = !isCrewInGitignore(in.RepoRoot)
	}

	return &InitRepoOutput{
		CrewDir:            in.CrewDir,
		AlreadyInitialized: alreadyInitialized,
		Repaired:           repaired,
		GitignoreNeedsAdd:  gitignoreNeedsAdd,
	}, nil
}

// isCrewInGitignore checks if .crew/ is in .gitignore.
func isCrewInGitignore(repoRoot string) bool {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return false
	}

	lines := splitLines(string(content))
	for _, line := range lines {
		// Check for exact match or with trailing slash
		if line == ".crew" || line == ".crew/" {
			return true
		}
	}
	return false
}

// splitLines splits a string by newlines, handling both Unix and Windows line endings.
func splitLines(s string) []string {
	var lines []string
	var current string
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else if c != '\r' {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
