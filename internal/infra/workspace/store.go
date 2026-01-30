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

// Store implements WorkspaceRepository for file-based persistence.
type Store struct {
	filePath string
}

// NewStore creates a new workspace store.
// globalCrewDir is typically ~/.config/crew.
func NewStore(globalCrewDir string) *Store {
	return &Store{
		filePath: domain.WorkspacesFilePath(globalCrewDir),
	}
}

// Load reads the workspace file and returns the repos list.
// Returns empty file with version 1 if file doesn't exist.
func (s *Store) Load() (*domain.WorkspaceFile, error) {
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
		// If file is corrupted, start fresh
		if errors.Is(err, domain.ErrWorkspaceFileCorrupted) {
			file = &domain.WorkspaceFile{Version: 1}
		} else {
			return err
		}
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
func (s *Store) RemoveRepo(path string) error {
	// Normalize path for comparison
	absPath, err := normalizePath(path)
	if err != nil {
		absPath = path // Use as-is if normalization fails
	}

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
