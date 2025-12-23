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
	if len(infoTitle) < width {
		infoTitle += strings.Repeat(" ", width-len(infoTitle))
	}
	b.WriteString(titleStyle.Render(infoTitle))
	b.WriteString("\n")

	if m.infoContainer == nil {
		b.WriteString(normalStyle.Render("  No container selected"))
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

	// Render info fields
	for i := 0; i < len(infoFields) && i < maxInfoLines; i++ {
		field := infoFields[i]
		value := field.value
		if value == "" {
			value = "─"
		}
		// Truncate value if too long
		maxValueLen := width - len(field.label) - 6
		if maxValueLen > 0 && len(value) > maxValueLen {
			value = value[:maxValueLen-3] + "..."
		}
		infoLine := fmt.Sprintf("  %s: %s", infoLabelStyle.Render(field.label), infoValueStyle.Render(value))
		if visibleLen(infoLine) < width {
			infoLine += strings.Repeat(" ", width-visibleLen(infoLine))
		}
		b.WriteString(normalStyle.Render(infoLine))
		b.WriteString("\n")
	}

	// Fill remaining lines with empty space
	renderedLines := len(infoFields)
	if renderedLines > maxInfoLines {
		renderedLines = maxInfoLines
	}
	for i := renderedLines; i < maxInfoLines; i++ {
		b.WriteString(normalStyle.Render(strings.Repeat(" ", width)))
		b.WriteString("\n")
	}

	return b.String()
}
