package usecase

import (
	"context"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestACPRunClientRequestPermissionSelected(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	permissionCh := make(chan domain.ACPCommand, 1)
	stopCh := make(chan struct{})

	client := &acpRunClient{
		permissionCh: permissionCh,
		stopCh:       stopCh,
	}

	params := permissionRequestParams()
	permissionCh <- domain.ACPCommand{Type: domain.ACPCommandPermission, OptionID: "opt-1"}

	resp, err := client.RequestPermission(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Selected)
	require.Equal(t, "opt-1", string(resp.Outcome.Selected.OptionId))
	require.Equal(t, "selected", resp.Outcome.Selected.Outcome)
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
