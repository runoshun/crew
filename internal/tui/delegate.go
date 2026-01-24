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

	prefixWidth := 19 // "  > 123  ● Todo   " = 19 chars
	listWidth := m.Width()
	maxTitleLen := listWidth - prefixWidth - 2
	if maxTitleLen < 10 {
		maxTitleLen = 10
	}

	title := task.Title
	if task.IsBlocked() {
		title = "[BLOCKED] " + title
	}
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

		line = "  " + indicator + " " + idPart + "  " + iconPart + " " + textPart + "  " + titlePart
	} else {
		indicator := d.styles.SelectionIndicator.Render(indicatorChar)
		idPart := d.styles.TaskID.Render(idStr)
		iconPart := d.styles.StatusStyle(task.Status).Render(statusIcon)
		textPart := d.styles.StatusStyle(task.Status).Render(statusText)
		titlePart := d.styles.TaskTitle.Render(title)

		line = "  " + indicator + " " + idPart + "  " + iconPart + " " + textPart + "  " + titlePart
	}
	lineWidth := runewidth.StringWidth(line)
	if lineWidth < listWidth {
		line += fmt.Sprintf("%*s", listWidth-lineWidth, "")
	}
	_, _ = fmt.Fprintln(w, line)

	// Build metadata line
	// Collect plain text and styled text separately for width calculation
	type metaPart struct {
		plain  string // plain text for width calculation
		styled string // styled text for display
	}
	var metaParts []metaPart

	grayStyle := lipgloss.NewStyle().Foreground(Colors.DescNormal) // Gray for metadata
	greenStyle := lipgloss.NewStyle().Foreground(Colors.Success)   // Green for play icon
	blueStyle := lipgloss.NewStyle().Foreground(Colors.Primary)    // Blue for GitHub

	// 1. Base branch (always shown)
	metaParts = append(metaParts, metaPart{
		plain:  task.BaseBranch,
		styled: grayStyle.Render(task.BaseBranch),
	})

	// 2. Created (always shown)
	createdStr := task.Created.Format("Jan 02")
	metaParts = append(metaParts, metaPart{
		plain:  createdStr,
		styled: grayStyle.Render(createdStr),
	})

	// 3. Labels (if any)
	if len(task.Labels) > 0 {
		labelsStrs := []string{}
		for _, label := range task.Labels {
			labelStyle := lipgloss.NewStyle().Bold(true).Foreground(labelColor(label))
			labelsStrs = append(labelsStrs, labelStyle.Render(label))
		}
		metaParts = append(metaParts, metaPart{
			plain:  strings.Join(task.Labels, " "),
			styled: strings.Join(labelsStrs, " "),
		})
	}

	// 4. Parent (if has parent)
	if task.ParentID != nil {
		parentStr := fmt.Sprintf("^%d", *task.ParentID)
		metaParts = append(metaParts, metaPart{
			plain:  parentStr,
			styled: grayStyle.Render(parentStr),
		})
	}

	// 5. Session + elapsed time (if running)
	if task.Session != "" {
		elapsed := time.Since(task.Started)
		elapsedFmt := formatElapsedTime(elapsed)
		metaParts = append(metaParts, metaPart{
			plain:  "▶ " + elapsedFmt,
			styled: greenStyle.Render("▶") + " " + grayStyle.Render(elapsedFmt),
		})
	}

	// 6. Comments (if any)
	if ti.commentCount > 0 {
		commentStr := fmt.Sprintf("%d comment", ti.commentCount)
		if ti.commentCount > 1 {
			commentStr += "s"
		}
		metaParts = append(metaParts, metaPart{
			plain:  commentStr,
			styled: grayStyle.Render(commentStr),
		})
	}

	// 7. GitHub (if linked)
	ghParts := []string{}
	if task.Issue > 0 {
		ghParts = append(ghParts, fmt.Sprintf("GH#%d", task.Issue))
	}
	if task.PR > 0 {
		ghParts = append(ghParts, fmt.Sprintf("PR#%d", task.PR))
	}
	if len(ghParts) > 0 {
		ghStr := strings.Join(ghParts, " ")
		metaParts = append(metaParts, metaPart{
			plain:  ghStr,
			styled: blueStyle.Render(ghStr),
		})
	}

	// Calculate available width
	prefix := "                   " // 19 spaces to align with title
	sep := "  |  "
	maxMetaLen := listWidth - runewidth.StringWidth(prefix) - 2
	if maxMetaLen < 10 {
		maxMetaLen = 10
	}

	// Try to fit metadata, removing items from the end if necessary
	numItems := len(metaParts)
	for numItems > 0 {
		// Calculate width of plain text
		plainTexts := make([]string, numItems)
		for i := 0; i < numItems; i++ {
			plainTexts[i] = metaParts[i].plain
		}
		plainStr := strings.Join(plainTexts, sep)
		if runewidth.StringWidth(plainStr) <= maxMetaLen {
			break
		}
		numItems--
	}

	// Build styled string
	styledParts := make([]string, numItems)
	for i := 0; i < numItems; i++ {
		styledParts[i] = metaParts[i].styled
	}
	styledSep := grayStyle.Render(sep)
	metaStr := strings.Join(styledParts, styledSep)

	metaLine := prefix + metaStr

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
// Uses a palette of colors for variety from existing Colors.
func labelColor(label string) lipgloss.Color {
	palette := []lipgloss.Color{
		Colors.Error,      // Red - #F38BA8
		Colors.Stopped,    // Peach - #FAB387
		Colors.Warning,    // Yellow - #F9E2AF
		Colors.Success,    // Green - #A6E3A1
		Colors.NeedsInput, // Teal - #94E2D5
		Colors.Primary,    // Blue - #89B4FA
		Colors.Secondary,  // Mauve - #CBA6F7
		Colors.ForReview,  // Also Mauve - #CBA6F7 (keeping for 8 colors)
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(label))
	paletteSize := uint32(len(palette)) // #nosec G115 - palette size is small (8), no overflow risk
	idx := h.Sum32() % paletteSize
	return palette[idx]
}
