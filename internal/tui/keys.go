package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings for the TUI.
type KeyMap struct {
	// Navigation
	Up   key.Binding
	Down key.Binding

	// Actions
	Enter  key.Binding // Smart action (start if todo, attach if running)
	Start  key.Binding // Start task with agent
	Stop   key.Binding // Stop running session
	Attach key.Binding // Attach to session

	// Task management
	New    key.Binding // Create new task
	Copy   key.Binding // Copy task
	Delete key.Binding // Delete task

	// Workflow
	Merge key.Binding // Merge task
	Close key.Binding // Close task
	PR    key.Binding // Create PR (future)

	// View
	Refresh       key.Binding // Refresh task list
	Filter        key.Binding // Enter filter mode
	Sort          key.Binding // Toggle sort mode
	Help          key.Binding // Show help
	Detail        key.Binding // Toggle detail view
	ToggleShowAll key.Binding // Toggle show all (including closed/done)

	// General
	Quit    key.Binding // Quit application
	Escape  key.Binding // Cancel/back
	Confirm key.Binding // Confirm action (in confirm mode)
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "smart action"),
		),
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start"),
		),
		Stop: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "stop"),
		),
		Attach: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "attach"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new task"),
		),
		Copy: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Merge: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "merge"),
		),
		Close: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "close"),
		),
		PR: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "create PR"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Sort: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "sort"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Detail: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "detail"),
		),
		ToggleShowAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle all"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("y", "confirm"),
		),
	}
}

// ShortHelp returns keybindings to show in the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Start, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},                         // Navigation
		{k.Start, k.Stop, k.Attach},                     // Session
		{k.New, k.Copy, k.Delete},                       // Task management
		{k.Merge, k.Close},                              // Workflow
		{k.Refresh, k.Filter, k.Detail, k.Help, k.Quit}, // View & general
	}
}
