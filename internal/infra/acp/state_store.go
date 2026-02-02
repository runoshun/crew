package acp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

const acpStateFileName = "state.json"

// FileStateStore persists ACP state in the filesystem.
type FileStateStore struct {
	crewDir string
}

// NewFileStateStore creates a new file-based ACP state store.
func NewFileStateStore(crewDir string) *FileStateStore {
	return &FileStateStore{crewDir: crewDir}
}

type acpStatePayload struct {
	ExecutionSubstate string `json:"execution_substate"`
	SessionID         string `json:"session_id,omitempty"`
}

// Load reads the current ACP state for a task.
func (s *FileStateStore) Load(ctx context.Context, namespace string, taskID int) (domain.ACPExecutionState, error) {
	if err := ctx.Err(); err != nil {
		return domain.ACPExecutionState{}, err
	}
	path := s.statePath(namespace, taskID)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.ACPExecutionState{}, domain.ErrACPStateNotFound
		}
		return domain.ACPExecutionState{}, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return domain.ACPExecutionState{}, domain.ErrEmptyFile
	}
	var payload acpStatePayload
	if err := decodeJSONStrict(data, &payload); err != nil {
		return domain.ACPExecutionState{}, err
	}
	state := domain.ACPExecutionState{
		ExecutionSubstate: domain.ACPExecutionSubstate(payload.ExecutionSubstate),
		SessionID:         payload.SessionID,
	}
	if !state.ExecutionSubstate.IsValid() {
		return domain.ACPExecutionState{}, domain.ErrInvalidACPExecutionSubstate
	}
	return state, nil
}

// Save writes the ACP state for a task.
func (s *FileStateStore) Save(ctx context.Context, namespace string, taskID int, state domain.ACPExecutionState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !state.ExecutionSubstate.IsValid() {
		return domain.ErrInvalidACPExecutionSubstate
	}
	base := domain.ACPDir(s.crewDir, s.normalizeNamespace(namespace), taskID)
	if err := os.MkdirAll(base, 0o750); err != nil {
		return err
	}
	payload := acpStatePayload{
		ExecutionSubstate: string(state.ExecutionSubstate),
		SessionID:         state.SessionID,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(base, acpStateFileName), data, 0o644)
}

func (s *FileStateStore) statePath(namespace string, taskID int) string {
	base := domain.ACPDir(s.crewDir, s.normalizeNamespace(namespace), taskID)
	return filepath.Join(base, acpStateFileName)
}

func (s *FileStateStore) normalizeNamespace(namespace string) string {
	if namespace == "" {
		return "default"
	}
	return namespace
}

func decodeJSONStrict(content []byte, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(content)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("unexpected trailing content")
	}
	return nil
}

func writeAtomic(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	file, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return err
	}
	name := file.Name()
	cleanup := func() {
		_ = os.Remove(name)
	}
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		cleanup()
		return err
	}
	if err := file.Chmod(perm); err != nil {
		_ = file.Close()
		cleanup()
		return err
	}
	if err := file.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(name, path); err != nil {
		cleanup()
		return err
	}
	return nil
}
