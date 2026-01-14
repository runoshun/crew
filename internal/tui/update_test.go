package tui

import (
	"testing"
	"time"

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

func TestUpdate_MsgReviewCompleted(t *testing.T) {
	m := &Model{
		mode:         ModeReviewing,
		reviewTaskID: 42,
	}

	msg := MsgReviewCompleted{TaskID: 42, Review: "LGTM - code looks good!"}

	updatedModel, _ := m.Update(msg)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeReviewResult, result.mode)
	assert.Equal(t, "LGTM - code looks good!", result.reviewResult)
	assert.Equal(t, 42, result.reviewTaskID)
}

func TestUpdate_MsgReviewCompleted_Cancelled(t *testing.T) {
	m := &Model{
		mode:            ModeNormal, // Already returned to normal
		reviewCancelled: true,       // User cancelled
		reviewTaskID:    42,
	}

	msg := MsgReviewCompleted{TaskID: 42, Review: "LGTM - code looks good!"}

	updatedModel, _ := m.Update(msg)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	// Should stay in normal mode and clear state
	assert.Equal(t, ModeNormal, result.mode)
	assert.False(t, result.reviewCancelled)
	assert.Equal(t, 0, result.reviewTaskID)
	assert.Equal(t, "", result.reviewResult)
}

func TestUpdate_MsgReviewError(t *testing.T) {
	m := &Model{
		mode:         ModeReviewing,
		reviewTaskID: 42,
		reviewResult: "some partial result",
	}

	msg := MsgReviewError{TaskID: 42, Err: assert.AnError}

	updatedModel, _ := m.Update(msg)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	assert.Equal(t, ModeNormal, result.mode)
	assert.Equal(t, assert.AnError, result.err)
	assert.Equal(t, 0, result.reviewTaskID)
	assert.Equal(t, "", result.reviewResult)
}

func TestUpdate_MsgReviewError_Cancelled(t *testing.T) {
	m := &Model{
		mode:            ModeNormal, // Already returned to normal
		reviewCancelled: true,       // User cancelled
		reviewTaskID:    42,
	}

	msg := MsgReviewError{TaskID: 42, Err: assert.AnError}

	updatedModel, _ := m.Update(msg)
	result, ok := updatedModel.(*Model)
	assert.True(t, ok)
	// Should stay in normal mode, no error shown
	assert.Equal(t, ModeNormal, result.mode)
	assert.Nil(t, result.err)
	assert.False(t, result.reviewCancelled)
	assert.Equal(t, 0, result.reviewTaskID)
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
