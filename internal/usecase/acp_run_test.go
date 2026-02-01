package usecase

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/require"
)

type acpRunStateStore struct {
	namespace string
	taskID    int
	calls     []domain.ACPExecutionSubstate
}

func (s *acpRunStateStore) Load(context.Context, string, int) (domain.ACPExecutionState, error) {
	return domain.ACPExecutionState{}, domain.ErrACPStateNotFound
}

func (s *acpRunStateStore) Save(_ context.Context, namespace string, taskID int, state domain.ACPExecutionState) error {
	s.namespace = namespace
	s.taskID = taskID
	s.calls = append(s.calls, state.ExecutionSubstate)
	return nil
}

func TestACPRunClientRequestPermissionSelected(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	permissionCh := make(chan domain.ACPCommand, 1)
	stopCh := make(chan struct{})
	stateStore := &acpRunStateStore{}

	client := &acpRunClient{
		permissionCh: permissionCh,
		stopCh:       stopCh,
		stateStore:   stateStore,
		stateNS:      "default",
		taskID:       1,
	}

	params := permissionRequestParams()
	permissionCh <- domain.ACPCommand{Type: domain.ACPCommandPermission, OptionID: "opt-1"}

	resp, err := client.RequestPermission(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Selected)
	require.Equal(t, "opt-1", string(resp.Outcome.Selected.OptionId))
	require.Equal(t, "selected", resp.Outcome.Selected.Outcome)
	require.Equal(t, "default", stateStore.namespace)
	require.Equal(t, 1, stateStore.taskID)
	require.Len(t, stateStore.calls, 2)
	require.Equal(t, domain.ACPExecutionAwaitingPermission, stateStore.calls[0])
	require.Equal(t, domain.ACPExecutionRunning, stateStore.calls[1])
}

func TestACPRunClientRequestPermissionStop(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	permissionCh := make(chan domain.ACPCommand, 1)
	stopCh := make(chan struct{})
	close(stopCh)

	client := &acpRunClient{
		permissionCh: permissionCh,
		stopCh:       stopCh,
	}

	resp, err := client.RequestPermission(ctx, permissionRequestParams())
	require.NoError(t, err)
	require.NotNil(t, resp.Outcome.Cancelled)
	require.Equal(t, "cancelled", resp.Outcome.Cancelled.Outcome)
}

func permissionRequestParams() acpsdk.RequestPermissionRequest {
	return acpsdk.RequestPermissionRequest{
		SessionId: acpsdk.SessionId("session-1"),
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: acpsdk.ToolCallId("tool-1"),
		},
		Options: []acpsdk.PermissionOption{
			{
				OptionId: acpsdk.PermissionOptionId("opt-1"),
				Name:     "Allow once",
				Kind:     acpsdk.PermissionOptionKind("allow_once"),
			},
		},
	}
}

func TestACPRunStopResetsSubstate(t *testing.T) {
	t.Parallel()

	called := false
	ipc := &stubACPIPC{
		next: func(ctx context.Context) (domain.ACPCommand, error) {
			if called {
				<-ctx.Done()
				return domain.ACPCommand{}, ctx.Err()
			}
			called = true
			return domain.ACPCommand{Type: domain.ACPCommandStop}, nil
		},
	}

	uc, repo, stateStore, task := newACPRunTest(t, "hold", ipc)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := uc.Execute(ctx, ACPRunInput{Agent: "helper", TaskID: task.ID})
	require.NoError(t, err)
	require.Equal(t, domain.StatusInProgress, repo.Tasks[task.ID].Status)
	require.Len(t, stateStore.calls, 2)
	require.Equal(t, domain.ACPExecutionRunning, stateStore.calls[0])
	require.Equal(t, domain.ACPExecutionIdle, stateStore.calls[1])
}

func TestACPRunNormalExitResetsSubstate(t *testing.T) {
	t.Parallel()

	ipc := &stubACPIPC{
		next: func(ctx context.Context) (domain.ACPCommand, error) {
			<-ctx.Done()
			return domain.ACPCommand{}, ctx.Err()
		},
	}

	uc, repo, stateStore, task := newACPRunTest(t, "exit_success", ipc)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := uc.Execute(ctx, ACPRunInput{Agent: "helper", TaskID: task.ID})
	require.NoError(t, err)
	require.Equal(t, domain.StatusInProgress, repo.Tasks[task.ID].Status)
	require.Len(t, stateStore.calls, 2)
	require.Equal(t, domain.ACPExecutionRunning, stateStore.calls[0])
	require.Equal(t, domain.ACPExecutionIdle, stateStore.calls[1])
}

func TestACPRunProcessErrorMarksErrorAndIdle(t *testing.T) {
	t.Parallel()

	ipc := &stubACPIPC{
		next: func(ctx context.Context) (domain.ACPCommand, error) {
			<-ctx.Done()
			return domain.ACPCommand{}, ctx.Err()
		},
	}

	uc, repo, stateStore, task := newACPRunTest(t, "exit_error", ipc)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := uc.Execute(ctx, ACPRunInput{Agent: "helper", TaskID: task.ID})
	require.Error(t, err)
	require.Equal(t, domain.StatusError, repo.Tasks[task.ID].Status)
	require.GreaterOrEqual(t, len(stateStore.calls), 1)
	require.Equal(t, domain.ACPExecutionIdle, stateStore.calls[len(stateStore.calls)-1])
}

func TestACPRunRouterErrorMarksErrorAndIdle(t *testing.T) {
	t.Parallel()

	ipc := &stubACPIPC{
		next: func(context.Context) (domain.ACPCommand, error) {
			return domain.ACPCommand{}, errors.New("router failed")
		},
	}

	uc, repo, stateStore, task := newACPRunTest(t, "hold", ipc)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := uc.Execute(ctx, ACPRunInput{Agent: "helper", TaskID: task.ID})
	require.Error(t, err)
	require.Equal(t, domain.StatusError, repo.Tasks[task.ID].Status)
	require.GreaterOrEqual(t, len(stateStore.calls), 1)
	require.Equal(t, domain.ACPExecutionIdle, stateStore.calls[len(stateStore.calls)-1])
}

func newACPRunTest(t *testing.T, mode string, ipc domain.ACPIPC) (*ACPRun, *testutil.MockTaskRepository, *acpRunStateStore, *domain.Task) {
	t.Helper()

	repo := testutil.NewMockTaskRepository()
	task := &domain.Task{
		ID:     1,
		Title:  "test",
		Status: domain.StatusTodo,
	}
	repo.Tasks[task.ID] = task

	worktrees := testutil.NewMockWorktreeManager()
	worktrees.ExistsVal = true
	worktrees.ResolvePath = t.TempDir()

	cfgLoader := testutil.NewMockConfigLoader()
	cfg := cfgLoader.Config
	if cfg.Agents == nil {
		cfg.Agents = make(map[string]domain.Agent)
	}
	cfg.Tasks.Namespace = "test-namespace"
	cfg.Agents["helper"] = domain.Agent{
		CommandTemplate: "{{.Args}}",
		Args:            os.Args[0] + " -test.run TestACPHelperProcess",
		DefaultModel:    "test-model",
		Env: map[string]string{
			"GO_WANT_HELPER_PROCESS": "1",
			"ACP_HELPER_MODE":        mode,
		},
	}
	require.NoError(t, cfg.ResolveInheritance())
	cfgLoader.Config = cfg

	git := &testutil.MockGit{
		DefaultBranchName: testutil.StringPtr("main"),
	}

	stateStore := &acpRunStateStore{}
	clock := &testutil.MockClock{NowTime: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewACPRun(
		repo,
		worktrees,
		cfgLoader,
		git,
		testutil.NewMockScriptRunner(),
		stubACPIPCFactory{ipc: ipc},
		stateStore,
		clock,
		t.TempDir(),
		io.Discard,
		io.Discard,
	)

	return uc, repo, stateStore, task
}

type stubACPIPCFactory struct {
	ipc domain.ACPIPC
}

func (f stubACPIPCFactory) ForTask(string, int) domain.ACPIPC {
	return f.ipc
}

type stubACPIPC struct {
	next func(context.Context) (domain.ACPCommand, error)
}

func (s *stubACPIPC) Next(ctx context.Context) (domain.ACPCommand, error) {
	return s.next(ctx)
}

func (s *stubACPIPC) Send(context.Context, domain.ACPCommand) error {
	return nil
}

type acpTestAgent struct {
	newSessionCh chan struct{}
}

func (a *acpTestAgent) Authenticate(context.Context, acpsdk.AuthenticateRequest) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (a *acpTestAgent) Initialize(context.Context, acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	return acpsdk.InitializeResponse{ProtocolVersion: acpsdk.ProtocolVersionNumber}, nil
}

func (a *acpTestAgent) Cancel(context.Context, acpsdk.CancelNotification) error {
	return nil
}

func (a *acpTestAgent) NewSession(context.Context, acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	select {
	case <-a.newSessionCh:
	default:
		close(a.newSessionCh)
	}
	return acpsdk.NewSessionResponse{SessionId: acpsdk.SessionId("session-1")}, nil
}

func (a *acpTestAgent) Prompt(context.Context, acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
}

func (a *acpTestAgent) SetSessionMode(context.Context, acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

func TestACPHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	mode := os.Getenv("ACP_HELPER_MODE")
	agent := &acpTestAgent{newSessionCh: make(chan struct{})}
	acpsdk.NewAgentSideConnection(agent, os.Stdout, os.Stdin)

	select {
	case <-agent.newSessionCh:
	case <-time.After(5 * time.Second):
		os.Exit(2)
	}

	if mode == "exit_success" || mode == "exit_error" {
		time.Sleep(50 * time.Millisecond)
	}

	switch mode {
	case "hold":
		select {}
	case "exit_error":
		os.Exit(1)
	default:
		os.Exit(0)
	}
}
