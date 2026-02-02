package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel_sessionLogPath(t *testing.T) {
	crewDir := t.TempDir()
	model := &Model{container: &app.Container{Config: app.Config{CrewDir: crewDir}}}

	workerPath, err := model.sessionLogPath(1, false)
	require.NoError(t, err)
	assert.Equal(t, domain.SessionLogPath(crewDir, domain.SessionName(1)), workerPath)

	reviewPath, err := model.sessionLogPath(1, true)
	require.NoError(t, err)
	assert.Equal(t, domain.SessionLogPath(crewDir, domain.ReviewSessionName(1)), reviewPath)
}

func TestModel_hasSessionLog(t *testing.T) {
	crewDir := t.TempDir()
	model := &Model{container: &app.Container{Config: app.Config{CrewDir: crewDir}}}

	assert.False(t, model.hasSessionLog(1, false))
	assert.NoError(t, model.err)

	logPath, err := model.sessionLogPath(1, false)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(logPath), 0o755))
	require.NoError(t, os.WriteFile(logPath, []byte("log"), 0o644))

	assert.True(t, model.hasSessionLog(1, false))
}

func TestModel_showLogInPager_ReturnsExecLogMsg(t *testing.T) {
	crewDir := t.TempDir()
	model := &Model{container: &app.Container{Config: app.Config{CrewDir: crewDir}}}

	logPath, err := model.sessionLogPath(1, false)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(logPath), 0o755))
	require.NoError(t, os.WriteFile(logPath, []byte("log"), 0o644))

	cmd := model.showLogInPager(1, false)
	msg := cmd()

	execMsg, ok := msg.(execLogMsg)
	require.True(t, ok)
	assert.Equal(t, "less", execMsg.cmd.Program)
	assert.Equal(t, []string{"-R", logPath}, execMsg.cmd.Args)
}

func TestModel_showLogInPager_MissingLog(t *testing.T) {
	crewDir := t.TempDir()
	model := &Model{container: &app.Container{Config: app.Config{CrewDir: crewDir}}}

	cmd := model.showLogInPager(1, false)
	msg := cmd()

	errMsg, ok := msg.(MsgError)
	require.True(t, ok)
	assert.ErrorIs(t, errMsg.Err, domain.ErrNoSession)
}
