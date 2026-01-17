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

func TestSessionEnded_Execute_NormalExit(t *testing.T) {
	crewDir := t.TempDir()
	createScriptFiles(t, crewDir, 1)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Test task",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}

	uc := NewSessionEnded(repo, crewDir)

	// Execute with normal exit (code 0)
	out, err := uc.Execute(context.Background(), SessionEndedInput{
		TaskID:   1,
		ExitCode: 0,
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.Ignored)

	// Verify task updated
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusForReview, task.Status)
	// Session info should be kept for review/merge operations
	assert.Equal(t, "claude", task.Agent)
	assert.Equal(t, "crew-1", task.Session)

	// Verify script file cleaned up
	assert.NoFileExists(t, domain.ScriptPath(crewDir, 1))
}

func TestSessionEnded_Execute_AbnormalExit(t *testing.T) {
	crewDir := t.TempDir()
	createScriptFiles(t, crewDir, 1)

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Test task",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}

	uc := NewSessionEnded(repo, crewDir)

	// Execute with abnormal exit (code 1)
	out, err := uc.Execute(context.Background(), SessionEndedInput{
		TaskID:   1,
		ExitCode: 1,
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.Ignored)

	// Verify task updated to error status
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusError, task.Status)
	assert.Empty(t, task.Agent)
	assert.Empty(t, task.Session)
}

func TestSessionEnded_Execute_CtrlC(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Test task",
		Status:  domain.StatusInProgress,
		Agent:   "claude",
		Session: "crew-1",
	}

	uc := NewSessionEnded(repo, crewDir)

	// Execute with Ctrl+C exit (code 130)
	out, err := uc.Execute(context.Background(), SessionEndedInput{
		TaskID:   1,
		ExitCode: 130,
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.Ignored)

	// Verify task updated to error status
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusError, task.Status)
}

func TestSessionEnded_Execute_AlreadyCleared(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Test task",
		Status:  domain.StatusForReview,
		Agent:   "", // Already cleared
		Session: "", // Already cleared
	}

	uc := NewSessionEnded(repo, crewDir)

	// Execute
	out, err := uc.Execute(context.Background(), SessionEndedInput{
		TaskID:   1,
		ExitCode: 0,
	})

	// Assert - should be ignored
	require.NoError(t, err)
	assert.True(t, out.Ignored)

	// Status should not change
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusForReview, task.Status)
}

func TestSessionEnded_Execute_TaskNotFound(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()

	uc := NewSessionEnded(repo, crewDir)

	// Execute with non-existent task
	out, err := uc.Execute(context.Background(), SessionEndedInput{
		TaskID:   999,
		ExitCode: 0,
	})

	// Assert - should be ignored
	require.NoError(t, err)
	assert.True(t, out.Ignored)
}

func TestSessionEnded_Execute_MaintainInReviewStatus(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Test task",
		Status:  domain.StatusForReview, // Already for_review
		Agent:   "claude",
		Session: "crew-1",
	}

	uc := NewSessionEnded(repo, crewDir)

	// Execute with normal exit
	out, err := uc.Execute(context.Background(), SessionEndedInput{
		TaskID:   1,
		ExitCode: 0,
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.Ignored)

	// Status should remain for_review and session info should be kept
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusForReview, task.Status)
	assert.Equal(t, "claude", task.Agent)
	assert.Equal(t, "crew-1", task.Session)
}

func TestSessionEnded_Execute_DoneStatusUnchanged(t *testing.T) {
	crewDir := t.TempDir()

	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{
		ID:      1,
		Title:   "Test task",
		Status:  domain.StatusClosed,
		Agent:   "claude", // Shouldn't happen, but test edge case
		Session: "crew-1",
	}

	uc := NewSessionEnded(repo, crewDir)

	// Execute
	out, err := uc.Execute(context.Background(), SessionEndedInput{
		TaskID:   1,
		ExitCode: 1, // Even with error exit
	})

	// Assert
	require.NoError(t, err)
	assert.False(t, out.Ignored)

	// Status should remain done
	task := repo.Tasks[1]
	assert.Equal(t, domain.StatusClosed, task.Status)
}

// Helper function to create script file for testing cleanup
func createScriptFiles(t *testing.T, crewDir string, taskID int) {
	t.Helper()
	scriptsDir := filepath.Join(crewDir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0755))
	require.NoError(t, os.WriteFile(domain.ScriptPath(crewDir, taskID), []byte("test"), 0755))
}
