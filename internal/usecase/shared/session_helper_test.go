package shared_test

import (
	"errors"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckSessionRunning_Success(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true

	running, err := shared.CheckSessionRunning(sessions, 1)

	require.NoError(t, err)
	assert.True(t, running)
}

func TestCheckSessionRunning_NotRunning(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false

	running, err := shared.CheckSessionRunning(sessions, 1)

	require.NoError(t, err)
	assert.False(t, running)
}

func TestCheckSessionRunning_Error(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningErr = assert.AnError

	_, err := shared.CheckSessionRunning(sessions, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check session running")
}

func TestStopSession_Running(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true

	sessionName, err := shared.StopSession(sessions, 1)

	require.NoError(t, err)
	assert.Equal(t, domain.SessionName(1), sessionName)
	assert.True(t, sessions.StopCalled)
}

func TestStopSession_NotRunning(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false

	sessionName, err := shared.StopSession(sessions, 1)

	require.NoError(t, err)
	assert.Empty(t, sessionName)
	assert.False(t, sessions.StopCalled)
}

func TestStopSession_IsRunningError(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningErr = assert.AnError

	_, err := shared.StopSession(sessions, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check session running")
}

func TestStopSession_StopError(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.StopErr = assert.AnError

	_, err := shared.StopSession(sessions, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stop session")
}

func TestSendSessionNotification_Running(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true

	err := shared.SendSessionNotification(sessions, 1, "test message")

	require.NoError(t, err)
	assert.True(t, sessions.SendCalled)
}

func TestSendSessionNotification_NotRunning(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false

	err := shared.SendSessionNotification(sessions, 1, "test message")

	require.NoError(t, err)
	assert.False(t, sessions.SendCalled)
}

func TestSendSessionNotification_SendError(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.SendErr = assert.AnError

	err := shared.SendSessionNotification(sessions, 1, "test message")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send notification")
}

func TestSendSessionNotification_EnterSendError(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true

	// Use SendFunc to fail only on the second call (Enter)
	callCount := 0
	sessions.SendFunc = func(_ string, keys string) error {
		callCount++
		if keys == "Enter" {
			return assert.AnError
		}
		return nil
	}

	err := shared.SendSessionNotification(sessions, 1, "test message")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send enter")
	assert.Equal(t, 2, callCount) // Both message and Enter were attempted
}

func TestRequireRunningSession_Running(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true

	sessionName, err := shared.RequireRunningSession(sessions, 1)

	require.NoError(t, err)
	assert.Equal(t, domain.SessionName(1), sessionName)
}

func TestRequireRunningSession_NotRunning(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false

	_, err := shared.RequireRunningSession(sessions, 1)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrNoSession))
}

func TestRequireRunningSession_Error(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningErr = assert.AnError

	_, err := shared.RequireRunningSession(sessions, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check session")
}

func TestEnsureNoRunningSession_NotRunning(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false

	err := shared.EnsureNoRunningSession(sessions, 1)

	require.NoError(t, err)
}

func TestEnsureNoRunningSession_Running(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true

	err := shared.EnsureNoRunningSession(sessions, 1)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrSessionRunning))
	assert.Contains(t, err.Error(), "task #1")
}

func TestEnsureNoRunningSession_Error(t *testing.T) {
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningErr = assert.AnError

	err := shared.EnsureNoRunningSession(sessions, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check session")
}
