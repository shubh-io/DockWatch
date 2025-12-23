package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

// ============================================================================
// Keyboard shortcuts
// ============================================================================

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Start    key.Binding
	Stop     key.Binding
	Restart  key.Binding
	Logs     key.Binding
	Info     key.Binding
	Exec     key.Binding
	Remove   key.Binding
	Refresh  key.Binding
	PageUp   key.Binding
	NextPage key.Binding
	PrevPage key.Binding
	PageDown key.Binding
	Quit     key.Binding
	Help     key.Binding
}

var Keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k")),
	Down:     key.NewBinding(key.WithKeys("down", "j")),
	Start:    key.NewBinding(key.WithKeys("s", "S")),
	Stop:     key.NewBinding(key.WithKeys("x", "X")),
	Logs:     key.NewBinding(key.WithKeys("l", "L")),
	Info:     key.NewBinding(key.WithKeys("i", "I")),
	Exec:     key.NewBinding(key.WithKeys("e", "E")),
	Restart:  key.NewBinding(key.WithKeys("r", "R")),
	Remove:   key.NewBinding(key.WithKeys("d", "D")),
	Refresh:  key.NewBinding(key.WithKeys("f5")),
	PageUp:   key.NewBinding(key.WithKeys("pgup", "left")),
	NextPage: key.NewBinding(key.WithKeys("n", "pagedown")),
	PrevPage: key.NewBinding(key.WithKeys("p", "pageup")),
	PageDown: key.NewBinding(key.WithKeys("pgdown", "right")),
	Quit:     key.NewBinding(key.WithKeys("q", "Q", "ctrl+c", "f10")),
	Help:     key.NewBinding(key.WithKeys("f1", "?")),
}
