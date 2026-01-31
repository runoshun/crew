package acp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

const defaultPollInterval = 200 * time.Millisecond

// FileIPCFactory creates file-based ACP IPC instances.
type FileIPCFactory struct {
	crewDir      string
	pollInterval time.Duration
}

// NewFileIPCFactory creates a new file-based IPC factory.
func NewFileIPCFactory(crewDir string) *FileIPCFactory {
	return &FileIPCFactory{
		crewDir:      crewDir,
		pollInterval: defaultPollInterval,
	}
}

// ForTask returns a file-based IPC instance for a task.
func (f *FileIPCFactory) ForTask(namespace string, taskID int) domain.ACPIPC {
	base := domain.ACPDir(f.crewDir, namespace, taskID)
	return &FileIPC{
		commandsDir:  filepath.Join(base, "commands"),
		pollInterval: f.pollInterval,
	}
}

// FileIPC implements ACP IPC using filesystem polling.
type FileIPC struct {
	commandsDir  string
	pollInterval time.Duration
}

// Next blocks until a command is available or context is canceled.
func (f *FileIPC) Next(ctx context.Context) (domain.ACPCommand, error) {
	for {
		if err := ctx.Err(); err != nil {
			return domain.ACPCommand{}, err
		}

		if err := f.ensureDir(); err != nil {
			return domain.ACPCommand{}, err
		}

		entries, err := os.ReadDir(f.commandsDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				time.Sleep(f.pollInterval)
				continue
			}
			return domain.ACPCommand{}, fmt.Errorf("read commands dir: %w", err)
		}

		files := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if filepath.Ext(name) != ".json" {
				continue
			}
			files = append(files, name)
		}

		sort.Strings(files)
		for _, name := range files {
			path := filepath.Join(f.commandsDir, name)
			cmd, ok := f.readCommand(path)
			if !ok {
				continue
			}
			return cmd, nil
		}

		time.Sleep(f.pollInterval)
	}
}

// Send enqueues a command for the runner.
func (f *FileIPC) Send(ctx context.Context, cmd domain.ACPCommand) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := f.ensureDir(); err != nil {
		return err
	}

	if cmd.ID == "" {
		cmd.ID = newCommandID()
	}
	if cmd.CreatedAt.IsZero() {
		cmd.CreatedAt = time.Now().UTC()
	}
	if err := cmd.Validate(); err != nil {
		return err
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal command: %w", err)
	}

	tmpPath := filepath.Join(f.commandsDir, ".tmp-"+cmd.ID)
	finalPath := filepath.Join(f.commandsDir, cmd.ID+".json")
	if err := os.WriteFile(tmpPath, payload, 0600); err != nil {
		return fmt.Errorf("write command temp file: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("finalize command file: %w", err)
	}

	return nil
}

func (f *FileIPC) ensureDir() error {
	if err := os.MkdirAll(f.commandsDir, 0750); err != nil {
		return fmt.Errorf("create commands dir: %w", err)
	}
	return nil
}

func (f *FileIPC) readCommand(path string) (domain.ACPCommand, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		f.moveToFailed(path)
		return domain.ACPCommand{}, false
	}

	var cmd domain.ACPCommand
	if err := json.Unmarshal(data, &cmd); err != nil {
		f.moveToFailed(path)
		return domain.ACPCommand{}, false
	}

	if err := cmd.Validate(); err != nil {
		f.moveToFailed(path)
		return domain.ACPCommand{}, false
	}

	if err := os.Remove(path); err != nil {
		return domain.ACPCommand{}, false
	}

	return cmd, true
}

func (f *FileIPC) moveToFailed(path string) {
	failedDir := filepath.Join(f.commandsDir, "failed")
	if err := os.MkdirAll(failedDir, 0750); err == nil {
		base := filepath.Base(path)
		target := filepath.Join(failedDir, base)
		if err := os.Rename(path, target); err == nil {
			return
		}
	}
	_ = os.Rename(path, path+".bad")
}

func newCommandID() string {
	now := time.Now().UTC().UnixNano()
	return fmt.Sprintf("%020d-%s", now, randomSuffix(4))
}

func randomSuffix(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(buf)
}
