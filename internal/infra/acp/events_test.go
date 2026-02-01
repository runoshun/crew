package acp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileEventWriterFactory_ForTask(t *testing.T) {
	t.Parallel()

	crewDir := t.TempDir()
	factory := NewFileEventWriterFactory(crewDir)

	writer, err := factory.ForTask("default", 1)
	require.NoError(t, err)
	defer func() { _ = writer.Close() }()

	// Verify directory was created
	eventsDir := filepath.Join(crewDir, "acp", "default", "1")
	_, err = os.Stat(eventsDir)
	require.NoError(t, err)
}

func TestFileEventWriter_Write(t *testing.T) {
	t.Parallel()

	crewDir := t.TempDir()
	factory := NewFileEventWriterFactory(crewDir)

	writer, err := factory.ForTask("default", 1)
	require.NoError(t, err)
	defer func() { _ = writer.Close() }()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	event := domain.ACPEvent{
		Timestamp: now,
		Type:      domain.ACPEventToolCall,
		SessionID: "session-1",
		Payload:   json.RawMessage(`{"tool":"test"}`),
	}

	err = writer.Write(ctx, event)
	require.NoError(t, err)

	// Close to flush
	err = writer.Close()
	require.NoError(t, err)

	// Verify file content
	eventsFile := filepath.Join(crewDir, "acp", "default", "1", "events.jsonl")
	data, err := os.ReadFile(eventsFile)
	require.NoError(t, err)

	var readEvent domain.ACPEvent
	err = json.Unmarshal(data, &readEvent)
	require.NoError(t, err)
	assert.Equal(t, event.Type, readEvent.Type)
	assert.Equal(t, event.SessionID, readEvent.SessionID)
}

func TestFileEventWriter_WriteMultiple(t *testing.T) {
	t.Parallel()

	crewDir := t.TempDir()
	factory := NewFileEventWriterFactory(crewDir)

	writer, err := factory.ForTask("default", 1)
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now().UTC()

	events := []domain.ACPEvent{
		{Timestamp: now, Type: domain.ACPEventToolCall, SessionID: "s1"},
		{Timestamp: now.Add(time.Second), Type: domain.ACPEventToolCallUpdate, SessionID: "s1"},
		{Timestamp: now.Add(2 * time.Second), Type: domain.ACPEventAgentMessageChunk, SessionID: "s1"},
	}

	for _, event := range events {
		err = writer.Write(ctx, event)
		require.NoError(t, err)
	}

	err = writer.Close()
	require.NoError(t, err)

	// Read back and verify
	reader := NewFileEventReader(crewDir, "default", 1)
	readEvents, err := reader.ReadAll(ctx)
	require.NoError(t, err)
	assert.Len(t, readEvents, 3)
	assert.Equal(t, domain.ACPEventToolCall, readEvents[0].Type)
	assert.Equal(t, domain.ACPEventToolCallUpdate, readEvents[1].Type)
	assert.Equal(t, domain.ACPEventAgentMessageChunk, readEvents[2].Type)
}

func TestFileEventReader_ReadAll_NoFile(t *testing.T) {
	t.Parallel()

	crewDir := t.TempDir()
	reader := NewFileEventReader(crewDir, "default", 999)

	ctx := context.Background()
	events, err := reader.ReadAll(ctx)
	require.NoError(t, err)
	assert.Nil(t, events)
}

func TestFileEventReader_ReadAll_EmptyFile(t *testing.T) {
	t.Parallel()

	crewDir := t.TempDir()
	eventsDir := filepath.Join(crewDir, "acp", "default", "1")
	err := os.MkdirAll(eventsDir, 0750)
	require.NoError(t, err)

	eventsFile := filepath.Join(eventsDir, "events.jsonl")
	err = os.WriteFile(eventsFile, []byte{}, 0600)
	require.NoError(t, err)

	reader := NewFileEventReader(crewDir, "default", 1)

	ctx := context.Background()
	events, err := reader.ReadAll(ctx)
	require.NoError(t, err)
	assert.Nil(t, events)
}

func TestFileEventReader_ReadAll_SkipMalformed(t *testing.T) {
	t.Parallel()

	crewDir := t.TempDir()
	eventsDir := filepath.Join(crewDir, "acp", "default", "1")
	err := os.MkdirAll(eventsDir, 0750)
	require.NoError(t, err)

	// Write a mix of valid and invalid lines
	eventsFile := filepath.Join(eventsDir, "events.jsonl")
	content := `{"ts":"2025-01-01T00:00:00Z","type":"tool_call","session_id":"s1"}
invalid json
{"ts":"2025-01-01T00:00:01Z","type":"tool_call_update","session_id":"s1"}
`
	err = os.WriteFile(eventsFile, []byte(content), 0600)
	require.NoError(t, err)

	reader := NewFileEventReader(crewDir, "default", 1)

	ctx := context.Background()
	events, err := reader.ReadAll(ctx)
	require.NoError(t, err)
	// 2 valid events + 1 warning event about skipped lines
	assert.Len(t, events, 3)
	// Last event should be warning about skipped lines
	assert.Equal(t, domain.ACPEventType("_warning"), events[2].Type)
}
