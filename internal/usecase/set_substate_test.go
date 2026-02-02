package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetSubstate_Execute(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Namespace: "alpha"}
	uc := NewSetSubstate(repo)

	_, err := uc.Execute(context.Background(), SetSubstateInput{
		TaskID:   1,
		Substate: domain.SubstateRunning,
	})

	require.NoError(t, err)
	require.Contains(t, repo.Tasks, 1)
	assert.Equal(t, domain.SubstateRunning, repo.Tasks[1].ExecutionSubstate)
}

func TestSetSubstate_InvalidSubstate(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	uc := NewSetSubstate(repo)

	_, err := uc.Execute(context.Background(), SetSubstateInput{
		TaskID:   1,
		Substate: domain.ExecutionSubstate("bad"),
	})

	require.ErrorIs(t, err, domain.ErrInvalidExecutionSubstate)
}

func TestSetSubstate_TaskNotFound(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	uc := NewSetSubstate(repo)

	_, err := uc.Execute(context.Background(), SetSubstateInput{
		TaskID:   1,
		Substate: domain.SubstateRunning,
	})

	require.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestSetSubstate_SaveError(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.SaveErr = assert.AnError
	repo.Tasks[1] = &domain.Task{ID: 1, Namespace: domain.DefaultNamespace}
	uc := NewSetSubstate(repo)

	_, err := uc.Execute(context.Background(), SetSubstateInput{
		TaskID:   1,
		Substate: domain.SubstateRunning,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}
