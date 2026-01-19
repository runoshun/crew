package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
)

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

func TestUpdate_ActionMenuSpace(t *testing.T) {
	m := &Model{
		mode: ModeActionMenu,
		actionMenuItems: []actionMenuItem{
			{ActionID: "detail", IsDefault: true},
		},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode)
	assert.Nil(t, result.actionMenuItems)
}

func TestUpdate_NormalModeSpace_TodoTask(t *testing.T) {
	// Test that Space key in normal mode triggers default action (start) for todo task
	task := &domain.Task{ID: 1, Title: "Task", Status: domain.StatusTodo}
	items := []list.Item{taskItem{task: task}}

	m := &Model{
		keys:     DefaultKeyMap(),
		mode:     ModeNormal,
		tasks:    []*domain.Task{task},
		taskList: list.New(items, newTaskDelegate(DefaultStyles()), 0, 0),
	}

	// Using tea.KeySpace simulates pressing the space key
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeStart, result.mode, "Space should trigger default action (start) for todo task")
}

func TestUpdate_NormalModeSpace_NoSelection(t *testing.T) {
	// Test that Space key with no selection does nothing
	m := &Model{
		keys:     DefaultKeyMap(),
		mode:     ModeNormal,
		taskList: list.New([]list.Item{}, newTaskDelegate(DefaultStyles()), 0, 0),
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Nil(t, cmd)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode)
}
