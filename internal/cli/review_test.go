package cli

import (
	"testing"

	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// parseReviewAllArgs Tests
// =============================================================================

// Note: argsLenAtDash simulates cmd.ArgsLenAtDash() behavior:
//   - -1 if "--" was not present
//   - N if "--" was present, where N is the number of args before "--"
// Cobra removes flags from args and removes "--" itself, so args only contains
// positional arguments (before and after dash combined).

func TestParseReviewAllArgs_NoArgs_OnCrewBranch(t *testing.T) {
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-42")}

	taskID, agent, message, err := parseReviewAllArgs([]string{}, -1, git)

	assert.NoError(t, err)
	assert.Equal(t, 42, taskID)
	assert.Empty(t, agent)
	assert.Empty(t, message)
}

func TestParseReviewAllArgs_NoArgs_OnNonCrewBranch(t *testing.T) {
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}

	_, _, _, err := parseReviewAllArgs([]string{}, -1, git)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task ID is required")
}

func TestParseReviewAllArgs_ExplicitID(t *testing.T) {
	// When ID is provided, git is not used
	taskID, agent, message, err := parseReviewAllArgs([]string{"123"}, -1, nil)

	assert.NoError(t, err)
	assert.Equal(t, 123, taskID)
	assert.Empty(t, agent)
	assert.Empty(t, message)
}

func TestParseReviewAllArgs_ExplicitIDWithHash(t *testing.T) {
	taskID, agent, message, err := parseReviewAllArgs([]string{"#456"}, -1, nil)

	assert.NoError(t, err)
	assert.Equal(t, 456, taskID)
	assert.Empty(t, agent)
	assert.Empty(t, message)
}

func TestParseReviewAllArgs_ExplicitIDAndAgent(t *testing.T) {
	taskID, agent, message, err := parseReviewAllArgs([]string{"123", "claude-reviewer"}, -1, nil)

	assert.NoError(t, err)
	assert.Equal(t, 123, taskID)
	assert.Equal(t, "claude-reviewer", agent)
	assert.Empty(t, message)
}

func TestParseReviewAllArgs_AgentOnly_OnCrewBranch(t *testing.T) {
	// Agent name (non-numeric) resolves ID from branch
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-99")}

	taskID, agent, message, err := parseReviewAllArgs([]string{"claude-reviewer"}, -1, git)

	assert.NoError(t, err)
	assert.Equal(t, 99, taskID)
	assert.Equal(t, "claude-reviewer", agent)
	assert.Empty(t, message)
}

func TestParseReviewAllArgs_AgentOnly_OnNonCrewBranch(t *testing.T) {
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("feature/test")}

	_, _, _, err := parseReviewAllArgs([]string{"claude-reviewer"}, -1, git)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task ID is required")
}

func TestParseReviewAllArgs_DashWithMessage_OnCrewBranch(t *testing.T) {
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-10")}

	// Simulates: crew review -- "Focus" "on" "security"
	// Cobra passes args=["Focus", "on", "security"] with argsLenAtDash=0
	taskID, agent, message, err := parseReviewAllArgs([]string{"Focus", "on", "security"}, 0, git)

	assert.NoError(t, err)
	assert.Equal(t, 10, taskID)
	assert.Empty(t, agent)
	assert.Equal(t, "Focus on security", message)
}

func TestParseReviewAllArgs_ExplicitIDWithMessage(t *testing.T) {
	// Simulates: crew review 7 -- "Check" "performance"
	// Cobra passes args=["7", "Check", "performance"] with argsLenAtDash=1
	taskID, agent, message, err := parseReviewAllArgs([]string{"7", "Check", "performance"}, 1, nil)

	assert.NoError(t, err)
	assert.Equal(t, 7, taskID)
	assert.Empty(t, agent)
	assert.Equal(t, "Check performance", message)
}

func TestParseReviewAllArgs_ExplicitIDAgentAndMessage(t *testing.T) {
	// Simulates: crew review 7 claude-reviewer -- "Be" "thorough"
	// Cobra passes args=["7", "claude-reviewer", "Be", "thorough"] with argsLenAtDash=2
	taskID, agent, message, err := parseReviewAllArgs([]string{"7", "claude-reviewer", "Be", "thorough"}, 2, nil)

	assert.NoError(t, err)
	assert.Equal(t, 7, taskID)
	assert.Equal(t, "claude-reviewer", agent)
	assert.Equal(t, "Be thorough", message)
}

func TestParseReviewAllArgs_AgentAndMessage_OnCrewBranch(t *testing.T) {
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-55")}

	// Simulates: crew review claude-reviewer -- "Focus" "on" "tests"
	// Cobra passes args=["claude-reviewer", "Focus", "on", "tests"] with argsLenAtDash=1
	taskID, agent, message, err := parseReviewAllArgs([]string{"claude-reviewer", "Focus", "on", "tests"}, 1, git)

	assert.NoError(t, err)
	assert.Equal(t, 55, taskID)
	assert.Equal(t, "claude-reviewer", agent)
	assert.Equal(t, "Focus on tests", message)
}

func TestParseReviewAllArgs_DashOnly_OnCrewBranch(t *testing.T) {
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-30")}

	// Simulates: crew review --
	// Cobra passes args=[] with argsLenAtDash=0
	taskID, agent, message, err := parseReviewAllArgs([]string{}, 0, git)

	assert.NoError(t, err)
	assert.Equal(t, 30, taskID)
	assert.Empty(t, agent)
	assert.Empty(t, message)
}

func TestParseReviewAllArgs_InvalidID_Zero(t *testing.T) {
	// "0" looks like a task ID but is invalid - should error
	_, _, _, err := parseReviewAllArgs([]string{"0"}, -1, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestParseReviewAllArgs_InvalidID_HashZero(t *testing.T) {
	// "#0" looks like a task ID but is invalid - should error
	_, _, _, err := parseReviewAllArgs([]string{"#0"}, -1, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestParseReviewAllArgs_InvalidID_Negative(t *testing.T) {
	// "-5" looks like a task ID but is invalid - should error
	_, _, _, err := parseReviewAllArgs([]string{"-5"}, -1, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestParseReviewAllArgs_BranchWithIssueSuffix(t *testing.T) {
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-15-gh-999")}

	taskID, agent, message, err := parseReviewAllArgs([]string{}, -1, git)

	assert.NoError(t, err)
	assert.Equal(t, 15, taskID)
	assert.Empty(t, agent)
	assert.Empty(t, message)
}

func TestParseReviewAllArgs_GitError(t *testing.T) {
	git := &testutil.MockGit{CurrentBranchErr: assert.AnError}

	_, _, _, err := parseReviewAllArgs([]string{}, -1, git)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to detect current branch")
}

func TestParseReviewAllArgs_WithFlags_ExplicitID(t *testing.T) {
	// Simulates: crew review 123 -m gpt-4o
	// Cobra removes flags, so args=["123"] with argsLenAtDash=-1
	taskID, agent, message, err := parseReviewAllArgs([]string{"123"}, -1, nil)

	assert.NoError(t, err)
	assert.Equal(t, 123, taskID)
	assert.Empty(t, agent)
	assert.Empty(t, message)
}

func TestParseReviewAllArgs_WithFlags_AutoDetect(t *testing.T) {
	// Simulates: crew review -m gpt-4o (on crew branch)
	// Cobra removes flags, so args=[] with argsLenAtDash=-1
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-88")}

	taskID, agent, message, err := parseReviewAllArgs([]string{}, -1, git)

	assert.NoError(t, err)
	assert.Equal(t, 88, taskID)
	assert.Empty(t, agent)
	assert.Empty(t, message)
}

func TestParseReviewAllArgs_WithFlags_AgentAndMessage(t *testing.T) {
	// Simulates: crew review -v claude-reviewer -- "Focus on tests"
	// Cobra removes flags, so args=["claude-reviewer", "Focus on tests"] with argsLenAtDash=1
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-77")}

	taskID, agent, message, err := parseReviewAllArgs([]string{"claude-reviewer", "Focus on tests"}, 1, git)

	assert.NoError(t, err)
	assert.Equal(t, 77, taskID)
	assert.Equal(t, "claude-reviewer", agent)
	assert.Equal(t, "Focus on tests", message)
}

func TestParseReviewAllArgs_TooManyArgsWithoutDash(t *testing.T) {
	// Simulates: crew review 1 claude-reviewer extra
	// Without "--", more than 2 positional args is an error
	_, _, _, err := parseReviewAllArgs([]string{"1", "claude-reviewer", "extra"}, -1, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many arguments")
	assert.Contains(t, err.Error(), "use \"--\" before message text")
}

func TestParseReviewAllArgs_ForgottenDashBeforeMessage(t *testing.T) {
	// Simulates: crew review 1 "Focus on X" (forgot "--")
	// This looks like the user meant to pass a message but forgot "--"
	_, _, _, err := parseReviewAllArgs([]string{"1", "Focus", "on", "X"}, -1, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many arguments")
	assert.Contains(t, err.Error(), "use \"--\" before message text")
}

func TestParseReviewAllArgs_ManyArgsWithDash_OK(t *testing.T) {
	// Simulates: crew review 1 claude-reviewer -- "Focus" "on" "X"
	// With "--", any number of message args is OK
	taskID, agent, message, err := parseReviewAllArgs([]string{"1", "claude-reviewer", "Focus", "on", "X"}, 2, nil)

	assert.NoError(t, err)
	assert.Equal(t, 1, taskID)
	assert.Equal(t, "claude-reviewer", agent)
	assert.Equal(t, "Focus on X", message)
}

func TestParseReviewAllArgs_TooManyPositionalArgsWithDash(t *testing.T) {
	// Simulates: crew review 1 agent extra -- msg
	// Even with "--", more than 2 positional args (before "--") is an error
	_, _, _, err := parseReviewAllArgs([]string{"1", "agent", "extra", "msg"}, 3, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many arguments")
}

func TestLooksLikeTaskID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"#123", true},
		{"0", true},
		{"#0", true},
		{"-5", true},
		{"claude-reviewer", false},
		{"agent123", false},
		{"123agent", false},
		{"4o-reviewer", false}, // mixed digits and letters - not ID-like
		{"", false},
		{"-", false},
		{"#", true}, // starts with #, so looks like ID attempt
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := looksLikeTaskID(tt.input)
			assert.Equal(t, tt.expected, result, "looksLikeTaskID(%q)", tt.input)
		})
	}
}

func TestParseReviewAllArgs_AgentLooksLikePartialID(t *testing.T) {
	// "4o-reviewer" should NOT be parsed as task ID 4
	// It should be treated as agent name and resolve ID from branch
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-100")}

	taskID, agent, message, err := parseReviewAllArgs([]string{"4o-reviewer"}, -1, git)

	assert.NoError(t, err)
	assert.Equal(t, 100, taskID)          // resolved from branch, NOT 4
	assert.Equal(t, "4o-reviewer", agent) // treated as agent name
	assert.Empty(t, message)
}

func TestParseReviewAllArgs_AgentStartsWithDigits(t *testing.T) {
	// "123abc" should NOT be parsed as task ID 123
	// It should be treated as agent name and resolve ID from branch
	git := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("crew-50")}

	taskID, agent, message, err := parseReviewAllArgs([]string{"123abc"}, -1, git)

	assert.NoError(t, err)
	assert.Equal(t, 50, taskID)      // resolved from branch, NOT 123
	assert.Equal(t, "123abc", agent) // treated as agent name
	assert.Empty(t, message)
}
