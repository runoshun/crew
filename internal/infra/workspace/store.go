// Package workspace provides workspace repository persistence.
package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Ensure Store implements domain.WorkspaceRepository.
var _ domain.WorkspaceRepository = (*Store)(nil)

// ErrNoHomeDir is returned when the home directory cannot be determined.
var ErrNoHomeDir = errors.New("home directory could not be determined")

// Store implements WorkspaceRepository for file-based persistence.
type Store struct {
	filePath string
}

// NewStore creates a new workspace store.
// globalCrewDir is typically ~/.config/crew.
// Returns error if globalCrewDir is empty (cannot determine where to store workspaces).
func NewStore(globalCrewDir string) (*Store, error) {
	if globalCrewDir == "" {
		return nil, ErrNoHomeDir
	}
	filePath := domain.WorkspacesFilePath(globalCrewDir)
	// Ensure the path is absolute to prevent writing to CWD
	if !filepath.IsAbs(filePath) {
		return nil, ErrNoHomeDir
	}
	return &Store{
		filePath: filePath,
	}, nil
}

// Load reads the workspace file and returns the repos list.
// Returns empty file with version 1 if file doesn't exist.
func (s *Store) Load() (*domain.WorkspaceFile, error) {
	if s.filePath == "" {
		return nil, ErrNoHomeDir
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &domain.WorkspaceFile{
				Version: 1,
				Repos:   []domain.WorkspaceRepo{},
			}, nil
		}
		return nil, err
	}

	var file domain.WorkspaceFile
	if err := toml.Unmarshal(data, &file); err != nil {
		// Return error but allow caller to handle it gracefully
		return nil, domain.ErrWorkspaceFileCorrupted
	}

	// Deduplicate repos by path (keep first occurrence)
	file.Repos = deduplicateRepos(file.Repos)

	return &file, nil
}

// Save writes the workspace file.
func (s *Store) Save(file *domain.WorkspaceFile) error {
	if s.filePath == "" {
		return ErrNoHomeDir
	}

	// Ensure directory exists with proper permissions (0700)
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Sort repos: pinned first, then by last_opened desc, then by name asc
	sortRepos(file.Repos)

	data, err := toml.Marshal(file)
	if err != nil {
		return err
	}

	// Write with 0600 permissions (user read/write only)
	return os.WriteFile(s.filePath, data, 0600)
}

// AddRepo adds a repository to the workspace.
// The path is normalized to an absolute path and resolved to git repo root.
func (s *Store) AddRepo(path string) error {
	// Normalize path
	absPath, err := normalizePath(path)
	if err != nil {
		return domain.ErrWorkspaceInvalidPath
	}

	// Resolve to git root if the path is a subdirectory
	gitRoot, err := resolveGitRoot(absPath)
	if err != nil {
		return domain.ErrWorkspaceInvalidPath
	}

	// Load current file
	file, err := s.Load()
	if err != nil {
		// If file is corrupted, don't overwrite - return error
		// This prevents data loss per spec: "壊れている場合は warning + 空リストで起動（ファイルは上書きしない）"
		return err
	}

	// Check for duplicates
	for _, repo := range file.Repos {
		if repo.Path == gitRoot {
			return domain.ErrWorkspaceRepoExists
		}
	}

	// Add new repo
	file.Repos = append(file.Repos, domain.WorkspaceRepo{
		Path: gitRoot,
	})

	return s.Save(file)
}

// RemoveRepo removes a repository from the workspace by path.
// Like AddRepo, this resolves the path to the git repository root,
// so removing via a subdirectory path works correctly.
func (s *Store) RemoveRepo(path string) error {
	// Normalize path
	absPath, err := normalizePath(path)
	if err != nil {
		absPath = path // Use as-is if normalization fails
	}

	// Try to resolve to git root (like AddRepo does)
	// This allows removing by subdir path
	gitRoot, err := resolveGitRoot(absPath)
	if err == nil {
		absPath = gitRoot
	}
	// If resolveGitRoot fails (path doesn't exist or isn't a git repo),
	// we still try to match against the normalized path

	file, err := s.Load()
	if err != nil {
		return err
	}

	found := false
	newRepos := make([]domain.WorkspaceRepo, 0, len(file.Repos))
	for _, repo := range file.Repos {
		if repo.Path == absPath {
			found = true
			continue
		}
		newRepos = append(newRepos, repo)
	}

	if !found {
		return domain.ErrWorkspaceRepoNotFound
	}

	file.Repos = newRepos
	return s.Save(file)
}

// UpdateLastOpened updates the last_opened timestamp for a repo.
func (s *Store) UpdateLastOpened(path string) error {
	absPath, err := normalizePath(path)
	if err != nil {
		absPath = path
	}

	file, err := s.Load()
	if err != nil {
		return err
	}

	found := false
	for i := range file.Repos {
		if file.Repos[i].Path == absPath {
			file.Repos[i].LastOpened = time.Now()
			found = true
			break
		}
	}

	if !found {
		return domain.ErrWorkspaceRepoNotFound
	}

	return s.Save(file)
}

// normalizePath converts a path to an absolute, cleaned path.
func normalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

// resolveGitRoot finds the git repository root from a path.
// Returns the path if it's already the root, or walks up to find .git.
func resolveGitRoot(path string) (string, error) {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	// If it's a file, start from its directory
	if !info.IsDir() {
		path = filepath.Dir(path)
	}

	// Walk up looking for .git
	current := path
	for {
		gitPath := filepath.Join(current, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			return "", domain.ErrWorkspaceInvalidPath
		}
		current = parent
	}
}

// deduplicateRepos removes duplicate repos by path, keeping the first occurrence.
func deduplicateRepos(repos []domain.WorkspaceRepo) []domain.WorkspaceRepo {
	seen := make(map[string]bool)
	result := make([]domain.WorkspaceRepo, 0, len(repos))
	for _, repo := range repos {
		if !seen[repo.Path] {
			seen[repo.Path] = true
			result = append(result, repo)
		}
	}
	return result
}

// sortRepos sorts repos by: pinned desc, last_opened desc, name asc.
func sortRepos(repos []domain.WorkspaceRepo) {
	sort.SliceStable(repos, func(i, j int) bool {
		// Pinned repos first
		if repos[i].Pinned != repos[j].Pinned {
			return repos[i].Pinned
		}

		// Then by last_opened (most recent first)
		if !repos[i].LastOpened.Equal(repos[j].LastOpened) {
			return repos[i].LastOpened.After(repos[j].LastOpened)
		}

		// Finally by display name
		return repos[i].DisplayName() < repos[j].DisplayName()
	})
}
