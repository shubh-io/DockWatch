package tui

import (
	"fmt"
	"strings"
)

// renderHelp shows a full-screen help view with all keyboard shortcuts
func (m model) renderHelp(width int) string {
	var b strings.Builder

	title := titleStyle.Render("┌─ Help ─┐")
	padding := (width - visibleLen(title)) / 2
	if padding < 0 {
		padding = 0
	}
	header := strings.Repeat(" ", padding) + title
	if visibleLen(header) < width {
		header += strings.Repeat(" ", width-visibleLen(header))
	}
	b.WriteString(header)
	b.WriteString("\n\n")

	// Define help sections with their keybindings
	helpSections := []struct {
		title string
		items []struct {
			key  string
			desc string
		}
	}{
		{
			title: "Navigation",
			items: []struct {
				key  string
				desc string
			}{
				{"↑ / ↓", "Move cursor up/down"},
				{"← / →", "Navigate between pages"},
				{"Tab", "Toggle column selection mode"},
				{"Enter", "Sort by selected column (in column mode)"},
			},
		},
		{
			title: "Container Actions",
			items: []struct {
				key  string
				desc string
			}{
				{"S", "Start selected container"},
				{"X", "Stop selected container"},
				{"R", "Restart selected container"},
				{"D", "Remove selected container"},
				{"E", fmt.Sprintf("Open interactive shell (%s)", m.settings.Shell)},
			},
		},
		{
			title: "View & Information",
			items: []struct {
				key  string
				desc string
			}{
				{"L", "View/Toggle container logs"},
				{"I", "View/Toggle container info"},
				{"C", "Toggle compose/normal view"},
			},
		},
		{
			title: "Application",
			items: []struct {
				key  string
				desc string
			}{
				{"F2", "Open settings"},
				{"?", "Show this help"},
				{"q", "Quit application"},
				{"Esc", "Back/Cancel"},
			},
		},
	}

	// Render each section, one by one
	for _, section := range helpSections {
		// Section title
		sectionTitle := infoLabelStyle.Render("━━ " + section.title + " ━━━━━━━━━━━━━━━━━━━━━━")
		b.WriteString(sectionTitle)
		b.WriteString("\n\n")

		// Section items
		for _, item := range section.items {
			keyPart := footerKeyStyle.Render(fmt.Sprintf("  %-12s", item.key))
			descPart := normalStyle.Render(item.desc)
			line := keyPart + " " + descPart
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	instr := "[?] or [Esc] to close  •  Visit https://github.com/shubh-io/dockmate for more info"
	if visibleLen(instr) < width {
		instr += strings.Repeat(" ", width-visibleLen(instr))
	}
	b.WriteString(infoValueStyle.Render(instr))
	b.WriteString("\n")

	return b.String()
}
