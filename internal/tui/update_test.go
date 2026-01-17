package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
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
