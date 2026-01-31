// Package filestore provides a file-based implementation of TaskRepository.
package filestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

const (
	namespaceMetaSchema = 1
	taskMetaSchema      = 1
)

// Store implements domain.TaskRepository using files under .crew/tasks.
type Store struct {
	rootDir   string
	namespace string
	lockPath  string
}

// New creates a new Store rooted at .crew/tasks.
func New(crewDir, namespace string) *Store {
	if namespace == "" {
		namespace = "default"
	}
	rootDir := filepath.Join(crewDir, "tasks")
	lockPath := filepath.Join(rootDir, namespace, ".lock")
	return &Store{rootDir: rootDir, namespace: namespace, lockPath: lockPath}
}

// Get retrieves a task by ID.
func (s *Store) Get(id int) (*domain.Task, error) {
	var task *domain.Task
	err := s.withLock(func() error {
		t, _, err := s.readTask(id)
		task = t
		return err
	})
	return task, err
}

// List retrieves tasks matching the filter.
func (s *Store) List(filter domain.TaskFilter) ([]*domain.Task, error) {
	var tasks []*domain.Task
	err := s.withLock(func() error {
		ids, err := s.listTaskIDs()
		if err != nil {
			return err
		}
		for _, id := range ids {
			task, _, err := s.readTask(id)
			if err != nil {
				return err
			}
			if task == nil {
				continue
			}
			if filter.ParentID != nil {
				if task.ParentID == nil || *task.ParentID != *filter.ParentID {
					continue
				}
			}
			if len(filter.Labels) > 0 {
				if !containsAll(task.Labels, filter.Labels) {
					continue
				}
			}
			tasks = append(tasks, task)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

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
	return s.withLockWrite(func() error {
		if task == nil {
			return errors.New("task is nil")
		}

		// Normalize status for legacy values
		domain.NormalizeStatus(task)

		_, comments, err := s.readTask(task.ID)
		if err != nil {
			return err
		}
		if comments == nil {
			comments = []domain.Comment{}
		}

		return s.writeTask(task, comments)
	})
}

// Delete removes a task by ID.
func (s *Store) Delete(id int) error {
	return s.withLockWrite(func() error {
		mdPath := s.taskMarkdownPath(id)
		metaPath := s.taskMetaPath(id)
		if err := os.Remove(mdPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove task markdown: %w", err)
		}
		if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove task meta: %w", err)
		}
		return nil
	})
}

// NextID returns the next available task ID.
func (s *Store) NextID() (int, error) {
	var id int
	err := s.withLockWrite(func() error {
		meta, err := s.readNamespaceMeta()
		if err != nil {
			return err
		}
		id = meta.NextID
		meta.NextID++
		return s.writeNamespaceMeta(meta)
	})
	return id, err
}

// GetComments retrieves comments for a task.
func (s *Store) GetComments(taskID int) ([]domain.Comment, error) {
	var comments []domain.Comment
	err := s.withLock(func() error {
		_, parsed, err := s.readTask(taskID)
		if err != nil {
			return err
		}
		if parsed == nil {
			comments = []domain.Comment{}
			return nil
		}
		comments = parsed
		return nil
	})
	return comments, err
}

// AddComment adds a comment to a task.
func (s *Store) AddComment(taskID int, comment domain.Comment) error {
	return s.withLockWrite(func() error {
		task, comments, err := s.readTask(taskID)
		if err != nil {
			return err
		}
		if task == nil {
			return domain.ErrTaskNotFound
		}
		comments = append(comments, comment)
		return s.writeTask(task, comments)
	})
}

// UpdateComment updates an existing comment of a task.
func (s *Store) UpdateComment(taskID, index int, comment domain.Comment) error {
	return s.withLockWrite(func() error {
		task, comments, err := s.readTask(taskID)
		if err != nil {
			return err
		}
		if task == nil {
			return domain.ErrTaskNotFound
		}
		if index < 0 || index >= len(comments) {
			return domain.ErrCommentNotFound
		}
		comments[index] = comment
		return s.writeTask(task, comments)
	})
}

// SaveTaskWithComments atomically saves a task and its comments.
func (s *Store) SaveTaskWithComments(task *domain.Task, comments []domain.Comment) error {
	return s.withLockWrite(func() error {
		if task == nil {
			return errors.New("task is nil")
		}

		domain.NormalizeStatus(task)

		if comments == nil {
			comments = []domain.Comment{}
		}
		return s.writeTask(task, comments)
	})
}

// IsInitialized checks if the store has been initialized.
func (s *Store) IsInitialized() bool {
	_, err := os.Stat(s.namespaceMetaPath())
	return err == nil
}

// Initialize creates the store if it doesn't exist and repairs next_id if needed.
func (s *Store) Initialize() (bool, error) {
	lock, err := s.acquireLock(syscall.LOCK_EX)
	if err != nil {
		return false, err
	}
	defer s.releaseLock(lock)

	err = os.MkdirAll(s.namespaceDir(), 0o750)
	if err != nil {
		return false, fmt.Errorf("create namespace dir: %w", err)
	}

	repaired := false
	meta, err := s.readNamespaceMeta()
	if err != nil {
		if !errors.Is(err, domain.ErrNotInitialized) {
			return false, err
		}
		meta = namespaceMeta{Schema: namespaceMetaSchema, Namespace: s.namespace, NextID: 1}
		err = s.writeNamespaceMeta(meta)
		if err != nil {
			return false, err
		}
	}

	minNextID, err := s.minNextID()
	if err != nil {
		return false, err
	}
	if meta.NextID < minNextID {
		meta.NextID = minNextID
		err = s.writeNamespaceMeta(meta)
		if err != nil {
			return false, err
		}
		repaired = true
	}

	return repaired, nil
}

// === Snapshot operations (no-op for file store) ===

// SaveSnapshot is a no-op for file store.
func (s *Store) SaveSnapshot(mainSHA string) error {
	return nil
}

// RestoreSnapshot is a no-op for file store.
func (s *Store) RestoreSnapshot(snapshotRef string) error {
	return nil
}

// ListSnapshots returns empty for file store.
func (s *Store) ListSnapshots(mainSHA string) ([]domain.SnapshotInfo, error) {
	return nil, nil
}

// SyncSnapshot is a no-op for file store.
func (s *Store) SyncSnapshot() error {
	return nil
}

// PruneSnapshots is a no-op for file store.
func (s *Store) PruneSnapshots(keepCount int) error {
	return nil
}

// === Remote sync operations (no-op for file store) ===

// Push is a no-op for file store.
func (s *Store) Push() error {
	return nil
}

// Fetch is a no-op for file store.
func (s *Store) Fetch(namespace string) error {
	return nil
}

// ListNamespaces returns empty for file store.
func (s *Store) ListNamespaces() ([]string, error) {
	return nil, nil
}

// Ensure Store implements TaskRepository.
var _ domain.TaskRepository = (*Store)(nil)

// Ensure Store implements StoreInitializer.
var _ domain.StoreInitializer = (*Store)(nil)

type namespaceMetaPayload struct {
	Schema    *int    `json:"schema"`
	Namespace *string `json:"namespace"`
	NextID    *int    `json:"next_id"`
}

type namespaceMeta struct {
	Namespace string
	Schema    int
	NextID    int
}

type taskMetaPayload struct {
	Schema        *int    `json:"schema"`
	Status        *string `json:"status"`
	Created       *string `json:"created"`
	Started       *string `json:"started,omitempty"`
	BaseBranch    *string `json:"base_branch"`
	StatusVersion *int    `json:"status_version"`

	Agent       string `json:"agent,omitempty"`
	Session     string `json:"session,omitempty"`
	CloseReason string `json:"close_reason,omitempty"`
	BlockReason string `json:"block_reason,omitempty"`

	Issue             int `json:"issue,omitempty"`
	PR                int `json:"pr,omitempty"`
	AutoFixRetryCount int `json:"auto_fix_retry_count,omitempty"`
}

type taskMeta struct {
	Created time.Time
	Started time.Time

	Status      domain.Status
	CloseReason domain.CloseReason
	Agent       string
	Session     string
	BaseBranch  string
	BlockReason string

	Schema            int
	Issue             int
	PR                int
	AutoFixRetryCount int
	StatusVersion     int
}

type taskFrontmatter struct {
	Title      string
	ParentID   *int
	SkipReview *bool
	Labels     []string

	LabelsFound     bool
	ParentFound     bool
	SkipReviewFound bool
}

func (s *Store) namespaceDir() string {
	return filepath.Join(s.rootDir, s.namespace)
}

func (s *Store) namespaceMetaPath() string {
	return filepath.Join(s.namespaceDir(), "meta.json")
}

func (s *Store) taskMarkdownPath(id int) string {
	return filepath.Join(s.namespaceDir(), fmt.Sprintf("%d.md", id))
}

func (s *Store) taskMetaPath(id int) string {
	return filepath.Join(s.namespaceDir(), fmt.Sprintf("%d.meta.json", id))
}

func (s *Store) withLock(fn func() error) error {
	lock, err := s.acquireLock(syscall.LOCK_SH)
	if err != nil {
		return err
	}
	defer s.releaseLock(lock)
	return fn()
}

func (s *Store) withLockWrite(fn func() error) error {
	lock, err := s.acquireLock(syscall.LOCK_EX)
	if err != nil {
		return err
	}
	defer s.releaseLock(lock)
	return fn()
}

func (s *Store) acquireLock(lockType int) (*os.File, error) {
	if err := os.MkdirAll(s.namespaceDir(), 0o750); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	lock, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	if err := syscall.Flock(int(lock.Fd()), lockType); err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("acquire lock: %w", err)
	}

	return lock, nil
}

func (s *Store) releaseLock(lock *os.File) {
	_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	_ = lock.Close()
}

func (s *Store) readNamespaceMeta() (namespaceMeta, error) {
	content, err := os.ReadFile(s.namespaceMetaPath())
	if err != nil {
		if os.IsNotExist(err) {
			return namespaceMeta{}, domain.ErrNotInitialized
		}
		return namespaceMeta{}, fmt.Errorf("read namespace meta: %w", err)
	}

	var payload namespaceMetaPayload
	if err := decodeJSONStrict(content, &payload); err != nil {
		return namespaceMeta{}, fmt.Errorf("parse namespace meta: %w", err)
	}

	if payload.Schema == nil || payload.Namespace == nil || payload.NextID == nil {
		return namespaceMeta{}, errors.New("namespace meta missing required fields")
	}
	if *payload.Schema != namespaceMetaSchema {
		return namespaceMeta{}, fmt.Errorf("namespace meta schema mismatch: %d", *payload.Schema)
	}
	if *payload.Namespace != s.namespace {
		return namespaceMeta{}, fmt.Errorf("namespace meta mismatch: %s", *payload.Namespace)
	}
	if *payload.NextID < 1 {
		return namespaceMeta{}, errors.New("namespace meta next_id must be positive")
	}

	return namespaceMeta{
		Schema:    *payload.Schema,
		Namespace: *payload.Namespace,
		NextID:    *payload.NextID,
	}, nil
}

func (s *Store) writeNamespaceMeta(meta namespaceMeta) error {
	payload := namespaceMetaPayload{
		Schema:    &meta.Schema,
		Namespace: &meta.Namespace,
		NextID:    &meta.NextID,
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal namespace meta: %w", err)
	}
	return writeAtomic(s.namespaceMetaPath(), content, 0o644)
}

func (s *Store) minNextID() (int, error) {
	ids, err := s.listTaskIDs()
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 1, nil
	}
	maxID := ids[len(ids)-1]
	if maxID < 0 {
		return 1, nil
	}
	return maxID + 1, nil
}

func (s *Store) listTaskIDs() ([]int, error) {
	if err := s.ensureInitialized(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.namespaceDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrNotInitialized
		}
		return nil, fmt.Errorf("read namespace dir: %w", err)
	}
	ids := make([]int, 0)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		idStr := strings.TrimSuffix(name, ".md")
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, nil
}

func (s *Store) readTask(id int) (*domain.Task, []domain.Comment, error) {
	if err := s.ensureInitialized(); err != nil {
		return nil, nil, err
	}
	mdPath := s.taskMarkdownPath(id)
	metaPath := s.taskMetaPath(id)

	mdContent, mdErr := os.ReadFile(mdPath)
	metaContent, metaErr := os.ReadFile(metaPath)

	if mdErr != nil || metaErr != nil {
		if os.IsNotExist(mdErr) && os.IsNotExist(metaErr) {
			return nil, nil, nil
		}
		if os.IsNotExist(mdErr) || os.IsNotExist(metaErr) {
			return nil, nil, fmt.Errorf("task files missing for id %d", id)
		}
		if mdErr != nil {
			return nil, nil, fmt.Errorf("read task markdown: %w", mdErr)
		}
		return nil, nil, fmt.Errorf("read task meta: %w", metaErr)
	}

	frontmatter, description, comments, err := parseTaskMarkdown(string(mdContent))
	if err != nil {
		return nil, nil, fmt.Errorf("parse task markdown: %w", err)
	}

	meta, err := parseTaskMeta(metaContent)
	if err != nil {
		return nil, nil, fmt.Errorf("parse task meta: %w", err)
	}

	task := &domain.Task{
		ID:                id,
		Namespace:         s.namespace,
		Title:             frontmatter.Title,
		Description:       description,
		Labels:            frontmatter.Labels,
		ParentID:          frontmatter.ParentID,
		SkipReview:        frontmatter.SkipReview,
		Created:           meta.Created,
		Started:           meta.Started,
		Agent:             meta.Agent,
		Session:           meta.Session,
		BaseBranch:        meta.BaseBranch,
		Status:            meta.Status,
		CloseReason:       meta.CloseReason,
		BlockReason:       meta.BlockReason,
		Issue:             meta.Issue,
		PR:                meta.PR,
		AutoFixRetryCount: meta.AutoFixRetryCount,
		StatusVersion:     meta.StatusVersion,
	}

	domain.NormalizeStatus(task)

	return task, comments, nil
}

func (s *Store) ensureInitialized() error {
	_, err := s.readNamespaceMeta()
	return err
}

func parseTaskMeta(content []byte) (taskMeta, error) {
	var payload taskMetaPayload
	if err := decodeJSONStrict(content, &payload); err != nil {
		return taskMeta{}, err
	}

	if payload.Schema == nil || payload.Status == nil || payload.Created == nil || payload.BaseBranch == nil || payload.StatusVersion == nil {
		return taskMeta{}, errors.New("task meta missing required fields")
	}
	if *payload.Schema != taskMetaSchema {
		return taskMeta{}, fmt.Errorf("task meta schema mismatch: %d", *payload.Schema)
	}
	status := domain.Status(*payload.Status)
	if !status.IsValid() && !status.IsLegacy() {
		return taskMeta{}, fmt.Errorf("invalid status: %s", *payload.Status)
	}
	created, err := time.Parse(time.RFC3339, *payload.Created)
	if err != nil {
		return taskMeta{}, fmt.Errorf("invalid created time: %w", err)
	}
	var started time.Time
	if payload.Started != nil && *payload.Started != "" {
		started, err = time.Parse(time.RFC3339, *payload.Started)
		if err != nil {
			return taskMeta{}, fmt.Errorf("invalid started time: %w", err)
		}
	}
	closeReason := domain.CloseReason(payload.CloseReason)
	if closeReason != domain.CloseReasonNone && closeReason != domain.CloseReasonMerged && closeReason != domain.CloseReasonAbandoned {
		return taskMeta{}, fmt.Errorf("invalid close_reason: %s", payload.CloseReason)
	}
	if *payload.StatusVersion < 0 {
		return taskMeta{}, errors.New("status_version must be non-negative")
	}

	return taskMeta{
		Schema:            *payload.Schema,
		Status:            status,
		Created:           created,
		Started:           started,
		Agent:             payload.Agent,
		Session:           payload.Session,
		BaseBranch:        *payload.BaseBranch,
		CloseReason:       closeReason,
		BlockReason:       payload.BlockReason,
		Issue:             payload.Issue,
		PR:                payload.PR,
		AutoFixRetryCount: payload.AutoFixRetryCount,
		StatusVersion:     *payload.StatusVersion,
	}, nil
}

func (s *Store) writeTask(task *domain.Task, comments []domain.Comment) error {
	if task == nil {
		return errors.New("task is nil")
	}
	task.Namespace = s.namespace
	if err := os.MkdirAll(s.namespaceDir(), 0o750); err != nil {
		return fmt.Errorf("create namespace dir: %w", err)
	}

	markdown := task.ToMarkdownWithComments(comments)
	mdPath := s.taskMarkdownPath(task.ID)
	metaPath := s.taskMetaPath(task.ID)

	metaPayload := taskMetaPayload{
		Schema:        intPtr(taskMetaSchema),
		Status:        strPtr(string(task.Status)),
		Created:       strPtr(task.Created.Format(time.RFC3339)),
		Agent:         task.Agent,
		Session:       task.Session,
		BaseBranch:    strPtr(task.BaseBranch),
		CloseReason:   string(task.CloseReason),
		BlockReason:   task.BlockReason,
		Issue:         task.Issue,
		PR:            task.PR,
		StatusVersion: intPtr(task.StatusVersion),
	}
	if !task.Started.IsZero() {
		metaPayload.Started = strPtr(task.Started.Format(time.RFC3339))
	}
	if task.AutoFixRetryCount != 0 {
		metaPayload.AutoFixRetryCount = task.AutoFixRetryCount
	}

	metaContent, err := json.MarshalIndent(metaPayload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task meta: %w", err)
	}

	if err := writeAtomic(mdPath, []byte(markdown), 0o644); err != nil {
		return err
	}
	if err := writeAtomic(metaPath, metaContent, 0o644); err != nil {
		return err
	}

	return nil
}

func parseTaskMarkdown(content string) (taskFrontmatter, string, []domain.Comment, error) {
	if len(content) < 4 || !strings.HasPrefix(content, "---\n") {
		return taskFrontmatter{}, "", nil, errors.New("invalid frontmatter: missing opening ---")
	}

	lines := splitLines(content)
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			endIdx = i
			break
		}
	}
	if endIdx == -1 {
		return taskFrontmatter{}, "", nil, errors.New("invalid frontmatter: missing closing ---")
	}

	frontmatterLines := lines[1:endIdx]
	frontmatter, err := parseFrontmatter(frontmatterLines)
	if err != nil {
		return taskFrontmatter{}, "", nil, err
	}

	bodyStart := endIdx + 1
	body := ""
	if bodyStart < len(lines) {
		body = trimLeadingNewlines(joinLines(lines[bodyStart:]))
	}

	description := body
	var commentSection string
	commentSeparator := "\n\n---\n# Comment:"
	if idx := strings.Index(body, commentSeparator); idx >= 0 {
		description = body[:idx]
		commentSection = body[idx+2:]
	} else if strings.HasPrefix(body, "---\n# Comment:") {
		description = ""
		commentSection = body
	}

	comments, err := parseCommentBlocks(commentSection)
	if err != nil {
		return taskFrontmatter{}, "", nil, err
	}

	return frontmatter, description, comments, nil
}

func parseFrontmatter(lines []string) (taskFrontmatter, error) {
	seen := make(map[string]bool)
	fm := taskFrontmatter{}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			return taskFrontmatter{}, errors.New("invalid frontmatter: missing ':'")
		}
		key := strings.TrimSpace(line[:idx])
		value := ""
		if idx+1 < len(line) {
			value = strings.TrimSpace(line[idx+1:])
		}
		if key == "" {
			return taskFrontmatter{}, errors.New("invalid frontmatter: empty key")
		}
		if seen[key] {
			return taskFrontmatter{}, fmt.Errorf("duplicate frontmatter key: %s", key)
		}
		seen[key] = true

		switch key {
		case "title":
			fm.Title = value
		case "labels":
			fm.LabelsFound = true
			if value != "" {
				fm.Labels = parseLabelsValue(value)
			}
		case "parent":
			fm.ParentFound = true
			if value != "" {
				id, err := strconv.Atoi(value)
				if err != nil || id < 0 {
					return taskFrontmatter{}, domain.ErrInvalidParentID
				}
				if id > 0 {
					fm.ParentID = &id
				}
			}
		case "skip_review":
			fm.SkipReviewFound = true
			if value != "" {
				parsed, err := parseBool(value)
				if err != nil {
					return taskFrontmatter{}, err
				}
				fm.SkipReview = &parsed
			}
		default:
			return taskFrontmatter{}, fmt.Errorf("unknown frontmatter key: %s", key)
		}
	}

	if fm.Title == "" {
		return taskFrontmatter{}, domain.ErrEmptyTitle
	}

	return fm, nil
}

func parseLabelsValue(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	}

	parts := strings.Split(value, ",")
	labels := make([]string, 0, len(parts))
	seen := make(map[string]bool)
	for _, part := range parts {
		label := strings.TrimSpace(part)
		if label == "" {
			continue
		}
		if !seen[label] {
			seen[label] = true
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		return nil
	}
	slices.Sort(labels)
	return labels
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, errors.New("invalid skip_review value")
	}
}

func parseCommentBlocks(section string) ([]domain.Comment, error) {
	if strings.TrimSpace(section) == "" {
		return nil, nil
	}
	blocks := splitByCommentSeparator(section)
	comments := make([]domain.Comment, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		comment, err := parseCommentBlock(block)
		if err != nil {
			return nil, err
		}
		if comment.Index != len(comments) {
			return nil, domain.ErrInvalidCommentMeta
		}
		comments = append(comments, comment.Comment)
	}
	return comments, nil
}

type indexedComment struct {
	Comment domain.Comment
	Index   int
}

func parseCommentBlock(block string) (indexedComment, error) {
	lines := splitLines(block)
	if len(lines) < 4 {
		return indexedComment{}, domain.ErrInvalidCommentMeta
	}
	if lines[0] != "---" {
		return indexedComment{}, domain.ErrInvalidCommentMeta
	}
	if len(lines[1]) < 12 || lines[1][:11] != "# Comment: " {
		return indexedComment{}, domain.ErrInvalidCommentMeta
	}
	idxStr := strings.TrimSpace(lines[1][11:])
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		return indexedComment{}, domain.ErrInvalidCommentMeta
	}
	if len(lines[2]) < 9 || lines[2][:9] != "# Author:" {
		return indexedComment{}, domain.ErrInvalidCommentMeta
	}
	author := strings.TrimSpace(lines[2][9:])
	if len(lines[3]) < 8 || lines[3][:7] != "# Time:" {
		return indexedComment{}, domain.ErrInvalidCommentMeta
	}
	timeStr := strings.TrimSpace(lines[3][7:])
	commentTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return indexedComment{}, domain.ErrInvalidCommentMeta
	}

	textStart := 4
	for textStart < len(lines) && lines[textStart] == "" {
		textStart++
	}
	text := ""
	if textStart < len(lines) {
		text = joinLines(lines[textStart:])
	}
	if strings.TrimSpace(text) == "" {
		return indexedComment{}, domain.ErrCommentTextEmpty
	}

	return indexedComment{
		Index: idx,
		Comment: domain.Comment{
			Text:   text,
			Time:   commentTime,
			Author: author,
		},
	}, nil
}

func splitByCommentSeparator(s string) []string {
	separator := "---\n# Comment:"
	var blocks []string
	start := 0
	for {
		idx := strings.Index(s[start:], separator)
		if idx < 0 {
			if start < len(s) {
				blocks = append(blocks, s[start:])
			}
			break
		}
		if idx > 0 {
			blocks = append(blocks, s[start:start+idx])
		}
		start = start + idx
		nextIdx := strings.Index(s[start+len(separator):], separator)
		if nextIdx < 0 {
			blocks = append(blocks, s[start:])
			break
		}
		blocks = append(blocks, s[start:start+len(separator)+nextIdx])
		start = start + len(separator) + nextIdx
	}
	return blocks
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\n" + lines[i]
	}
	return result
}

func trimLeadingNewlines(s string) string {
	start := 0
	for start < len(s) && s[start] == '\n' {
		start++
	}
	return s[start:]
}

func decodeJSONStrict(content []byte, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(content)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("unexpected trailing content")
	}
	return nil
}

func writeAtomic(path string, content []byte, perm os.FileMode) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, content, perm); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

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

func intPtr(v int) *int {
	return &v
}

func strPtr(v string) *string {
	return &v
}
