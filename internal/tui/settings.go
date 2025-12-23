package tui

import (
	"fmt"
	"strings"
)

func (m model) renderSettings(width int) string {
	var b strings.Builder

	title := titleStyle.Render("┌─ Settings 🛠️─┐")
	padding := (width - visibleLen(title)) / 2
	if padding < 0 {
		padding = 0
	}
	header := strings.Repeat(" ", padding) + title
	if visibleLen(header) < width {
		header += strings.Repeat(" ", width-visibleLen(header))
	}
	b.WriteString(header)
	b.WriteString("\n")

	// Column list
	colNames := []string{"CONTAINER ID", "NAME", "MEMORY", "CPU", "NET I/O", "Disk I/O", "IMAGE", "STATUS", "PORTS"}
	if m.settings.ColumnPercents == nil || len(m.settings.ColumnPercents) != 9 {
		m.settings.ColumnPercents = []int{8, 14, 6, 6, 10, 12, 18, 13, 13}
	}

	for i, name := range colNames {
		pct := m.settings.ColumnPercents[i]
		line := fmt.Sprintf(" %2d%%  %s", pct, name)
		if m.settingsSelected == i {
			// highlight selected
			b.WriteString(selectedStyle.Render(padRight(line, width)))
		} else {
			b.WriteString(normalStyle.Render(padRight(line, width)))
		}
		b.WriteString("\n")
	}

	// Refresh interval row (index 9)
	b.WriteString("\n")
	refreshLine := fmt.Sprintf(" %2ds  Refresh Interval", m.settings.RefreshInterval)
	if m.settingsSelected == 9 {
		b.WriteString(selectedStyle.Render(padRight(refreshLine, width)))
	} else {
		b.WriteString(normalStyle.Render(padRight(refreshLine, width)))
	}
	b.WriteString("\n")

	// runtime row (index 10)
	b.WriteString("\n")
	runtime := fmt.Sprintf("Runtime: %s", m.settings.Runtime)
	if m.settingsSelected == 10 {
		b.WriteString(selectedStyle.Render(padRight(runtime, width)))
	} else {
		b.WriteString(normalStyle.Render(padRight(runtime, width)))
	}
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("Changing the runtime will trigger a RESTART!"))

	// shell row (index 11)
	b.WriteString("\n\n")
	shellLine := fmt.Sprintf("Shell: %s", m.settings.Shell)
	if m.settingsSelected == 11 {
		b.WriteString(selectedStyle.Render(padRight(shellLine, width)))
	} else {
		b.WriteString(normalStyle.Render(padRight(shellLine, width)))
	}
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("Shell used for container exec (fallback: /bin/sh)"))

	b.WriteString("\n")
	instr := "[←/→] or [+/-] adjust  •  [↑/↓] navigate • [s] save  •   [Esc] cancel"
	if visibleLen(instr) < width {
		instr += strings.Repeat(" ", width-visibleLen(instr))
	}
	b.WriteString(infoValueStyle.Render(instr))
	b.WriteString("\n")

	return b.String()
}
