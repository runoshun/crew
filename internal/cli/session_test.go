package cli

import (
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestIsLGTM(t *testing.T) {
	tests := []struct {
		name         string
		reviewResult string
		expected     bool
	}{
		{
			name:         "LGTM with emoji prefix",
			reviewResult: domain.ReviewLGTMPrefix + "\n\nNo issues found.",
			expected:     true,
		},
		{
			name:         "LGTM exact prefix",
			reviewResult: domain.ReviewLGTMPrefix,
			expected:     true,
		},
		{
			name:         "Minor issues",
			reviewResult: "⚠️ Minor issues\n\n- Fix typo in line 10",
			expected:     false,
		},
		{
			name:         "Needs changes",
			reviewResult: "❌ Needs changes\n\n- Missing error handling",
			expected:     false,
		},
		{
			name:         "Empty result",
			reviewResult: "",
			expected:     false,
		},
		{
			name:         "Random text",
			reviewResult: "This code looks good but needs some work.",
			expected:     false,
		},
		{
			name:         "LGTM in middle of text",
			reviewResult: "The code is ✅ LGTM but wait...",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLGTM(tt.reviewResult)
			assert.Equal(t, tt.expected, result)
		})
	}
}
