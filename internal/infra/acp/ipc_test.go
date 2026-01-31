package acp_test

import (
	"context"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	acpinfra "github.com/runoshun/git-crew/v2/internal/infra/acp"
	"github.com/stretchr/testify/require"
)

func TestFileIPC_SendNext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	factory := acpinfra.NewFileIPCFactory(t.TempDir(), nil)
	ipc := factory.ForTask("default", 1)

	cmd1 := domain.ACPCommand{ID: "0001", Type: domain.ACPCommandPrompt, Text: "hello"}
	cmd2 := domain.ACPCommand{ID: "0002", Type: domain.ACPCommandStop}

	require.NoError(t, ipc.Send(ctx, cmd1))
	require.NoError(t, ipc.Send(ctx, cmd2))

	got1, err := ipc.Next(ctx)
	require.NoError(t, err)
	require.Equal(t, cmd1.ID, got1.ID)
	require.Equal(t, cmd1.Type, got1.Type)
	require.Equal(t, cmd1.Text, got1.Text)

	got2, err := ipc.Next(ctx)
	require.NoError(t, err)
	require.Equal(t, cmd2.ID, got2.ID)
	require.Equal(t, cmd2.Type, got2.Type)
}
