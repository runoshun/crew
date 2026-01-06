package tui

import (
	"fmt"
	"hash/fnv"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

type taskItem struct {
	task         *domain.Task
	commentCount int
}

func (t taskItem) FilterValue() string {
	return t.task.Title
}

// escapeNewlines replaces newline characters with spaces for single-line display.
func escapeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

type taskDelegate struct {
	styles Styles
}

func newTaskDelegate(styles Styles) taskDelegate {
	return taskDelegate{styles: styles}
}

func (d taskDelegate) Height() int {
	return 2
}

func (d taskDelegate) Spacing() int {
	return 1
}

func (d taskDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d taskDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ti, ok := item.(taskItem)
	if !ok {
		return
	}
	task := ti.task
	selected := index == m.Index()

	indicatorChar := " "
	if selected {
		indicatorChar = ">"
	}

	idStr := fmt.Sprintf("%3d", task.ID)
	statusIcon := StatusIcon(task.Status)
	statusText := fmt.Sprintf("%-5s", StatusText(task.Status))

	// Labels
	var labelsStr string
	if len(task.Labels) > 0 {
		for _, l := range task.Labels {
			labelsStr += "[" + l + "] "
		}
	}

	prefixWidth := 19 + runewidth.StringWidth(labelsStr)
	listWidth := m.Width()
	maxTitleLen := listWidth - prefixWidth - 2
	if maxTitleLen < 10 {
		maxTitleLen = 10
	}

	title := task.Title
	if runewidth.StringWidth(title) > maxTitleLen {
		title = runewidth.Truncate(title, maxTitleLen-3, "...")
	}

	var line string
	if selected {
		indicator := d.styles.SelectionIndicator.Bold(true).Render(indicatorChar)
		idPart := d.styles.TaskID.Bold(true).Render(idStr)
		iconPart := d.styles.StatusStyle(task.Status).Bold(true).Render(statusIcon)
		textPart := d.styles.StatusStyle(task.Status).Bold(true).Render(statusText)
		titlePart := d.styles.TaskTitle.Bold(true).Render(title)

		line = "  " + indicator + " " + idPart + "  " + iconPart + " " + textPart + "  "
		if labelsStr != "" {
			line += d.styles.TaskLabelSelected.Render(labelsStr)
		}
		line += titlePart
	} else {
		indicator := d.styles.SelectionIndicator.Render(indicatorChar)
		idPart := d.styles.TaskID.Render(idStr)
		iconPart := d.styles.StatusStyle(task.Status).Render(statusIcon)
		textPart := d.styles.StatusStyle(task.Status).Render(statusText)
		titlePart := d.styles.TaskTitle.Render(title)

		line = "  " + indicator + " " + idPart + "  " + iconPart + " " + textPart + "  "
		if labelsStr != "" {
			line += d.styles.TaskLabel.Render(labelsStr)
		}
		line += titlePart
	}
	lineWidth := runewidth.StringWidth(line)
	if lineWidth < listWidth {
		line += fmt.Sprintf("%*s", listWidth-lineWidth, "")
	}
	_, _ = fmt.Fprintln(w, line)

	// Build metadata line
	metaParts := []string{}
	grayStyle := lipgloss.NewStyle().Foreground(Colors.Muted)
	sepStyle := lipgloss.NewStyle().Foreground(Colors.Muted)

	// 1. Created (always shown)
	createdStr := task.Created.Format("Jan 02")
	metaParts = append(metaParts, grayStyle.Render(createdStr))

	// 2. Labels (if any)
	if len(task.Labels) > 0 {
		labelsStrs := []string{}
		for _, label := range task.Labels {
			labelStyle := lipgloss.NewStyle().Bold(true).Foreground(labelColor(label))
			labelsStrs = append(labelsStrs, labelStyle.Render(label))
		}
		metaParts = append(metaParts, strings.Join(labelsStrs, " "))
	}

	// 3. Parent (if has parent)
	if task.ParentID != nil {
		parentStr := fmt.Sprintf("^%d", *task.ParentID)
		metaParts = append(metaParts, grayStyle.Render(parentStr))
	}

	// 4. Session + elapsed time (if running)
	if task.Session != "" {
		elapsed := time.Since(task.Started)
		playIcon := lipgloss.NewStyle().Foreground(Colors.Success).Render("▶")
		elapsedStr := grayStyle.Render(formatElapsedTime(elapsed))
		metaParts = append(metaParts, playIcon+" "+elapsedStr)
	}

	// 5. Comments (if any)
	if ti.commentCount > 0 {
		commentStr := fmt.Sprintf("%d comment", ti.commentCount)
		if ti.commentCount > 1 {
			commentStr += "s"
		}
		metaParts = append(metaParts, grayStyle.Render(commentStr))
	}

	// 6. GitHub (if linked)
	if task.Issue > 0 {
		ghStr := fmt.Sprintf("GH#%d", task.Issue)
		ghStyle := lipgloss.NewStyle().Foreground(Colors.Primary)
		metaParts = append(metaParts, ghStyle.Render(ghStr))
	} else if task.PR > 0 {
		ghStr := fmt.Sprintf("PR#%d", task.PR)
		ghStyle := lipgloss.NewStyle().Foreground(Colors.Primary)
		metaParts = append(metaParts, ghStyle.Render(ghStr))
	}

	// Join with separator
	sep := sepStyle.Render("  |  ")
	metaLine := "                   " + strings.Join(metaParts, sep)

	// Pad to full width
	metaLineWidth := runewidth.StringWidth(metaLine)
	if metaLineWidth < listWidth {
		metaLine += fmt.Sprintf("%*s", listWidth-metaLineWidth, "")
	}
	_, _ = fmt.Fprint(w, metaLine)
}

// formatElapsedTime formats a duration into a human-readable string.
// Examples: "2h", "15m", "3d", "2w"
func formatElapsedTime(d time.Duration) string {
	if d < time.Minute {
		return "< 1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days < 7 {
		return fmt.Sprintf("%dd", days)
	}
	weeks := days / 7
	return fmt.Sprintf("%dw", weeks)
}

// labelColor returns a color for a label based on its hash.
// Uses a palette of colors for variety.
func labelColor(label string) lipgloss.Color {
	palette := []lipgloss.Color{
		lipgloss.Color("#F38BA8"), // Red
		lipgloss.Color("#FAB387"), // Peach
		lipgloss.Color("#F9E2AF"), // Yellow
		lipgloss.Color("#A6E3A1"), // Green
		lipgloss.Color("#94E2D5"), // Teal
		lipgloss.Color("#89B4FA"), // Blue
		lipgloss.Color("#CBA6F7"), // Mauve
		lipgloss.Color("#F5C2E7"), // Pink
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(label))
	paletteSize := uint32(len(palette)) // #nosec G115 - palette size is small (8), no overflow risk
	idx := h.Sum32() % paletteSize
	return palette[idx]
}
