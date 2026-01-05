package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_GetRepoConfigInfo(t *testing.T) {
	t.Run("returns info when file exists", func(t *testing.T) {
		crewDir := t.TempDir()
		configContent := "default_agent = \"claude\""
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(configContent), 0644)
		require.NoError(t, err)

		manager := NewManagerWithGlobalDir(crewDir, "")
		info := manager.GetRepoConfigInfo()

		assert.Equal(t, filepath.Join(crewDir, domain.ConfigFileName), info.Path)
		assert.Equal(t, configContent, info.Content)
		assert.True(t, info.Exists)
	})

	t.Run("returns info when file does not exist", func(t *testing.T) {
		crewDir := t.TempDir()

		manager := NewManagerWithGlobalDir(crewDir, "")
		info := manager.GetRepoConfigInfo()

		assert.Equal(t, filepath.Join(crewDir, domain.ConfigFileName), info.Path)
		assert.Empty(t, info.Content)
		assert.False(t, info.Exists)
	})
}

func TestManager_GetGlobalConfigInfo(t *testing.T) {
	t.Run("returns info when file exists", func(t *testing.T) {
		globalDir := t.TempDir()
		configContent := "[log]\nlevel = \"debug\""
		err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte(configContent), 0644)
		require.NoError(t, err)

		manager := NewManagerWithGlobalDir("", globalDir)
		info := manager.GetGlobalConfigInfo()

		assert.Equal(t, filepath.Join(globalDir, domain.ConfigFileName), info.Path)
		assert.Equal(t, configContent, info.Content)
		assert.True(t, info.Exists)
	})

	t.Run("returns info when file does not exist", func(t *testing.T) {
		globalDir := t.TempDir()

		manager := NewManagerWithGlobalDir("", globalDir)
		info := manager.GetGlobalConfigInfo()

		assert.Equal(t, filepath.Join(globalDir, domain.ConfigFileName), info.Path)
		assert.Empty(t, info.Content)
		assert.False(t, info.Exists)
	})

	t.Run("returns empty info when global dir is empty", func(t *testing.T) {
		manager := NewManagerWithGlobalDir("", "")
		info := manager.GetGlobalConfigInfo()

		assert.Empty(t, info.Path)
		assert.Empty(t, info.Content)
		assert.False(t, info.Exists)
	})
}

func TestManager_InitRepoConfig(t *testing.T) {
	t.Run("creates config file", func(t *testing.T) {
		crewDir := t.TempDir()
		cfg := domain.NewDefaultConfig()
		cfg.Workers["test-worker"] = domain.Worker{
			Agent:       "test-agent",
			Description: "Test worker",
		}

		manager := NewManagerWithGlobalDir(crewDir, "")
		err := manager.InitRepoConfig(cfg)

		require.NoError(t, err)

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigFileName))
		require.NoError(t, err)
		assert.Contains(t, string(content), "git-crew configuration")
		assert.Contains(t, string(content), "[workers]")
		assert.Contains(t, string(content), "default = ")
		// Verify dynamic content from registered workers
		assert.Contains(t, string(content), "[workers.test-worker]")
	})

	t.Run("returns error if file already exists", func(t *testing.T) {
		crewDir := t.TempDir()
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte("existing"), 0644)
		require.NoError(t, err)
		cfg := domain.NewDefaultConfig()

		manager := NewManagerWithGlobalDir(crewDir, "")
		err = manager.InitRepoConfig(cfg)

		assert.Error(t, err)
	})
}

func TestManager_InitGlobalConfig(t *testing.T) {
	t.Run("creates config file and parent directory", func(t *testing.T) {
		tempDir := t.TempDir()
		globalDir := filepath.Join(tempDir, "git-crew") // This doesn't exist yet
		cfg := domain.NewDefaultConfig()

		manager := NewManagerWithGlobalDir("", globalDir)
		err := manager.InitGlobalConfig(cfg)

		require.NoError(t, err)

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(globalDir, domain.ConfigFileName))
		require.NoError(t, err)
		assert.Contains(t, string(content), "git-crew configuration")
	})

	t.Run("returns error if file already exists", func(t *testing.T) {
		globalDir := t.TempDir()
		err := os.WriteFile(filepath.Join(globalDir, domain.ConfigFileName), []byte("existing"), 0644)
		require.NoError(t, err)
		cfg := domain.NewDefaultConfig()

		manager := NewManagerWithGlobalDir("", globalDir)
		err = manager.InitGlobalConfig(cfg)

		assert.Error(t, err)
	})

	t.Run("returns error if global dir is empty", func(t *testing.T) {
		cfg := domain.NewDefaultConfig()
		manager := NewManagerWithGlobalDir("", "")
		err := manager.InitGlobalConfig(cfg)

		assert.Error(t, err)
	})
}
