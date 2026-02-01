package usecase

import (
	"context"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/require"
)

type acpRunStateStore struct {
	namespace string
	taskID    int
	calls     []domain.ACPExecutionSubstate
}

func (s *acpRunStateStore) Load(context.Context, string, int) (domain.ACPExecutionState, error) {
	return domain.ACPExecutionState{}, domain.ErrACPStateNotFound
}

func (s *acpRunStateStore) Save(_ context.Context, namespace string, taskID int, state domain.ACPExecutionState) error {
	s.namespace = namespace
	s.taskID = taskID
	s.calls = append(s.calls, state.ExecutionSubstate)
	return nil
}

func TestACPRunClientRequestPermissionSelected(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	permissionCh := make(chan domain.ACPCommand, 1)
	stopCh := make(chan struct{})
	stateStore := &acpRunStateStore{}

	client := &acpRunClient{
		permissionCh: permissionCh,
		stopCh:       stopCh,
		stateStore:   stateStore,
		stateNS:      "default",
		taskID:       1,
	}

	params := permissionRequestParams()
	permissionCh <- domain.ACPCommand{Type: domain.ACPCommandPermission, OptionID: "opt-1"}

	resp, err := client.RequestPermission(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Selected)
	require.Equal(t, "opt-1", string(resp.Outcome.Selected.OptionId))
	require.Equal(t, "selected", resp.Outcome.Selected.Outcome)
	require.Equal(t, "default", stateStore.namespace)
	require.Equal(t, 1, stateStore.taskID)
	require.Len(t, stateStore.calls, 2)
	require.Equal(t, domain.ACPExecutionAwaitingPermission, stateStore.calls[0])
	require.Equal(t, domain.ACPExecutionRunning, stateStore.calls[1])
}

func TestACPRunClientRequestPermissionStop(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	permissionCh := make(chan domain.ACPCommand, 1)
	stopCh := make(chan struct{})
	close(stopCh)

	client := &acpRunClient{
		permissionCh: permissionCh,
		stopCh:       stopCh,
	}

	resp, err := client.RequestPermission(ctx, permissionRequestParams())
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Cancelled)
	require.Equal(t, "cancelled", resp.Outcome.Cancelled.Outcome)
}

func permissionRequestParams() acpsdk.RequestPermissionRequest {
	return acpsdk.RequestPermissionRequest{
		SessionId: acpsdk.SessionId("session-1"),
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: acpsdk.ToolCallId("tool-1"),
		},
		Options: []acpsdk.PermissionOption{
			{
				OptionId: acpsdk.PermissionOptionId("opt-1"),
				Name:     "Allow once",
				Kind:     acpsdk.PermissionOptionKind("allow_once"),
			},
		},
	}
}
