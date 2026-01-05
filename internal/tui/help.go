package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
)

// renderHelp shows a full-screen help view with all keyboard shortcuts
func (m model) renderHelp(width int) string {

	return m.helpList.View()
}

func getHelpItems(m model) []list.Item {
	return []list.Item{
		item{"↑ / ↓", "Move cursor up/down"},
		item{"← / →", "Navigate between pages"},
		item{"Tab", "Toggle column selection mode"},
		item{"Enter", "Sort by selected column (in column mode)"},
		item{"S", "Start selected container"},
		item{"X", "Stop selected container"},
		item{"R", "Restart selected container"},
		item{"D", "Remove selected container"},
		item{"E", fmt.Sprintf("Open interactive shell (%s)", m.settings.Shell)},
		item{"L", "View/Toggle logs (container or compose project)"},
		item{"I", "View/Toggle container info"},
		item{"U", "Compose: up / start project"},
		item{"D", "Compose: down / stop project"},
		item{"R", "Compose: restart project"},
		item{"P", "Compose: pause/unpause project"},
		item{"X", "Compose: stop all containers in project"},
		item{"C", "Toggle compose/normal view"},
		item{"F2", "Open settings"},
		item{"F1", "Show this help"},
		item{"q", "Quit application"},
		item{"Esc", "Back/Cancel"},
	}

}

type item struct {
	key, desc string
}

func (i item) Title() string       { return i.key }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return "" }
