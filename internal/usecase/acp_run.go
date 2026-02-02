package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ACPRunInput contains parameters for ACP runner.
// Fields are ordered to minimize memory padding.
type ACPRunInput struct {
	Agent  string // Agent name
	Model  string // Model override (optional)
	TaskID int    // Task ID
}

// ACPRunOutput contains the result of ACP run initialization.
type ACPRunOutput struct {
	SessionID string
}

// ACPRun is the use case for running an ACP client session.
type ACPRun struct {
	tasks              domain.TaskRepository
	worktrees          domain.WorktreeManager
	configLoader       domain.ConfigLoader
	git                domain.Git
	runner             domain.ScriptRunner
	ipcFactory         domain.ACPIPCFactory
	acpStates          domain.ACPStateStore
	eventWriterFactory domain.ACPEventWriterFactory
	clock              domain.Clock
	stdout             io.Writer
	stderr             io.Writer
	repoRoot           string
}

// NewACPRun creates a new ACPRun use case.
func NewACPRun(
	tasks domain.TaskRepository,
	worktrees domain.WorktreeManager,
	configLoader domain.ConfigLoader,
	git domain.Git,
	runner domain.ScriptRunner,
	ipcFactory domain.ACPIPCFactory,
	acpStates domain.ACPStateStore,
	eventWriterFactory domain.ACPEventWriterFactory,
	clock domain.Clock,
	repoRoot string,
	stdout io.Writer,
	stderr io.Writer,
) *ACPRun {
	return &ACPRun{
		tasks:              tasks,
		worktrees:          worktrees,
		configLoader:       configLoader,
		git:                git,
		runner:             runner,
		ipcFactory:         ipcFactory,
		acpStates:          acpStates,
		eventWriterFactory: eventWriterFactory,
		clock:              clock,
		repoRoot:           repoRoot,
		stdout:             stdout,
		stderr:             stderr,
	}
}

// Execute starts an ACP client connected to a wrapper agent process.
func (uc *ACPRun) Execute(ctx context.Context, in ACPRunInput) (*ACPRunOutput, error) {
	if in.Agent == "" {
		return nil, domain.ErrNoAgent
	}

	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}
	if !task.Status.CanStart() {
		return nil, fmt.Errorf("cannot start task with status %q: %w", task.Status, domain.ErrInvalidTransition)
	}
	if task.IsBlocked() {
		return nil, fmt.Errorf("%w: %q", domain.ErrTaskBlocked, task.BlockReason)
	}

	cfg, err := uc.configLoader.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	namespace := shared.ResolveACPNamespace(cfg, uc.git)

	agent, ok := cfg.EnabledAgents()[in.Agent]
	if !ok {
		if _, exists := cfg.Agents[in.Agent]; exists {
			return nil, fmt.Errorf("agent %q is disabled: %w", in.Agent, domain.ErrAgentDisabled)
		}
		return nil, fmt.Errorf("agent %q: %w", in.Agent, domain.ErrAgentNotFound)
	}

	model := in.Model
	if model == "" {
		model = agent.DefaultModel
	}

	wtPath, _, err := ensureACPWorktree(task, cfg, agent, uc.worktrees, uc.git, uc.runner, uc.repoRoot)
	if err != nil {
		return nil, err
	}

	command, err := uc.buildAgentCommand(task, wtPath, agent, model)
	if err != nil {
		return nil, err
	}

	env, err := buildEnv(agent.Env)
	if err != nil {
		return nil, err
	}

	cmdCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	args := splitCommand(command)
	if len(args) == 0 {
		cancel()
		return nil, fmt.Errorf("parse command: empty command")
	}

	// #nosec G204 - command comes from trusted agent configuration
	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...)
	cmd.Dir = wtPath
	cmd.Env = env
	if uc.stderr != nil {
		cmd.Stderr = uc.stderr
	} else {
		cmd.Stderr = os.Stderr
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	if startErr := cmd.Start(); startErr != nil {
		return nil, fmt.Errorf("start agent process: %w", startErr)
	}

	procErrCh := make(chan error, 1)
	go func() {
		procErrCh <- cmd.Wait()
	}()

	stopCh := make(chan struct{})
	permissionCh := make(chan domain.ACPCommand, 10)
	promptCh := make(chan domain.ACPCommand, 10)
	cancelCh := make(chan struct{}, 10)

	client := &acpRunClient{
		permissionCh: permissionCh,
		stopCh:       stopCh,
		stdout:       uc.stdout,
		stderr:       uc.stderr,
		stateStore:   uc.acpStates,
		stateNS:      namespace,
		taskID:       task.ID,
	}
	conn := acpsdk.NewClientSideConnection(client, stdin, stdout)

	if _, initErr := conn.Initialize(ctx, acpsdk.InitializeRequest{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		ClientCapabilities: acpsdk.ClientCapabilities{
			Fs:       acpsdk.FileSystemCapability{ReadTextFile: false, WriteTextFile: false},
			Terminal: false,
		},
	}); initErr != nil {
		cancel()
		return nil, fmt.Errorf("acp initialize: %w", initErr)
	}

	session, err := conn.NewSession(ctx, acpsdk.NewSessionRequest{
		Cwd:        wtPath,
		McpServers: []acpsdk.McpServer{},
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("acp new session: %w", err)
	}
	sessionID := string(session.SessionId)
	client.sessionID = sessionID
	if err := uc.markACPRunning(ctx, task, namespace, in.Agent, sessionID); err != nil {
		cancel()
		return nil, err
	}

	ipc := uc.ipcFactory.ForTask(namespace, task.ID)
	router := newACPCommandRouter(ipc, permissionCh, promptCh, cancelCh, stopCh)
	routerErrCh := router.Start(cmdCtx)

	// Create event writer for logging
	if uc.eventWriterFactory != nil {
		eventWriter, err := uc.eventWriterFactory.ForTask(namespace, task.ID)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("create event writer: %w", err)
		}
		defer func() { _ = eventWriter.Close() }()
		client.eventWriter = eventWriter
	}

	var endOnce sync.Once
	emitSessionEnd := func(reason string, err error) {
		endOnce.Do(func() {
			client.writeSessionEndEvent(context.Background(), reason, err)
		})
	}

	for {
		select {
		case cmd := <-promptCh:
			client.writePromptSentEvent(cmdCtx, cmd)
			if err := uc.handlePrompt(cmdCtx, conn, session.SessionId, cmd, namespace, task.ID); err != nil {
				uc.writeError("prompt", err)
			}
		case <-cancelCh:
			_ = conn.Cancel(context.Background(), acpsdk.CancelNotification{SessionId: session.SessionId})
		case <-stopCh:
			_ = conn.Cancel(context.Background(), acpsdk.CancelNotification{SessionId: session.SessionId})
			emitSessionEnd("stop", nil)
			if idleErr := uc.setExecutionSubstate(context.Background(), namespace, task.ID, domain.ACPExecutionIdle); idleErr != nil {
				cancel()
				return nil, fmt.Errorf("update state: %w", idleErr)
			}
			cancel()
			return &ACPRunOutput{SessionID: string(session.SessionId)}, nil
		case err, ok := <-routerErrCh:
			if !ok {
				routerErrCh = nil
				continue
			}
			if err != nil {
				emitSessionEnd("router_error", err)
				if stateErr := uc.markACPError(ctx, task, namespace); stateErr != nil {
					return nil, fmt.Errorf("acp router error: %w (update state: %v)", err, stateErr)
				}
				return nil, err
			}
		case err := <-procErrCh:
			if err == nil || errors.Is(cmdCtx.Err(), context.Canceled) {
				emitSessionEnd("process_exit", nil)
				if idleErr := uc.setExecutionSubstate(context.Background(), namespace, task.ID, domain.ACPExecutionIdle); idleErr != nil {
					return nil, fmt.Errorf("update state: %w", idleErr)
				}
				return &ACPRunOutput{SessionID: string(session.SessionId)}, nil
			}
			emitSessionEnd("process_error", err)
			if stateErr := uc.markACPError(ctx, task, namespace); stateErr != nil {
				return nil, fmt.Errorf("agent process exited: %w (update state: %v)", err, stateErr)
			}
			return nil, fmt.Errorf("agent process exited: %w", err)
		case <-conn.Done():
			wasCanceled := cmdCtx.Err() != nil
			cancel()
			if err := <-procErrCh; err != nil && !wasCanceled {
				emitSessionEnd("process_error", err)
				if stateErr := uc.markACPError(ctx, task, namespace); stateErr != nil {
					return nil, fmt.Errorf("agent process exited: %w (update state: %v)", err, stateErr)
				}
				return nil, fmt.Errorf("agent process exited: %w", err)
			}
			emitSessionEnd("connection_closed", nil)
			if idleErr := uc.setExecutionSubstate(context.Background(), namespace, task.ID, domain.ACPExecutionIdle); idleErr != nil {
				return nil, fmt.Errorf("update state: %w", idleErr)
			}
			return &ACPRunOutput{SessionID: string(session.SessionId)}, nil
		case <-ctx.Done():
			emitSessionEnd("context_canceled", ctx.Err())
			if idleErr := uc.setExecutionSubstate(context.Background(), namespace, task.ID, domain.ACPExecutionIdle); idleErr != nil {
				return nil, fmt.Errorf("context canceled: %w (update state: %v)", ctx.Err(), idleErr)
			}
			return nil, ctx.Err()
		}
	}
}

func (uc *ACPRun) buildAgentCommand(task *domain.Task, worktreePath string, agent domain.Agent, model string) (string, error) {
	cmdData := domain.CommandData{
		GitDir:      filepath.Join(uc.repoRoot, ".git"),
		RepoRoot:    uc.repoRoot,
		Worktree:    worktreePath,
		Title:       task.Title,
		Description: task.Description,
		Branch:      domain.BranchName(task.ID, task.Issue),
		Issue:       task.Issue,
		TaskID:      task.ID,
		Model:       model,
		Continue:    false,
	}

	result, err := agent.RenderCommand(cmdData, "", "", "")
	if err != nil {
		return "", fmt.Errorf("render agent command: %w", err)
	}
	command := strings.TrimSpace(result.Command)
	if command == "" {
		return "", fmt.Errorf("render agent command: empty command")
	}
	return command, nil
}

func (uc *ACPRun) handlePrompt(ctx context.Context, conn *acpsdk.ClientSideConnection, sessionID acpsdk.SessionId, cmd domain.ACPCommand, namespace string, taskID int) error {
	resp, err := conn.Prompt(ctx, acpsdk.PromptRequest{
		SessionId: sessionID,
		Prompt:    []acpsdk.ContentBlock{acpsdk.TextBlock(cmd.Text)},
	})
	if err != nil {
		return err
	}
	if resp.StopReason == acpsdk.StopReasonEndTurn {
		return uc.setExecutionSubstate(ctx, namespace, taskID, domain.ACPExecutionAwaitingUser)
	}
	return nil
}

func (uc *ACPRun) markACPRunning(ctx context.Context, task *domain.Task, namespace string, agentName string, sessionID string) error {
	if err := uc.saveACPState(ctx, namespace, task.ID, domain.ACPExecutionRunning, sessionID); err != nil {
		return err
	}

	prevStatus := task.Status
	prevAgent := task.Agent
	prevStarted := task.Started

	task.Status = domain.StatusInProgress
	task.Agent = agentName
	if uc.clock != nil {
		task.Started = uc.clock.Now()
	}
	if err := uc.tasks.Save(task); err != nil {
		task.Status = prevStatus
		task.Agent = prevAgent
		task.Started = prevStarted
		if stateErr := uc.setExecutionSubstate(ctx, namespace, task.ID, domain.ACPExecutionIdle); stateErr != nil {
			return fmt.Errorf("save task: %w (reset state: %v)", err, stateErr)
		}
		return fmt.Errorf("save task: %w", err)
	}
	return nil
}

func (uc *ACPRun) markACPError(ctx context.Context, task *domain.Task, namespace string) error {
	task.Status = domain.StatusError
	if err := uc.tasks.Save(task); err != nil {
		return fmt.Errorf("save task: %w", err)
	}
	return uc.setExecutionSubstate(ctx, namespace, task.ID, domain.ACPExecutionIdle)
}

func (uc *ACPRun) setExecutionSubstate(ctx context.Context, namespace string, taskID int, substate domain.ACPExecutionSubstate) error {
	if uc.acpStates == nil {
		return nil
	}
	// Load existing state to preserve session ID
	state, err := uc.acpStates.Load(ctx, namespace, taskID)
	if err != nil && !errors.Is(err, domain.ErrACPStateNotFound) {
		return err
	}
	state.ExecutionSubstate = substate
	return uc.acpStates.Save(ctx, namespace, taskID, state)
}

func (uc *ACPRun) saveACPState(ctx context.Context, namespace string, taskID int, substate domain.ACPExecutionSubstate, sessionID string) error {
	if uc.acpStates == nil {
		return nil
	}
	state := domain.ACPExecutionState{
		ExecutionSubstate: substate,
		SessionID:         sessionID,
	}
	return uc.acpStates.Save(ctx, namespace, taskID, state)
}

func (uc *ACPRun) writeError(stage string, err error) {
	if uc.stderr == nil {
		return
	}
	_, _ = fmt.Fprintf(uc.stderr, "[acp:%s] %v\n", stage, err)
}

type acpCommandRouter struct {
	ipc          domain.ACPIPC
	permissionCh chan<- domain.ACPCommand
	promptCh     chan<- domain.ACPCommand
	cancelCh     chan<- struct{}
	stopCh       chan<- struct{}
	stopOnce     sync.Once
}

func newACPCommandRouter(
	ipc domain.ACPIPC,
	permissionCh chan<- domain.ACPCommand,
	promptCh chan<- domain.ACPCommand,
	cancelCh chan<- struct{},
	stopCh chan<- struct{},
) *acpCommandRouter {
	return &acpCommandRouter{
		ipc:          ipc,
		permissionCh: permissionCh,
		promptCh:     promptCh,
		cancelCh:     cancelCh,
		stopCh:       stopCh,
	}
}

func (r *acpCommandRouter) Start(ctx context.Context) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		for {
			cmd, err := r.ipc.Next(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				errCh <- err
				return
			}
			switch cmd.Type {
			case domain.ACPCommandPrompt:
				r.promptCh <- cmd
			case domain.ACPCommandPermission:
				r.permissionCh <- cmd
			case domain.ACPCommandCancel:
				r.cancelCh <- struct{}{}
			case domain.ACPCommandStop:
				r.stopOnce.Do(func() { close(r.stopCh) })
				return
			}
		}
	}()
	return errCh
}

type acpRunClient struct {
	eventWriter  domain.ACPEventWriter
	stdout       io.Writer
	stderr       io.Writer
	stateStore   domain.ACPStateStore
	permissionCh <-chan domain.ACPCommand
	stopCh       <-chan struct{}
	stateNS      string
	sessionID    string
	taskID       int
}

var _ acpsdk.Client = (*acpRunClient)(nil)

func (c *acpRunClient) RequestPermission(ctx context.Context, params acpsdk.RequestPermissionRequest) (acpsdk.RequestPermissionResponse, error) {
	c.setExecutionSubstate(ctx, domain.ACPExecutionAwaitingPermission)
	c.writePermissionRequest(params)
	c.writeEvent(ctx, domain.ACPEventRequestPermission, params)

	options := make(map[string]struct{}, len(params.Options))
	for _, opt := range params.Options {
		options[string(opt.OptionId)] = struct{}{}
	}

	for {
		select {
		case cmd := <-c.permissionCh:
			if _, ok := options[cmd.OptionID]; !ok {
				c.writeWarning(fmt.Sprintf("unknown permission option_id: %s", cmd.OptionID))
				continue
			}
			c.setExecutionSubstate(ctx, domain.ACPExecutionRunning)
			resp := acpsdk.RequestPermissionResponse{
				Outcome: acpsdk.RequestPermissionOutcome{
					Selected: &acpsdk.RequestPermissionOutcomeSelected{
						OptionId: acpsdk.PermissionOptionId(cmd.OptionID),
						Outcome:  "selected",
					},
				},
			}
			c.writeEvent(ctx, domain.ACPEventPermissionResponse, resp)
			return resp, nil
		case <-c.stopCh:
			resp := cancelPermissionResponse()
			c.writeEvent(ctx, domain.ACPEventPermissionResponse, resp)
			return resp, nil
		case <-ctx.Done():
			resp := cancelPermissionResponse()
			c.writeEvent(ctx, domain.ACPEventPermissionResponse, resp)
			return resp, nil
		}
	}
}
func (c *acpRunClient) setExecutionSubstate(ctx context.Context, substate domain.ACPExecutionSubstate) {
	if c.stateStore == nil {
		return
	}
	state := domain.ACPExecutionState{
		ExecutionSubstate: substate,
		SessionID:         c.sessionID,
	}
	if err := c.stateStore.Save(ctx, c.stateNS, c.taskID, state); err != nil {
		c.writeWarning(fmt.Sprintf("update execution substate: %v", err))
	}
}

func (c *acpRunClient) SessionUpdate(ctx context.Context, params acpsdk.SessionNotification) error {
	// Write event based on update type
	c.writeSessionUpdateEvent(ctx, params)
	if c.stdout == nil {
		return nil
	}
	update := params.Update
	if update.AgentMessageChunk != nil && update.AgentMessageChunk.Content.Text != nil {
		_, _ = fmt.Fprint(c.stdout, update.AgentMessageChunk.Content.Text.Text)
	}
	return nil
}

func (c *acpRunClient) writeSessionUpdateEvent(ctx context.Context, params acpsdk.SessionNotification) {
	update := params.Update

	// Determine event type based on update content
	var eventType domain.ACPEventType
	switch {
	case update.AgentMessageChunk != nil:
		eventType = domain.ACPEventAgentMessageChunk
	case update.AgentThoughtChunk != nil:
		eventType = domain.ACPEventAgentThoughtChunk
	case update.ToolCall != nil:
		eventType = domain.ACPEventToolCall
	case update.ToolCallUpdate != nil:
		eventType = domain.ACPEventToolCallUpdate
	case update.UserMessageChunk != nil:
		eventType = domain.ACPEventUserMessageChunk
	case update.Plan != nil:
		eventType = domain.ACPEventPlan
	case update.CurrentModeUpdate != nil:
		eventType = domain.ACPEventCurrentModeUpdate
	case update.AvailableCommandsUpdate != nil:
		eventType = domain.ACPEventAvailableCommands
	default:
		eventType = domain.ACPEventSessionUpdate
	}

	c.writeEvent(ctx, eventType, params)
}

func (c *acpRunClient) WriteTextFile(_ context.Context, _ acpsdk.WriteTextFileRequest) (acpsdk.WriteTextFileResponse, error) {
	return acpsdk.WriteTextFileResponse{}, acpsdk.NewMethodNotFound(acpsdk.ClientMethodFsWriteTextFile)
}

func (c *acpRunClient) ReadTextFile(_ context.Context, _ acpsdk.ReadTextFileRequest) (acpsdk.ReadTextFileResponse, error) {
	return acpsdk.ReadTextFileResponse{}, acpsdk.NewMethodNotFound(acpsdk.ClientMethodFsReadTextFile)
}

func (c *acpRunClient) CreateTerminal(_ context.Context, _ acpsdk.CreateTerminalRequest) (acpsdk.CreateTerminalResponse, error) {
	return acpsdk.CreateTerminalResponse{}, acpsdk.NewMethodNotFound(acpsdk.ClientMethodTerminalCreate)
}

func (c *acpRunClient) TerminalOutput(_ context.Context, _ acpsdk.TerminalOutputRequest) (acpsdk.TerminalOutputResponse, error) {
	return acpsdk.TerminalOutputResponse{}, acpsdk.NewMethodNotFound(acpsdk.ClientMethodTerminalOutput)
}

func (c *acpRunClient) ReleaseTerminal(_ context.Context, _ acpsdk.ReleaseTerminalRequest) (acpsdk.ReleaseTerminalResponse, error) {
	return acpsdk.ReleaseTerminalResponse{}, acpsdk.NewMethodNotFound(acpsdk.ClientMethodTerminalRelease)
}

func (c *acpRunClient) WaitForTerminalExit(_ context.Context, _ acpsdk.WaitForTerminalExitRequest) (acpsdk.WaitForTerminalExitResponse, error) {
	return acpsdk.WaitForTerminalExitResponse{}, acpsdk.NewMethodNotFound(acpsdk.ClientMethodTerminalWaitForExit)
}

func (c *acpRunClient) KillTerminalCommand(_ context.Context, _ acpsdk.KillTerminalCommandRequest) (acpsdk.KillTerminalCommandResponse, error) {
	return acpsdk.KillTerminalCommandResponse{}, acpsdk.NewMethodNotFound(acpsdk.ClientMethodTerminalKill)
}

func (c *acpRunClient) writePermissionRequest(params acpsdk.RequestPermissionRequest) {
	if c.stderr == nil {
		return
	}
	title := ""
	if params.ToolCall.Title != nil {
		title = *params.ToolCall.Title
	}
	_, _ = fmt.Fprintf(c.stderr, "Permission requested: %s\n", title)
	for _, opt := range params.Options {
		_, _ = fmt.Fprintf(c.stderr, "- %s (%s) id=%s\n", opt.Name, opt.Kind, opt.OptionId)
	}
}

func (c *acpRunClient) writeWarning(msg string) {
	if c.stderr == nil {
		return
	}
	_, _ = fmt.Fprintf(c.stderr, "[acp] %s\n", msg)
}

func (c *acpRunClient) writeEvent(ctx context.Context, eventType domain.ACPEventType, payload any) {
	if c.eventWriter == nil {
		return
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		if c.stderr != nil {
			_, _ = fmt.Fprintf(c.stderr, "[acp:event] marshal error: %v\n", err)
		}
		return
	}

	event := domain.ACPEvent{
		Timestamp: time.Now().UTC(),
		Type:      eventType,
		SessionID: c.sessionID,
		Payload:   payloadBytes,
	}

	if err := c.eventWriter.Write(ctx, event); err != nil {
		if c.stderr != nil {
			_, _ = fmt.Fprintf(c.stderr, "[acp:event] write error: %v\n", err)
		}
	}
}

// promptSentPayload is the payload for ACPEventPromptSent events.
type promptSentPayload struct {
	Text string `json:"text"`
}

func (c *acpRunClient) writePromptSentEvent(ctx context.Context, cmd domain.ACPCommand) {
	c.writeEvent(ctx, domain.ACPEventPromptSent, promptSentPayload{Text: cmd.Text})
}

type sessionEndPayload struct {
	Reason string `json:"reason"`
	Error  string `json:"error,omitempty"`
}

func (c *acpRunClient) writeSessionEndEvent(ctx context.Context, reason string, err error) {
	payload := sessionEndPayload{Reason: reason}
	if err != nil {
		payload.Error = err.Error()
	}
	c.writeEvent(ctx, domain.ACPEventSessionEnd, payload)
}

func cancelPermissionResponse() acpsdk.RequestPermissionResponse {
	return acpsdk.RequestPermissionResponse{
		Outcome: acpsdk.RequestPermissionOutcome{
			Cancelled: &acpsdk.RequestPermissionOutcomeCancelled{
				Outcome: "cancelled",
			},
		},
	}
}

func buildEnv(env map[string]string) ([]string, error) {
	base := os.Environ()
	merged := make(map[string]string, len(base)+len(env))
	for _, entry := range base {
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		merged[parts[0]] = parts[1]
	}

	for key, value := range env {
		if !shared.IsValidEnvVarName(key) {
			return nil, fmt.Errorf("%w: %q", domain.ErrInvalidEnvVarName, key)
		}
		merged[key] = value
	}

	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+merged[key])
	}
	return out, nil
}
