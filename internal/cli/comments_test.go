package cli

import (
	"bytes"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewCommentsCommand_FilterByType(t *testing.T) {
	repo := testutil.NewMockTaskRepository()
	repo.Tasks[1] = &domain.Task{ID: 1, Title: "Task", Status: domain.StatusTodo}
	repo.Comments[1] = []domain.Comment{
		{Text: "Friction comment", Time: time.Now(), Type: domain.CommentTypeFriction},
		{Text: "Report comment", Time: time.Now(), Type: domain.CommentTypeReport},
	}
	container := newTestContainer(repo)

	cmd := newCommentsCommand(container)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--type", "friction"})

	err := cmd.Execute()

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Friction comment")
	assert.NotContains(t, output, "Report comment")
}
