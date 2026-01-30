package workspace

import (
	"os"
	"path/filepath"

	"github.com/runoshun/git-crew/v2/internal/domain"
	infraWorkspace "github.com/runoshun/git-crew/v2/internal/infra/workspace"
)

// NewStoreFromDefault creates a workspace store using the default global config directory.
func NewStoreFromDefault() *infraWorkspace.Store {
	return infraWorkspace.NewStore(defaultGlobalConfigDir())
}

// defaultGlobalConfigDir returns the default global config directory.
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
