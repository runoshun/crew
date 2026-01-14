package config

import (
	"strings"
	"testing"
)

func TestClaudeSetupScriptTrustConfiguration(t *testing.T) {
	script := claudeSetupScript

	// Check that trust block is included
	if !strings.Contains(script, "Trust worktree in Claude") {
		t.Error("trust configuration block should be present in claude setup script")
	}

	// Check that mktemp is used for thread-safe temporary file handling
	if !strings.Contains(script, "mktemp") {
		t.Error("setup script should use mktemp for thread-safe temporary file handling")
	}

	// Check that jq is used
	if !strings.Contains(script, "jq") {
		t.Error("setup script should use jq for JSON manipulation")
	}

	// Check that CLAUDE_JSON is defined
	if !strings.Contains(script, "CLAUDE_JSON=~/.claude.json") {
		t.Error("setup script should define CLAUDE_JSON path")
	}

	// Check that jq guard is present
	if !strings.Contains(script, `command -v jq`) {
		t.Error("setup script should check if jq is available")
	}

	// Check that file existence guard is present
	if !strings.Contains(script, `[ -f "$CLAUDE_JSON" ]`) {
		t.Error("setup script should check if ~/.claude.json exists")
	}

	// Check that projects path merging logic is present
	if !strings.Contains(script, ".projects[$path] //=") {
		t.Error("setup script should use //= operator to preserve existing project settings")
	}

	// Check that hasTrustDialogAccepted flag is set
	if !strings.Contains(script, "hasTrustDialogAccepted") {
		t.Error("setup script should set hasTrustDialogAccepted flag")
	}
}
