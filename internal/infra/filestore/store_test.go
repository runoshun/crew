package filestore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Initialize_RepairsNextID(t *testing.T) {
	crewDir := filepath.Join(t.TempDir(), ".crew")
	store := New(crewDir, "default")

	repaired, err := store.Initialize()
	require.NoError(t, err)
	assert.False(t, repaired)

	metaPath := filepath.Join(crewDir, "tasks", "default", "meta.json")
	meta := readNamespaceMetaForTest(t, metaPath)
	assert.Equal(t, 1, meta.NextID)
	assert.Equal(t, "default", meta.Namespace)

	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	task := &domain.Task{
		ID:            2,
		Title:         "Repair test",
		Description:   "Description",
		Status:        domain.StatusTodo,
		Created:       now,
		BaseBranch:    "main",
		StatusVersion: domain.StatusVersionCurrent,
	}
	require.NoError(t, store.Save(task))

	meta.NextID = 1
	writeNamespaceMetaForTest(t, metaPath, meta)

	repaired, err = store.Initialize()
	require.NoError(t, err)
	assert.True(t, repaired)

	meta = readNamespaceMetaForTest(t, metaPath)
	assert.Equal(t, 3, meta.NextID)
}

func TestStore_Save_Get_Comments(t *testing.T) {
	crewDir := filepath.Join(t.TempDir(), ".crew")
	store := New(crewDir, "default")
	_, err := store.Initialize()
	require.NoError(t, err)

	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	parentID := 10
	skipReview := true

	task := &domain.Task{
		ID:            1,
		Title:         "Task",
		Description:   "Body",
		Labels:        []string{"bug", "urgent"},
		ParentID:      &parentID,
		SkipReview:    &skipReview,
		Status:        domain.StatusTodo,
		Created:       now,
		BaseBranch:    "main",
		StatusVersion: domain.StatusVersionCurrent,
	}
	require.NoError(t, store.Save(task))

	loaded, err := store.Get(1)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "Task", loaded.Title)
	assert.Equal(t, "Body", loaded.Description)
	assert.Equal(t, []string{"bug", "urgent"}, loaded.Labels)
	require.NotNil(t, loaded.ParentID)
	assert.Equal(t, parentID, *loaded.ParentID)
	require.NotNil(t, loaded.SkipReview)
	assert.True(t, *loaded.SkipReview)

	comment := domain.Comment{Text: "First", Author: "worker", Time: now}
	require.NoError(t, store.AddComment(1, comment))

	comments, err := store.GetComments(1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "First", comments[0].Text)

	updated := domain.Comment{Text: "Updated", Author: "worker", Time: now}
	require.NoError(t, store.UpdateComment(1, 0, updated))

	comments, err = store.GetComments(1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "Updated", comments[0].Text)
}

func TestStore_StrictValidation(t *testing.T) {
	crewDir := filepath.Join(t.TempDir(), ".crew")
	store := New(crewDir, "default")
	_, err := store.Initialize()
	require.NoError(t, err)

	mdPath := filepath.Join(crewDir, "tasks", "default", "1.md")
	metaPath := filepath.Join(crewDir, "tasks", "default", "1.meta.json")

	mdContent := "---\ntitle: Task\nunknown: value\n---\n\nBody"
	require.NoError(t, os.WriteFile(mdPath, []byte(mdContent), 0o644))

	meta := taskMetaPayload{
		Schema:        intPtr(taskMetaSchema),
		Status:        strPtr(string(domain.StatusTodo)),
		Created:       strPtr(time.Now().UTC().Format(time.RFC3339)),
		BaseBranch:    strPtr("main"),
		StatusVersion: intPtr(domain.StatusVersionCurrent),
	}
	metaBytes, err := json.Marshal(meta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(metaPath, metaBytes, 0o644))

	_, err = store.Get(1)
	require.Error(t, err)

	mdContent = "---\ntitle: Task\n---\n\nBody"
	require.NoError(t, os.WriteFile(mdPath, []byte(mdContent), 0o644))

	badMeta := taskMetaPayload{
		Schema:        intPtr(taskMetaSchema),
		Status:        strPtr("invalid"),
		Created:       strPtr(time.Now().UTC().Format(time.RFC3339)),
		BaseBranch:    strPtr("main"),
		StatusVersion: intPtr(domain.StatusVersionCurrent),
	}
	metaBytes, err = json.Marshal(badMeta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(metaPath, metaBytes, 0o644))

	_, err = store.Get(1)
	require.Error(t, err)
}

type namespaceMetaFile struct {
	Schema    int    `json:"schema"`
	Namespace string `json:"namespace"`
	NextID    int    `json:"next_id"`
}

func readNamespaceMetaForTest(t *testing.T, path string) namespaceMetaFile {
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	var meta namespaceMetaFile
	require.NoError(t, json.Unmarshal(content, &meta))
	return meta
}

func writeNamespaceMetaForTest(t *testing.T, path string, meta namespaceMetaFile) {
	content, err := json.MarshalIndent(meta, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, content, 0o644))
}
