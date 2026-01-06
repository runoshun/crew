// Package config provides configuration loading functionality.
package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Ensure Manager implements domain.ConfigManager.
var _ domain.ConfigManager = (*Manager)(nil)

// Manager manages configuration files.
type Manager struct {
	crewDir       string // Path to .git/crew directory
	repoRoot      string // Path to repository root
	globalConfDir string // Path to global config directory (e.g., ~/.config/git-crew)
}

// NewManager creates a new Manager.
func NewManager(crewDir, repoRoot string) *Manager {
	return &Manager{
		crewDir:       crewDir,
		repoRoot:      repoRoot,
		globalConfDir: defaultGlobalConfigDir(),
	}
}

// NewManagerWithGlobalDir creates a new Manager with a custom global config directory.
// This is useful for testing.
func NewManagerWithGlobalDir(crewDir, repoRoot, globalConfDir string) *Manager {
	return &Manager{
		crewDir:       crewDir,
		repoRoot:      repoRoot,
		globalConfDir: globalConfDir,
	}
}

// GetRepoConfigInfo returns information about the repository config file.
func (m *Manager) GetRepoConfigInfo() domain.ConfigInfo {
	path := filepath.Join(m.crewDir, domain.ConfigFileName)
	return m.getConfigInfo(path)
}

// GetGlobalConfigInfo returns information about the global config file.
func (m *Manager) GetGlobalConfigInfo() domain.ConfigInfo {
	if m.globalConfDir == "" {
		return domain.ConfigInfo{
			Path:   "",
			Exists: false,
		}
	}
	path := filepath.Join(m.globalConfDir, domain.ConfigFileName)
	return m.getConfigInfo(path)
}

// GetRootRepoConfigInfo returns information about the root repository config file (.crew.toml).
func (m *Manager) GetRootRepoConfigInfo() domain.ConfigInfo {
	if m.repoRoot == "" {
		return domain.ConfigInfo{
			Path:   "",
			Exists: false,
		}
	}
	path := domain.RepoRootConfigPath(m.repoRoot)
	return m.getConfigInfo(path)
}

// getConfigInfo reads a config file and returns its info.
func (m *Manager) getConfigInfo(path string) domain.ConfigInfo {
	content, err := os.ReadFile(path)
	if err != nil {
		return domain.ConfigInfo{
			Path:   path,
			Exists: false,
		}
	}
	return domain.ConfigInfo{
		Path:    path,
		Content: string(content),
		Exists:  true,
	}
}

// InitRepoConfig creates a repository config file with default template.
// The cfg parameter should have builtin agents registered (via builtin.Register).
func (m *Manager) InitRepoConfig(cfg *domain.Config) error {
	path := filepath.Join(m.crewDir, domain.ConfigFileName)
	return m.initConfig(path, cfg)
}

// InitGlobalConfig creates a global config file with default template.
// The cfg parameter should have builtin agents registered (via builtin.Register).
func (m *Manager) InitGlobalConfig(cfg *domain.Config) error {
	if m.globalConfDir == "" {
		return errors.New("global config directory not available")
	}
	path := filepath.Join(m.globalConfDir, domain.ConfigFileName)

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(m.globalConfDir, 0700); err != nil {
		return err
	}

	return m.initConfig(path, cfg)
}

// initConfig creates a config file with default template.
func (m *Manager) initConfig(path string, cfg *domain.Config) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return domain.ErrConfigExists
	}

	// Render template dynamically from the registered config
	content := domain.RenderConfigTemplate(cfg)

	return os.WriteFile(path, []byte(content), 0600)
}
