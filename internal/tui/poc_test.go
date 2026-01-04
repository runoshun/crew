package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// このファイルは lipgloss の Place/Whitespace API を使って
// 「テキストが存在しない余白部分」にもダイアログ背景色を適用するための PoC。
// 既存処理では Place が挿入する余白が無着色のスペースになるため、
// 背景色が抜け落ちてしまう。WithWhitespaceBackground/Chars を渡すと
// Place が生成するダミー文字列にも背景色が付くことを確認する。

func forceTrueColorProfile(t *testing.T) {
	t.Helper()
	prev := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(prev)
	})
}

func tintedSpaces(bg lipgloss.Color, count int) string {
	return lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", count))
}

func tintedBlankLine(bg lipgloss.Color, width int) string {
	return lipgloss.Place(width, 1, lipgloss.Left, lipgloss.Top, "",
		lipgloss.WithWhitespaceBackground(bg),
		lipgloss.WithWhitespaceChars(" "),
	)
}

func dialogContent(bg lipgloss.Color) string {
	return lipgloss.NewStyle().Background(bg).Width(6).Render("OK")
}

func TestPlaceWithoutWhitespaceBackgroundLeavesTransparentPadding(t *testing.T) {
	forceTrueColorProfile(t)
	bg := lipgloss.Color("#44475a")

	content := dialogContent(bg)
	raw := lipgloss.Place(12, 1, lipgloss.Center, lipgloss.Center, content)

	if !strings.HasPrefix(raw, "   ") {
		t.Fatalf("expected raw prefix to be plain spaces, got %q", raw[:3])
	}

	if strings.HasPrefix(raw, tintedSpaces(bg, 3)) {
		t.Fatalf("expected no tinted padding, but found ANSI background codes: %q", raw)
	}
}

func TestPlaceWithWhitespaceBackgroundFillsHorizontalPadding(t *testing.T) {
	forceTrueColorProfile(t)
	bg := lipgloss.Color("#44475a")

	content := dialogContent(bg)
	tinted := lipgloss.Place(12, 1, lipgloss.Center, lipgloss.Center, content,
		lipgloss.WithWhitespaceBackground(bg),
		lipgloss.WithWhitespaceChars(" "),
	)

	tintedPad := tintedSpaces(bg, 3)
	if !strings.HasPrefix(tinted, tintedPad) {
		t.Fatalf("expected tinted prefix, got %q", tinted)
	}
	if !strings.HasSuffix(tinted, tintedPad) {
		t.Fatalf("expected tinted suffix, got %q", tinted)
	}
}

func TestPlaceWithWhitespaceBackgroundFillsVerticalPadding(t *testing.T) {
	forceTrueColorProfile(t)
	bg := lipgloss.Color("#44475a")

	content := dialogContent(bg)
	tinted := lipgloss.Place(12, 3, lipgloss.Center, lipgloss.Center, content,
		lipgloss.WithWhitespaceBackground(bg),
		lipgloss.WithWhitespaceChars(" "),
	)

	lines := strings.Split(tinted, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	blankLine := tintedBlankLine(bg, 12)
	if lines[0] != blankLine {
		t.Fatalf("top padding is not tinted: %q", lines[0])
	}
	if lines[2] != blankLine {
		t.Fatalf("bottom padding is not tinted: %q", lines[2])
	}
}
