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
	assert.Contains(t, content, "CLAUDE.md")
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
	assert.Contains(t, content, "## ⚡ Quick Start")
	assert.Contains(t, content, "## ⚠️ Critical Notes")
	assert.Contains(t, content, "crew new")
	assert.Contains(t, content, "crew start")
	assert.Contains(t, content, "crew peek")
	assert.Contains(t, content, "crew merge")
	assert.Contains(t, content, "crew send` requires Enter")
	assert.Contains(t, content, "--from")
	assert.Contains(t, content, "## Interaction Style")

	// Check available workers section
	assert.Contains(t, content, "## Available Workers")
	assert.Contains(t, content, "| worker1 | model1 | desc1 |")
	assert.Contains(t, content, "| worker2 | model2 | desc2 |")
}

func TestShowManagerHelp_OnboardingSection(t *testing.T) {
	t.Run("shows onboarding note when not done", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := &domain.Config{
			OnboardingDone: false,
		}

		err := showManagerHelp(&buf, cfg)

		require.NoError(t, err)
		content := buf.String()
		assert.Contains(t, content, "## Setup & Onboarding")
		assert.Contains(t, content, "Onboarding not completed")
		assert.Contains(t, content, "crew --help-manager-onboarding")
	})

	t.Run("hides onboarding note when done", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := &domain.Config{
			OnboardingDone: true,
		}

		err := showManagerHelp(&buf, cfg)

		require.NoError(t, err)
		content := buf.String()
		assert.NotContains(t, content, "Onboarding not completed")
	})
}

func TestShowManagerOnboardingHelp(t *testing.T) {
	var buf bytes.Buffer

	err := showManagerOnboardingHelp(&buf)

	require.NoError(t, err)
	content := buf.String()

	// Check that content contains key sections
	assert.Contains(t, content, "# git-crew Onboarding Guide")
	assert.Contains(t, content, "## Onboarding Checklist")
	assert.Contains(t, content, "### 1. Basic Configuration")
	assert.Contains(t, content, "### 2. Project Information for AI")
	assert.Contains(t, content, "CLAUDE.md")
	assert.Contains(t, content, "onboarding_done = true")
}
