package acp_test

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	acpinfra "github.com/runoshun/git-crew/v2/internal/infra/acp"
	"github.com/stretchr/testify/require"
)

func TestFileStateStore_SaveLoad(t *testing.T) {
	t.Parallel()

	store := acpinfra.NewFileStateStore(t.TempDir())
	state := domain.ACPExecutionState{ExecutionSubstate: domain.ACPExecutionAwaitingPermission}

	require.NoError(t, store.Save(context.Background(), "default", 1, state))
	got, err := store.Load(context.Background(), "default", 1)
	require.NoError(t, err)
	require.Equal(t, state.ExecutionSubstate, got.ExecutionSubstate)
}

func TestFileStateStore_LoadNotFound(t *testing.T) {
	t.Parallel()

	store := acpinfra.NewFileStateStore(t.TempDir())
	_, err := store.Load(context.Background(), "default", 1)
	require.ErrorIs(t, err, domain.ErrACPStateNotFound)
}

func TestFileStateStore_SaveInvalidSubstate(t *testing.T) {
	t.Parallel()

	store := acpinfra.NewFileStateStore(t.TempDir())
	err := store.Save(context.Background(), "default", 1, domain.ACPExecutionState{ExecutionSubstate: domain.ACPExecutionSubstate("bad")})
	require.ErrorIs(t, err, domain.ErrInvalidACPExecutionSubstate)
}
