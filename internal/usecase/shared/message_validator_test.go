package shared

import (
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMessage_Success(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal message",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "message with leading spaces",
			input:    "   Hello!",
			expected: "Hello!",
		},
		{
			name:     "message with trailing spaces",
			input:    "Hello!   ",
			expected: "Hello!",
		},
		{
			name:     "message with leading and trailing spaces",
			input:    "   Hello!   ",
			expected: "Hello!",
		},
		{
			name:     "message with newlines",
			input:    "\n\nHello!\n\n",
			expected: "Hello!",
		},
		{
			name:     "message with tabs",
			input:    "\t\tHello!\t\t",
			expected: "Hello!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateMessage(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateMessage_EmptyMessage(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "only spaces",
			input: "   ",
		},
		{
			name:  "only tabs",
			input: "\t\t\t",
		},
		{
			name:  "only newlines",
			input: "\n\n\n",
		},
		{
			name:  "mixed whitespace",
			input: "  \t\n  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateMessage(tt.input)

			assert.Empty(t, result)
			require.ErrorIs(t, err, domain.ErrEmptyMessage)
		})
	}
}
