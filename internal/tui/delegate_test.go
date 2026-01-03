package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeNewlines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no newlines",
			input: "simple text",
			want:  "simple text",
		},
		{
			name:  "single LF",
			input: "line1\nline2",
			want:  "line1 line2",
		},
		{
			name:  "multiple LF",
			input: "line1\nline2\nline3",
			want:  "line1 line2 line3",
		},
		{
			name:  "CRLF",
			input: "line1\r\nline2",
			want:  "line1 line2",
		},
		{
			name:  "single CR",
			input: "line1\rline2",
			want:  "line1 line2",
		},
		{
			name:  "mixed newlines",
			input: "line1\nline2\r\nline3\rline4",
			want:  "line1 line2 line3 line4",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only newlines",
			input: "\n\r\n\r",
			want:  "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeNewlines(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
