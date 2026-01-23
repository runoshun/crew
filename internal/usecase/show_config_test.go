package usecase_test

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowConfig_Execute(t *testing.T) {
	t.Run("returns both config infos and effective config", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.RepoConfigInfo = domain.ConfigInfo{
			Path:    "/test/.git/crew/config.toml",
			Content: "default_agent = \"claude\"",
			Exists:  true,
		}
		manager.GlobalConfigInfo = domain.ConfigInfo{
			Path:    "/home/test/.config/crew/config.toml",
			Content: "[log]\nlevel = \"debug\"",
			Exists:  true,
		}
		manager.RootRepoConfigInfo = domain.ConfigInfo{
			Path:    "/test/.crew.toml",
			Content: "[agents]\nworker_default = \"claude\"",
			Exists:  true,
		}

		loader := testutil.NewMockConfigLoader()
		loader.Config = &domain.Config{
			AgentsConfig: domain.AgentsConfig{},
			Agents: map[string]domain.Agent{
				"claude": {Role: domain.RoleWorker},
			},
		}

		uc := usecase.NewShowConfig(manager, loader)
		out, err := uc.Execute(context.Background(), usecase.ShowConfigInput{})

		require.NoError(t, err)
		assert.Equal(t, "/test/.git/crew/config.toml", out.RepoConfig.Path)
		assert.Equal(t, "default_agent = \"claude\"", out.RepoConfig.Content)
		assert.True(t, out.RepoConfig.Exists)
		assert.Equal(t, "/home/test/.config/crew/config.toml", out.GlobalConfig.Path)
		assert.Equal(t, "[log]\nlevel = \"debug\"", out.GlobalConfig.Content)
		assert.True(t, out.GlobalConfig.Exists)
		assert.Equal(t, "/test/.crew.toml", out.RootRepoConfig.Path)
		assert.Equal(t, "[agents]\nworker_default = \"claude\"", out.RootRepoConfig.Content)
		assert.True(t, out.RootRepoConfig.Exists)
		assert.NotNil(t, out.EffectiveConfig)
	})

	t.Run("handles non-existent files", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.RepoConfigInfo = domain.ConfigInfo{
			Path:   "/test/.git/crew/config.toml",
			Exists: false,
		}
		manager.GlobalConfigInfo = domain.ConfigInfo{
			Path:   "/home/test/.config/crew/config.toml",
			Exists: false,
		}

		loader := testutil.NewMockConfigLoader()
		loader.Config = domain.NewDefaultConfig()

		uc := usecase.NewShowConfig(manager, loader)
		out, err := uc.Execute(context.Background(), usecase.ShowConfigInput{})

		require.NoError(t, err)
		assert.False(t, out.RepoConfig.Exists)
		assert.False(t, out.GlobalConfig.Exists)
		assert.False(t, out.RootRepoConfig.Exists)
		assert.Empty(t, out.RepoConfig.Content)
		assert.Empty(t, out.GlobalConfig.Content)
		assert.NotNil(t, out.EffectiveConfig)
	})

	t.Run("ignores global config when flag is set", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.RepoConfigInfo = domain.ConfigInfo{
			Path:   "/test/.git/crew/config.toml",
			Exists: true,
		}
		manager.GlobalConfigInfo = domain.ConfigInfo{
			Path:   "/home/test/.config/crew/config.toml",
			Exists: true,
		}

		loader := testutil.NewMockConfigLoader()
		loader.Config = domain.NewDefaultConfig()

		uc := usecase.NewShowConfig(manager, loader)
		out, err := uc.Execute(context.Background(), usecase.ShowConfigInput{
			IgnoreGlobal: true,
		})

		require.NoError(t, err)
		// GlobalConfig should be empty when ignored
		assert.Empty(t, out.GlobalConfig.Path)
		assert.False(t, out.GlobalConfig.Exists)
		// RepoConfig should still be present
		assert.Equal(t, "/test/.git/crew/config.toml", out.RepoConfig.Path)
		// RootRepoConfig should still be present
		assert.Equal(t, "/test/.crew.toml", out.RootRepoConfig.Path)
	})

	t.Run("ignores root repo config when flag is set", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.RepoConfigInfo = domain.ConfigInfo{
			Path:   "/test/.git/crew/config.toml",
			Exists: true,
		}
		manager.RootRepoConfigInfo = domain.ConfigInfo{
			Path:   "/test/.crew.toml",
			Exists: true,
		}

		loader := testutil.NewMockConfigLoader()
		loader.Config = domain.NewDefaultConfig()

		uc := usecase.NewShowConfig(manager, loader)
		out, err := uc.Execute(context.Background(), usecase.ShowConfigInput{
			IgnoreRootRepo: true,
		})

		require.NoError(t, err)
		// RootRepoConfig should be empty when ignored
		assert.Empty(t, out.RootRepoConfig.Path)
		assert.False(t, out.RootRepoConfig.Exists)
		// RepoConfig should still be present
		assert.Equal(t, "/test/.git/crew/config.toml", out.RepoConfig.Path)
	})

	t.Run("ignores repo config when flag is set", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.RepoConfigInfo = domain.ConfigInfo{
			Path:   "/test/.git/crew/config.toml",
			Exists: true,
		}
		manager.GlobalConfigInfo = domain.ConfigInfo{
			Path:   "/home/test/.config/crew/config.toml",
			Exists: true,
		}

		loader := testutil.NewMockConfigLoader()
		loader.Config = domain.NewDefaultConfig()

		uc := usecase.NewShowConfig(manager, loader)
		out, err := uc.Execute(context.Background(), usecase.ShowConfigInput{
			IgnoreRepo: true,
		})

		require.NoError(t, err)
		// RepoConfig should be empty when ignored
		assert.Empty(t, out.RepoConfig.Path)
		assert.False(t, out.RepoConfig.Exists)
		// GlobalConfig should still be present
		assert.Equal(t, "/home/test/.config/crew/config.toml", out.GlobalConfig.Path)
		// RootRepoConfig should still be present
		assert.Equal(t, "/test/.crew.toml", out.RootRepoConfig.Path)
	})

	t.Run("returns runtime config info", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.RuntimeConfigInfo = domain.ConfigInfo{
			Path:    "/test/.git/crew/config.runtime.toml",
			Content: "[complete]\nauto_fix = true",
			Exists:  true,
		}

		loader := testutil.NewMockConfigLoader()
		loader.Config = domain.NewDefaultConfig()

		uc := usecase.NewShowConfig(manager, loader)
		out, err := uc.Execute(context.Background(), usecase.ShowConfigInput{})

		require.NoError(t, err)
		assert.Equal(t, "/test/.git/crew/config.runtime.toml", out.RuntimeConfig.Path)
		assert.Equal(t, "[complete]\nauto_fix = true", out.RuntimeConfig.Content)
		assert.True(t, out.RuntimeConfig.Exists)
	})

	t.Run("ignores runtime config when flag is set", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.RuntimeConfigInfo = domain.ConfigInfo{
			Path:   "/test/.git/crew/config.runtime.toml",
			Exists: true,
		}
		manager.RepoConfigInfo = domain.ConfigInfo{
			Path:   "/test/.git/crew/config.toml",
			Exists: true,
		}

		loader := testutil.NewMockConfigLoader()
		loader.Config = domain.NewDefaultConfig()

		uc := usecase.NewShowConfig(manager, loader)
		out, err := uc.Execute(context.Background(), usecase.ShowConfigInput{
			IgnoreRuntime: true,
		})

		require.NoError(t, err)
		// RuntimeConfig should be empty when ignored
		assert.Empty(t, out.RuntimeConfig.Path)
		assert.False(t, out.RuntimeConfig.Exists)
		// RepoConfig should still be present
		assert.Equal(t, "/test/.git/crew/config.toml", out.RepoConfig.Path)
		// Verify that IgnoreRuntime was passed to loader
		assert.True(t, loader.LastOpts.IgnoreRuntime)
	})
}
