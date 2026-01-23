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

		manager := NewManagerWithGlobalDir(crewDir, "", "")
		info := manager.GetRepoConfigInfo()

		assert.Equal(t, filepath.Join(crewDir, domain.ConfigFileName), info.Path)
		assert.Equal(t, configContent, info.Content)
		assert.True(t, info.Exists)
	})

	t.Run("returns info when file does not exist", func(t *testing.T) {
		crewDir := t.TempDir()

		manager := NewManagerWithGlobalDir(crewDir, "", "")
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

		manager := NewManagerWithGlobalDir("", "", globalDir)
		info := manager.GetGlobalConfigInfo()

		assert.Equal(t, filepath.Join(globalDir, domain.ConfigFileName), info.Path)
		assert.Equal(t, configContent, info.Content)
		assert.True(t, info.Exists)
	})

	t.Run("returns info when file does not exist", func(t *testing.T) {
		globalDir := t.TempDir()

		manager := NewManagerWithGlobalDir("", "", globalDir)
		info := manager.GetGlobalConfigInfo()

		assert.Equal(t, filepath.Join(globalDir, domain.ConfigFileName), info.Path)
		assert.Empty(t, info.Content)
		assert.False(t, info.Exists)
	})

	t.Run("returns empty info when global dir is empty", func(t *testing.T) {
		manager := NewManagerWithGlobalDir("", "", "")
		info := manager.GetGlobalConfigInfo()

		assert.Empty(t, info.Path)
		assert.Empty(t, info.Content)
		assert.False(t, info.Exists)
	})
}

func TestManager_GetRootRepoConfigInfo(t *testing.T) {
	t.Run("returns info when file exists", func(t *testing.T) {
		repoRoot := t.TempDir()
		configContent := "[agents]\nworker_default = \"sonnet\""
		err := os.WriteFile(filepath.Join(repoRoot, domain.RootConfigFileName), []byte(configContent), 0644)
		require.NoError(t, err)

		manager := NewManagerWithGlobalDir("", repoRoot, "")
		info := manager.GetRootRepoConfigInfo()

		assert.Equal(t, filepath.Join(repoRoot, domain.RootConfigFileName), info.Path)
		assert.Equal(t, configContent, info.Content)
		assert.True(t, info.Exists)
	})

	t.Run("returns info when file does not exist", func(t *testing.T) {
		repoRoot := t.TempDir()

		manager := NewManagerWithGlobalDir("", repoRoot, "")
		info := manager.GetRootRepoConfigInfo()

		assert.Equal(t, filepath.Join(repoRoot, domain.RootConfigFileName), info.Path)
		assert.Empty(t, info.Content)
		assert.False(t, info.Exists)
	})

	t.Run("returns empty info when repo root is empty", func(t *testing.T) {
		manager := NewManagerWithGlobalDir("", "", "")
		info := manager.GetRootRepoConfigInfo()

		assert.Empty(t, info.Path)
		assert.Empty(t, info.Content)
		assert.False(t, info.Exists)
	})
}

func TestManager_InitRepoConfig(t *testing.T) {
	t.Run("creates config file", func(t *testing.T) {
		crewDir := t.TempDir()
		cfg := domain.NewDefaultConfig()
		cfg.Agents["test-worker"] = domain.Agent{
			Role:        domain.RoleWorker,
			Description: "Test worker",
		}

		manager := NewManagerWithGlobalDir(crewDir, "", "")
		err := manager.InitRepoConfig(cfg)

		require.NoError(t, err)

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigFileName))
		require.NoError(t, err)
		assert.Contains(t, string(content), "git-crew configuration")
		assert.Contains(t, string(content), "[agents]")
		assert.Contains(t, string(content), "worker_default = ")
		// Verify dynamic content from registered agents
		assert.Contains(t, string(content), "[agents.test-worker]")
	})

	t.Run("returns error if file already exists", func(t *testing.T) {
		crewDir := t.TempDir()
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte("existing"), 0644)
		require.NoError(t, err)
		cfg := domain.NewDefaultConfig()

		manager := NewManagerWithGlobalDir(crewDir, "", "")
		err = manager.InitRepoConfig(cfg)

		assert.Error(t, err)
	})
}

func TestManager_InitGlobalConfig(t *testing.T) {
	t.Run("creates config file and parent directory", func(t *testing.T) {
		tempDir := t.TempDir()
		globalDir := filepath.Join(tempDir, "git-crew") // This doesn't exist yet
		cfg := domain.NewDefaultConfig()

		manager := NewManagerWithGlobalDir("", "", globalDir)
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

		manager := NewManagerWithGlobalDir("", "", globalDir)
		err = manager.InitGlobalConfig(cfg)

		assert.Error(t, err)
	})

	t.Run("returns error if global dir is empty", func(t *testing.T) {
		cfg := domain.NewDefaultConfig()
		manager := NewManagerWithGlobalDir("", "", "")
		err := manager.InitGlobalConfig(cfg)

		assert.Error(t, err)
	})
}

func TestManager_SetReviewMode(t *testing.T) {
	t.Run("creates runtime config file if not exists and sets review_mode to auto", func(t *testing.T) {
		crewDir := t.TempDir()
		manager := NewManagerWithGlobalDir(crewDir, "", "")

		err := manager.SetReviewMode(domain.ReviewModeAuto)
		require.NoError(t, err)

		// Verify file was created with correct content
		content, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName))
		require.NoError(t, err)
		assert.Contains(t, string(content), "[complete]")
		assert.Contains(t, string(content), "review_mode")
		assert.Contains(t, string(content), "auto")
	})

	t.Run("creates runtime config file if not exists and sets review_mode to manual", func(t *testing.T) {
		crewDir := t.TempDir()
		manager := NewManagerWithGlobalDir(crewDir, "", "")

		err := manager.SetReviewMode(domain.ReviewModeManual)
		require.NoError(t, err)

		// Verify file was created with correct content
		content, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName))
		require.NoError(t, err)
		assert.Contains(t, string(content), "[complete]")
		assert.Contains(t, string(content), "review_mode")
		assert.Contains(t, string(content), "manual")
	})

	t.Run("creates runtime config file if not exists and sets review_mode to auto_fix", func(t *testing.T) {
		crewDir := t.TempDir()
		manager := NewManagerWithGlobalDir(crewDir, "", "")

		err := manager.SetReviewMode(domain.ReviewModeAutoFix)
		require.NoError(t, err)

		// Verify file was created with correct content
		content, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName))
		require.NoError(t, err)
		assert.Contains(t, string(content), "[complete]")
		assert.Contains(t, string(content), "review_mode")
		assert.Contains(t, string(content), "auto_fix")
	})

	t.Run("returns error for invalid review_mode", func(t *testing.T) {
		crewDir := t.TempDir()
		manager := NewManagerWithGlobalDir(crewDir, "", "")

		err := manager.SetReviewMode(domain.ReviewMode("invalid"))

		assert.ErrorIs(t, err, domain.ErrInvalidReviewMode)
	})

	t.Run("updates existing runtime config without destroying other settings", func(t *testing.T) {
		crewDir := t.TempDir()
		existingContent := `[agents]
worker_default = "claude"

[log]
level = "debug"
`
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName), []byte(existingContent), 0644)
		require.NoError(t, err)

		manager := NewManagerWithGlobalDir(crewDir, "", "")
		err = manager.SetReviewMode(domain.ReviewModeAutoFix)
		require.NoError(t, err)

		// Verify content preserved and review_mode added
		content, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName))
		require.NoError(t, err)
		contentStr := string(content)
		assert.Contains(t, contentStr, "worker_default")
		assert.Contains(t, contentStr, "level")
		assert.Contains(t, contentStr, "[complete]")
		assert.Contains(t, contentStr, "auto_fix")
	})

	t.Run("updates existing complete section", func(t *testing.T) {
		crewDir := t.TempDir()
		existingContent := `[complete]
command = "mise run ci"
review_mode = "auto"
`
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName), []byte(existingContent), 0644)
		require.NoError(t, err)

		manager := NewManagerWithGlobalDir(crewDir, "", "")
		err = manager.SetReviewMode(domain.ReviewModeManual)
		require.NoError(t, err)

		// Verify command preserved and review_mode updated
		content, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName))
		require.NoError(t, err)
		contentStr := string(content)
		assert.Contains(t, contentStr, "mise run ci")
		assert.Contains(t, contentStr, "manual")
	})

	t.Run("cycles review_mode from auto_fix to auto", func(t *testing.T) {
		crewDir := t.TempDir()
		existingContent := `[complete]
review_mode = "auto_fix"
`
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName), []byte(existingContent), 0644)
		require.NoError(t, err)

		manager := NewManagerWithGlobalDir(crewDir, "", "")
		err = manager.SetReviewMode(domain.ReviewModeAuto)
		require.NoError(t, err)

		// Verify review_mode changed to auto
		content, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName))
		require.NoError(t, err)
		contentStr := string(content)
		assert.Contains(t, contentStr, "review_mode")
		assert.Contains(t, contentStr, "auto")
		assert.NotContains(t, contentStr, "auto_fix") // Should be replaced, not appended
	})

	t.Run("handles empty runtime config file without panic", func(t *testing.T) {
		crewDir := t.TempDir()
		// Create empty file
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName), []byte(""), 0644)
		require.NoError(t, err)

		manager := NewManagerWithGlobalDir(crewDir, "", "")
		err = manager.SetReviewMode(domain.ReviewModeAutoFix)
		require.NoError(t, err)

		// Verify review_mode was set
		content, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName))
		require.NoError(t, err)
		assert.Contains(t, string(content), "[complete]")
		assert.Contains(t, string(content), "auto_fix")
	})

	t.Run("handles comment-only runtime config file without panic", func(t *testing.T) {
		crewDir := t.TempDir()
		// Create file with only comments
		commentOnlyContent := `# This is a comment
# Another comment
`
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName), []byte(commentOnlyContent), 0644)
		require.NoError(t, err)

		manager := NewManagerWithGlobalDir(crewDir, "", "")
		err = manager.SetReviewMode(domain.ReviewModeManual)
		require.NoError(t, err)

		// Verify review_mode was set
		content, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName))
		require.NoError(t, err)
		assert.Contains(t, string(content), "[complete]")
		assert.Contains(t, string(content), "manual")
	})

	t.Run("does not modify base config.toml", func(t *testing.T) {
		crewDir := t.TempDir()
		baseContent := `[log]
level = "debug"
`
		err := os.WriteFile(filepath.Join(crewDir, domain.ConfigFileName), []byte(baseContent), 0644)
		require.NoError(t, err)

		manager := NewManagerWithGlobalDir(crewDir, "", "")
		err = manager.SetReviewMode(domain.ReviewModeAutoFix)
		require.NoError(t, err)

		// Verify base config.toml was not modified
		content, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigFileName))
		require.NoError(t, err)
		assert.Equal(t, baseContent, string(content))

		// Verify runtime config has the review_mode setting
		runtimeContent, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName))
		require.NoError(t, err)
		assert.Contains(t, string(runtimeContent), "auto_fix")
	})

	t.Run("creates crew directory if it does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		crewDir := filepath.Join(tempDir, "nonexistent", "crew")
		manager := NewManagerWithGlobalDir(crewDir, "", "")

		err := manager.SetReviewMode(domain.ReviewModeAuto)
		require.NoError(t, err)

		// Verify directory was created
		info, err := os.Stat(crewDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Verify runtime config has the review_mode setting
		runtimeContent, err := os.ReadFile(filepath.Join(crewDir, domain.ConfigRuntimeFileName))
		require.NoError(t, err)
		assert.Contains(t, string(runtimeContent), "auto")
	})
}
