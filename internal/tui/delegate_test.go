package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
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

func TestFormatElapsedTime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "less than 1 minute",
			duration: 30 * time.Second,
			want:     "< 1m",
		},
		{
			name:     "exactly 1 minute",
			duration: time.Minute,
			want:     "1m",
		},
		{
			name:     "15 minutes",
			duration: 15 * time.Minute,
			want:     "15m",
		},
		{
			name:     "1 hour",
			duration: time.Hour,
			want:     "1h",
		},
		{
			name:     "2 hours",
			duration: 2 * time.Hour,
			want:     "2h",
		},
		{
			name:     "1 day",
			duration: 24 * time.Hour,
			want:     "1d",
		},
		{
			name:     "3 days",
			duration: 3 * 24 * time.Hour,
			want:     "3d",
		},
		{
			name:     "1 week",
			duration: 7 * 24 * time.Hour,
			want:     "1w",
		},
		{
			name:     "2 weeks",
			duration: 14 * 24 * time.Hour,
			want:     "2w",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatElapsedTime(tt.duration)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLabelColor(t *testing.T) {
	// Test that same label always returns same color
	label := "bug"
	color1 := labelColor(label)
	color2 := labelColor(label)
	assert.Equal(t, color1, color2, "same label should return same color")

	// Test that different labels may return different colors
	// (not guaranteed, but highly likely with 8 colors)
	label2 := "feature"
	color3 := labelColor(label2)
	// We can't assert inequality since hash collision is possible,
	// but we can verify it returns a valid lipgloss.Color
	assert.IsType(t, lipgloss.Color(""), color3)
}
