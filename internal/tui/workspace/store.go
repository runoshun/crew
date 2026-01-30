package workspace

import (
	"os"
	"path/filepath"

	"github.com/runoshun/git-crew/v2/internal/domain"
	infraWorkspace "github.com/runoshun/git-crew/v2/internal/infra/workspace"
)

// NewStoreFromDefault creates a workspace store using the default global config directory.
// Returns the store and any error that occurred during creation.
func NewStoreFromDefault() (*infraWorkspace.Store, error) {
	globalDir := defaultGlobalConfigDir()
	if globalDir == "" {
		return nil, infraWorkspace.ErrNoHomeDir
	}
	return infraWorkspace.NewStore(globalDir)
}

// defaultGlobalConfigDir returns the default global config directory.
// Returns empty string if home directory cannot be determined.
func defaultGlobalConfigDir() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configHome = filepath.Join(home, ".config")
	}
	return domain.GlobalCrewDir(configHome)
}
