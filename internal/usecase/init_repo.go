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
	CrewDir   string // Path to .git/crew directory
	StorePath string // Path to tasks.json
}

// InitRepoOutput contains the output from InitRepo.
type InitRepoOutput struct {
	CrewDir string // Path to created crew directory
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
// It creates the .git/crew/ directory, tmux.conf, and empty tasks.json.
func (uc *InitRepo) Execute(_ context.Context, in InitRepoInput) (*InitRepoOutput, error) {
	// Check if already initialized
	if uc.storeInit.IsInitialized() {
		return nil, domain.ErrAlreadyInitialized
	}

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

	// Initialize task store (creates empty tasks.json)
	if err := uc.storeInit.Initialize(); err != nil {
		return nil, fmt.Errorf("initialize task store: %w", err)
	}

	return &InitRepoOutput{
		CrewDir: in.CrewDir,
	}, nil
}
