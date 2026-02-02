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

	comment := domain.Comment{
		Text:     "First",
		Author:   "worker",
		Time:     now,
		Type:     domain.CommentTypeReport,
		Tags:     []string{"docs"},
		Metadata: map[string]string{"source": "cli"},
	}
	require.NoError(t, store.AddComment(1, comment))

	comments, err := store.GetComments(1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "First", comments[0].Text)
	assert.Equal(t, domain.CommentTypeReport, comments[0].Type)
	assert.Equal(t, []string{"docs"}, comments[0].Tags)
	assert.Equal(t, map[string]string{"source": "cli"}, comments[0].Metadata)

	updated := domain.Comment{
		Text:     "Updated",
		Author:   "worker",
		Time:     now,
		Type:     domain.CommentTypeFriction,
		Tags:     []string{"testing"},
		Metadata: map[string]string{"priority": "high"},
	}
	require.NoError(t, store.UpdateComment(1, 0, updated))

	comments, err = store.GetComments(1)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "Updated", comments[0].Text)
	assert.Equal(t, domain.CommentTypeFriction, comments[0].Type)
	assert.Equal(t, []string{"testing"}, comments[0].Tags)
	assert.Equal(t, map[string]string{"priority": "high"}, comments[0].Metadata)
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

func TestStore_ListAll(t *testing.T) {
	crewDir := filepath.Join(t.TempDir(), ".crew")
	storeAlpha := New(crewDir, "alpha")
	storeBeta := New(crewDir, "beta")

	_, err := storeAlpha.Initialize()
	require.NoError(t, err)
	_, err = storeBeta.Initialize()
	require.NoError(t, err)

	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	require.NoError(t, storeAlpha.Save(&domain.Task{
		ID:            1,
		Title:         "Alpha task",
		Description:   "Alpha body",
		Status:        domain.StatusTodo,
		Created:       now,
		BaseBranch:    "main",
		StatusVersion: domain.StatusVersionCurrent,
	}))
	require.NoError(t, storeBeta.Save(&domain.Task{
		ID:            1,
		Title:         "Beta task",
		Description:   "Beta body",
		Status:        domain.StatusTodo,
		Created:       now,
		BaseBranch:    "main",
		StatusVersion: domain.StatusVersionCurrent,
	}))

	tasks, err := storeAlpha.ListAll(domain.TaskFilter{})
	require.NoError(t, err)
	require.Len(t, tasks, 2)

	assert.Equal(t, "alpha", tasks[0].Namespace)
	assert.Equal(t, 1, tasks[0].ID)
	assert.Equal(t, "beta", tasks[1].Namespace)
	assert.Equal(t, 1, tasks[1].ID)
}

func TestStore_List_IgnoresCommentMarkersInCodeFence(t *testing.T) {
	crewDir := filepath.Join(t.TempDir(), ".crew")
	store := New(crewDir, "default")
	_, err := store.Initialize()
	require.NoError(t, err)

	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	task := &domain.Task{
		ID:            1,
		Title:         "Has example",
		Description:   "before\n\n~~~\n---\n# Comment: 0\n# Author: reviewer\n# Time: 2026-...\ntext\n~~~\n\nafter",
		Status:        domain.StatusTodo,
		Created:       now,
		BaseBranch:    "main",
		StatusVersion: domain.StatusVersionCurrent,
	}
	require.NoError(t, store.Save(task))

	tasks, err := store.List(domain.TaskFilter{})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, 1, tasks[0].ID)
	assert.Equal(t, domain.StatusTodo, tasks[0].Status)
}

func TestStore_List_DoesNotTreatInvalidCommentHeaderAsComments(t *testing.T) {
	crewDir := filepath.Join(t.TempDir(), ".crew")
	store := New(crewDir, "default")
	_, err := store.Initialize()
	require.NoError(t, err)

	mdPath := filepath.Join(crewDir, "tasks", "default", "1.md")
	metaPath := filepath.Join(crewDir, "tasks", "default", "1.meta.json")

	md := "---\ntitle: Task\n---\n\nBody\n\n---\n# Comment: 0\n# Author: reviewer\n# Time: 2026-...\n\nNot a real comment"
	require.NoError(t, os.WriteFile(mdPath, []byte(md), 0o644))

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

	tasks, err := store.List(domain.TaskFilter{})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
}

func TestStore_ListWithLabelFilter(t *testing.T) {
	crewDir := filepath.Join(t.TempDir(), ".crew")
	store := New(crewDir, "default")
	_, err := store.Initialize()
	require.NoError(t, err)

	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	tasks := []*domain.Task{
		{ID: 1, Title: "Task 1", Status: domain.StatusTodo, Created: now, BaseBranch: "main", Labels: []string{"bug"}, StatusVersion: domain.StatusVersionCurrent},
		{ID: 2, Title: "Task 2", Status: domain.StatusTodo, Created: now, BaseBranch: "main", Labels: []string{"feature"}, StatusVersion: domain.StatusVersionCurrent},
		{ID: 3, Title: "Task 3", Status: domain.StatusTodo, Created: now, BaseBranch: "main", Labels: []string{"bug", "feature"}, StatusVersion: domain.StatusVersionCurrent},
	}
	for _, task := range tasks {
		require.NoError(t, store.Save(task))
	}

	bugTasks, err := store.List(domain.TaskFilter{Labels: []string{"bug"}})
	require.NoError(t, err)
	require.Len(t, bugTasks, 2)
	assert.Equal(t, 1, bugTasks[0].ID)
	assert.Equal(t, 3, bugTasks[1].ID)

	// Label filter uses AND semantics (task must contain all labels).
	both, err := store.List(domain.TaskFilter{Labels: []string{"bug", "feature"}})
	require.NoError(t, err)
	require.Len(t, both, 1)
	assert.Equal(t, 3, both[0].ID)
}

func TestStore_ListWithParentFilter(t *testing.T) {
	crewDir := filepath.Join(t.TempDir(), ".crew")
	store := New(crewDir, "default")
	_, err := store.Initialize()
	require.NoError(t, err)

	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	parentID := 1
	tasks := []*domain.Task{
		{ID: 1, Title: "Parent", Status: domain.StatusTodo, Created: now, BaseBranch: "main", StatusVersion: domain.StatusVersionCurrent},
		{ID: 2, Title: "Child 1", ParentID: &parentID, Status: domain.StatusTodo, Created: now, BaseBranch: "main", StatusVersion: domain.StatusVersionCurrent},
		{ID: 3, Title: "Child 2", ParentID: &parentID, Status: domain.StatusTodo, Created: now, BaseBranch: "main", StatusVersion: domain.StatusVersionCurrent},
		{ID: 4, Title: "Orphan", Status: domain.StatusTodo, Created: now, BaseBranch: "main", StatusVersion: domain.StatusVersionCurrent},
	}
	for _, task := range tasks {
		require.NoError(t, store.Save(task))
	}

	children, err := store.List(domain.TaskFilter{ParentID: &parentID})
	require.NoError(t, err)
	require.Len(t, children, 2)
	assert.Equal(t, 2, children[0].ID)
	assert.Equal(t, 3, children[1].ID)
}

func TestStore_GetChildren(t *testing.T) {
	crewDir := filepath.Join(t.TempDir(), ".crew")
	store := New(crewDir, "default")
	_, err := store.Initialize()
	require.NoError(t, err)

	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	parentID := 1
	for _, task := range []*domain.Task{
		{ID: 1, Title: "Parent", Status: domain.StatusTodo, Created: now, BaseBranch: "main", StatusVersion: domain.StatusVersionCurrent},
		{ID: 2, Title: "Child 1", ParentID: &parentID, Status: domain.StatusTodo, Created: now, BaseBranch: "main", StatusVersion: domain.StatusVersionCurrent},
		{ID: 3, Title: "Child 2", ParentID: &parentID, Status: domain.StatusTodo, Created: now, BaseBranch: "main", StatusVersion: domain.StatusVersionCurrent},
	} {
		require.NoError(t, store.Save(task))
	}

	children, err := store.GetChildren(1)
	require.NoError(t, err)
	require.Len(t, children, 2)
	assert.Equal(t, 2, children[0].ID)
	assert.Equal(t, 3, children[1].ID)
}

func TestStore_Delete(t *testing.T) {
	crewDir := filepath.Join(t.TempDir(), ".crew")
	store := New(crewDir, "default")
	_, err := store.Initialize()
	require.NoError(t, err)

	now := time.Date(2026, 1, 18, 10, 0, 0, 0, time.UTC)
	task := &domain.Task{
		ID:            1,
		Title:         "To Delete",
		Status:        domain.StatusTodo,
		Created:       now,
		BaseBranch:    "main",
		StatusVersion: domain.StatusVersionCurrent,
	}
	require.NoError(t, store.Save(task))

	comment := domain.Comment{Text: "Test comment", Author: "worker", Time: now}
	require.NoError(t, store.AddComment(1, comment))

	require.NoError(t, store.Delete(1))

	loaded, err := store.Get(1)
	require.NoError(t, err)
	assert.Nil(t, loaded)

	comments, err := store.GetComments(1)
	require.NoError(t, err)
	assert.Empty(t, comments)
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
