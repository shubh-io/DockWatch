package tui

import (
	"fmt"
	"strings"
)

func (m model) renderInfoPanel(width int) string {
	var b strings.Builder

	b.WriteString(dividerStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	containerName := ""
	if m.infoContainer != nil && len(m.infoContainer.Names) > 0 {
		containerName = m.infoContainer.Names[0]
	}
	infoTitle := fmt.Sprintf("Container Info: %s ", containerName)
	if visibleLen(infoTitle) < width {
		infoTitle += strings.Repeat(" ", width-visibleLen(infoTitle))
	}
	b.WriteString(titleStyle.Render(infoTitle))
	b.WriteString("\n")

	if m.infoContainer == nil {
		noContainerMsg := "  No container selected"
		if visibleLen(noContainerMsg) < width {
			noContainerMsg += strings.Repeat(" ", width-visibleLen(noContainerMsg))
		}
		b.WriteString(normalStyle.Render(noContainerMsg))
		b.WriteString("\n")
		return b.String()
	}

	c := m.infoContainer

	// Display container information fields
	infoFields := []struct {
		label string
		value string
	}{
		{"Container ID", c.ID},
		{"Name", containerName},
		{"Image", c.Image},
		{"Status", c.Status},
		{"State", c.State},
		{"CPU Usage", c.CPU},
		{"Memory Usage", c.Memory},
		{"Network I/O", c.NetIO},
		{"Block I/O", c.BlockIO},
		{"Ports", c.Ports},
	}

	// Add compose-specific fields if available
	if c.ComposeProject != "" {
		infoFields = append(infoFields, struct {
			label string
			value string
		}{"Compose Project", c.ComposeProject})
	}
	if c.ComposeService != "" {
		infoFields = append(infoFields, struct {
			label string
			value string
		}{"Compose Service", c.ComposeService})
	}

	maxInfoLines := m.infoPanelHeight - 2 // account for divider and title
	if maxInfoLines < 1 {
		maxInfoLines = 1
	}

	// Render info fields with wrapping
	renderedLines := 0
	for _, field := range infoFields {
		if renderedLines >= maxInfoLines {
			break
		}

		value := field.value
		if value == "" {
			value = "─"
		}

		labelRendered := infoLabelStyle.Render(field.label)
		labelPart := fmt.Sprintf("  %s: ", labelRendered)

		valueMaxWidth := width - visibleLen(labelPart)

		if valueMaxWidth <= 0 {
			if renderedLines < maxInfoLines {
				line := labelPart
				if visibleLen(line) < width {
					line += strings.Repeat(" ", width-visibleLen(line))
				}
				b.WriteString(normalStyle.Render(line))
				b.WriteString("\n")
				renderedLines++
			}
			continue
		}

		valueLines := wrapText(value, valueMaxWidth)

		// First line with label
		if len(valueLines) > 0 {
			if renderedLines < maxInfoLines {
				line := labelPart + infoValueStyle.Render(valueLines[0])
				if visibleLen(line) < width {
					line += strings.Repeat(" ", width-visibleLen(line))
				}
				b.WriteString(normalStyle.Render(line))
				b.WriteString("\n")
				renderedLines++
			}
		} else {
			if renderedLines < maxInfoLines {
				line := labelPart
				if visibleLen(line) < width {
					line += strings.Repeat(" ", width-visibleLen(line))
				}
				b.WriteString(normalStyle.Render(line))
				b.WriteString("\n")
				renderedLines++
			}
		}

		// Subsequent lines, indented
		if len(valueLines) > 1 {
			indent := strings.Repeat(" ", visibleLen(labelPart))
			for i := 1; i < len(valueLines); i++ {
				if renderedLines >= maxInfoLines {
					break
				}
				line := indent + infoValueStyle.Render(valueLines[i])
				if visibleLen(line) < width {
					line += strings.Repeat(" ", width-visibleLen(line))
				}
				b.WriteString(normalStyle.Render(line))
				b.WriteString("\n")
				renderedLines++
			}
		}
	}

	// Fill remaining lines with empty space
	for i := renderedLines; i < maxInfoLines; i++ {
		b.WriteString(normalStyle.Render(strings.Repeat(" ", width)))
		b.WriteString("\n")
	}

	return b.String()
}

// wrapText performs hard wrapping on a string.
func wrapText(text string, maxWidth int) []string {
	var lines []string
	if maxWidth <= 0 || text == "" {
		lines = append(lines, text)
		return lines
	}

	runes := []rune(text)
	for len(runes) > 0 {
		width := maxWidth
		if len(runes) < width {
			width = len(runes)
		}
		lines = append(lines, string(runes[:width]))
		runes = runes[width:]
	}
	return lines
}
