package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shubh-io/dockmate/internal/config"
	"github.com/shubh-io/dockmate/internal/docker"
)

// layout sizing constants
const (
	HEADER_HEIGHT        = 8
	CONTAINER_ROW_HEIGHT = 1
	LOG_PANEL_HEIGHT     = 15
	INFO_PANEL_HEIGHT    = 16
)

func InitialModel() model {
	// Load configuration from file
	cfg, _ := config.Load()

	columnPercents := []int{
		cfg.Layout.ContainerId,
		cfg.Layout.ContainerNameWidth,
		cfg.Layout.MemoryWidth,
		cfg.Layout.CPUWidth,
		cfg.Layout.NetIOWidth,
		cfg.Layout.DiskIOWidth,
		cfg.Layout.ImageWidth,
		cfg.Layout.StatusWidth,
		cfg.Layout.PortWidth,
	}
	// runtime load

	return model{
		loading:              true,
		startTime:            time.Now(),
		page:                 0,
		maxContainersPerPage: 12,
		terminalWidth:        0,
		terminalHeight:       0,
		projects:             make(map[string]*docker.ComposeProject),
		expandedProjects:     make(map[string]bool),
		flatList:             []treeRow{},
		logsVisible:          false, // logs hidden by default
		logPanelHeight:       LOG_PANEL_HEIGHT,
		infoVisible:          false,
		infoPanelHeight:      INFO_PANEL_HEIGHT,
		infoContainer:        nil,
		sortBy:               sortByStatus,
		sortAsc:              false, // descending
		columnMode:           false,
		selectedColumn:       7,
		currentMode:          modeNormal,

		// Load settings from config file
		settings: Settings{
			ColumnPercents:  columnPercents,
			RefreshInterval: cfg.Performance.PollRate,
			Runtime:         ContainerRuntime(cfg.Runtime.Type),
			Shell:           cfg.Exec.Shell,
		},
		suspendRefresh:   false,
		settingsSelected: 0,
	}
}

// called once at startup
// kicks off container fetch and timer
func (m model) Init() tea.Cmd {

	return tea.Batch(fetchContainers(), tickCmd(time.Duration(m.settings.RefreshInterval)*time.Second))
}

// sort containers by current column and direction
func (m *model) sortContainers() {
	lessContainer := func(a, b docker.Container) bool {

		switch m.sortBy {
		case sortByID:
			return a.ID < b.ID

		case sortByName:
			ai, aj := "", ""

			if len(a.Names) > 0 {
				ai = a.Names[0]
			}

			if len(b.Names) > 0 {
				aj = b.Names[0]
			}
			return strings.ToLower(ai) < strings.ToLower(aj)

		case sortByMemory:
			return parsePercent(a.Memory) < parsePercent(b.Memory)

		case sortByCPU:
			return parsePercent(a.CPU) < parsePercent(b.CPU)
		case sortByImage:
			return strings.ToLower(a.Image) < strings.ToLower(b.Image)

		case sortByStatus:
			return strings.ToLower(a.Status) < strings.ToLower(b.Status)

		case sortByPorts:
			return strings.ToLower(a.Ports) < strings.ToLower(b.Ports)

		case sortByNetIO:

			return parseNetIO(a.NetIO) < parseNetIO(b.NetIO)

		case sortByBlockIO:
			return parseNetIO(a.BlockIO) < parseNetIO(b.BlockIO)
		default:
			return a.ID < b.ID
		}
	}

	// sort main container slice
	sort.Slice(m.containers, func(i, j int) bool {
		if m.sortAsc {
			return lessContainer(m.containers[i], m.containers[j])
		}
		return !lessContainer(m.containers[i], m.containers[j])
	})

	// also sort containers inside each compose project so compose view  matches column sorting
	if len(m.projects) > 0 {
		for _, p := range m.projects {
			sort.Slice(p.Containers, func(i, j int) bool {
				if m.sortAsc {
					return lessContainer(p.Containers[i], p.Containers[j])
				}
				return !lessContainer(p.Containers[i], p.Containers[j])
			})
		}
		if m.composeViewMode {
			m.buildFlatList()
		}
	}
}

// calculateMaxContainers determines how many containers fit on screen given current layout state
func (m *model) calculateMaxContainers() int {
	availableHeight := m.terminalHeight - HEADER_HEIGHT
	if m.logsVisible {
		availableHeight -= m.logPanelHeight
	}
	if m.infoVisible {
		availableHeight -= INFO_PANEL_HEIGHT
	}
	maxContainers := availableHeight / CONTAINER_ROW_HEIGHT
	if maxContainers < 1 {
		return 1
	}
	return maxContainers
}

// updatePagination recalculates page sizing and keeps cursor/page within bounds
func (m *model) updatePagination() {
	m.maxContainersPerPage = m.calculateMaxContainers()
	if m.maxContainersPerPage < 1 {
		m.maxContainersPerPage = 1
	}

	if len(m.containers) == 0 {
		m.cursor = 0
		m.page = 0
		return
	}

	if m.cursor >= len(m.containers) {
		m.cursor = len(m.containers) - 1
	}

	maxPage := (len(m.containers) - 1) / m.maxContainersPerPage
	if maxPage < 0 {
		maxPage = 0
	}
	if m.page > maxPage {
		m.page = maxPage
	}

	if m.cursor < m.page*m.maxContainersPerPage {
		m.page = m.cursor / m.maxContainersPerPage
	}
	if m.cursor >= (m.page+1)*m.maxContainersPerPage {
		m.page = m.cursor / m.maxContainersPerPage
	}

	// keep persistent page indicator up-to-date
	if m.maxContainersPerPage > 0 {
		maxPage = (len(m.containers) - 1) / m.maxContainersPerPage
		if maxPage < 0 {
			maxPage = 0
		}
		m.message = fmt.Sprintf("Page %d/%d", m.page+1, maxPage+1)
	} else {
		m.message = fmt.Sprintf("Page %d/%d", m.page+1, 1)
	}
}

// ============================================================================
// Update (event handler)
// ============================================================================

// handle all incoming events
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		// terminal resized
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		m.updatePagination()
		return m, nil

	case docker.ContainersMsg:
		// got container list
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.containers = msg.Containers
			m.err = nil
			// sort with current settings
			m.sortContainers()
			// If in compose view, just rebuild!!
			if m.currentMode == modeComposeView {
				m.buildFlatList()
			}
		}

		// keep cursor in bounds
		if m.cursor >= len(m.containers) {
			m.cursor = max(0, len(m.containers)-1)
		}

		m.updatePagination()
		return m, nil

	case composeProjectsMsg:
		// received compose projects
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			m.statusMessage = fmt.Sprintf("Error fetching compose projects: %v", msg.Err)
		} else {
			m.projects = msg.Projects
			if m.expandedProjects == nil {
				m.expandedProjects = make(map[string]bool)
			}
			// default expand any projects
			for name := range m.projects {
				if _, exists := m.expandedProjects[name]; !exists {
					m.expandedProjects[name] = true
				}
			}

			// standalone section for lonely containers (not in compose projects)
			if _, ok := m.expandedProjects["Standalone Containers"]; !ok {
				m.expandedProjects["Standalone Containers"] = true
			}
			m.buildFlatList()
			// keep cursor in bounds
			if m.cursor >= len(m.flatList) {
				m.cursor = max(0, len(m.flatList)-1)
			}
		}
		// just update pagination
		m.updatePagination()
		return m, nil

	case docker.LogsMsg:
		// got logs
		if msg.Err != nil {
			m.statusMessage = fmt.Sprintf("Logs error: %v", msg.Err)
			m.logsLines = nil
			m.logsVisible = false
		} else {
			m.logsLines = msg.Lines
			m.logsContainer = msg.ID
			m.logsVisible = true
		}
		m.updatePagination()
		return m, nil

	case actionDoneMsg:
		// docker action finished
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.statusMessage = "Action completed successfully"
		}

		return m, fetchContainers()

	case tickMsg:

		if m.suspendRefresh {
			return m, tickCmd(time.Duration(m.settings.RefreshInterval) * time.Second)
		}
		if m.logsVisible && m.logsContainer != "" {
			return m, tea.Batch(fetchContainers(), tickCmd(time.Duration(m.settings.RefreshInterval)*time.Second), fetchLogsCmd(m.logsContainer))
		}
		if m.composeViewMode {
			// in compose view , refresh both compose projects and containers as per refresh interval
			return m, tea.Batch(fetchComposeProjects(), tickCmd(time.Duration(m.settings.RefreshInterval)*time.Second))
		}
		return m, tea.Batch(fetchContainers(), tickCmd(time.Duration(m.settings.RefreshInterval)*time.Second))

	case tea.KeyMsg:
		// keyboard input
		m.statusMessage = ""

		if msg.String() == "esc" {
			if m.columnMode {
				m.columnMode = false
				m.currentMode = modeNormal
				m.statusMessage = "Back to normal mode"
				return m, nil
			}
			if m.logsVisible {
				m.logsVisible = false
				m.currentMode = modeNormal
				m.updatePagination()
				m.statusMessage = "Logs closed"
				return m, nil
			}
			if m.infoVisible {
				m.infoVisible = false
				m.infoContainer = nil
				m.currentMode = modeNormal
				m.updatePagination()
				m.statusMessage = "Info panel closed"
				return m, nil
			}
		}

		switch msg.String() {

		case "`":
			debugLogger.Printf(
				"STATE SNAPSHOT: width=%d height=%d page=%d cursor=%d perPage=%d selectedColumn=%d",
				m.terminalWidth, m.terminalHeight, m.page, m.cursor, m.maxContainersPerPage, m.selectedColumn,
			)
			m.statusMessage = "Dumped debug snapshot"
			return m, nil

		case "tab":
			// toggle column/row mode
			m.columnMode = !m.columnMode
			if m.columnMode {
				m.currentMode = modeColumnSelect
				m.statusMessage = "Column mode: Use ← → to navigate, Enter to sort"
			} else {
				m.currentMode = modeNormal
				m.statusMessage = "Row mode: Use ↑ ↓ and ← → to navigate containers"
			}
			return m, nil

		case "f2":
			// toggle settings mode - say yes to settings or no to settings
			if m.currentMode == modeSettings {
				m.currentMode = modeNormal
				m.suspendRefresh = false
				m.statusMessage = "Settings closed"
				// normalize percents to sum 100
				total := 0
				for _, p := range m.settings.ColumnPercents {
					total += p
				}
				if total == 0 {
					m.settings.ColumnPercents = []int{8, 14, 6, 6, 10, 12, 18, 13, 13}
				} else if total != 100 {
					// normalize proportionally
					newp := make([]int, len(m.settings.ColumnPercents))
					acc := 0
					for i, p := range m.settings.ColumnPercents {
						np := (p * 100) / total
						newp[i] = np
						acc += np
					}
					// fix rounding
					if acc < 100 {
						newp[0] += 100 - acc
					}
					m.settings.ColumnPercents = newp
				}
				return m, nil
			}
			m.currentMode = modeSettings
			m.suspendRefresh = true
			m.statusMessage = "Settings: adjust column % and refresh interval"
			return m, nil

		case "?":
			// toggle help mode
			if m.currentMode == modeHelp {
				m.currentMode = modeNormal
				m.suspendRefresh = false
				m.statusMessage = "Help closed"
			} else {
				m.currentMode = modeHelp
				m.suspendRefresh = true
				m.statusMessage = "Help: Keyboard shortcuts"
			}
			return m, nil

		case "l", "L":

			var containerID string
			if m.infoVisible {
				return m, nil
			}
			if m.composeViewMode {
				if m.cursor < len(m.flatList) && !m.flatList[m.cursor].isProject {
					containerID = m.flatList[m.cursor].container.ID
				}
			} else {
				if len(m.containers) > 0 {
					containerID = m.containers[m.cursor].ID
				}
			}
			if containerID != "" {
				if m.logsVisible {
					m.logsVisible = false
					m.currentMode = modeNormal
					m.statusMessage = "Logs closed"
					m.updatePagination()
				} else {
					m.logsVisible = true
					m.currentMode = modeLogs
					m.statusMessage = "Fetching logs..."
					m.updatePagination()
					return m, fetchLogsCmd(containerID)
				}
			}
			return m, nil

		case "enter":

			if m.columnMode {
				var col sortColumn
				var canSort bool = true
				switch m.selectedColumn {
				case 0:
					col = sortByID
				case 1:
					col = sortByName
				case 2:
					col = sortByMemory
				case 3:
					col = sortByCPU
				case 4:
					col = sortByNetIO
				case 5:
					col = sortByBlockIO
				case 6:
					col = sortByImage
				case 7:
					col = sortByStatus
				case 8:
					col = sortByPorts
				}

				if canSort {

					if m.sortBy == col {
						m.sortAsc = !m.sortAsc
					} else {
						m.sortBy = col
						m.sortAsc = true
					}
					m.sortContainers()

					dir := "asc"
					if !m.sortAsc {
						dir = "desc"
					}
					colNames := []string{"ID", "Name", "Memory", "CPU", "NET I/O", "Disk I/O", "Image", "Status", "PORTS"}
					m.statusMessage = fmt.Sprintf("Sorted by %s (%s)", colNames[m.selectedColumn], dir)
				}
			}
			return m, nil

		case "left", "h":

			if m.columnMode {
				if m.selectedColumn > 0 {
					m.selectedColumn--
				}
				return m, nil
			}

		case "right":

			if m.columnMode {
				if m.selectedColumn < 8 {
					m.selectedColumn++
				}
				return m, nil
			}
		}

		if m.currentMode == modeSettings {
			switch msg.String() {
			case "up", "k":
				if m.settingsSelected > 0 {
					m.settingsSelected--
				}
				return m, nil
			case "down", "j":
				if m.settingsSelected < 11 {
					m.settingsSelected++
				}
				return m, nil
			case "left", "h", "-":
				if m.settings.ColumnPercents == nil || len(m.settings.ColumnPercents) != 9 {
					m.settings.ColumnPercents = []int{8, 14, 6, 6, 10, 12, 18, 13, 13}
				}
				if m.settingsSelected >= 0 && m.settingsSelected <= 8 {
					if m.settings.ColumnPercents[m.settingsSelected] > 1 {
						m.settings.ColumnPercents[m.settingsSelected]--
					}
				} else if m.settingsSelected == 9 {
					if m.settings.RefreshInterval > 1 {
						m.settings.RefreshInterval--
					}
				} else if m.settingsSelected == 10 {
					// toggle runtime option btwn docker and podman
					if m.settings.Runtime == RuntimeDocker {
						m.settings.Runtime = RuntimePodman
					} else {
						m.settings.Runtime = RuntimeDocker
					}
				} else if m.settingsSelected == 11 {
					// cycle shell options backward
					idx := slices.Index(ShellOptions, m.settings.Shell)
					m.settings.Shell = ShellOptions[(idx-1+len(ShellOptions))%len(ShellOptions)]
				}
				return m, nil
			case "right", "l", "+":
				if m.settings.ColumnPercents == nil || len(m.settings.ColumnPercents) != 9 {
					m.settings.ColumnPercents = []int{8, 14, 6, 6, 10, 12, 18, 13, 13}
				}
				if m.settingsSelected >= 0 && m.settingsSelected <= 8 {
					m.settings.ColumnPercents[m.settingsSelected]++
				} else if m.settingsSelected == 9 {
					if m.settings.RefreshInterval < 300 {
						m.settings.RefreshInterval++
					}
				} else if m.settingsSelected == 10 {
					if m.settings.Runtime == RuntimeDocker {
						m.settings.Runtime = RuntimePodman
					} else {
						m.settings.Runtime = RuntimeDocker
					}
				} else if m.settingsSelected == 11 {
					// cycle shell options forward
					idx := slices.Index(ShellOptions, m.settings.Shell)
					m.settings.Shell = ShellOptions[(idx+1)%len(ShellOptions)]
				}
				return m, nil
			case "s", "S":
				// save settings to yaml and restart
				currentCfg, _ := config.Load()
				// check if runtime is changed
				runtimeChanged := string(m.settings.Runtime) != currentCfg.Runtime.Type
				// Create .yaml config from current settings
				cfg := &config.Config{
					Layout: config.LayoutConfig{
						ContainerId:        m.settings.ColumnPercents[0],
						ContainerNameWidth: m.settings.ColumnPercents[1],
						MemoryWidth:        m.settings.ColumnPercents[2],
						CPUWidth:           m.settings.ColumnPercents[3],
						NetIOWidth:         m.settings.ColumnPercents[4],
						DiskIOWidth:        m.settings.ColumnPercents[5],
						ImageWidth:         m.settings.ColumnPercents[6],
						StatusWidth:        m.settings.ColumnPercents[7],
						PortWidth:          m.settings.ColumnPercents[8],
					},
					Performance: config.PerformanceConfig{
						PollRate: m.settings.RefreshInterval,
					},
					Runtime: config.RuntimeConfig{
						Type: string(m.settings.Runtime),
					},
					Exec: config.ExecConfig{
						Shell: m.settings.Shell,
					},
				}

				// Save to config
				if err := cfg.Save(); err != nil {
					m.statusMessage = fmt.Sprintf("Failed to save config: %v", err)
				} else {
					if runtimeChanged {
						m.statusMessage = "Settings saved! Restarting app..."
						// Create restart marker file so our app can detect and restart
						markerPath := filepath.Join(os.TempDir(), ".dockmate_restart")
						os.WriteFile(markerPath, []byte{}, 0644)
						// Exit app to restart with new settings
						return m, tea.Sequence(
							tea.Tick(800*time.Millisecond, func(t time.Time) tea.Msg { return tea.QuitMsg{} }),
						)
					}
					total := 0
					for _, p := range m.settings.ColumnPercents {
						total += p
					}
					if total == 0 {
						m.settings.ColumnPercents = []int{8, 14, 6, 6, 10, 12, 18, 13, 13}
					} else if total != 100 {
						newp := make([]int, len(m.settings.ColumnPercents))
						acc := 0
						for i, p := range m.settings.ColumnPercents {
							np := (p * 100) / total
							newp[i] = np
							acc += np
						}
						if acc < 100 {
							newp[0] += 100 - acc
						}
						m.settings.ColumnPercents = newp
					}
					m.currentMode = modeNormal
					m.suspendRefresh = false
					m.statusMessage = "Settings saved!"
					return m, tea.Batch(fetchContainers(), tickCmd(time.Duration(m.settings.RefreshInterval)*time.Second))
				}
				return m, nil
			case "esc":
				m.currentMode = modeNormal
				m.suspendRefresh = false
				m.statusMessage = "Settings closed"
				return m, nil
			}
		}

		// Handle key bindings
		switch {
		case key.Matches(msg, Keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, Keys.Up):
			if !m.columnMode {
				if m.composeViewMode {
					if len(m.flatList) > 0 {
						m.moveCursorUpTree()
					}
				} else {
					if m.cursor > 0 {
						m.cursor--
					}
				}
				if m.maxContainersPerPage > 0 && m.cursor < m.page*m.maxContainersPerPage {
					m.page--
					if m.page < 0 {
						m.page = 0
					}
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}

		case key.Matches(msg, Keys.Down):
			if !m.columnMode {
				if m.composeViewMode {
					if len(m.flatList) > 0 {
						m.moveCursorDownTree()
					}
				} else {
					maxItems := len(m.containers) - 1
					if m.cursor < maxItems {
						m.cursor++
					}
				}
				if m.maxContainersPerPage > 0 && m.cursor >= (m.page+1)*m.maxContainersPerPage {
					m.page++
				}
			}

		case key.Matches(msg, Keys.PageUp):
			if m.page > 0 {
				m.page--
				if m.maxContainersPerPage > 0 {
					if m.composeViewMode {
						pageStart := m.page * m.maxContainersPerPage
						if pageStart < 0 {
							pageStart = 0
						}
						pageEnd := pageStart + m.maxContainersPerPage
						if pageEnd > len(m.flatList) {
							pageEnd = len(m.flatList)
						}
						found := -1
						for i := pageStart; i < pageEnd && i < len(m.flatList); i++ {
							if !m.flatList[i].isProject {
								found = i
								break
							}
						}
						if found != -1 {
							m.cursor = found
						} else if len(m.flatList) > 0 {
							for i := pageStart - 1; i >= 0; i-- {
								if !m.flatList[i].isProject {
									m.cursor = i
									break
								}
							}
						}
					} else {
						m.cursor = m.page * m.maxContainersPerPage
						if m.cursor >= len(m.containers) {
							m.cursor = max(0, len(m.containers)-1)
						}
					}
				}
			}
			m.updatePagination()

		case key.Matches(msg, Keys.PageDown):
			// Go to next page (right arrow)
			maxPage := 0
			if m.maxContainersPerPage > 0 {
				count := len(m.containers)
				if m.composeViewMode {
					count = len(m.flatList)
				}
				maxPage = (count - 1) / m.maxContainersPerPage
			}
			if maxPage < 0 {
				maxPage = 0
			}
			if m.page < maxPage {
				m.page++
				if m.maxContainersPerPage > 0 {
					if m.composeViewMode {
						pageStart := m.page * m.maxContainersPerPage
						if pageStart < 0 {
							pageStart = 0
						}
						pageEnd := pageStart + m.maxContainersPerPage
						if pageEnd > len(m.flatList) {
							pageEnd = len(m.flatList)
						}
						found := -1
						for i := pageStart; i < pageEnd && i < len(m.flatList); i++ {
							if !m.flatList[i].isProject {
								found = i
								break
							}
						}
						if found != -1 {
							m.cursor = found
						} else if len(m.flatList) > 0 {
							for i := pageStart; i < len(m.flatList); i++ {
								if !m.flatList[i].isProject {
									m.cursor = i
									break
								}
							}
						}
					} else {
						m.cursor = m.page * m.maxContainersPerPage
						if m.cursor >= len(m.containers) {
							m.cursor = max(0, len(m.containers)-1)
						}
					}
				}
			}
			m.updatePagination()

		case key.Matches(msg, Keys.Refresh):
			// Manually refresh container list
			m.loading = true
			m.logsVisible = false
			m.infoVisible = false
			m.infoContainer = nil
			m.updatePagination()
			return m, fetchContainers()

		case msg.String() == "c", msg.String() == "C":
			m.composeViewMode = !m.composeViewMode
			m.currentMode = modeComposeView
			if m.composeViewMode {
				m.statusMessage = "Switched to Compose view "
				m.expandedProjects = make(map[string]bool)
				m.expandedProjects["Standalone Containers"] = true
				m.cursor = 0
				m.page = 0

				// to save up performance and API calls
				return m, tea.Batch(fetchComposeProjects(), tickCmd(time.Duration(m.settings.RefreshInterval)*time.Second))
			}
			// Exiting compose view  - back to normal
			m.statusMessage = "Switched to Container View"
			m.cursor = 0
			m.page = 0
			m.updatePagination()
			return m, nil

		case key.Matches(msg, Keys.Start):
			// Start selected container
			if m.composeViewMode {
				// In compose view mode, get container from flatList
				if m.cursor < len(m.flatList) && !m.flatList[m.cursor].isProject {
					container := m.flatList[m.cursor].container
					m.statusMessage = "Starting container..."
					return m, doAction("start", container.ID)
				}
			} else {
				// Normal mode
				if len(m.containers) > 0 {
					m.statusMessage = "Starting container..."
					return m, doAction("start", m.containers[m.cursor].ID)
				}
			}

		case key.Matches(msg, Keys.Stop):
			// Stop selected container
			if m.composeViewMode {
				if m.cursor < len(m.flatList) && !m.flatList[m.cursor].isProject {
					container := m.flatList[m.cursor].container
					m.statusMessage = "Stopping container..."
					return m, doAction("stop", container.ID)
				}
			} else {
				// Normal mode
				if len(m.containers) > 0 {
					m.statusMessage = "Stopping container..."
					return m, doAction("stop", m.containers[m.cursor].ID)
				}
			}

		case key.Matches(msg, Keys.Info):
			// Toggle info panel for selected container
			var selected *docker.Container
			if m.logsVisible {
				return m, nil
			}
			if m.composeViewMode {
				if m.cursor < len(m.flatList) && !m.flatList[m.cursor].isProject {
					selected = m.flatList[m.cursor].container
				}
			} else {
				if len(m.containers) > 0 {
					selected = &m.containers[m.cursor]
				}
			}
			if selected != nil {
				// toggle visibility; when opening set infoContainer pointer, when closing clear it
				m.infoVisible = !m.infoVisible
				if m.infoVisible {
					m.infoContainer = selected
					m.currentMode = modeInfo
					m.statusMessage = "Showing container info"
				} else {
					m.infoContainer = nil
					m.currentMode = modeNormal
					m.statusMessage = "Info panel closed"
				}
				m.updatePagination()
			}

		case key.Matches(msg, Keys.Exec):
			// Open interactive shell in selected container (only if running)
			var container *docker.Container
			if m.composeViewMode {
				if m.cursor < len(m.flatList) && !m.flatList[m.cursor].isProject {
					container = m.flatList[m.cursor].container
				}
			} else {
				if len(m.containers) > 0 {
					container = &m.containers[m.cursor]
				}
			}
			if container != nil && container.State == "running" {
				containerID := container.ID
				m.statusMessage = "Opening interactive shell..."
				// Falls back to /bin/sh if configured shell is not available in container
				shell := m.settings.Shell
				shellCmd := fmt.Sprintf("if [ -x %s ]; then exec %s; else exec /bin/sh; fi", shell, shell)
				cmdStr := fmt.Sprintf("echo '# you are in interactive shell'; exec %s exec -it %s sh -c '%s'", string(m.settings.Runtime), containerID, shellCmd)
				c := exec.Command("bash", "-lc", cmdStr)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					if err != nil {
						return actionDoneMsg{err: fmt.Errorf("shell error: %v", err)}
					}
					return actionDoneMsg{err: nil}
				})
			}

		case key.Matches(msg, Keys.Restart):
			// Restart selected container
			if m.composeViewMode {

				if m.cursor < len(m.flatList) && !m.flatList[m.cursor].isProject {
					container := m.flatList[m.cursor].container
					m.statusMessage = "Restarting container..."
					return m, doAction("restart", container.ID)
				}
			} else {
				// Normal mode
				if len(m.containers) > 0 {
					m.statusMessage = "Restarting container..."
					return m, doAction("restart", m.containers[m.cursor].ID)
				}
			}

		case key.Matches(msg, Keys.Remove):
			// Remove selected container
			if m.composeViewMode {
				if m.cursor < len(m.flatList) && !m.flatList[m.cursor].isProject {
					container := m.flatList[m.cursor].container
					m.statusMessage = "Removing container..."
					return m, doAction("rm", container.ID)
				}
			} else {
				// Normal mode
				if len(m.containers) > 0 {
					m.statusMessage = "Removing container..."
					return m, doAction("rm", m.containers[m.cursor].ID)
				}
			}
		}
	}
	return m, nil
}

// ============================================================================
// View (render UI)
// ============================================================================

// render everything
func (m model) View() string {
	if m.terminalWidth == 0 {
		return "Initializing..."
	}

	if m.currentMode == modeSettings {
		return m.renderSettings(m.terminalWidth)
	}

	if m.currentMode == modeHelp {
		return m.renderHelp(m.terminalWidth)
	}

	var b strings.Builder

	// Ensure minimum width
	width := m.terminalWidth
	if width < 80 {
		width = 80
	}

	// title bar

	titleBar := m.renderTitleBar(width)
	b.WriteString(titleBar)
	b.WriteString("\n")

	running := 0
	stopped := 0
	for _, c := range m.containers {
		if strings.ToLower(c.State) == "running" {
			running++
		} else {
			stopped++
		}
	}
	total := len(m.containers)
	uptime := time.Since(m.startTime).Round(time.Second)

	statsSection := m.renderStatsSection(running, stopped, total, uptime, width)
	b.WriteString(statsSection)
	b.WriteString("\n")

	usableWidth := width - 2

	mins := []int{13, 17, 8, 6, 10, 11, 11, 13, 15}

	percents := m.settings.ColumnPercents
	if len(percents) != 9 {
		percents = []int{8, 14, 6, 6, 10, 12, 11, 13, 15}
	}

	// allocate widths by percent, respecting minimums
	widths := make([]int, len(mins))
	allocated := 0
	for i := range mins {
		desired := (usableWidth * percents[i]) / 100
		widths[i] = max(mins[i], desired)
		allocated += widths[i]
	}

	// if we have remaining space, distribute one char at a time across columns
	if allocated < usableWidth {
		remaining := usableWidth - allocated
		for remaining > 0 {
			for i := range widths {
				if remaining == 0 {
					break
				}
				widths[i]++
				remaining--
			}
		}
	}
	idW := widths[0]
	nameW := widths[1]
	memoryW := widths[2]
	cpuW := widths[3]
	netIOW := widths[4]
	blockIOW := widths[5]
	imageW := widths[6]
	statusW := widths[7]
	portsW := widths[8]

	sortIndicator := func(col sortColumn) string {
		if m.sortBy == col {
			if m.sortAsc {
				return " ▲"
			}
			return " ▼"
		}
		return ""
	}

	// highlight selected column in column mode
	highlightStyle := lipgloss.NewStyle().Background(lipgloss.Color("#58cdffff")).Foreground(lipgloss.Color("#000000")).Bold(true)

	// buildColumn builds a complete cell with spacing, padding, and title
	buildColumn := func(colIdx int, title string, width int, indicator string) string {
		text := title + indicator

		paddingNeeded := width - visibleLen(text)
		if paddingNeeded > 0 {
			text += strings.Repeat(" ", paddingNeeded)
		}
		// Add leading space and apply style
		cell := " " + text
		if m.columnMode && m.selectedColumn == colIdx {
			return highlightStyle.Render(cell)
		}
		return headerStyle.Render(cell)
	}

	// build all 9 columns
	col0 := buildColumn(0, "CONTAINER ID", idW-1, sortIndicator(sortByID))
	col1 := buildColumn(1, "NAME", nameW-1, sortIndicator(sortByName))
	col2 := buildColumn(2, "MEMORY", memoryW-2, sortIndicator(sortByMemory))
	col3 := buildColumn(3, "CPU", cpuW-2, sortIndicator(sortByCPU))
	col4 := buildColumn(4, "NET I/O", netIOW-1, sortIndicator(sortByNetIO))
	col5 := buildColumn(5, "DISK I/O", blockIOW-1, sortIndicator(sortByBlockIO))
	col6 := buildColumn(6, "IMAGE", imageW-1, sortIndicator(sortByImage))
	col7 := buildColumn(7, "STATUS", statusW, sortIndicator(sortByStatus))
	col8 := buildColumn(8, "PORTS", portsW, sortIndicator(sortByPorts))

	// combine into header - separators only
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(meterGreen)
	sep := sepStyle.Render("│")

	var hdrBuilder strings.Builder
	hdrBuilder.WriteString(col0)
	hdrBuilder.WriteString(sep)
	hdrBuilder.WriteString(col1)
	hdrBuilder.WriteString(sep)
	hdrBuilder.WriteString(col2)
	hdrBuilder.WriteString(sep)
	hdrBuilder.WriteString(col3)
	hdrBuilder.WriteString(sep)
	hdrBuilder.WriteString(col4)
	hdrBuilder.WriteString(sep)
	hdrBuilder.WriteString(col5)
	hdrBuilder.WriteString(sep)
	hdrBuilder.WriteString(col6)
	hdrBuilder.WriteString(sep)
	hdrBuilder.WriteString(col7)
	hdrBuilder.WriteString(sep)
	hdrBuilder.WriteString(col8)

	hdr := hdrBuilder.String()
	// pad header to fill width
	if visibleLen(hdr) < width {
		hdr += headerStyle.Render(strings.Repeat(" ", width-visibleLen(hdr)))
	}
	b.WriteString(hdr)
	b.WriteString("\n")
	// container list (paginated)

	rowsToShow := m.maxContainersPerPage
	if rowsToShow < 1 {
		rowsToShow = m.calculateMaxContainers()
	}
	if rowsToShow < 1 {
		rowsToShow = 1
	}

	// render rows
	rowsRendered := 0

	if m.composeViewMode {
		// Compose view mode -- render from flatList
		pageStart := m.page * rowsToShow
		if pageStart > len(m.flatList) {
			pageStart = 0
			if len(m.flatList) > rowsToShow {
				pageStart = len(m.flatList) - rowsToShow
			}
		}
		pageEnd := pageStart + rowsToShow
		if pageEnd > len(m.flatList) {
			pageEnd = len(m.flatList)
		}

		for i := pageStart; i < pageEnd; i++ {
			row := m.renderTreeRow(m.flatList[i], i == m.cursor, idW, nameW, memoryW, cpuW, netIOW, blockIOW, imageW, statusW, portsW, width)
			b.WriteString(row)
			b.WriteString("\n")
			rowsRendered++
		}
	} else {
		// Normal mode: render from containers
		pageStart := m.page * rowsToShow
		if pageStart > len(m.containers) {
			pageStart = 0
			if len(m.containers) > rowsToShow {
				pageStart = len(m.containers) - rowsToShow
			}
		}
		pageEnd := pageStart + rowsToShow
		if pageEnd > len(m.containers) {
			pageEnd = len(m.containers)
		}

		for i := pageStart; i < pageEnd; i++ {
			c := m.containers[i]
			row := m.renderContainerRow(c, i == m.cursor, idW, nameW, memoryW, cpuW, netIOW, blockIOW, imageW, statusW, portsW, width)
			b.WriteString(row)
			b.WriteString("\n")
			rowsRendered++
		}
	}

	// If no rows were rendered and app isnt loading, show the message.
	emptyNow := false
	if m.composeViewMode {
		emptyNow = !m.loading && len(m.flatList) == 0
	} else {
		emptyNow = !m.loading && len(m.containers) == 0
	}

	if emptyNow && rowsRendered == 0 {
		text := "No containers to display"
		pad := (width - visibleLen(text)) / 2
		if pad < 0 {
			pad = 0
		}
		line := strings.Repeat(" ", pad) + text
		line = padRight(line, width)
		b.WriteString(messageStyle.Render(line))
		b.WriteString("\n")
		rowsRendered++
	}

	// fill empty space
	emptyRow := normalStyle.Render(strings.Repeat(" ", width))
	for i := rowsRendered; i < rowsToShow; i++ {
		b.WriteString(emptyRow)
		b.WriteString("\n")
	}

	if m.logsVisible && !m.infoVisible {
		b.WriteString(m.renderLogsPanel(width))
	}
	if m.infoVisible && !m.logsVisible {
		b.WriteString(m.renderInfoPanel(width))
	}

	pageLine := m.message
	if pageLine == "" {
		pageLine = fmt.Sprintf("Page %d/%d", m.page+1, 1)
	}
	if len(pageLine) < width {
		pageLine += strings.Repeat(" ", width-len(pageLine))
	}
	b.WriteString(messageStyle.Render(pageLine))
	b.WriteString("\n")

	if m.statusMessage != "" {
		sm := m.statusMessage
		if len(sm) < width {
			sm += strings.Repeat(" ", width-len(sm))
		}
		b.WriteString(messageStyle.Render(sm))
		b.WriteString("\n")
	}

	b.WriteString(normalStyle.Render(strings.Repeat(" ", width)))
	b.WriteString("\n")

	// footer (keybinds)
	footer := m.renderFooter(width)
	b.WriteString(footer)

	return b.String()
}

// ============================================================================
// Rendering helpers
// ============================================================================

// render centered title bar
func (m model) renderTitleBar(width int) string {
	appName := appNameStyle.Render("┌─ DockMate🐳 ─┐")

	// center it
	padding := (width - visibleLen(appName)) / 2
	if padding < 0 {
		padding = 0
	}

	line := strings.Repeat(" ", padding) + appName
	if visibleLen(line) < width {
		line += strings.Repeat(" ", width-visibleLen(line))
	}

	return line
}

func (m model) renderStatsSection(running, stopped, total int, uptime time.Duration, width int) string {
	var b strings.Builder

	// calculate bar widths
	halfWidth := (width - 6) / 2
	barWidth := halfWidth - 16
	if barWidth < 10 {
		barWidth = 10
	}

	runPct := 0.0
	if total > 0 {
		runPct = float64(running) / float64(total)
	}
	runBar := renderBar(runPct, barWidth, meterGreen, textMuted)
	runningLine := fmt.Sprintf(" %s%s%s%s %s",
		meterLabelStyle.Render("Running "),
		meterBracketStyle.Render("["),
		runBar,
		meterBracketStyle.Render("]"),
		infoValueStyle.Render(fmt.Sprintf("%d/%d", running, total)))

	infoLine := fmt.Sprintf("%s %s  %s %s  %s %s %s %s",
		infoLabelStyle.Render("Total:"),
		infoValueStyle.Render(fmt.Sprintf("%d", total)),
		infoLabelStyle.Render("Session:"),
		infoValueStyle.Render(formatDuration(uptime)),
		infoLabelStyle.Render("Refresh:"),
		infoValueStyle.Render(fmt.Sprintf("%ds", m.settings.RefreshInterval)),
		infoLabelStyle.Render("Runtime:"),
		infoValueStyle.Render(string(m.settings.Runtime)))

	leftLen := visibleLen(runningLine)
	rightLen := visibleLen(infoLine)
	middlePad := width - leftLen - rightLen - 2
	if middlePad < 2 {
		middlePad = 2
	}

	b.WriteString(runningLine)
	b.WriteString(strings.Repeat(" ", middlePad))
	b.WriteString(infoLine)
	b.WriteString("\n")

	// line 2: stopped bar + loading indicator
	stopPct := 0.0
	if total > 0 {
		stopPct = float64(stopped) / float64(total)
	}
	stopBar := renderBar(stopPct, barWidth, meterRed, textMuted)
	stoppedLine := fmt.Sprintf(" %s%s%s%s %s",
		meterLabelStyle.Render("Stopped "),
		meterBracketStyle.Render("["),
		stopBar,
		meterBracketStyle.Render("]"),
		infoValueStyle.Render(fmt.Sprintf("%d/%d", stopped, total)))

	b.WriteString(stoppedLine)

	// loading spinner if fetching
	if m.loading {
		loadingPad := width - visibleLen(stoppedLine) - 12
		if loadingPad > 0 {
			b.WriteString(strings.Repeat(" ", loadingPad))
			b.WriteString(messageStyle.Render("⟳ Loading..."))
		}
	}

	return b.String()
}

func renderBar(pct float64, width int, fgColor, bgColor lipgloss.Color) string {
	// clamp percentage
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	// Calculate filled and empty counts
	filled := int(pct * float64(width))
	empty := width - filled

	filledStyle := lipgloss.NewStyle().Foreground(fgColor).Bold(true)
	emptyStyle := lipgloss.NewStyle().Foreground(bgColor)

	bar := ""
	if filled > 0 {
		bar += filledStyle.Render(strings.Repeat("█", filled))
	}
	if empty > 0 {
		bar += emptyStyle.Render(strings.Repeat("░", empty))
	}

	return bar
}

func visibleLen(s string) int {
	count := 0
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
		} else if inEscape && r == 'm' {
			inEscape = false
		} else if !inEscape {
			count++
		}
	}
	return count
}

func truncateToWidth(s string, width int) string {
	if width < 1 {
		return ""
	}

	visLen := visibleLen(s)
	if visLen <= width {
		return s
	}

	targetWidth := width - 1
	if targetWidth < 1 {
		return "…"
	}

	visCount := 0
	inEscape := false
	result := ""

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result += string(r)
		} else if inEscape {
			result += string(r)
			if r == 'm' {
				inEscape = false
			}
		} else {
			if visCount >= targetWidth {
				break
			}
			result += string(r)
			visCount++
		}
	}

	return result + "…"
}

// render one container row
// applies styles based on selection and state
func (m model) renderContainerRow(c docker.Container, selected bool, idW, nameW, memoryW, cpuW, netIOW, blockIOW, imageW, statusW, portsW, totalWidth int) string {
	// get name from names array
	name := ""
	if len(c.Names) > 0 {
		name = c.Names[0]
	}

	// truncate fields to fit
	id := c.ID
	if visibleLen(id) > idW-2 {
		id = truncateToWidth(id, idW-2)
	}

	if visibleLen(name) > nameW-2 {
		name = truncateToWidth(name, nameW-2)
	}
	img := c.Image
	if visibleLen(img) > imageW-2 {
		img = truncateToWidth(img, imageW-2)
	}
	status := c.Status
	if visibleLen(status) > statusW-2 {
		status = truncateToWidth(status, statusW-2)
	}
	// STATE column hidden for testing; keep state value for styling only
	// state := c.State

	// net IO
	netio := c.NetIO
	if netio == "" {
		netio = "─"
	}
	if visibleLen(netio) > netIOW-2 {
		// truncate to fit with ellipsis
		netio = truncateToWidth(netio, netIOW-2)
	}

	// block IO
	blockio := c.BlockIO
	if blockio == "" {
		blockio = "─"
	}
	if visibleLen(blockio) > blockIOW-2 {
		blockio = truncateToWidth(blockio, blockIOW-2)
	}

	// placeholder for empty stuff
	mem := c.Memory
	if mem == "" {
		mem = "─"
	}
	if visibleLen(mem) > memoryW-2 {
		mem = truncateToWidth(mem, memoryW-2)
	}

	cpu := c.CPU
	if visibleLen(cpu) > cpuW-2 {
		// truncate to fit with ellipsis
		cpu = truncateToWidth(cpu, cpuW-2)
	}
	if cpu == "" {
		cpu = "─"
	}
	ports := c.Ports
	if ports == "" {
		ports = "─"
	}
	if visibleLen(ports) > portsW-7 {
		ports = truncateToWidth(ports, portsW-6)
	}

	// Format row (STATE column omitted)
	row := fmt.Sprintf(" %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s",
		idW-1, id,
		nameW-1, name,
		memoryW-2, mem,
		cpuW-2, cpu,
		netIOW-1, netio,
		blockIOW-1, blockio,
		imageW-1, img,
		statusW, status,
		portsW-2, ports)

	// Pad row to totalWidth BEFORE styling to ensure color extends to edge
	if visibleLen(row) < totalWidth {
		row += strings.Repeat(" ", totalWidth-visibleLen(row))
	}

	// Apply style based on selection and state
	if selected {
		return selectedStyle.Render(row)
	}

	switch strings.ToLower(c.State) {
	case "running":
		return runningStyle.Render(row)
	case "paused":
		return pausedStyle.Render(row)
	case "exited", "dead":
		return stoppedStyle.Render(row)
	default:
		return normalStyle.Render(row)
	}
}

func padRight(s string, width int) string {
	if visibleLen(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visibleLen(s))
}

func (m model) renderFooter(width int) string {
	var keys []struct {
		key  string
		desc string
	}

	// Show different shortcuts based on current mode
	switch m.currentMode {
	case modeColumnSelect:
		keys = []struct {
			key  string
			desc string
		}{
			{"←→", "Select Col"},
			{"Enter", "Sort"},
			{"Esc", "Back"},
		}
	case modeLogs:
		keys = []struct {
			key  string
			desc string
		}{
			{"l", "Close Logs"},
			{"↑↓", "Scroll"},
			{"E", "Interactive Shell"},
			{"Esc", "Back"},
		}
	case modeInfo:
		keys = []struct {
			key  string
			desc string
		}{
			{"i", "Close info"},
			{"↑↓", "Scroll"},
			{"E", "Interactive Shell"},
			{"Esc", "Back"},
		}
	case modeHelp:
		keys = []struct {
			key  string
			desc string
		}{
			{"?", "Close Help"},
			{"Esc", "Back"},
		}
	default: // modeNormal
		keys = []struct {
			key  string
			desc string
		}{
			{"↑↓", "Nav"},
			{"←→", "Nav pages"},
			{"Tab", "Col Mode"},
			{"c", "Compose View"},
			{"?", "Keyboard shortcuts"},
			{"f2", "Settings"},
			{"q", "Quit"},
		}
		if m.composeViewMode {
			keys = []struct {
				key  string
				desc string
			}{
				{"↑↓", "Nav"},
				{"←→", "Nav pages"},
				{"Tab", "Col Mode"},

				{"c", "Normal View"},
				{"?", "Keyboard shortcuts"},
				{"f2", "Settings"},
				{"q", "Quit"},
			}
		}
	}

	var footer strings.Builder
	footer.WriteString(" ")

	// build key action format
	for i, k := range keys {
		footer.WriteString(meterBracketStyle.Render("["))
		footer.WriteString(footerKeyStyle.Render(k.key))
		footer.WriteString(meterBracketStyle.Render("]"))
		footer.WriteString(footerArrowStyle.Render("→"))
		footer.WriteString(footerDescStyle.Render(k.desc))
		if i < len(keys)-1 {
			footer.WriteString("  ")
		}
	}

	// pad footer
	footerStr := footer.String()
	footerVisible := visibleLen(footerStr)
	if footerVisible < width {
		footerStr += strings.Repeat(" ", width-footerVisible)
	}

	return footerStr
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// format duration like HH:MM:SS
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
