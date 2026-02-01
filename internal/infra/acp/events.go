package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

const eventsFileName = "events.jsonl"

// FileEventWriterFactory creates file-based event writers.
type FileEventWriterFactory struct {
	crewDir string
}

// NewFileEventWriterFactory creates a new file-based event writer factory.
func NewFileEventWriterFactory(crewDir string) *FileEventWriterFactory {
	return &FileEventWriterFactory{crewDir: crewDir}
}

// ForTask returns a file-based event writer for a task.
func (f *FileEventWriterFactory) ForTask(namespace string, taskID int) (domain.ACPEventWriter, error) {
	base := domain.ACPDir(f.crewDir, namespace, taskID)
	if err := os.MkdirAll(base, 0750); err != nil {
		return nil, fmt.Errorf("create ACP dir: %w", err)
	}

	path := filepath.Join(base, eventsFileName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("open events file: %w", err)
	}

	return &FileEventWriter{
		file: file,
		enc:  json.NewEncoder(file),
	}, nil
}

// FileEventWriter writes events to a JSONL file.
// Fields are ordered to minimize memory padding.
type FileEventWriter struct {
	file *os.File
	enc  *json.Encoder
	mu   sync.Mutex
}

// Write appends an event to the event log.
func (w *FileEventWriter) Write(_ context.Context, event domain.ACPEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.enc.Encode(event); err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	return nil
}

// Close releases any resources held by the writer.
func (w *FileEventWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// FileEventReader reads events from a JSONL file.
type FileEventReader struct {
	path string
}

// NewFileEventReader creates a new file-based event reader.
func NewFileEventReader(crewDir, namespace string, taskID int) *FileEventReader {
	base := domain.ACPDir(crewDir, namespace, taskID)
	return &FileEventReader{
		path: filepath.Join(base, eventsFileName),
	}
}

// ReadAll returns all events from the event log.
func (r *FileEventReader) ReadAll(_ context.Context) ([]domain.ACPEvent, error) {
	file, err := os.Open(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open events file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var events []domain.ACPEvent
	var skippedLines int
	scanner := bufio.NewScanner(file)

	// Start with default buffer, allow up to 1MB for large JSON lines
	const (
		initialBufSize = 64 * 1024   // 64KB
		maxLineSize    = 1024 * 1024 // 1MB
	)
	scanner.Buffer(make([]byte, initialBufSize), maxLineSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event domain.ACPEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Count malformed lines but continue
			skippedLines++
			continue
		}
		events = append(events, event)
	}

	// Log skipped lines count if any
	if skippedLines > 0 {
		// Append warning to events as a pseudo-event
		// This is informational; the caller can filter it out
		events = append(events, domain.ACPEvent{
			Type:    domain.ACPEventType("_warning"),
			Payload: json.RawMessage(fmt.Sprintf(`{"message":"skipped %d malformed lines"}`, skippedLines)),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan events file: %w", err)
	}

	return events, nil
}
