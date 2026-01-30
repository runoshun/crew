package tui

import (
	"strings"
	"testing"

	"github.com/muesli/reflow/ansi"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

func TestStyles_StatusStyle(t *testing.T) {
	styles := DefaultStyles()

	tests := []struct {
		status domain.Status
	}{
		{domain.StatusTodo},
		{domain.StatusInProgress},
		{domain.StatusDone},
		{domain.StatusMerged},
		{domain.StatusError},
		{domain.StatusClosed},
	}

	for _, tt := range tests {
		t.Run(tt.status.Display(), func(t *testing.T) {
			// StatusStyle should not panic for any valid status
			style := styles.StatusStyle(tt.status)
			// Verify we get a non-empty rendered output
			rendered := style.Render(tt.status.Display())
			if rendered == "" {
				t.Errorf("StatusStyle(%v).Render() returned empty string", tt.status)
			}
		})
	}
}

func TestStyles_StatusStyle_UnknownStatus(t *testing.T) {
	styles := DefaultStyles()
	// Test that unknown status doesn't panic (uses default case)
	unknownStatus := domain.Status("unknown")
	style := styles.StatusStyle(unknownStatus)
	// Should use default style without panicking
	_ = style.Render("unknown")
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status domain.Status
		want   string
	}{
		{domain.StatusTodo, "●"},
		{domain.StatusInProgress, "➜"},
		{domain.StatusDone, "✔"},
		{domain.StatusMerged, "◆"},
		{domain.StatusError, "✕"},
		{domain.StatusClosed, "−"},
	}

	for _, tt := range tests {
		t.Run(tt.status.Display(), func(t *testing.T) {
			if got := StatusIcon(tt.status); got != tt.want {
				t.Errorf("StatusIcon(%v) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestStatusIcon_UnknownStatus(t *testing.T) {
	unknownStatus := domain.Status("unknown")
	got := StatusIcon(unknownStatus)
	if got != "?" {
		t.Errorf("StatusIcon(unknown) = %v, want ?", got)
	}
}

func TestStyles_RenderMarkdown(t *testing.T) {
	styles := DefaultStyles()

	tests := []struct {
		name  string
		text  string
		width int
	}{
		{
			name:  "basic markdown",
			text:  "# Hello\n\nThis is a test.",
			width: 40,
		},
		{
			name:  "long URL should hard wrap",
			text:  "Check https://example.com/very/long/url/that/exceeds/width/limit/and/needs/wrapping",
			width: 30,
		},
		{
			name:  "long word should hard wrap",
			text:  "ThisIsAVeryLongWordThatShouldBeHardWrappedBecauseItExceedsTheWidthLimit",
			width: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := styles.RenderMarkdown(tt.text, tt.width)

			// Result should not be empty
			if result == "" {
				t.Error("RenderMarkdown returned empty string")
			}

			// Verify hard wrap: no line should exceed the specified width
			// Use ANSI-aware width calculation
			lines := strings.Split(result, "\n")
			for i, line := range lines {
				lineWidth := ansi.PrintableRuneWidth(line)
				if lineWidth > tt.width {
					t.Errorf("line %d exceeds width %d: got %d chars (%q)", i, tt.width, lineWidth, line)
				}
			}
		})
	}
}

func TestStyles_RenderMarkdownWithBg(t *testing.T) {
	styles := DefaultStyles()

	tests := []struct {
		name  string
		text  string
		width int
	}{
		{
			name:  "basic text with background",
			text:  "Hello",
			width: 20,
		},
		{
			name:  "multiline with background",
			text:  "Line 1\nLine 2",
			width: 20,
		},
		{
			name:  "long word should hard wrap with background",
			text:  "ThisIsAVeryLongWordThatNeedsWrapping",
			width: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := styles.RenderMarkdownWithBg(tt.text, tt.width, Colors.Background)

			// Result should not be empty
			if result == "" {
				t.Error("RenderMarkdownWithBg returned empty string")
			}

			// Verify each line has consistent width (lipgloss.Width applies padding)
			lines := strings.Split(result, "\n")
			for i, line := range lines {
				// Skip empty lines (may appear at end)
				if line == "" {
					continue
				}
				lineWidth := ansi.PrintableRuneWidth(line)
				if lineWidth != tt.width {
					t.Errorf("line %d width mismatch: got %d, want %d (%q)", i, lineWidth, tt.width, line)
				}
			}
		})
	}
}
