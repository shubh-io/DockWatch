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
		if selected {
			return selectedStyle.Render(projectLabel)
		}
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

	visible := m.settings.VisibleColumns
	if visible == nil || len(visible) != 9 {
		visible = []bool{true, true, true, true, true, true, true, true, true}
		m.settings.VisibleColumns = visible
	}
	rows := []struct {
		idx   int
		width int
		val   string
	}{
		{0, idW - 1, id},
		{1, nameW - 1, containerName},
		{2, memoryW - 2, mem},
		{3, cpuW - 2, cpu},
		{4, netIOW - 1, netio},
		{5, blockIOW - 1, blockio},
		{6, imageW - 1, img},
		{7, statusW, status},
		{8, portsW - 2, ports},
	}

	var rowStr string
	first_row := true
	for _, row := range rows {
		if !visible[row.idx] {
			continue
		}
		if first_row {
			rowStr += fmt.Sprintf("%-*s", row.width, row.val)
			first_row = false

		} else {
			rowStr += fmt.Sprintf("│ %-*s", row.width, row.val)
		}

	}

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
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *model) moveCursorDownTree() {
	if len(m.flatList) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < len(m.flatList)-1 {
		m.cursor++
	}
}

// refreshInfoContainer rebinds m.infoContainer to the current container instance
func (m *model) refreshInfoContainer() {
	if m.infoContainer == nil {
		return
	}
	id := m.infoContainer.ID

	// determine whether to prefer compose project containers
	preferProject := m.infoContainer.ComposeProject != "" || m.composeViewMode

	if preferProject {
		// search project containers first
		for _, p := range m.projects {
			for i := range p.Containers {
				if p.Containers[i].ID == id {
					m.infoContainer = &p.Containers[i]
					return
				}
			}
		}
		// fallback to normal containers
		for i := range m.containers {
			if m.containers[i].ID == id {
				m.infoContainer = &m.containers[i]
				return
			}
		}
	} else {

		for i := range m.containers {
			if m.containers[i].ID == id {
				m.infoContainer = &m.containers[i]
				return
			}
		}
		for _, p := range m.projects {
			for i := range p.Containers {
				if p.Containers[i].ID == id {
					m.infoContainer = &p.Containers[i]
					return
				}
			}
		}
	}
}

func (m *model) getSelectedProject() (string, string) {
	if !m.composeViewMode || len(m.flatList) == 0 {
		return "", ""
	}
	if m.cursor >= len(m.flatList) {
		return "", ""
	}
	row := m.flatList[m.cursor]
	projectName := row.projectName
	if !row.isProject {
		if row.container != nil {
			projectName = row.container.ComposeProject
		}
	}

	if projectName == "" || projectName == "Standalone Containers" {
		return "", ""
	}

	if proj, ok := m.projects[projectName]; ok {
		return projectName, proj.WorkingDir
	}
	return projectName, ""
}

func (m *model) isProjectSelected() bool {
	if !m.composeViewMode || len(m.flatList) == 0 || m.cursor >= len(m.flatList) {
		return false
	}
	return m.flatList[m.cursor].isProject
}
