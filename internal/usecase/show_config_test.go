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
	t.Run("returns both config infos", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.RepoConfigInfo = domain.ConfigInfo{
			Path:    "/test/.git/crew/config.toml",
			Content: "default_agent = \"claude\"",
			Exists:  true,
		}
		manager.GlobalConfigInfo = domain.ConfigInfo{
			Path:    "/home/test/.config/git-crew/config.toml",
			Content: "[log]\nlevel = \"debug\"",
			Exists:  true,
		}

		uc := usecase.NewShowConfig(manager)
		out, err := uc.Execute(context.Background(), usecase.ShowConfigInput{})

		require.NoError(t, err)
		assert.Equal(t, "/test/.git/crew/config.toml", out.RepoConfig.Path)
		assert.Equal(t, "default_agent = \"claude\"", out.RepoConfig.Content)
		assert.True(t, out.RepoConfig.Exists)
		assert.Equal(t, "/home/test/.config/git-crew/config.toml", out.GlobalConfig.Path)
		assert.Equal(t, "[log]\nlevel = \"debug\"", out.GlobalConfig.Content)
		assert.True(t, out.GlobalConfig.Exists)
	})

	t.Run("handles non-existent files", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.RepoConfigInfo = domain.ConfigInfo{
			Path:   "/test/.git/crew/config.toml",
			Exists: false,
		}
		manager.GlobalConfigInfo = domain.ConfigInfo{
			Path:   "/home/test/.config/git-crew/config.toml",
			Exists: false,
		}

		uc := usecase.NewShowConfig(manager)
		out, err := uc.Execute(context.Background(), usecase.ShowConfigInput{})

		require.NoError(t, err)
		assert.False(t, out.RepoConfig.Exists)
		assert.False(t, out.GlobalConfig.Exists)
		assert.Empty(t, out.RepoConfig.Content)
		assert.Empty(t, out.GlobalConfig.Content)
	})
}
