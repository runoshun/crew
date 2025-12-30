package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func TestAllAgents(t *testing.T) {
	// Create a minimal model with test data
	m := &Model{
		builtinAgents: []string{"claude", "opencode", "codex"},
		customAgents:  []string{"my-agent", "another-agent"},
	}

	got := m.allAgents()
	want := []string{"claude", "opencode", "codex", "my-agent", "another-agent"}

	if len(got) != len(want) {
		t.Errorf("allAgents() returned %d agents, want %d", len(got), len(want))
		return
	}

	for i, agent := range want {
		if got[i] != agent {
			t.Errorf("allAgents()[%d] = %q, want %q", i, got[i], agent)
		}
	}
}

func TestAllAgents_NoCustom(t *testing.T) {
	m := &Model{
		builtinAgents: []string{"claude", "opencode"},
		customAgents:  nil,
	}

	got := m.allAgents()

	if len(got) != 2 {
		t.Errorf("allAgents() returned %d agents, want 2", len(got))
	}
}

func TestStartModeInitialState(t *testing.T) {
	m := &Model{
		mode:             ModeStart,
		agentCursor:      0,
		startFocusCustom: false,
	}

	// Initial state should have list focus
	if m.startFocusCustom {
		t.Error("Initial startFocusCustom should be false (list focused)")
	}

	// Agent cursor should be at 0
	if m.agentCursor != 0 {
		t.Errorf("Initial agentCursor = %d, want 0", m.agentCursor)
	}

	// Mode should be ModeStart
	if m.mode != ModeStart {
		t.Errorf("mode = %v, want ModeStart", m.mode)
	}
}

func TestStartModeFocusToggle(t *testing.T) {
	m := &Model{
		startFocusCustom: false,
	}

	// Toggle to custom input
	m.startFocusCustom = !m.startFocusCustom
	if !m.startFocusCustom {
		t.Error("After toggle, startFocusCustom should be true")
	}

	// Toggle back to list
	m.startFocusCustom = !m.startFocusCustom
	if m.startFocusCustom {
		t.Error("After second toggle, startFocusCustom should be false")
	}
}

func TestStartModeAgentCursorBounds(t *testing.T) {
	m := &Model{
		builtinAgents: []string{"claude", "opencode", "codex"},
		customAgents:  []string{"custom1"},
		agentCursor:   0,
	}

	allAgents := m.allAgents()
	maxIndex := len(allAgents) - 1

	// Test cursor at start
	if m.agentCursor != 0 {
		t.Errorf("Initial cursor = %d, want 0", m.agentCursor)
	}

	// Test cursor at end
	m.agentCursor = maxIndex
	if m.agentCursor != 3 {
		t.Errorf("Cursor at end = %d, want 3", m.agentCursor)
	}

	// Test selected agent at different positions
	m.agentCursor = 0
	if allAgents[m.agentCursor] != "claude" {
		t.Errorf("Agent at cursor 0 = %q, want 'claude'", allAgents[m.agentCursor])
	}

	m.agentCursor = 3
	if allAgents[m.agentCursor] != "custom1" {
		t.Errorf("Agent at cursor 3 = %q, want 'custom1'", allAgents[m.agentCursor])
	}
}

func TestStartModeCustomInputValue(t *testing.T) {
	ti := textinput.New()
	m := &Model{
		customInput: ti,
	}

	// Focus on custom input
	m.startFocusCustom = true
	if !m.startFocusCustom {
		t.Error("startFocusCustom should be true after setting")
	}

	// Set custom input value
	m.customInput.SetValue("my-custom-command arg1 arg2")

	got := m.customInput.Value()
	want := "my-custom-command arg1 arg2"

	if got != want {
		t.Errorf("customInput.Value() = %q, want %q", got, want)
	}
}

func TestStartModeAgentCommands(t *testing.T) {
	m := &Model{
		agentCommands: map[string]string{
			"claude":   "claude --dangerously-skip-permissions",
			"opencode": "opencode",
			"custom1":  "my-custom-agent --flag",
		},
	}

	tests := []struct {
		agent string
		want  string
	}{
		{"claude", "claude --dangerously-skip-permissions"},
		{"opencode", "opencode"},
		{"custom1", "my-custom-agent --flag"},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			got := m.agentCommands[tt.agent]
			if got != tt.want {
				t.Errorf("agentCommands[%q] = %q, want %q", tt.agent, got, tt.want)
			}
		})
	}
}

func TestStartModeSeparatorIndex(t *testing.T) {
	tests := []struct {
		name          string
		builtinAgents []string
		customAgents  []string
		wantSeparator int // index after which separator should appear (-1 if no separator)
	}{
		{
			name:          "with custom agents",
			builtinAgents: []string{"claude", "opencode", "codex"},
			customAgents:  []string{"custom1", "custom2"},
			wantSeparator: 2, // separator after index 2 (codex)
		},
		{
			name:          "no custom agents",
			builtinAgents: []string{"claude", "opencode"},
			customAgents:  nil,
			wantSeparator: -1, // no separator needed
		},
		{
			name:          "single builtin with custom",
			builtinAgents: []string{"claude"},
			customAgents:  []string{"custom1"},
			wantSeparator: 0, // separator after index 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				builtinAgents: tt.builtinAgents,
				customAgents:  tt.customAgents,
			}

			// Calculate separator index (last builtin index, or -1 if no custom)
			var separatorIndex int
			if len(m.customAgents) > 0 {
				separatorIndex = len(m.builtinAgents) - 1
			} else {
				separatorIndex = -1
			}

			if separatorIndex != tt.wantSeparator {
				t.Errorf("separator index = %d, want %d", separatorIndex, tt.wantSeparator)
			}
		})
	}
}
