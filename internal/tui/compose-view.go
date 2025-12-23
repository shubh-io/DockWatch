package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/shubh-io/dockmate/internal/docker"
)

func (m *model) buildFlatList() {
	m.flatList = []treeRow{}

	// sort projects by name
	projectNames := []string{}
	for name := range m.projects {
		projectNames = append(projectNames, name)
	}
	sort.Strings(projectNames)

	// Add compose projects
	for _, projectName := range projectNames {
		project := m.projects[projectName]
		running := 0
		for _, c := range project.Containers {
			if strings.ToLower(c.State) == "running" {
				running++
			}
		}
		total := len(project.Containers)

		// Add project row
		m.flatList = append(m.flatList, treeRow{
			isProject:   true,
			projectName: projectName,
			running:     running,
			total:       total,
			indent:      0,
		})

		// Add container rows if expanded
		if m.expandedProjects[projectName] {
			for i := range project.Containers {
				m.flatList = append(m.flatList, treeRow{
					isProject: false,
					container: &project.Containers[i],
					indent:    1,
				})
			}
		}
	}

	// Find standalone containers
	standaloneContainers := []*docker.Container{}
	composeContainerIDs := make(map[string]bool)

	for _, project := range m.projects {
		for _, c := range project.Containers {
			composeContainerIDs[c.ID] = true
		}
	}

	for i := range m.containers {
		if !composeContainerIDs[m.containers[i].ID] {
			standaloneContainers = append(standaloneContainers, &m.containers[i])
		}
	}

	// Add standalone section if any exist
	if len(standaloneContainers) > 0 {
		m.flatList = append(m.flatList, treeRow{
			isProject:   true,
			projectName: "Standalone Containers",
			total:       len(standaloneContainers),
			indent:      0,
		})

		if m.expandedProjects["Standalone Containers"] {
			for _, container := range standaloneContainers {
				m.flatList = append(m.flatList, treeRow{
					isProject: false,
					container: container,
					indent:    1,
				})
			}
		}
	}
}

func (m model) renderTreeRow(row treeRow, selected bool, idW, nameW, memoryW, cpuW, netIOW, blockIOW, imageW, statusW, portsW, totalWidth int) string {
	if row.isProject {
		// Project header row
		expandIcon := "▼"
		if !m.expandedProjects[row.projectName] {
			expandIcon = "▶"
		}

		projectLabel := fmt.Sprintf(" %s %s [%d/%d running]", expandIcon, row.projectName, row.running, row.total)
		if visibleLen(projectLabel) < totalWidth {
			projectLabel += strings.Repeat(" ", totalWidth-visibleLen(projectLabel))
		}

		// Project row style
		projectStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		return projectStyle.Render(projectLabel)
	}

	c := row.container
	if c == nil {
		return normalStyle.Render(strings.Repeat(" ", totalWidth))
	}

	name := ""
	if len(c.Names) > 0 {
		name = c.Names[0]
		name = strings.TrimPrefix(name, "/")
	}

	indentStr := ""
	if row.indent > 0 {
		indentStr = " ├─ "
	}

	id := c.ID
	if visibleLen(id) > idW-2 {
		id = truncateToWidth(id, idW-2)
	}

	containerName := indentStr + name
	if visibleLen(containerName) > nameW-2 {
		containerName = truncateToWidth(containerName, nameW-2)
	}

	img := c.Image
	if visibleLen(img) > imageW-2 {
		img = truncateToWidth(img, imageW-2)
	}

	status := c.Status
	if visibleLen(status) > statusW-2 {
		status = truncateToWidth(status, statusW-2)
	}

	mem := c.Memory
	if mem == "" {
		mem = "─"
	}
	if visibleLen(mem) > memoryW-2 {
		mem = truncateToWidth(mem, memoryW-2)
	}

	cpu := c.CPU
	if cpu == "" {
		cpu = "─"
	}
	if visibleLen(cpu) > cpuW-2 {
		cpu = truncateToWidth(cpu, cpuW-2)
	}

	netio := c.NetIO
	if netio == "" {
		netio = "─"
	}
	if visibleLen(netio) > netIOW-2 {
		netio = truncateToWidth(netio, netIOW-2)
	}

	blockio := c.BlockIO
	if blockio == "" {
		blockio = "─"
	}
	if visibleLen(blockio) > blockIOW-2 {
		blockio = truncateToWidth(blockio, blockIOW-2)
	}

	ports := c.Ports
	if ports == "" {
		ports = "─"
	}
	if visibleLen(ports) > portsW-7 {
		ports = truncateToWidth(ports, portsW-6)
	}

	rowStr := fmt.Sprintf(" %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s",
		idW-1, id,
		nameW-1, containerName,
		memoryW-2, mem,
		cpuW-2, cpu,
		netIOW-1, netio,
		blockIOW-1, blockio,
		imageW-1, img,
		statusW, status,
		portsW-2, ports)

	if visibleLen(rowStr) < totalWidth {
		rowStr += strings.Repeat(" ", totalWidth-visibleLen(rowStr))
	}

	if selected {
		return selectedStyle.Render(rowStr)
	}

	switch strings.ToLower(c.State) {
	case "running":
		return runningStyle.Render(rowStr)
	case "paused":
		return pausedStyle.Render(rowStr)
	case "exited", "dead":
		return stoppedStyle.Render(rowStr)
	default:
		return normalStyle.Render(rowStr)
	}
}

func (m *model) moveCursorUpTree() {
	if len(m.flatList) == 0 {
		m.cursor = 0
		return
	}
	i := m.cursor - 1
	for i >= 0 && m.flatList[i].isProject {
		i--
	}
	if i >= 0 {
		m.cursor = i
	} else {
		// clamp to first non-project if any
		for j := 0; j < len(m.flatList); j++ {
			if !m.flatList[j].isProject {
				m.cursor = j
				return
			}
		}
		m.cursor = 0
	}
}

func (m *model) moveCursorDownTree() {
	if len(m.flatList) == 0 {
		m.cursor = 0
		return
	}
	i := m.cursor + 1
	for i < len(m.flatList) && m.flatList[i].isProject {
		i++
	}
	if i < len(m.flatList) {
		m.cursor = i
	} else {
		// clamp to last non-project if any
		for j := len(m.flatList) - 1; j >= 0; j-- {
			if !m.flatList[j].isProject {
				m.cursor = j
				return
			}
		}
		m.cursor = len(m.flatList) - 1
	}
}
