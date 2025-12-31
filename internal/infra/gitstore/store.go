// Package gitstore provides a Git plumbing-based implementation of TaskRepository.
package gitstore

import (
	"fmt"
	"slices"
	"strconv"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"gopkg.in/yaml.v3"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Store implements domain.TaskRepository using Git plumbing (refs and blobs).
//
// Data structure:
//
//	refs/<namespace>/
//	  meta      → blob (nextTaskID, etc.)
//	  current   → latest snapshot ref
//	  tasks/
//	    <id>    → blob (task YAML)
//	  comments/
//	    <id>    → blob (comments YAML)
//	  snapshots/
//	    <main-sha>_<seq> → tree
type Store struct {
	repo      *git.Repository
	namespace string // e.g., "crew-shun"
	mu        sync.RWMutex
}

// meta contains store metadata.
type meta struct {
	NextTaskID int `yaml:"nextTaskID"`
}

// commentsData holds comments for a task.
type commentsData struct {
	Comments []domain.Comment `yaml:"comments"`
}

// New creates a new Store for the given repository.
func New(repoPath, namespace string) (*Store, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("open git repository: %w", err)
	}

	return &Store{
		repo:      repo,
		namespace: namespace,
	}, nil
}

// NewWithRepo creates a new Store with an existing repository instance.
func NewWithRepo(repo *git.Repository, namespace string) *Store {
	return &Store{
		repo:      repo,
		namespace: namespace,
	}
}

// refPrefix returns the ref prefix for this namespace.
func (s *Store) refPrefix() string {
	return "refs/" + s.namespace + "/"
}

// taskRef returns the ref name for a task.
func (s *Store) taskRef(id int) plumbing.ReferenceName {
	return plumbing.ReferenceName(s.refPrefix() + "tasks/" + strconv.Itoa(id))
}

// commentsRef returns the ref name for task comments.
func (s *Store) commentsRef(id int) plumbing.ReferenceName {
	return plumbing.ReferenceName(s.refPrefix() + "comments/" + strconv.Itoa(id))
}

// metaRef returns the ref name for metadata.
func (s *Store) metaRef() plumbing.ReferenceName {
	return plumbing.ReferenceName(s.refPrefix() + "meta")
}

// initializedRef returns the ref name for the initialized marker.
func (s *Store) initializedRef() plumbing.ReferenceName {
	return plumbing.ReferenceName(s.refPrefix() + "initialized")
}

// Get retrieves a task by ID.
func (s *Store) Get(id int) (*domain.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ref, err := s.repo.Reference(s.taskRef(id), true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("get task ref: %w", err)
	}

	blob, err := s.repo.BlobObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("get task blob: %w", err)
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("read task blob: %w", err)
	}
	defer func() { _ = reader.Close() }()

	var task domain.Task
	if err := yaml.NewDecoder(reader).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode task: %w", err)
	}
	task.ID = id

	return &task, nil
}

// List retrieves tasks matching the filter.
func (s *Store) List(filter domain.TaskFilter) ([]*domain.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*domain.Task
	prefix := s.refPrefix() + "tasks/"

	refs, err := s.repo.References()
	if err != nil {
		return nil, fmt.Errorf("list refs: %w", err)
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		refName := string(ref.Name())
		if len(refName) <= len(prefix) || refName[:len(prefix)] != prefix {
			return nil
		}

		idStr := refName[len(prefix):]
		taskID, parseErr := strconv.Atoi(idStr)
		if parseErr != nil {
			return nil // Skip invalid refs
		}

		blob, blobErr := s.repo.BlobObject(ref.Hash())
		if blobErr != nil {
			return fmt.Errorf("get task blob: %w", blobErr)
		}

		reader, readerErr := blob.Reader()
		if readerErr != nil {
			return fmt.Errorf("read task blob: %w", readerErr)
		}
		defer func() { _ = reader.Close() }()

		var task domain.Task
		if decodeErr := yaml.NewDecoder(reader).Decode(&task); decodeErr != nil {
			return fmt.Errorf("decode task: %w", decodeErr)
		}
		task.ID = taskID

		// Apply ParentID filter
		if filter.ParentID != nil {
			if task.ParentID == nil || *task.ParentID != *filter.ParentID {
				return nil
			}
		}

		// Apply Labels filter (AND condition)
		if len(filter.Labels) > 0 {
			if !containsAll(task.Labels, filter.Labels) {
				return nil
			}
		}

		tasks = append(tasks, &task)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by ID for consistent ordering
	slices.SortFunc(tasks, func(a, b *domain.Task) int {
		return a.ID - b.ID
	})

	return tasks, nil
}

// GetChildren retrieves direct children of a task.
func (s *Store) GetChildren(parentID int) ([]*domain.Task, error) {
	return s.List(domain.TaskFilter{ParentID: &parentID})
}

// Save creates or updates a task.
func (s *Store) Save(task *domain.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize task to YAML
	data, err := yaml.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	// Create blob
	hash, err := s.writeBlob(data)
	if err != nil {
		return err
	}

	// Update ref
	ref := plumbing.NewHashReference(s.taskRef(task.ID), hash)
	if err := s.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("set task ref: %w", err)
	}

	return nil
}

// Delete removes a task.
func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete task ref
	if err := s.repo.Storer.RemoveReference(s.taskRef(id)); err != nil {
		if err != plumbing.ErrReferenceNotFound {
			return fmt.Errorf("remove task ref: %w", err)
		}
	}

	// Also delete comments ref if exists
	if err := s.repo.Storer.RemoveReference(s.commentsRef(id)); err != nil {
		if err != plumbing.ErrReferenceNotFound {
			return fmt.Errorf("remove comments ref: %w", err)
		}
	}

	return nil
}

// NextID returns the next available task ID.
func (s *Store) NextID() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, err := s.loadMeta()
	if err != nil {
		return 0, err
	}

	id := m.NextTaskID
	m.NextTaskID++

	if err := s.saveMeta(m); err != nil {
		return 0, err
	}

	return id, nil
}

// GetComments retrieves comments for a task.
func (s *Store) GetComments(taskID int) ([]domain.Comment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getCommentsLocked(taskID)
}

// AddComment adds a comment to a task.
func (s *Store) AddComment(taskID int, comment domain.Comment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load existing comments
	comments, err := s.getCommentsLocked(taskID)
	if err != nil {
		return err
	}

	// Add new comment
	comments = append(comments, comment)

	// Save comments
	data := commentsData{Comments: comments}
	yamlData, err := yaml.Marshal(&data)
	if err != nil {
		return fmt.Errorf("marshal comments: %w", err)
	}

	hash, err := s.writeBlob(yamlData)
	if err != nil {
		return err
	}

	ref := plumbing.NewHashReference(s.commentsRef(taskID), hash)
	if err := s.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("set comments ref: %w", err)
	}

	return nil
}

// getCommentsLocked loads comments without locking (caller must hold lock).
func (s *Store) getCommentsLocked(taskID int) ([]domain.Comment, error) {
	ref, err := s.repo.Reference(s.commentsRef(taskID), true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return []domain.Comment{}, nil
		}
		return nil, fmt.Errorf("get comments ref: %w", err)
	}

	blob, err := s.repo.BlobObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("get comments blob: %w", err)
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("read comments blob: %w", err)
	}
	defer func() { _ = reader.Close() }()

	var data commentsData
	if err := yaml.NewDecoder(reader).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode comments: %w", err)
	}

	return data.Comments, nil
}

// loadMeta loads metadata from the meta ref.
func (s *Store) loadMeta() (*meta, error) {
	ref, err := s.repo.Reference(s.metaRef(), true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return &meta{NextTaskID: 1}, nil
		}
		return nil, fmt.Errorf("get meta ref: %w", err)
	}

	blob, err := s.repo.BlobObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("get meta blob: %w", err)
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("read meta blob: %w", err)
	}
	defer func() { _ = reader.Close() }()

	var m meta
	if err := yaml.NewDecoder(reader).Decode(&m); err != nil {
		return nil, fmt.Errorf("decode meta: %w", err)
	}

	return &m, nil
}

// saveMeta saves metadata to the meta ref.
func (s *Store) saveMeta(m *meta) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}

	hash, err := s.writeBlob(data)
	if err != nil {
		return err
	}

	ref := plumbing.NewHashReference(s.metaRef(), hash)
	if err := s.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("set meta ref: %w", err)
	}

	return nil
}

// writeBlob writes data to a blob and returns the hash.
func (s *Store) writeBlob(data []byte) (plumbing.Hash, error) {
	obj := s.repo.Storer.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(int64(len(data)))

	writer, err := obj.Writer()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("create blob writer: %w", err)
	}

	if _, writeErr := writer.Write(data); writeErr != nil {
		_ = writer.Close()
		return plumbing.ZeroHash, fmt.Errorf("write blob: %w", writeErr)
	}
	_ = writer.Close()

	hash, err := s.repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("store blob: %w", err)
	}

	return hash, nil
}

// Initialize creates initial metadata if it doesn't exist.
func (s *Store) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already initialized
	_, err := s.repo.Reference(s.initializedRef(), true)
	if err == nil {
		return nil // Already initialized
	}
	if err != plumbing.ErrReferenceNotFound {
		return fmt.Errorf("check initialized ref: %w", err)
	}

	// Create initial metadata
	m := &meta{NextTaskID: 1}
	if err := s.saveMeta(m); err != nil {
		return err
	}

	// Create initialized marker (empty blob)
	hash, err := s.writeBlob([]byte("initialized"))
	if err != nil {
		return err
	}
	ref := plumbing.NewHashReference(s.initializedRef(), hash)
	if err := s.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("set initialized ref: %w", err)
	}

	return nil
}

// IsInitialized checks if the store has been initialized.
func (s *Store) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := s.repo.Reference(s.initializedRef(), true)
	return err == nil
}

// containsAll checks if slice contains all elements in required.
func containsAll(slice, required []string) bool {
	for _, r := range required {
		found := false
		for _, item := range slice {
			if item == r {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Ensure Store implements TaskRepository.
var _ domain.TaskRepository = (*Store)(nil)

// Ensure Store implements StoreInitializer.
var _ domain.StoreInitializer = (*Store)(nil)

// === Snapshot operations ===

// snapshotRef returns the ref name for a snapshot.
func (s *Store) snapshotRef(mainSHA string, seq int) plumbing.ReferenceName {
	return plumbing.ReferenceName(fmt.Sprintf("%ssnapshots/%s_%03d", s.refPrefix(), mainSHA, seq))
}

// currentRef returns the ref name for the current snapshot pointer.
func (s *Store) currentRef() plumbing.ReferenceName {
	return plumbing.ReferenceName(s.refPrefix() + "current")
}

// SaveSnapshot saves the current task state as a snapshot.
func (s *Store) SaveSnapshot(mainSHA string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find next sequence number for this mainSHA
	seq := 1
	snapshots, err := s.listSnapshotsLocked(mainSHA)
	if err != nil {
		return err
	}
	if len(snapshots) > 0 {
		seq = snapshots[len(snapshots)-1].Seq + 1
	}

	// Build tree from current tasks
	treeHash, err := s.buildTasksTree()
	if err != nil {
		return err
	}

	// Create snapshot ref
	snapshotRefName := s.snapshotRef(mainSHA, seq)
	ref := plumbing.NewHashReference(snapshotRefName, treeHash)
	if err := s.repo.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("set snapshot ref: %w", err)
	}

	// Update current to point to this snapshot
	currentRefObj := plumbing.NewSymbolicReference(s.currentRef(), snapshotRefName)
	if err := s.repo.Storer.SetReference(currentRefObj); err != nil {
		return fmt.Errorf("set current ref: %w", err)
	}

	return nil
}

// buildTasksTree creates a tree object from all current tasks.
func (s *Store) buildTasksTree() (plumbing.Hash, error) {
	var entries []object.TreeEntry
	prefix := s.refPrefix() + "tasks/"

	refs, err := s.repo.References()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("list refs: %w", err)
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		refName := string(ref.Name())
		if len(refName) <= len(prefix) || refName[:len(prefix)] != prefix {
			return nil
		}

		idStr := refName[len(prefix):]
		entries = append(entries, object.TreeEntry{
			Name: idStr,
			Mode: filemode.Regular,
			Hash: ref.Hash(),
		})
		return nil
	})
	if err != nil {
		return plumbing.ZeroHash, err
	}

	// Sort entries by name for consistent tree hash
	slices.SortFunc(entries, func(a, b object.TreeEntry) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	// Create tree object
	tree := &object.Tree{}
	tree.Entries = entries

	obj := s.repo.Storer.NewEncodedObject()
	if err := tree.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encode tree: %w", err)
	}

	hash, err := s.repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("store tree: %w", err)
	}

	return hash, nil
}

// RestoreSnapshot restores tasks from a snapshot.
func (s *Store) RestoreSnapshot(snapshotRefStr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get snapshot tree
	snapshotRefName := plumbing.ReferenceName(snapshotRefStr)
	ref, err := s.repo.Reference(snapshotRefName, true)
	if err != nil {
		return fmt.Errorf("get snapshot ref: %w", err)
	}

	tree, err := s.repo.TreeObject(ref.Hash())
	if err != nil {
		return fmt.Errorf("get snapshot tree: %w", err)
	}

	// Delete all current task refs
	if err := s.deleteAllTaskRefs(); err != nil {
		return err
	}

	// Restore tasks from tree
	for _, entry := range tree.Entries {
		taskRefName := plumbing.ReferenceName(s.refPrefix() + "tasks/" + entry.Name)
		taskRef := plumbing.NewHashReference(taskRefName, entry.Hash)
		if err := s.repo.Storer.SetReference(taskRef); err != nil {
			return fmt.Errorf("restore task ref %s: %w", entry.Name, err)
		}
	}

	// Update current to point to this snapshot
	currentRefObj := plumbing.NewSymbolicReference(s.currentRef(), snapshotRefName)
	if err := s.repo.Storer.SetReference(currentRefObj); err != nil {
		return fmt.Errorf("set current ref: %w", err)
	}

	return nil
}

// deleteAllTaskRefs removes all task refs.
func (s *Store) deleteAllTaskRefs() error {
	prefix := s.refPrefix() + "tasks/"
	var toDelete []plumbing.ReferenceName

	refs, err := s.repo.References()
	if err != nil {
		return fmt.Errorf("list refs: %w", err)
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		refName := string(ref.Name())
		if len(refName) > len(prefix) && refName[:len(prefix)] == prefix {
			toDelete = append(toDelete, ref.Name())
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, refName := range toDelete {
		if err := s.repo.Storer.RemoveReference(refName); err != nil {
			return fmt.Errorf("remove ref %s: %w", refName, err)
		}
	}

	return nil
}

// ListSnapshots returns all snapshots for a given main SHA.
func (s *Store) ListSnapshots(mainSHA string) ([]domain.SnapshotInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.listSnapshotsLocked(mainSHA)
}

// listSnapshotsLocked lists snapshots without locking.
func (s *Store) listSnapshotsLocked(mainSHA string) ([]domain.SnapshotInfo, error) {
	var snapshots []domain.SnapshotInfo
	prefix := s.refPrefix() + "snapshots/"

	refs, err := s.repo.References()
	if err != nil {
		return nil, fmt.Errorf("list refs: %w", err)
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		refName := string(ref.Name())
		if len(refName) <= len(prefix) || refName[:len(prefix)] != prefix {
			return nil
		}

		// Parse snapshot ref name: <mainSHA>_<seq>
		suffix := refName[len(prefix):]
		parts := splitLast(suffix, "_")
		if len(parts) != 2 {
			return nil
		}

		sha := parts[0]
		seq, parseErr := strconv.Atoi(parts[1])
		if parseErr != nil {
			return nil
		}

		// Filter by mainSHA if specified
		if mainSHA != "" && sha != mainSHA {
			return nil
		}

		snapshots = append(snapshots, domain.SnapshotInfo{
			Ref:     refName,
			MainSHA: sha,
			Seq:     seq,
			// CreatedAt: would need commit metadata
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by mainSHA, then seq
	slices.SortFunc(snapshots, func(a, b domain.SnapshotInfo) int {
		if a.MainSHA != b.MainSHA {
			if a.MainSHA < b.MainSHA {
				return -1
			}
			return 1
		}
		return a.Seq - b.Seq
	})

	return snapshots, nil
}

// splitLast splits s by the last occurrence of sep.
func splitLast(s, sep string) []string {
	idx := -1
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == sep[0] {
			idx = i
			break
		}
	}
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}

// SyncSnapshot syncs task state with the current git HEAD.
// If a snapshot exists for the current HEAD, restore from it.
func (s *Store) SyncSnapshot() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current HEAD
	head, err := s.repo.Head()
	if err != nil {
		return fmt.Errorf("get HEAD: %w", err)
	}
	currentSHA := head.Hash().String()

	// Check if we have snapshots for this SHA
	snapshots, err := s.listSnapshotsLocked(currentSHA)
	if err != nil {
		return err
	}

	if len(snapshots) == 0 {
		// No snapshot for current HEAD, nothing to do
		return nil
	}

	// Get the latest snapshot for this SHA
	latestSnapshot := snapshots[len(snapshots)-1]

	// Check current ref - if already pointing to this snapshot, skip
	currentRef, err := s.repo.Reference(s.currentRef(), true)
	if err == nil {
		// Resolve symbolic ref
		resolved, err := s.repo.Reference(currentRef.Target(), true)
		if err == nil && string(resolved.Name()) == latestSnapshot.Ref {
			// Already synced
			return nil
		}
	}

	// Restore from the latest snapshot (without lock since we already hold it)
	return s.restoreSnapshotLocked(latestSnapshot.Ref)
}

// restoreSnapshotLocked restores from a snapshot without acquiring lock.
func (s *Store) restoreSnapshotLocked(snapshotRefStr string) error {
	// Get snapshot tree
	snapshotRefName := plumbing.ReferenceName(snapshotRefStr)
	ref, err := s.repo.Reference(snapshotRefName, true)
	if err != nil {
		return fmt.Errorf("get snapshot ref: %w", err)
	}

	tree, err := s.repo.TreeObject(ref.Hash())
	if err != nil {
		return fmt.Errorf("get snapshot tree: %w", err)
	}

	// Delete all current task refs
	if err := s.deleteAllTaskRefs(); err != nil {
		return err
	}

	// Restore tasks from tree
	for _, entry := range tree.Entries {
		taskRefName := plumbing.ReferenceName(s.refPrefix() + "tasks/" + entry.Name)
		taskRef := plumbing.NewHashReference(taskRefName, entry.Hash)
		if err := s.repo.Storer.SetReference(taskRef); err != nil {
			return fmt.Errorf("restore task ref %s: %w", entry.Name, err)
		}
	}

	// Update current to point to this snapshot
	currentRefObj := plumbing.NewSymbolicReference(s.currentRef(), snapshotRefName)
	if err := s.repo.Storer.SetReference(currentRefObj); err != nil {
		return fmt.Errorf("set current ref: %w", err)
	}

	return nil
}

// PruneSnapshots removes old snapshots, keeping the most recent ones.
func (s *Store) PruneSnapshots(keepCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get all snapshots (empty string = all SHAs)
	snapshots, err := s.listSnapshotsLocked("")
	if err != nil {
		return err
	}

	// Group by mainSHA
	byMainSHA := make(map[string][]domain.SnapshotInfo)
	for _, snap := range snapshots {
		byMainSHA[snap.MainSHA] = append(byMainSHA[snap.MainSHA], snap)
	}

	// For each mainSHA, keep only the latest `keepCount` snapshots
	for _, snaps := range byMainSHA {
		if len(snaps) <= keepCount {
			continue
		}

		// Remove older ones (snaps are sorted by seq)
		toRemove := snaps[:len(snaps)-keepCount]
		for _, snap := range toRemove {
			refName := plumbing.ReferenceName(snap.Ref)
			if err := s.repo.Storer.RemoveReference(refName); err != nil {
				return fmt.Errorf("remove snapshot %s: %w", snap.Ref, err)
			}
		}
	}

	return nil
}
