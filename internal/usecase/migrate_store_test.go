package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateStore_Execute_MigratesTasks(t *testing.T) {
	source := testutil.NewMockTaskRepository()
	dest := testutil.NewMockTaskRepository()
	destInit := &testutil.MockStoreInitializer{}

	now := time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC)
	source.Tasks[1] = &domain.Task{
		ID:            1,
		Title:         "Legacy task",
		Description:   "Legacy description",
		Status:        domain.StatusTodo,
		Created:       now,
		BaseBranch:    "main",
		Labels:        []string{"beta", "alpha"},
		StatusVersion: 0,
	}
	source.Comments[1] = []domain.Comment{{Text: "note", Author: "worker", Time: now}}

	uc := NewMigrateStore(source, dest, destInit)
	out, err := uc.Execute(context.Background(), MigrateStoreInput{})

	require.NoError(t, err)
	assert.Equal(t, 1, out.Total)
	assert.Equal(t, 1, out.Migrated)
	assert.Equal(t, 0, out.Skipped)

	migrated := dest.Tasks[1]
	require.NotNil(t, migrated)
	assert.Equal(t, "Legacy task", migrated.Title)
	assert.Equal(t, "Legacy description", migrated.Description)
	assert.Equal(t, domain.StatusTodo, migrated.Status)
	assert.Equal(t, domain.StatusVersionCurrent, migrated.StatusVersion)

	comments := dest.Comments[1]
	require.Len(t, comments, 1)
	assert.Equal(t, "note", comments[0].Text)
}

func TestMigrateStore_Execute_SkipsIdentical(t *testing.T) {
	source := testutil.NewMockTaskRepository()
	dest := testutil.NewMockTaskRepository()
	destInit := &testutil.MockStoreInitializer{}

	now := time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC)
	task := &domain.Task{
		ID:            2,
		Title:         "Same task",
		Description:   "Same body",
		Status:        domain.StatusDone,
		Created:       now,
		BaseBranch:    "main",
		StatusVersion: 0,
	}
	source.Tasks[2] = task
	source.Comments[2] = []domain.Comment{{Text: "note", Author: "worker", Time: now}}

	dest.Tasks[2] = cloneTask(task)
	dest.Comments[2] = []domain.Comment{{Text: "note", Author: "worker", Time: now}}

	uc := NewMigrateStore(source, dest, destInit)
	out, err := uc.Execute(context.Background(), MigrateStoreInput{})

	require.NoError(t, err)
	assert.Equal(t, 1, out.Total)
	assert.Equal(t, 0, out.Migrated)
	assert.Equal(t, 1, out.Skipped)
}

func TestMigrateStore_Execute_Conflict(t *testing.T) {
	source := testutil.NewMockTaskRepository()
	dest := testutil.NewMockTaskRepository()
	destInit := &testutil.MockStoreInitializer{}

	source.Tasks[3] = &domain.Task{
		ID:         3,
		Title:      "Source task",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}
	dest.Tasks[3] = &domain.Task{
		ID:         3,
		Title:      "Different title",
		Status:     domain.StatusTodo,
		BaseBranch: "main",
	}

	uc := NewMigrateStore(source, dest, destInit)
	_, err := uc.Execute(context.Background(), MigrateStoreInput{})

	assert.ErrorIs(t, err, domain.ErrMigrationConflict)
}

func TestMigrateStore_Execute_ListError(t *testing.T) {
	source := &testutil.MockTaskRepositoryWithListError{
		MockTaskRepository: testutil.NewMockTaskRepository(),
		ListErr:            assert.AnError,
	}
	dest := testutil.NewMockTaskRepository()
	destInit := &testutil.MockStoreInitializer{}

	uc := NewMigrateStore(source, dest, destInit)
	_, err := uc.Execute(context.Background(), MigrateStoreInput{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list source tasks")
}

func TestMigrateStore_Execute_CommentsError(t *testing.T) {
	source := &testutil.MockTaskRepositoryWithCommentsError{
		MockTaskRepository: testutil.NewMockTaskRepository(),
		CommentsErr:        assert.AnError,
	}
	source.Tasks[1] = &domain.Task{ID: 1, Title: "Task", Status: domain.StatusTodo, BaseBranch: "main"}
	dest := testutil.NewMockTaskRepository()
	destInit := &testutil.MockStoreInitializer{}

	uc := NewMigrateStore(source, dest, destInit)
	_, err := uc.Execute(context.Background(), MigrateStoreInput{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get source comments")
}

func TestMigrateStore_Execute_SaveError(t *testing.T) {
	source := testutil.NewMockTaskRepository()
	dest := testutil.NewMockTaskRepository()
	dest.SaveErr = assert.AnError
	destInit := &testutil.MockStoreInitializer{}

	source.Tasks[1] = &domain.Task{ID: 1, Title: "Task", Status: domain.StatusTodo, BaseBranch: "main"}

	uc := NewMigrateStore(source, dest, destInit)
	_, err := uc.Execute(context.Background(), MigrateStoreInput{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save destination task")
}
