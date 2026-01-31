package cli

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMigrateCommand_JSON(t *testing.T) {
	repoDir := t.TempDir()
	crewDir := filepath.Join(repoDir, ".crew")
	gitDir := filepath.Join(repoDir, ".git")

	now := time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC)
	task := &domain.Task{
		ID:            1,
		Title:         "Legacy task",
		Description:   "Legacy body",
		Status:        domain.StatusTodo,
		Created:       now,
		BaseBranch:    "main",
		StatusVersion: 0,
	}
	container := app.NewWithDeps(
		app.Config{RepoRoot: repoDir, CrewDir: crewDir, GitDir: gitDir},
		testutil.NewMockTaskRepository(),
		&testutil.MockStoreInitializer{},
		&testutil.MockClock{NowTime: now},
		testutil.NewMockLogger(),
		testutil.NewMockCommandExecutor(),
	)
	container.ConfigLoader = testutil.NewMockConfigLoader()

	legacyPath := filepath.Join(crewDir, "tasks.json")
	legacyStore, legacyInit := container.JSONStore(legacyPath)
	_, err := legacyInit.Initialize()
	require.NoError(t, err)
	require.NoError(t, legacyStore.SaveTaskWithComments(task, []domain.Comment{{Text: "note", Author: "worker", Time: now}}))

	cmd := newMigrateCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--from", "json", "--namespace", "legacy"})

	err = cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Migrated 1 task(s) from json store")

	destStore, _ := container.FileStore("legacy")
	migrated, err := destStore.Get(1)
	require.NoError(t, err)
	require.NotNil(t, migrated)
	assert.Equal(t, "Legacy task", migrated.Title)

	comments, err := destStore.GetComments(1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "note", comments[0].Text)
}
