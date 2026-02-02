package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACPStart_Execute_Success_NewWorktree(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusInProgress}

	sessions := testutil.NewMockSessionManager()
	var gotSession string
	sessions.IsRunningFunc = func(name string) (bool, error) {
		gotSession = name
		return false, nil
	}

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = false

	configLoader := testutil.NewMockConfigLoader()
	git := &testutil.MockGit{}
	runner := testutil.NewMockScriptRunner()

	crewDir := t.TempDir()
	repoRoot := t.TempDir()

	uc := NewACPStart(repo, sessions, worktrees, configLoader, git, runner, crewDir, repoRoot)

	out, err := uc.Execute(context.Background(), ACPStartInput{TaskID: 1})
	require.NoError(t, err)

	assert.Equal(t, domain.ACPSessionName(1), gotSession)
	assert.Equal(t, domain.ACPSessionName(1), out.SessionName)
	assert.Equal(t, worktrees.CreatePath, out.WorktreePath)
	assert.True(t, worktrees.CreateCalled)
	assert.True(t, worktrees.SetupCalled)
	assert.True(t, runner.RunCalled)
	assert.True(t, sessions.StartCalled)
	assert.Equal(t, domain.ACPSessionName(1), sessions.StartOpts.Name)
	assert.Equal(t, worktrees.CreatePath, sessions.StartOpts.Dir)
	assert.Equal(t, configLoader.Config.AgentsConfig.DefaultWorker, sessions.StartOpts.TaskAgent)

	scriptPath := domain.ACPScriptPath(crewDir, 1)
	_, statErr := os.Stat(scriptPath)
	require.NoError(t, statErr)

	logPath := domain.SessionLogPath(crewDir, domain.ACPSessionName(1))
	_, logErr := os.Stat(logPath)
	require.NoError(t, logErr)
}

func TestACPStart_Execute_AgentNotFound(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusInProgress}

	sessions := testutil.NewMockSessionManager()
	worktrees := testutil.NewMockWorktreeManager()
	configLoader := testutil.NewMockConfigLoader()
	git := &testutil.MockGit{}
	runner := testutil.NewMockScriptRunner()

	crewDir := t.TempDir()
	repoRoot := t.TempDir()

	uc := NewACPStart(repo, sessions, worktrees, configLoader, git, runner, crewDir, repoRoot)

	_, err := uc.Execute(context.Background(), ACPStartInput{TaskID: 1, Agent: "missing-agent"})
	assert.ErrorIs(t, err, domain.ErrAgentNotFound)
}

func TestACPStart_Execute_SessionRunning(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Test task", Status: domain.StatusInProgress}

	sessions := testutil.NewMockSessionManager()
	sessions.IsRunningVal = true
	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	worktrees.ResolvePath = filepath.Join(t.TempDir(), "worktree")

	configLoader := testutil.NewMockConfigLoader()
	git := &testutil.MockGit{}
	runner := testutil.NewMockScriptRunner()

	crewDir := t.TempDir()
	repoRoot := t.TempDir()

	uc := NewACPStart(repo, sessions, worktrees, configLoader, git, runner, crewDir, repoRoot)

	_, err := uc.Execute(context.Background(), ACPStartInput{TaskID: 1})
	assert.ErrorIs(t, err, domain.ErrSessionRunning)
}
