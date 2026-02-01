package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACPAttach_Execute_Success(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusInProgress}
	var gotSession string
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningFunc = func(name string) (bool, error) {
		gotSession = name
		return true, nil
	}

	uc := NewACPAttach(repo, sessions)

	_, err := uc.Execute(context.Background(), ACPAttachInput{TaskID: 1})
	require.NoError(t, err)
	assert.Equal(t, domain.ACPSessionName(1), gotSession)
	assert.True(t, sessions.AttachCalled)
}

func TestACPAttach_Execute_TaskNotFound(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	sessions := testutil.NewMockSessionManager()

	uc := NewACPAttach(repo, sessions)

	_, err := uc.Execute(context.Background(), ACPAttachInput{TaskID: 999})
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestACPAttach_Execute_NoSession(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusTodo}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false

	uc := NewACPAttach(repo, sessions)

	_, err := uc.Execute(context.Background(), ACPAttachInput{TaskID: 1})
	assert.ErrorIs(t, err, domain.ErrNoSession)
	assert.Contains(t, err.Error(), "crew acp start 1")
}

func TestACPAttach_Execute_IsRunningError(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusInProgress}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningErr = assert.AnError

	uc := NewACPAttach(repo, sessions)

	_, err := uc.Execute(context.Background(), ACPAttachInput{TaskID: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check session")
}

func TestACPAttach_Execute_AttachError(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusInProgress}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.AttachErr = assert.AnError

	uc := NewACPAttach(repo, sessions)

	_, err := uc.Execute(context.Background(), ACPAttachInput{TaskID: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attach session")
}

func TestACPAttach_Execute_GetTaskError(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	sessions := testutil.NewMockSessionManager()

	uc := NewACPAttach(repo, sessions)

	_, err := uc.Execute(context.Background(), ACPAttachInput{TaskID: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}
