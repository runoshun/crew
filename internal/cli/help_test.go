package cli

import (
	"bytes"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowWorkerHelp(t *testing.T) {
	var buf bytes.Buffer

	err := showWorkerHelp(&buf)

	require.NoError(t, err)
	content := buf.String()

	// Check that content contains key sections
	assert.Contains(t, content, "# git-crew Worker Guide")
	assert.Contains(t, content, "crew show")
	assert.Contains(t, content, "crew complete")
	assert.Contains(t, content, "mise run ci")
	assert.Contains(t, content, "git push") // in prohibited actions section
}

func TestShowManagerHelp(t *testing.T) {
	var buf bytes.Buffer

	cfg := &domain.Config{
		Agents: map[string]domain.Agent{
			"worker1": {DefaultModel: "model1", Description: "desc1", Role: domain.RoleWorker},
			"worker2": {DefaultModel: "model2", Description: "desc2", Role: domain.RoleWorker},
		},
	}

	err := showManagerHelp(&buf, cfg)

	require.NoError(t, err)
	content := buf.String()

	// Check that content contains key sections
	assert.Contains(t, content, "# git-crew Manager Guide")
	assert.Contains(t, content, "crew new")
	assert.Contains(t, content, "crew start")
	assert.Contains(t, content, "crew peek")
	assert.Contains(t, content, "crew merge")
	assert.Contains(t, content, "Send Enter after send")
	assert.Contains(t, content, "## Interaction Style")
	assert.Contains(t, content, "Using AskUserQuestion Tool")

	// Check available workers section
	assert.Contains(t, content, "## Available Workers")
	assert.Contains(t, content, "| worker1 | model1 | desc1 |")
	assert.Contains(t, content, "| worker2 | model2 | desc2 |")
}
