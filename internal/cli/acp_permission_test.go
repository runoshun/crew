package cli

import (
	"encoding/json"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePermissionOptionID_Index(t *testing.T) {
	req := acpsdk.RequestPermissionRequest{
		Options: []acpsdk.PermissionOption{
			{OptionId: acpsdk.PermissionOptionId("opt-1"), Name: "Allow"},
			{OptionId: acpsdk.PermissionOptionId("opt-2"), Name: "Deny"},
		},
	}
	payload, err := json.Marshal(req)
	require.NoError(t, err)

	events := []domain.ACPEvent{
		{Type: domain.ACPEventRequestPermission, Payload: payload},
	}

	optionID, err := resolvePermissionOptionID("2", events)
	require.NoError(t, err)
	assert.Equal(t, "opt-2", optionID)
}

func TestResolvePermissionOptionID_NonNumeric(t *testing.T) {
	optionID, err := resolvePermissionOptionID("opt-1", nil)
	require.NoError(t, err)
	assert.Equal(t, "opt-1", optionID)
}

func TestResolvePermissionOptionID_OutOfRange(t *testing.T) {
	req := acpsdk.RequestPermissionRequest{
		Options: []acpsdk.PermissionOption{
			{OptionId: acpsdk.PermissionOptionId("opt-1"), Name: "Allow"},
		},
	}
	payload, err := json.Marshal(req)
	require.NoError(t, err)

	events := []domain.ACPEvent{
		{Type: domain.ACPEventRequestPermission, Payload: payload},
	}

	_, err = resolvePermissionOptionID("2", events)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestResolvePermissionOptionID_NoEvents(t *testing.T) {
	_, err := resolvePermissionOptionID("1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no permission requests")
}

func TestResolvePermissionOptionID_SkipsInvalidPayload(t *testing.T) {
	validReq := acpsdk.RequestPermissionRequest{
		Options: []acpsdk.PermissionOption{
			{OptionId: acpsdk.PermissionOptionId("opt-1"), Name: "Allow"},
		},
	}
	validPayload, err := json.Marshal(validReq)
	require.NoError(t, err)

	events := []domain.ACPEvent{
		{Type: domain.ACPEventRequestPermission, Payload: json.RawMessage("{invalid")},
		{Type: domain.ACPEventRequestPermission, Payload: validPayload},
	}

	optionID, err := resolvePermissionOptionID("1", events)
	require.NoError(t, err)
	assert.Equal(t, "opt-1", optionID)
}
