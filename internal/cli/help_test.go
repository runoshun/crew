package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderWorkerHelp(t *testing.T) {
	var buf bytes.Buffer

	err := RenderWorkerHelp(&buf, HelpTemplateData{})

	require.NoError(t, err)
	content := buf.String()

	// Check that content contains key sections
	assert.Contains(t, content, "# git-crew Worker Guide")
	assert.Contains(t, content, "crew show")
	assert.Contains(t, content, "crew complete")
	assert.Contains(t, content, "mise run ci")
	assert.Contains(t, content, "git push") // in prohibited actions section
}

func TestRenderManagerHelp(t *testing.T) {
	var buf bytes.Buffer

	data := HelpTemplateData{
		Workers: []WorkerInfo{
			{Name: "worker1", Model: "model1", Description: "desc1"},
			{Name: "worker2", Model: "model2", Description: "desc2"},
		},
	}

	err := RenderManagerHelp(&buf, data)

	require.NoError(t, err)
	content := buf.String()

	// Check that content contains key sections
	assert.Contains(t, content, "# git-crew Manager Guide")
	assert.Contains(t, content, "crew new")
	assert.Contains(t, content, "crew start")
	assert.Contains(t, content, "crew peek")
	assert.Contains(t, content, "crew merge")
	assert.Contains(t, content, "Send Enter after send")

	// Check available workers section
	assert.Contains(t, content, "## Available Workers")
	assert.Contains(t, content, "| worker1 | model1 | desc1 |")
	assert.Contains(t, content, "| worker2 | model2 | desc2 |")
}
