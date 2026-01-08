package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseReviewArgs_NoArgs(t *testing.T) {
	agent, message := parseReviewArgs([]string{})
	assert.Empty(t, agent)
	assert.Empty(t, message)
}

func TestParseReviewArgs_AgentOnly(t *testing.T) {
	agent, message := parseReviewArgs([]string{"claude-reviewer"})
	assert.Equal(t, "claude-reviewer", agent)
	assert.Empty(t, message)
}

func TestParseReviewArgs_DashOnly(t *testing.T) {
	agent, message := parseReviewArgs([]string{"--"})
	assert.Empty(t, agent)
	assert.Empty(t, message)
}

func TestParseReviewArgs_DashWithMessage(t *testing.T) {
	agent, message := parseReviewArgs([]string{"--", "Focus", "on", "security"})
	assert.Empty(t, agent)
	assert.Equal(t, "Focus on security", message)
}

func TestParseReviewArgs_AgentAndDash(t *testing.T) {
	agent, message := parseReviewArgs([]string{"claude-reviewer", "--"})
	assert.Equal(t, "claude-reviewer", agent)
	assert.Empty(t, message)
}

func TestParseReviewArgs_AgentAndDashWithMessage(t *testing.T) {
	agent, message := parseReviewArgs([]string{"claude-reviewer", "--", "Check", "performance"})
	assert.Equal(t, "claude-reviewer", agent)
	assert.Equal(t, "Check performance", message)
}

func TestParseReviewArgs_FlagLikeArg(t *testing.T) {
	// If the first arg looks like a flag, it's not treated as an agent
	agent, message := parseReviewArgs([]string{"-m"})
	assert.Empty(t, agent)
	assert.Empty(t, message)
}

func TestParseReviewArgs_MessageWithQuotes(t *testing.T) {
	// In reality, shell would handle quotes, but we test the join behavior
	agent, message := parseReviewArgs([]string{"claude-reviewer", "--", "Check the", "last commit"})
	assert.Equal(t, "claude-reviewer", agent)
	assert.Equal(t, "Check the last commit", message)
}
