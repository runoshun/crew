package tui

import (
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

func TestStyles_StatusStyle(t *testing.T) {
	styles := DefaultStyles()

	tests := []struct {
		status domain.Status
	}{
		{domain.StatusTodo},
		{domain.StatusInProgress},
		{domain.StatusInReview},
		{domain.StatusError},
		{domain.StatusDone},
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
		{domain.StatusTodo, "○"},
		{domain.StatusInProgress, "●"},
		{domain.StatusInReview, "◉"},
		{domain.StatusError, "✗"},
		{domain.StatusDone, "✓"},
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
