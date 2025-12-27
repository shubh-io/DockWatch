package tui

import (
	"fmt"
	"strings"

	"github.com/shubh-io/dockmate/internal/docker"
)

func (m model) renderInfoPanel(width int) string {
	var b strings.Builder

	b.WriteString(dividerStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	id := m.infoContainerID
	if id == "" && m.infoContainer != nil {
		id = m.infoContainer.ID
	}

	var container *docker.Container

	if id != "" {

		for _, p := range m.projects {
			for i := range p.Containers {
				if p.Containers[i].ID == id {
					container = &p.Containers[i]
					break
				}
			}
			if container != nil {
				break
			}
		}
		if container == nil {
			for i := range m.containers {
				if m.containers[i].ID == id {
					container = &m.containers[i]
					break
				}
			}
		}
	}

	containerName := ""
	if container != nil && len(container.Names) > 0 {
		containerName = container.Names[0]
	}

	infoTitle := fmt.Sprintf("Container Info: %s ", containerName)
	if visibleLen(infoTitle) < width {
		infoTitle += strings.Repeat(" ", width-visibleLen(infoTitle))
	}
	b.WriteString(titleStyle.Render(infoTitle))
	b.WriteString("\n")

	if container == nil {
		noContainerMsg := "  No container selected"
		if visibleLen(noContainerMsg) < width {
			noContainerMsg += strings.Repeat(" ", width-visibleLen(noContainerMsg))
		}
		b.WriteString(normalStyle.Render(noContainerMsg))
		b.WriteString("\n")
		return b.String()
	}

	// Display container information fields
	infoFields := []struct {
		label string
		value string
	}{
		{"Container ID", container.ID},
		{"Name", containerName},
		{"Image", container.Image},
		{"Status", container.Status},
		{"State", container.State},
		{"CPU Usage", container.CPU},
		{"Memory Usage", container.Memory},
		{"Network I/O", container.NetIO},
		{"Block I/O", container.BlockIO},
		{"Ports", container.Ports},
		// {"Compose Project", container.ComposeProject},
		// {"Compose File Directory", container.ComposeFileDirectory},
		// {"Compose Directory", container.ComposeDirectory},
		// {"Compose Service", container.ComposeService},
	}

	// Add compose-specific fields if available
	if container.ComposeProject != "" {
		infoFields = append(infoFields, struct {
			label string
			value string
		}{"Compose Project", container.ComposeProject})
	}
	if container.ComposeDirectory != "" {
		infoFields = append(infoFields, struct {
			label string
			value string
		}{"Compose Directory", container.ComposeDirectory})
	}
	if container.ComposeFileDirectory != "" {
		infoFields = append(infoFields, struct {
			label string
			value string
		}{"Compose File Directory", container.ComposeFileDirectory})
	}
	if container.ComposeService != "" {
		infoFields = append(infoFields, struct {
			label string
			value string
		}{"Compose Service", container.ComposeService})
	}
	panelHeight := m.infoPanelHeight
	if container != nil && container.ComposeFileDirectory == "" {
		panelHeight -= 4
	}

	maxInfoLines := panelHeight - 2 // account for divider and title
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
