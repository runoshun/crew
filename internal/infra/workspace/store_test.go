package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

func TestNewStore_EmptyDir(t *testing.T) {
	_, err := NewStore("")
	assert.ErrorIs(t, err, ErrNoHomeDir)
}

func TestNewStore_RelativePath(t *testing.T) {
	// Relative path should fail
	_, err := NewStore("relative/path")
	assert.ErrorIs(t, err, ErrNoHomeDir)
}

func TestStore_LoadEmpty(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	file, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, 1, file.Version)
	assert.Empty(t, file.Repos)
}

func TestStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	// Create a test file
	file := &domain.WorkspaceFile{
		Version: 1,
		Repos: []domain.WorkspaceRepo{
			{Path: "/path/to/repo1", Name: "repo1"},
			{Path: "/path/to/repo2", Name: ""},
		},
	}

	// Save
	err = store.Save(file)
	require.NoError(t, err)

	// Verify file was created with correct permissions
	info, err := os.Stat(domain.WorkspacesFilePath(dir))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Load
	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Version)
	assert.Len(t, loaded.Repos, 2)
}

func TestStore_AddRepo(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	// Create a git repo to add
	repoDir := t.TempDir()
	gitDir := filepath.Join(repoDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))

	// Add repo
	err = store.AddRepo(repoDir)
	require.NoError(t, err)

	// Verify it was added
	file, err := store.Load()
	require.NoError(t, err)
	require.Len(t, file.Repos, 1)
	assert.Equal(t, repoDir, file.Repos[0].Path)

	// Adding same repo again should fail
	err = store.AddRepo(repoDir)
	assert.ErrorIs(t, err, domain.ErrWorkspaceRepoExists)
}

func TestStore_AddRepoSubdirectory(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	// Create a git repo with subdirectory
	repoDir := t.TempDir()
	gitDir := filepath.Join(repoDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))

	subDir := filepath.Join(repoDir, "src", "pkg")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	// Add via subdirectory - should resolve to repo root
	err = store.AddRepo(subDir)
	require.NoError(t, err)

	// Verify it was added with the repo root path
	file, err := store.Load()
	require.NoError(t, err)
	require.Len(t, file.Repos, 1)
	assert.Equal(t, repoDir, file.Repos[0].Path)
}

func TestStore_AddRepoInvalidPath(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	// Try to add non-existent path
	err = store.AddRepo("/non/existent/path")
	assert.ErrorIs(t, err, domain.ErrWorkspaceInvalidPath)

	// Try to add non-git directory
	nonGitDir := t.TempDir()
	err = store.AddRepo(nonGitDir)
	assert.ErrorIs(t, err, domain.ErrWorkspaceInvalidPath)
}

func TestStore_AddRepoCorruptedFile(t *testing.T) {
	dir := t.TempDir()

	// Write a corrupted file
	filePath := domain.WorkspacesFilePath(dir)
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
	require.NoError(t, os.WriteFile(filePath, []byte("this is not valid toml [[["), 0600))

	store, err := NewStore(dir)
	require.NoError(t, err)

	// Create a git repo to add
	repoDir := t.TempDir()
	gitDir := filepath.Join(repoDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))

	// AddRepo should fail with corruption error, not overwrite
	err = store.AddRepo(repoDir)
	assert.ErrorIs(t, err, domain.ErrWorkspaceFileCorrupted)

	// Verify file was not overwritten
	content, _ := os.ReadFile(filePath)
	assert.Contains(t, string(content), "this is not valid toml")
}

func TestStore_RemoveRepo(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	// Create initial file with repos
	file := &domain.WorkspaceFile{
		Version: 1,
		Repos: []domain.WorkspaceRepo{
			{Path: "/path/to/repo1"},
			{Path: "/path/to/repo2"},
			{Path: "/path/to/repo3"},
		},
	}
	require.NoError(t, store.Save(file))

	// Remove middle repo
	err = store.RemoveRepo("/path/to/repo2")
	require.NoError(t, err)

	// Verify it was removed
	loaded, err := store.Load()
	require.NoError(t, err)
	require.Len(t, loaded.Repos, 2)
	assert.Equal(t, "/path/to/repo1", loaded.Repos[0].Path)
	assert.Equal(t, "/path/to/repo3", loaded.Repos[1].Path)

	// Removing non-existent repo should fail
	err = store.RemoveRepo("/path/to/nonexistent")
	assert.ErrorIs(t, err, domain.ErrWorkspaceRepoNotFound)
}

func TestStore_RemoveRepoBySubdirectory(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	// Create a git repo with subdirectory
	repoDir := t.TempDir()
	gitDir := filepath.Join(repoDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))

	subDir := filepath.Join(repoDir, "src", "pkg")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	// Add repo (will be stored as repoDir)
	err = store.AddRepo(repoDir)
	require.NoError(t, err)

	// Verify it was added
	file, err := store.Load()
	require.NoError(t, err)
	require.Len(t, file.Repos, 1)

	// Remove via subdirectory - should resolve to repo root and match
	err = store.RemoveRepo(subDir)
	require.NoError(t, err)

	// Verify it was removed
	file, err = store.Load()
	require.NoError(t, err)
	assert.Empty(t, file.Repos)
}

func TestStore_UpdateLastOpened(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	// Create a real path that exists for normalization to work
	repoPath := t.TempDir()

	// Create initial file with the real path
	file := &domain.WorkspaceFile{
		Version: 1,
		Repos: []domain.WorkspaceRepo{
			{Path: repoPath},
		},
	}
	require.NoError(t, store.Save(file))

	// Update last opened
	err = store.UpdateLastOpened(repoPath)
	require.NoError(t, err)

	// Verify it was updated
	loaded, err := store.Load()
	require.NoError(t, err)
	require.Len(t, loaded.Repos, 1)
	assert.True(t, !loaded.Repos[0].LastOpened.IsZero(), "LastOpened should be set")

	// Updating non-existent repo should fail
	err = store.UpdateLastOpened("/path/to/nonexistent")
	assert.ErrorIs(t, err, domain.ErrWorkspaceRepoNotFound)
}

func TestStore_Deduplication(t *testing.T) {
	dir := t.TempDir()

	// Write file with duplicates directly
	content := `version = 1

[[repos]]
path = "/path/to/repo"
name = "first"

[[repos]]
path = "/path/to/repo"
name = "second"
`
	filePath := domain.WorkspacesFilePath(dir)
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0600))

	store, err := NewStore(dir)
	require.NoError(t, err)

	// Load should deduplicate (keep first)
	file, err := store.Load()
	require.NoError(t, err)
	require.Len(t, file.Repos, 1)
	assert.Equal(t, "first", file.Repos[0].Name)
}

func TestStore_SortOrder(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	// Create file with unsorted repos
	file := &domain.WorkspaceFile{
		Version: 1,
		Repos: []domain.WorkspaceRepo{
			{Path: "/c", Name: "c-repo"},
			{Path: "/a", Name: "a-repo", Pinned: true},
			{Path: "/b", Name: "b-repo"},
		},
	}
	require.NoError(t, store.Save(file))

	// Load - should be sorted: pinned first, then by name
	loaded, err := store.Load()
	require.NoError(t, err)
	require.Len(t, loaded.Repos, 3)
	assert.Equal(t, "/a", loaded.Repos[0].Path) // pinned
	assert.Equal(t, "/b", loaded.Repos[1].Path) // by name
	assert.Equal(t, "/c", loaded.Repos[2].Path) // by name
}
