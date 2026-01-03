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
