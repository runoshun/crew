package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/require"
)

type acpControlTaskRepo struct {
	tasks map[int]*domain.Task
}

func (r *acpControlTaskRepo) Get(id int) (*domain.Task, error) {
	task, ok := r.tasks[id]
	if !ok {
		return nil, nil
	}
	copy := *task
	return &copy, nil
}

func (r *acpControlTaskRepo) Save(_ *domain.Task) error                          { return nil }
func (r *acpControlTaskRepo) List(_ domain.TaskFilter) ([]*domain.Task, error)   { return nil, nil }
func (r *acpControlTaskRepo) GetChildren(_ int) ([]*domain.Task, error)          { return nil, nil }
func (r *acpControlTaskRepo) Delete(_ int) error                                 { return nil }
func (r *acpControlTaskRepo) NextID() (int, error)                               { return 0, nil }
func (r *acpControlTaskRepo) AddComment(_ int, _ domain.Comment) error           { return nil }
func (r *acpControlTaskRepo) GetComments(_ int) ([]domain.Comment, error)        { return nil, nil }
func (r *acpControlTaskRepo) UpdateComment(_ int, _ int, _ domain.Comment) error { return nil }
func (r *acpControlTaskRepo) SaveTaskWithComments(_ *domain.Task, _ []domain.Comment) error {
	return nil
}
func (r *acpControlTaskRepo) SaveSnapshot(_ string) error                           { return nil }
func (r *acpControlTaskRepo) RestoreSnapshot(_ string) error                        { return nil }
func (r *acpControlTaskRepo) ListSnapshots(_ string) ([]domain.SnapshotInfo, error) { return nil, nil }
func (r *acpControlTaskRepo) SyncSnapshot() error                                   { return nil }
func (r *acpControlTaskRepo) PruneSnapshots(_ int) error                            { return nil }
func (r *acpControlTaskRepo) Push() error                                           { return nil }
func (r *acpControlTaskRepo) Fetch(_ string) error                                  { return nil }
func (r *acpControlTaskRepo) ListNamespaces() ([]string, error)                     { return nil, nil }

type acpControlConfigLoader struct {
	cfg *domain.Config
}

func (l acpControlConfigLoader) Load() (*domain.Config, error) {
	return l.cfg, nil
}
func (l acpControlConfigLoader) LoadGlobal() (*domain.Config, error) {
	return l.cfg, nil
}
func (l acpControlConfigLoader) LoadRepo() (*domain.Config, error) {
	return l.cfg, nil
}
func (l acpControlConfigLoader) LoadWithOptions(domain.LoadConfigOptions) (*domain.Config, error) {
	return l.cfg, nil
}

type acpControlGit struct{}

func (acpControlGit) CurrentBranch() (string, error)                { return "", nil }
func (acpControlGit) UserEmail() (string, error)                    { return "", nil }
func (acpControlGit) BranchExists(string) (bool, error)             { return false, nil }
func (acpControlGit) HasUncommittedChanges(string) (bool, error)    { return false, nil }
func (acpControlGit) HasMergeConflict(string, string) (bool, error) { return false, nil }
func (acpControlGit) GetMergeConflictFiles(string, string) ([]string, error) {
	return nil, nil
}
func (acpControlGit) Merge(string, bool) error          { return nil }
func (acpControlGit) DeleteBranch(string, bool) error   { return nil }
func (acpControlGit) ListBranches() ([]string, error)   { return nil, nil }
func (acpControlGit) GetDefaultBranch() (string, error) { return "", nil }

type acpControlIPC struct {
	last domain.ACPCommand
}

func (i *acpControlIPC) Next(context.Context) (domain.ACPCommand, error) {
	return domain.ACPCommand{}, nil
}
func (i *acpControlIPC) Send(_ context.Context, cmd domain.ACPCommand) error {
	i.last = cmd
	return nil
}

type acpControlIPCFactory struct {
	ipc       *acpControlIPC
	namespace string
	taskID    int
}

func (f *acpControlIPCFactory) ForTask(namespace string, taskID int) domain.ACPIPC {
	f.namespace = namespace
	f.taskID = taskID
	return f.ipc
}

type acpControlStateStore struct {
	namespace  string
	taskID     int
	last       domain.ACPExecutionState
	saveCalled bool
}

func (s *acpControlStateStore) Load(context.Context, string, int) (domain.ACPExecutionState, error) {
	return domain.ACPExecutionState{}, domain.ErrACPStateNotFound
}

func (s *acpControlStateStore) Save(_ context.Context, namespace string, taskID int, state domain.ACPExecutionState) error {
	s.namespace = namespace
	s.taskID = taskID
	s.last = state
	s.saveCalled = true
	return nil
}

func TestACPControlExecutePrompt(t *testing.T) {
	repo := &acpControlTaskRepo{tasks: map[int]*domain.Task{}}
	repo.tasks[1] = &domain.Task{
		ID:     1,
		Title:  "task",
		Status: domain.StatusTodo,
	}
	cfg := &domain.Config{Tasks: domain.TasksConfig{Namespace: "Team Alpha"}}
	loader := acpControlConfigLoader{cfg: cfg}
	ipc := &acpControlIPC{}
	factory := &acpControlIPCFactory{ipc: ipc}
	stateStore := &acpControlStateStore{}

	uc := NewACPControl(repo, loader, acpControlGit{}, factory, stateStore)

	_, err := uc.Execute(context.Background(), ACPControlInput{
		TaskID:      1,
		CommandType: domain.ACPCommandPrompt,
		Text:        "hello",
	})

	require.NoError(t, err)
	require.Equal(t, "team-alpha", factory.namespace)
	require.Equal(t, 1, factory.taskID)
	require.Equal(t, domain.ACPCommandPrompt, ipc.last.Type)
	require.Equal(t, "hello", ipc.last.Text)
	require.True(t, stateStore.saveCalled)
	require.Equal(t, "team-alpha", stateStore.namespace)
	require.Equal(t, 1, stateStore.taskID)
	require.Equal(t, domain.ACPExecutionRunning, stateStore.last.ExecutionSubstate)
}
