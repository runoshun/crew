package acpconsole

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/mattn/go-runewidth"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Config contains configuration for the ACP console.
type Config struct {
	EventReader domain.ACPEventReader
	StateLoader func() (domain.ACPExecutionState, error)
	SendPrompt  func(text string) error
	SendPerm    func(optionID string) error
	SendCancel  func() error
	SendStop    func() error
	Namespace   string
	TaskID      int
}

// Model is the bubbletea model for ACP console.
type Model struct {
	textarea textarea.Model
	err      error
	permReq  *permissionRequest
	state    domain.ACPExecutionState
	config   Config
	events   []domain.ACPEvent
	viewport viewport.Model
	width    int
	height   int
	quitting bool
}

type permissionRequest struct {
	message string
	options []permissionOption
}

type permissionOption struct {
	id    string
	label string
}

// Messages
type eventsLoadedMsg struct {
	events []domain.ACPEvent
}

type stateLoadedMsg struct {
	state domain.ACPExecutionState
}

type errMsg struct {
	err error
}

type tickMsg struct{}

// New creates a new ACP console model.
func New(cfg Config) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter prompt..."
	ta.CharLimit = 4096
	ta.SetHeight(3)
	ta.Focus()

	vp := viewport.New(80, 20)
	vp.SetContent("")

	return Model{
		config:   cfg,
		viewport: vp,
		textarea: ta,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.loadEvents,
		m.loadState,
		m.tick(),
	)
}

func (m Model) tick() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) loadEvents() tea.Msg {
	if m.config.EventReader == nil {
		return eventsLoadedMsg{events: nil}
	}
	events, err := m.config.EventReader.ReadAll(context.Background())
	if err != nil {
		return errMsg{err: err}
	}
	return eventsLoadedMsg{events: events}
}

func (m Model) loadState() tea.Msg {
	if m.config.StateLoader == nil {
		return stateLoadedMsg{}
	}
	state, err := m.config.StateLoader()
	if err != nil {
		// Ignore not found errors
		return stateLoadedMsg{}
	}
	return stateLoadedMsg{state: state}
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()

	case eventsLoadedMsg:
		m.events = msg.events
		m.updateViewportContent()
		m.extractPermissionRequest()

	case stateLoadedMsg:
		m.state = msg.state

	case errMsg:
		m.err = msg.err

	case tickMsg:
		// Reload events and state periodically
		cmds = append(cmds, m.loadEvents, m.loadState, m.tick())
	}

	// Update textarea
	var taCmd tea.Cmd
	m.textarea, taCmd = m.textarea.Update(msg)
	cmds = append(cmds, taCmd)

	// Update viewport
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	//nolint:exhaustive // Only handling specific keys, rest forwarded to textarea
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyCtrlD:
		// Stop session
		if m.config.SendStop != nil {
			_ = m.config.SendStop()
		}
		m.quitting = true
		return m, tea.Quit

	case tea.KeyEsc:
		// Cancel current operation
		if m.config.SendCancel != nil {
			_ = m.config.SendCancel()
		}
		return m, nil

	case tea.KeyEnter:
		// If permission request is active, don't send as prompt
		if m.permReq != nil {
			return m, nil
		}
		// Send prompt
		text := strings.TrimSpace(m.textarea.Value())
		if text != "" && m.config.SendPrompt != nil {
			if err := m.config.SendPrompt(text); err != nil {
				m.err = err
			} else {
				m.textarea.Reset()
			}
		}
		return m, nil

	case tea.KeyRunes:
		// Handle permission selection (1-9)
		if m.permReq != nil && len(msg.Runes) == 1 {
			r := msg.Runes[0]
			if r >= '1' && r <= '9' {
				idx := int(r - '1')
				if idx < len(m.permReq.options) {
					opt := m.permReq.options[idx]
					if m.config.SendPerm != nil {
						if err := m.config.SendPerm(opt.id); err != nil {
							m.err = err
						} else {
							m.permReq = nil
						}
					}
					return m, nil
				}
			}
		}

	default:
		// Forward other keys to textarea
	}

	// Forward to textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m *Model) updateLayout() {
	// Layout:
	// - Header: 1 line
	// - Viewport: remaining - input - status
	// - Permission (if any): 2 lines
	// - Input: 3 lines
	// - Status: 1 line

	headerHeight := 1
	inputHeight := 3
	statusHeight := 1
	permHeight := 0
	if m.permReq != nil {
		permHeight = 2
	}

	vpHeight := m.height - headerHeight - inputHeight - statusHeight - permHeight - 2 // borders
	if vpHeight < 3 {
		vpHeight = 3
	}

	m.viewport.Width = m.width - 2
	m.viewport.Height = vpHeight
	m.textarea.SetWidth(m.width - 4)
}

func (m *Model) updateViewportContent() {
	var lines []string
	var msgBuffer strings.Builder
	var lastMsgTime time.Time

	// Calculate wrap width (viewport width minus some padding)
	wrapWidth := m.viewport.Width - 4
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	flushMessageBuffer := func() {
		if msgBuffer.Len() == 0 {
			return
		}
		text := msgBuffer.String()
		prefix := fmt.Sprintf("[%s] AGENT: ", lastMsgTime.Format("15:04:05"))

		// Wrap text to viewport width
		wrapped := wrapText(text, wrapWidth-len(prefix))
		wrappedLines := strings.Split(wrapped, "\n")

		// First line with timestamp prefix
		if len(wrappedLines) > 0 {
			lines = append(lines, prefix+wrappedLines[0])
		}
		// Continuation lines with indent
		indent := strings.Repeat(" ", len(prefix))
		for i := 1; i < len(wrappedLines); i++ {
			lines = append(lines, indent+wrappedLines[i])
		}
		msgBuffer.Reset()
	}

	for _, ev := range m.events {
		// Accumulate agent message chunks
		if ev.Type == domain.ACPEventAgentMessageChunk {
			text := extractAgentText(ev.Payload)
			if text != "" {
				if msgBuffer.Len() == 0 {
					lastMsgTime = ev.Timestamp
				}
				msgBuffer.WriteString(text)
			}
			continue
		}

		// Flush accumulated messages before other events
		flushMessageBuffer()

		line := m.formatEvent(ev)
		if line != "" {
			lines = append(lines, line)
		}
	}

	// Flush any remaining messages
	flushMessageBuffer()

	content := strings.Join(lines, "\n")
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

// wrapText wraps text at the given width, preserving existing newlines.
// Uses runewidth for proper CJK/Japanese character width handling.
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	for i, paragraph := range strings.Split(text, "\n") {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(wrapParagraph(paragraph, width))
	}
	return result.String()
}

// wrapParagraph wraps a single paragraph (no newlines) at the given width.
// Handles CJK characters which can be broken at any point.
func wrapParagraph(text string, width int) string {
	if runewidth.StringWidth(text) <= width {
		return text
	}

	var result strings.Builder
	var lineWidth int

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		rw := runewidth.RuneWidth(r)

		// Check if adding this rune would exceed width
		if lineWidth+rw > width && lineWidth > 0 {
			result.WriteString("\n")
			lineWidth = 0
		}

		// For ASCII words, try to keep them together
		if isASCIIWordChar(r) && lineWidth > 0 {
			// Look ahead to find word width
			wordWidth := 0
			for j := i; j < len(runes) && isASCIIWordChar(runes[j]); j++ {
				wordWidth += runewidth.RuneWidth(runes[j])
			}

			// If word doesn't fit on current line but fits on new line, break before
			if lineWidth+wordWidth > width && wordWidth <= width {
				result.WriteString("\n")
				lineWidth = 0
			}
		}

		result.WriteRune(r)
		lineWidth += rw
	}

	return result.String()
}

// isASCIIWordChar returns true if the rune is an ASCII letter, digit, or common punctuation.
func isASCIIWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
}

func (m *Model) formatEvent(ev domain.ACPEvent) string {
	ts := ev.Timestamp.Format("15:04:05")
	switch ev.Type {
	case domain.ACPEventToolCall:
		detail := extractToolCallDetail(ev.Payload)
		if detail != "" {
			return fmt.Sprintf("[%s] TOOL: %s", ts, detail)
		}
		return fmt.Sprintf("[%s] TOOL: (tool call)", ts)
	case domain.ACPEventRequestPermission:
		return fmt.Sprintf("[%s] PERM: Permission requested", ts)
	case domain.ACPEventPermissionResponse:
		return fmt.Sprintf("[%s] PERM: Permission responded", ts)
	case domain.ACPEventPromptSent:
		detail := extractPromptDetail(ev.Payload)
		if detail != "" {
			return fmt.Sprintf("[%s] USER: %s", ts, detail)
		}
		return fmt.Sprintf("[%s] USER: Prompt sent", ts)
	case domain.ACPEventSessionEnd:
		return fmt.Sprintf("[%s] END: Session ended", ts)
	case domain.ACPEventAgentMessageChunk,
		domain.ACPEventSessionUpdate,
		domain.ACPEventAgentThoughtChunk,
		domain.ACPEventToolCallUpdate,
		domain.ACPEventUserMessageChunk,
		domain.ACPEventPlan,
		domain.ACPEventCurrentModeUpdate,
		domain.ACPEventAvailableCommands:
		// Skip these event types in log display
	}
	return ""
}

// extractToolCallDetail extracts tool call details from payload.
func extractToolCallDetail(payload []byte) string {
	var notification acpsdk.SessionNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		return ""
	}
	if notification.Update.ToolCall == nil {
		return ""
	}
	tc := notification.Update.ToolCall
	return tc.Title
}

// extractPromptDetail extracts the prompt text from payload.
func extractPromptDetail(payload []byte) string {
	// PromptSent payload contains the ACPCommand
	var cmd struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(payload, &cmd); err != nil {
		return ""
	}
	return cmd.Text
}

func (m *Model) extractPermissionRequest() {
	// Find the last permission request without a response
	var lastReq *domain.ACPEvent
	hasResponse := false

	for i := len(m.events) - 1; i >= 0; i-- {
		ev := m.events[i]
		if ev.Type == domain.ACPEventPermissionResponse {
			hasResponse = true
			break
		}
		if ev.Type == domain.ACPEventRequestPermission {
			lastReq = &m.events[i]
			break
		}
	}

	if lastReq == nil || hasResponse {
		m.permReq = nil
		return
	}

	// Parse permission request from payload
	m.permReq = parsePermissionRequest(lastReq.Payload)
	m.updateLayout()
}

func parsePermissionRequest(payload []byte) *permissionRequest {
	var req acpsdk.RequestPermissionRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return &permissionRequest{
			message: "Permission required",
			options: []permissionOption{
				{id: "allow", label: "Allow"},
				{id: "deny", label: "Deny"},
			},
		}
	}

	message := "Permission required"
	if req.ToolCall.Title != nil {
		message = *req.ToolCall.Title
	}

	var options []permissionOption
	for _, opt := range req.Options {
		options = append(options, permissionOption{
			id:    string(opt.OptionId),
			label: opt.Name,
		})
	}
	if len(options) == 0 {
		options = []permissionOption{
			{id: "allow", label: "Allow"},
			{id: "deny", label: "Deny"},
		}
	}

	return &permissionRequest{
		message: message,
		options: options,
	}
}

func extractAgentText(payload []byte) string {
	var notification acpsdk.SessionNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		return ""
	}
	if notification.Update.AgentMessageChunk == nil {
		return ""
	}
	chunk := notification.Update.AgentMessageChunk
	if chunk.Content.Text != nil {
		return chunk.Content.Text.Text
	}
	return ""
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#89B4FA")).
		Background(lipgloss.Color("#1E1E2E")).
		Padding(0, 1).
		Width(m.width)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6C7086"))

	permStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9E2AF")).
		Bold(true)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086")).
		Width(m.width)

	// Header
	header := titleStyle.Render(fmt.Sprintf(" ACP Console - Task #%d ", m.config.TaskID))
	b.WriteString(header)
	b.WriteString("\n")

	// Viewport (log area)
	vpView := borderStyle.Width(m.width - 2).Render(m.viewport.View())
	b.WriteString(vpView)
	b.WriteString("\n")

	// Permission request (if any)
	if m.permReq != nil {
		var permLine strings.Builder
		permLine.WriteString(permStyle.Render(" " + m.permReq.message + ": "))
		for i, opt := range m.permReq.options {
			permLine.WriteString(fmt.Sprintf("[%d] %s  ", i+1, opt.label))
		}
		b.WriteString(permLine.String())
		b.WriteString("\n")
	}

	// Input area
	inputView := borderStyle.Width(m.width - 2).Render(m.textarea.View())
	b.WriteString(inputView)
	b.WriteString("\n")

	// Status bar
	stateStr := string(m.state.ExecutionSubstate)
	if stateStr == "" {
		stateStr = "unknown"
	}
	status := fmt.Sprintf(" State: %s | Enter: send | Esc: cancel | Ctrl+D: stop | Ctrl+C: quit", stateStr)
	if m.err != nil {
		status = fmt.Sprintf(" Error: %v", m.err)
	}
	b.WriteString(statusStyle.Render(status))

	return b.String()
}

// Run starts the ACP console TUI.
func Run(cfg Config) error {
	m := New(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
