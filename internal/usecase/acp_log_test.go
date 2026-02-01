package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEventReader struct {
	events []domain.ACPEvent
	err    error
}

func (m *mockEventReader) ReadAll(_ context.Context) ([]domain.ACPEvent, error) {
	return m.events, m.err
}

func TestACPLog_Execute(t *testing.T) {
	t.Parallel()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}

	now := time.Now().UTC()
	events := []domain.ACPEvent{
		{Timestamp: now, Type: domain.ACPEventToolCall, SessionID: "s1", Payload: json.RawMessage(`{}`)},
		{Timestamp: now.Add(time.Second), Type: domain.ACPEventAgentMessageChunk, SessionID: "s1"},
	}

	mockConfig := testutil.NewMockConfigLoader()
	mockGit := &testutil.MockGit{}

	reader := &mockEventReader{events: events}
	readerFactory := func(_ string, _ int) domain.ACPEventReader {
		return reader
	}

	uc := NewACPLog(repo, mockConfig, mockGit, readerFactory)

	ctx := context.Background()
	out, err := uc.Execute(ctx, ACPLogInput{TaskID: 1})
	require.NoError(t, err)
	assert.Len(t, out.Events, 2)
	assert.Equal(t, domain.ACPEventToolCall, out.Events[0].Type)
}

func TestACPLog_Execute_TaskNotFound(t *testing.T) {
	t.Parallel()

	repo := testutil.NewMockTaskRepository()
	mockConfig := testutil.NewMockConfigLoader()
	mockGit := &testutil.MockGit{}

	readerFactory := func(_ string, _ int) domain.ACPEventReader {
		return &mockEventReader{}
	}

	uc := NewACPLog(repo, mockConfig, mockGit, readerFactory)

	ctx := context.Background()
	_, err := uc.Execute(ctx, ACPLogInput{TaskID: 999})
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestACPLog_Execute_NoEvents(t *testing.T) {
	t.Parallel()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Test task",
		Status: domain.StatusInProgress,
	}

	mockConfig := testutil.NewMockConfigLoader()
	mockGit := &testutil.MockGit{}

	reader := &mockEventReader{events: nil}
	readerFactory := func(_ string, _ int) domain.ACPEventReader {
		return reader
	}

	uc := NewACPLog(repo, mockConfig, mockGit, readerFactory)

	ctx := context.Background()
	out, err := uc.Execute(ctx, ACPLogInput{TaskID: 1})
	require.NoError(t, err)
	assert.Nil(t, out.Events)
}
