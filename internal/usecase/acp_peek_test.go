package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACPPeek_Execute_Success(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusInProgress}
	var gotSession string
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningFunc = func(name string) (bool, error) {
		gotSession = name
		return true, nil
	}
	sessions.PeekOutput = "line1\nline2"

	uc := NewACPPeek(repo, sessions)

	out, err := uc.Execute(context.Background(), ACPPeekInput{TaskID: 1, Lines: 10})
	require.NoError(t, err)
	assert.Equal(t, domain.ACPSessionName(1), gotSession)
	assert.Equal(t, 10, sessions.PeekLines)
	assert.Equal(t, "line1\nline2", out.Output)
}

func TestACPPeek_Execute_DefaultLines(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusInProgress}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.PeekOutput = "output"

	uc := NewACPPeek(repo, sessions)

	out, err := uc.Execute(context.Background(), ACPPeekInput{TaskID: 1, Lines: 0})
	require.NoError(t, err)
	assert.Equal(t, DefaultPeekLines, sessions.PeekLines)
	assert.Equal(t, "output", out.Output)
}

func TestACPPeek_Execute_TaskNotFound(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	sessions := testutil.NewMockSessionManager()

	uc := NewACPPeek(repo, sessions)

	_, err := uc.Execute(context.Background(), ACPPeekInput{TaskID: 999})
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestACPPeek_Execute_NoSession(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusTodo}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = false

	uc := NewACPPeek(repo, sessions)

	_, err := uc.Execute(context.Background(), ACPPeekInput{TaskID: 1})
	assert.ErrorIs(t, err, domain.ErrNoSession)
	assert.Contains(t, err.Error(), "crew acp start 1")
}

func TestACPPeek_Execute_IsRunningError(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusInProgress}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningErr = assert.AnError

	uc := NewACPPeek(repo, sessions)

	_, err := uc.Execute(context.Background(), ACPPeekInput{TaskID: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check session")
}

func TestACPPeek_Execute_PeekError(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusInProgress}
	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	sessions.PeekErr = assert.AnError

	uc := NewACPPeek(repo, sessions)

	_, err := uc.Execute(context.Background(), ACPPeekInput{TaskID: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "peek session")
}

func TestACPPeek_Execute_GetTaskError(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.GetErr = assert.AnError
	sessions := testutil.NewMockSessionManager()

	uc := NewACPPeek(repo, sessions)

	_, err := uc.Execute(context.Background(), ACPPeekInput{TaskID: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task")
}
