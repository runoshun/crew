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

func TestInitConfig_Execute(t *testing.T) {
	t.Run("creates repo config", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.RepoConfigInfo = domain.ConfigInfo{
			Path:   "/test/.git/crew/config.toml",
			Exists: false,
		}

		uc := usecase.NewInitConfig(manager)
		out, err := uc.Execute(context.Background(), usecase.InitConfigInput{
			Global: false,
		})

		require.NoError(t, err)
		assert.Equal(t, "/test/.git/crew/config.toml", out.Path)
		assert.True(t, manager.InitRepoCalled)
		assert.False(t, manager.InitGlobalCalled)
	})

	t.Run("creates global config", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.GlobalConfigInfo = domain.ConfigInfo{
			Path:   "/home/test/.config/git-crew/config.toml",
			Exists: false,
		}

		uc := usecase.NewInitConfig(manager)
		out, err := uc.Execute(context.Background(), usecase.InitConfigInput{
			Global: true,
		})

		require.NoError(t, err)
		assert.Equal(t, "/home/test/.config/git-crew/config.toml", out.Path)
		assert.False(t, manager.InitRepoCalled)
		assert.True(t, manager.InitGlobalCalled)
	})

	t.Run("returns error when repo config already exists", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.RepoConfigInfo = domain.ConfigInfo{
			Path:   "/test/.git/crew/config.toml",
			Exists: true,
		}
		manager.InitRepoErr = domain.ErrConfigExists

		uc := usecase.NewInitConfig(manager)
		_, err := uc.Execute(context.Background(), usecase.InitConfigInput{
			Global: false,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrConfigExists)
	})

	t.Run("returns error when global config already exists", func(t *testing.T) {
		manager := testutil.NewMockConfigManager()
		manager.GlobalConfigInfo = domain.ConfigInfo{
			Path:   "/home/test/.config/git-crew/config.toml",
			Exists: true,
		}
		manager.InitGlobalErr = domain.ErrConfigExists

		uc := usecase.NewInitConfig(manager)
		_, err := uc.Execute(context.Background(), usecase.InitConfigInput{
			Global: true,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrConfigExists)
	})
}
