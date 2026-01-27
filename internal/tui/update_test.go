package tui

import (
	"context"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
)

// mockSessionManager is a test double for domain.SessionManager.
type mockSessionManager struct {
	isRunningResult bool
	isRunningErr    error
}

func (m *mockSessionManager) Start(_ context.Context, _ domain.StartSessionOptions) error {
	return nil
}

func (m *mockSessionManager) Stop(_ string) error {
	return nil
}

func (m *mockSessionManager) Attach(_ string) error {
	return nil
}

func (m *mockSessionManager) Peek(_ string, _ int, _ bool) (string, error) {
	return "", nil
}

func (m *mockSessionManager) Send(_ string, _ string) error {
	return nil
}

func (m *mockSessionManager) IsRunning(_ string) (bool, error) {
	return m.isRunningResult, m.isRunningErr
}

func (m *mockSessionManager) GetPaneProcesses(_ string) ([]domain.ProcessInfo, error) {
	return nil, nil
}

func TestUpdate_MsgCommentsLoaded(t *testing.T) {
	m := &Model{
		comments: nil,
	}

	testComments := []domain.Comment{
		{
			Text: "Test comment 1",
			Time: time.Now(),
		},
		{
			Text: "Test comment 2",
			Time: time.Now(),
		},
	}

	msg := MsgCommentsLoaded{
		TaskID:   1,
		Comments: testComments,
	}

	updatedModel, _ := m.Update(msg)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok, "Update should return *Model")
	assert.Equal(t, testComments, result.comments, "Comments should be set")
}

func TestUpdateTaskList_PreservesFilter(t *testing.T) {
	filterInput := textinput.New()
	filterInput.SetValue("alpha")

	tasks := []*domain.Task{
		{ID: 1, Title: "alpha task", Status: domain.StatusTodo},
		{ID: 2, Title: "beta task", Status: domain.StatusTodo},
	}

	m := &Model{
		tasks:         tasks,
		commentCounts: map[int]int{1: 2, 2: 0},
		filterInput:   filterInput,
		taskList:      list.New([]list.Item{}, newTaskDelegate(DefaultStyles()), 0, 0),
	}

	m.updateTaskList()

	items := m.taskList.Items()
	assert.Len(t, items, 1)
	item, ok := items[0].(taskItem)
	assert.True(t, ok)
	assert.Equal(t, 1, item.task.ID)
	assert.Equal(t, 2, item.commentCount)
}

func TestUpdate_StopKey_NoSelection(t *testing.T) {
	m := &Model{
		keys:     DefaultKeyMap(),
		mode:     ModeNormal,
		taskList: list.New([]list.Item{}, newTaskDelegate(DefaultStyles()), 0, 0),
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	assert.Nil(t, cmd)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)

	assert.Equal(t, ModeNormal, result.mode)
	assert.Equal(t, ConfirmNone, result.confirmAction)
	assert.Equal(t, 0, result.confirmTaskID)
}

func TestUpdate_StopKey_NoSession(t *testing.T) {
	task := &domain.Task{ID: 1, Title: "Task", Status: domain.StatusTodo}
	items := []list.Item{taskItem{task: task}}

	m := &Model{
		keys:     DefaultKeyMap(),
		mode:     ModeNormal,
		tasks:    []*domain.Task{task},
		taskList: list.New(items, newTaskDelegate(DefaultStyles()), 0, 0),
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	assert.Nil(t, cmd)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)

	assert.Equal(t, ModeNormal, result.mode)
	assert.Equal(t, ConfirmNone, result.confirmAction)
	assert.Equal(t, 0, result.confirmTaskID)
}

func TestUpdate_StopKey_WithSession(t *testing.T) {
	task := &domain.Task{ID: 1, Title: "Task", Status: domain.StatusForReview, Session: "crew-1"}
	items := []list.Item{taskItem{task: task}}

	m := &Model{
		keys:     DefaultKeyMap(),
		mode:     ModeNormal,
		tasks:    []*domain.Task{task},
		taskList: list.New(items, newTaskDelegate(DefaultStyles()), 0, 0),
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	assert.Nil(t, cmd)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)

	assert.Equal(t, ModeConfirm, result.mode)
	assert.Equal(t, ConfirmStop, result.confirmAction)
	assert.Equal(t, 1, result.confirmTaskID)
}

func TestUpdate_StopKey_InProgress(t *testing.T) {
	task := &domain.Task{ID: 1, Title: "Task", Status: domain.StatusInProgress}
	items := []list.Item{taskItem{task: task}}

	m := &Model{
		keys:     DefaultKeyMap(),
		mode:     ModeNormal,
		tasks:    []*domain.Task{task},
		taskList: list.New(items, newTaskDelegate(DefaultStyles()), 0, 0),
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	assert.Nil(t, cmd)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)

	assert.Equal(t, ModeConfirm, result.mode)
	assert.Equal(t, ConfirmStop, result.confirmAction)
	assert.Equal(t, 1, result.confirmTaskID)
}

func TestUpdate_MsgCommentsLoaded_EmptyComments(t *testing.T) {
	m := &Model{
		comments: []domain.Comment{
			{Text: "Old comment", Time: time.Now()},
		},
	}

	msg := MsgCommentsLoaded{
		TaskID:   1,
		Comments: []domain.Comment{},
	}

	updatedModel, _ := m.Update(msg)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok, "Update should return *Model")
	assert.Empty(t, result.comments, "Comments should be empty")
}

func TestUpdate_MsgConfigLoaded_Warnings(t *testing.T) {
	m := &Model{}

	cfg := &domain.Config{
		Warnings: []string{"unknown key: xxx"},
	}

	msg := MsgConfigLoaded{
		Config: cfg,
	}

	updatedModel, _ := m.Update(msg)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, []string{"unknown key: xxx"}, result.warnings)
}

func TestUpdate_MsgReviewActionCompleted(t *testing.T) {
	m := &Model{
		mode:               ModeReviewAction,
		reviewTaskID:       42,
		reviewResult:       "Some review",
		reviewActionCursor: 1,
	}

	msg := MsgReviewActionCompleted{TaskID: 42, Action: ReviewActionNotifyWorker}

	updatedModel, _ := m.Update(msg)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode)
	assert.Equal(t, 0, result.reviewTaskID)
	assert.Equal(t, "", result.reviewResult)
	assert.Equal(t, 0, result.reviewActionCursor)
}

func TestUpdate_MsgPrepareEditComment(t *testing.T) {
	eci := textinput.New()

	m := &Model{
		mode:             ModeReviewAction,
		reviewTaskID:     42,
		editCommentInput: eci,
	}

	msg := MsgPrepareEditComment{
		TaskID:  42,
		Index:   0,
		Message: "Original review comment",
	}

	updatedModel, _ := m.Update(msg)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeEditReviewComment, result.mode)
	assert.Equal(t, 0, result.editCommentIndex)
	assert.Equal(t, "Original review comment", result.editCommentInput.Value())
}

func TestUpdate_MsgReviewResultLoaded(t *testing.T) {
	m := &Model{
		mode:         ModeNormal,
		reviewTaskID: 0,
		reviewResult: "",
	}

	msg := MsgReviewResultLoaded{
		TaskID: 42,
		Review: "Review content here",
	}

	updatedModel, _ := m.Update(msg)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeReviewResult, result.mode)
	assert.Equal(t, 42, result.reviewTaskID)
	assert.Equal(t, "Review content here", result.reviewResult)
}

func TestUpdate_ActionMenuEnter(t *testing.T) {
	actionCalled := false

	var m *Model
	m = &Model{
		keys: DefaultKeyMap(),
		mode: ModeActionMenu,
		actionMenuItems: []actionMenuItem{
			{
				ActionID:  "detail",
				IsDefault: true,
				Action: func() (tea.Model, tea.Cmd) {
					actionCalled = true
					return m, nil
				},
			},
		},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode)
	assert.Nil(t, result.actionMenuItems)
	assert.True(t, actionCalled)
}

func TestUpdate_NormalModeEnter_TodoTask(t *testing.T) {
	// Test that Enter key in normal mode triggers default action (start) for todo task
	task := &domain.Task{ID: 1, Title: "Task", Status: domain.StatusTodo}
	items := []list.Item{taskItem{task: task}}

	m := &Model{
		keys:     DefaultKeyMap(),
		mode:     ModeNormal,
		tasks:    []*domain.Task{task},
		taskList: list.New(items, newTaskDelegate(DefaultStyles()), 0, 0),
	}

	// Using tea.KeyEnter simulates pressing the Enter key
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeStart, result.mode, "Enter should trigger default action (start) for todo task")
}

func TestUpdate_NormalModeEnter_NoSelection(t *testing.T) {
	// Test that Enter key with no selection does nothing
	m := &Model{
		keys:     DefaultKeyMap(),
		mode:     ModeNormal,
		taskList: list.New([]list.Item{}, newTaskDelegate(DefaultStyles()), 0, 0),
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode)
}

func TestUpdate_NormalModeSpace_OpensActionMenu(t *testing.T) {
	m := &Model{
		keys: DefaultKeyMap(),
		mode: ModeNormal,
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode)
	assert.Empty(t, result.actionMenuItems)
}

func TestHandleSelectManagerMode_Escape(t *testing.T) {
	m := &Model{
		keys:          DefaultKeyMap(),
		mode:          ModeSelectManager,
		managerAgents: []string{"manager-1", "manager-2"},
	}

	updatedModel, cmd := m.handleSelectManagerMode(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Nil(t, cmd)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode, "Escape should return to normal mode")
}

func TestHandleSelectManagerMode_UpDown(t *testing.T) {
	m := &Model{
		keys:               DefaultKeyMap(),
		mode:               ModeSelectManager,
		managerAgents:      []string{"manager-1", "manager-2", "manager-3"},
		managerAgentCursor: 1,
	}

	// Test Up key
	updatedModel, cmd := m.handleSelectManagerMode(tea.KeyMsg{Type: tea.KeyUp})
	assert.Nil(t, cmd)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, 0, result.managerAgentCursor, "Up should decrease cursor")

	// Test Up at top (should not go negative)
	result.managerAgentCursor = 0
	updatedModel, cmd = result.handleSelectManagerMode(tea.KeyMsg{Type: tea.KeyUp})
	assert.Nil(t, cmd)
	result, ok = updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, 0, result.managerAgentCursor, "Cursor should stay at 0")

	// Test Down key
	result.managerAgentCursor = 1
	updatedModel, cmd = result.handleSelectManagerMode(tea.KeyMsg{Type: tea.KeyDown})
	assert.Nil(t, cmd)
	result, ok = updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, 2, result.managerAgentCursor, "Down should increase cursor")

	// Test Down at bottom (should not exceed length)
	updatedModel, cmd = result.handleSelectManagerMode(tea.KeyMsg{Type: tea.KeyDown})
	assert.Nil(t, cmd)
	result, ok = updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, 2, result.managerAgentCursor, "Cursor should stay at max")
}

func TestHandleSelectManagerMode_Enter_NoTask(t *testing.T) {
	m := &Model{
		keys:               DefaultKeyMap(),
		mode:               ModeSelectManager,
		managerAgents:      []string{"manager-1"},
		managerAgentCursor: 0,
		taskList:           list.New([]list.Item{}, newTaskDelegate(DefaultStyles()), 0, 0),
	}

	updatedModel, cmd := m.handleSelectManagerMode(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode, "Enter with no task should return to normal mode")
}

func TestHandleSelectManagerMode_Enter_NoAgents(t *testing.T) {
	task := &domain.Task{ID: 1, Title: "Task", Status: domain.StatusTodo}
	items := []list.Item{taskItem{task: task}}

	m := &Model{
		keys:               DefaultKeyMap(),
		mode:               ModeSelectManager,
		managerAgents:      []string{},
		managerAgentCursor: 0,
		tasks:              []*domain.Task{task},
		taskList:           list.New(items, newTaskDelegate(DefaultStyles()), 0, 0),
	}

	updatedModel, cmd := m.handleSelectManagerMode(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode, "Enter with no agents should return to normal mode")
}

func TestUpdate_MsgShowManagerSelect(t *testing.T) {
	m := &Model{
		mode: ModeNormal,
	}

	msg := MsgShowManagerSelect{}

	updatedModel, cmd := m.Update(msg)
	assert.Nil(t, cmd)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeSelectManager, result.mode, "MsgShowManagerSelect should switch to ModeSelectManager")
}

func TestCheckAndAttachOrSelectManager_SessionRunning(t *testing.T) {
	// When manager session is running, should return MsgAttachManagerSession
	mockSessions := &mockSessionManager{isRunningResult: true}
	container := &app.Container{Sessions: mockSessions}
	m := &Model{container: container}

	cmd := m.checkAndAttachOrSelectManager()
	msg := cmd()

	_, ok := msg.(MsgAttachManagerSession)
	assert.True(t, ok, "Should return MsgAttachManagerSession when session is running")
}

func TestCheckAndAttachOrSelectManager_SessionNotRunning(t *testing.T) {
	// When manager session is not running, should return MsgShowManagerSelect
	mockSessions := &mockSessionManager{isRunningResult: false}
	container := &app.Container{Sessions: mockSessions}
	m := &Model{container: container}

	cmd := m.checkAndAttachOrSelectManager()
	msg := cmd()

	_, ok := msg.(MsgShowManagerSelect)
	assert.True(t, ok, "Should return MsgShowManagerSelect when session is not running")
}

func TestCheckAndAttachOrSelectManager_Error(t *testing.T) {
	// When IsRunning returns an error, should return MsgError
	mockSessions := &mockSessionManager{isRunningErr: assert.AnError}
	container := &app.Container{Sessions: mockSessions}
	m := &Model{container: container}

	cmd := m.checkAndAttachOrSelectManager()
	msg := cmd()

	errMsg, ok := msg.(MsgError)
	assert.True(t, ok, "Should return MsgError when IsRunning fails")
	assert.Contains(t, errMsg.Err.Error(), "check manager session")
}

func TestManagerKey_WithTaskSelected_SessionRunning_FullFlow(t *testing.T) {
	// Test full flow: M key -> cmd() -> Update(msg) -> attachToManagerSession
	// When manager session is running, M key should lead to MsgAttachManagerSession -> attach command
	task := &domain.Task{ID: 1, Title: "Task", Status: domain.StatusTodo}
	items := []list.Item{taskItem{task: task}}

	mockSessions := &mockSessionManager{isRunningResult: true}
	container := &app.Container{
		Sessions: mockSessions,
		Config:   app.Config{SocketPath: "/tmp/test.sock"},
	}

	m := &Model{
		keys:      DefaultKeyMap(),
		mode:      ModeNormal,
		tasks:     []*domain.Task{task},
		taskList:  list.New(items, newTaskDelegate(DefaultStyles()), 0, 0),
		container: container,
	}

	// Step 1: Press M key
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'M'}})
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode, "Mode should not change immediately")
	assert.NotNil(t, cmd, "Should return a command")

	// Step 2: Execute the command to get the message
	msg := cmd()
	attachMsg, isMsgAttach := msg.(MsgAttachManagerSession)
	assert.True(t, isMsgAttach, "Command should return MsgAttachManagerSession when session is running")

	// Step 3: Pass MsgAttachManagerSession to Update to trigger attachToManagerSession
	updatedModel2, attachCmd := result.Update(attachMsg)
	result2, ok := updatedModel2.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result2.mode, "Mode should stay normal during attach")
	assert.NotNil(t, attachCmd, "Should return attach command (tea.Exec)")
}

func TestManagerKey_WithTaskSelected_SessionNotRunning_FullFlow(t *testing.T) {
	// Test full flow: M key -> cmd() -> Update(msg) -> show manager select
	// When manager session is not running, M key should lead to MsgShowManagerSelect -> ModeSelectManager
	task := &domain.Task{ID: 1, Title: "Task", Status: domain.StatusTodo}
	items := []list.Item{taskItem{task: task}}

	mockSessions := &mockSessionManager{isRunningResult: false}
	container := &app.Container{Sessions: mockSessions}

	m := &Model{
		keys:      DefaultKeyMap(),
		mode:      ModeNormal,
		tasks:     []*domain.Task{task},
		taskList:  list.New(items, newTaskDelegate(DefaultStyles()), 0, 0),
		container: container,
	}

	// Step 1: Press M key
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'M'}})
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode, "Mode should not change immediately")
	assert.NotNil(t, cmd, "Should return a command")

	// Step 2: Execute the command to get the message
	msg := cmd()
	_, isMsgShowSelect := msg.(MsgShowManagerSelect)
	assert.True(t, isMsgShowSelect, "Command should return MsgShowManagerSelect when session is not running")

	// Step 3: Pass the message to Update to verify mode transition
	updatedModel2, _ := result.Update(msg)
	result2, ok := updatedModel2.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeSelectManager, result2.mode, "Mode should change to ModeSelectManager after receiving MsgShowManagerSelect")
}

func TestIsManagerSessionRunning_NilContainer(t *testing.T) {
	// Test defensive guard: nil container should return error, not panic
	m := &Model{container: nil}

	running, err := m.isManagerSessionRunning()
	assert.False(t, running)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session manager not initialized")
}

func TestIsManagerSessionRunning_NilSessions(t *testing.T) {
	// Test defensive guard: nil Sessions should return error, not panic
	container := &app.Container{Sessions: nil}
	m := &Model{container: container}

	running, err := m.isManagerSessionRunning()
	assert.False(t, running)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session manager not initialized")
}
