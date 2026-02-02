package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingACPStateStore struct {
	loadState     domain.ACPExecutionState
	loadErr       error
	saveErr       error
	lastNamespace string
	lastTaskID    int
	savedState    domain.ACPExecutionState
	loadCalled    bool
	saveCalled    bool
}

func (s *recordingACPStateStore) Load(_ context.Context, namespace string, taskID int) (domain.ACPExecutionState, error) {
	s.loadCalled = true
	s.lastNamespace = namespace
	s.lastTaskID = taskID
	return s.loadState, s.loadErr
}

func (s *recordingACPStateStore) Save(_ context.Context, namespace string, taskID int, state domain.ACPExecutionState) error {
	s.saveCalled = true
	s.lastNamespace = namespace
	s.lastTaskID = taskID
	s.savedState = state
	return s.saveErr
}

func TestSetSubstate_Execute(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Namespace: "alpha"}
	store := &recordingACPStateStore{
		loadState: domain.ACPExecutionState{ExecutionSubstate: domain.ACPExecutionIdle, SessionID: "session-1"},
	}
	uc := NewSetSubstate(repo, store)

	_, err := uc.Execute(context.Background(), SetSubstateInput{
		TaskID:   1,
		Substate: domain.ACPExecutionRunning,
	})

	require.NoError(t, err)
	assert.True(t, store.loadCalled)
	assert.True(t, store.saveCalled)
	assert.Equal(t, "alpha", store.lastNamespace)
	assert.Equal(t, 1, store.lastTaskID)
	assert.Equal(t, domain.ACPExecutionRunning, store.savedState.ExecutionSubstate)
	assert.Equal(t, "session-1", store.savedState.SessionID)
}

func TestSetSubstate_InvalidSubstate(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	store := &recordingACPStateStore{}
	uc := NewSetSubstate(repo, store)

	_, err := uc.Execute(context.Background(), SetSubstateInput{
		TaskID:   1,
		Substate: domain.ACPExecutionSubstate("bad"),
	})

	require.ErrorIs(t, err, domain.ErrInvalidACPExecutionSubstate)
	assert.False(t, store.saveCalled)
}

func TestSetSubstate_TaskNotFound(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	store := &recordingACPStateStore{}
	uc := NewSetSubstate(repo, store)

	_, err := uc.Execute(context.Background(), SetSubstateInput{
		TaskID:   1,
		Substate: domain.ACPExecutionRunning,
	})

	require.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestSetSubstate_LoadError(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Namespace: "default"}
	store := &recordingACPStateStore{loadErr: assert.AnError}
	uc := NewSetSubstate(repo, store)

	_, err := uc.Execute(context.Background(), SetSubstateInput{
		TaskID:   1,
		Substate: domain.ACPExecutionRunning,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, assert.AnError))
}
