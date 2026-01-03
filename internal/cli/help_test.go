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

	err := RenderManagerHelp(&buf, HelpTemplateData{})

	require.NoError(t, err)
	content := buf.String()

	// Check that content contains key sections
	assert.Contains(t, content, "# git-crew Manager Guide")
	assert.Contains(t, content, "crew new")
	assert.Contains(t, content, "crew start")
	assert.Contains(t, content, "crew peek")
	assert.Contains(t, content, "crew merge")
	assert.Contains(t, content, "Send Enter after send")
}
