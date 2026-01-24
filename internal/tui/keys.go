package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings for the TUI.
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	PrevPage key.Binding
	NextPage key.Binding

	// Actions
	Enter   key.Binding // Execute default action
	Default key.Binding // Show actions menu
	Start   key.Binding // Start task with agent
	Stop    key.Binding // Stop running session
	Attach  key.Binding // Attach to session
	Exec    key.Binding // Execute command
	Review  key.Binding // Run review on task

	// Task management
	New        key.Binding // Create new task
	Copy       key.Binding // Copy task
	Delete     key.Binding // Delete task
	Edit       key.Binding // Edit task in editor (with comments)
	EditStatus key.Binding // Change task status

	// Workflow
	Merge   key.Binding // Merge task
	Close   key.Binding // Close task
	Block   key.Binding // Block/unblock task
	PR      key.Binding // Create PR (future)
	Manager key.Binding // Start/attach manager session

	// View
	Refresh          key.Binding // Refresh task list
	Filter           key.Binding // Enter filter mode
	Sort             key.Binding // Toggle sort mode
	Help             key.Binding // Show help
	Detail           key.Binding // Toggle detail view
	ToggleShowAll    key.Binding // Toggle show all (including closed)
	ToggleReviewMode key.Binding // Cycle review mode (auto -> manual -> auto_fix)

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
		PrevPage: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev page"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next page"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "default"),
		),
		Default: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "actions"),
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
			key.WithKeys("a"),
			key.WithHelp("a", "attach"),
		),
		Exec: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "execute"),
		),
		Review: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "review"),
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
		Edit: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "edit task"),
		),
		EditStatus: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "change status"),
		),
		Merge: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "merge"),
		),
		Close: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "close"),
		),
		Block: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "block"),
		),
		PR: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "create PR"),
		),
		Manager: key.NewBinding(
			key.WithKeys("M"),
			key.WithHelp("M", "manager"),
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
			key.WithKeys("A"),
			key.WithHelp("A", "toggle all"),
		),
		ToggleReviewMode: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "review mode"),
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
	return []key.Binding{k.Up, k.Down, k.PrevPage, k.NextPage, k.Enter, k.Default, k.Start, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PrevPage, k.NextPage, k.Enter}, // Navigation
		{k.Default}, // Default action

		{k.Start, k.Stop, k.Attach, k.Exec, k.Review},   // Session
		{k.New, k.Copy, k.Delete, k.Edit, k.EditStatus}, // Task management
		{k.Merge, k.Close, k.Block},                     // Workflow
		{k.Refresh, k.Filter, k.Detail, k.Help, k.Quit}, // View & general
	}
}

// GetBuiltinKeys returns a set of all keys used by builtin keybindings.
// This is used to detect conflicts with custom keybindings.
func (k KeyMap) GetBuiltinKeys() map[string]bool {
	keys := make(map[string]bool)

	// Add all keys from all bindings
	addKeys := func(binding key.Binding) {
		for _, k := range binding.Keys() {
			keys[k] = true
		}
	}

	addKeys(k.Up)
	addKeys(k.Down)
	addKeys(k.PrevPage)
	addKeys(k.NextPage)
	addKeys(k.Enter)
	addKeys(k.Default)
	addKeys(k.Start)
	addKeys(k.Stop)
	addKeys(k.Attach)
	addKeys(k.Exec)
	addKeys(k.Review)
	addKeys(k.New)
	addKeys(k.Copy)
	addKeys(k.Delete)
	addKeys(k.Edit)
	addKeys(k.EditStatus)
	addKeys(k.Merge)
	addKeys(k.Close)
	addKeys(k.Block)
	addKeys(k.PR)
	addKeys(k.Manager)
	addKeys(k.Refresh)
	addKeys(k.Filter)
	addKeys(k.Sort)
	addKeys(k.Help)
	addKeys(k.Detail)
	addKeys(k.ToggleShowAll)
	addKeys(k.ToggleReviewMode)
	addKeys(k.Quit)
	addKeys(k.Escape)
	addKeys(k.Confirm)

	return keys
}
