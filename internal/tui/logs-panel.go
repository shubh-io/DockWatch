package tui

import (
	"fmt"
	"strings"
)

func (m model) renderLogsPanel(width int) string {
	var b strings.Builder

	b.WriteString(dividerStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	logsTitle := fmt.Sprintf("Logs: %s ", m.logsContainer)
	if len(logsTitle) < width {
		logsTitle += strings.Repeat(" ", width-len(logsTitle))
	}
	b.WriteString(titleStyle.Render(logsTitle))
	b.WriteString("\n")

	maxLogLines := m.logPanelHeight - 2 // account for divider and title
	if maxLogLines < 1 {
		maxLogLines = 1
	}

	startLog := 0
	if len(m.logsLines) > maxLogLines {
		startLog = len(m.logsLines) - maxLogLines
	}

	for i := startLog; i < len(m.logsLines); i++ {
		logLine := m.logsLines[i]
		if len(logLine) > width-4 {
			logLine = logLine[:width-7] + "..."
		}
		b.WriteString(normalStyle.Render("  " + logLine))
		b.WriteString("\n")
	}

	renderedLines := len(m.logsLines) - startLog
	for i := renderedLines; i < maxLogLines; i++ {
		b.WriteString(normalStyle.Render(strings.Repeat(" ", width)))
		b.WriteString("\n")
	}

	return b.String()
}
