package tui

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shubh-io/dockmate/internal/docker"
)

// ============================================================================

var (
	// color palette
	// main accents
	accent = lipgloss.Color("#22D3EE")

	// text colors
	textPrimary   = lipgloss.Color("#F8FAFC") // near white
	textSecondary = lipgloss.Color("#94A3B8")
	textMuted     = lipgloss.Color("#475569")

	// backgrounds
	borderColor = lipgloss.Color("#334155")

	// status colors
	yellowColor = lipgloss.Color("#F59E0B") // warnings/actions
	cyanColor   = lipgloss.Color("#06B6D4") // selected

	// others
	meterGreen = lipgloss.Color("#4ADE80") // bright green for bars
	meterRed   = lipgloss.Color("#F87171") // bright red for bars

	// title style
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Padding(0, 1)

	// app name
	appNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textPrimary)

	// meter styles
	meterLabelStyle = lipgloss.NewStyle().
			Foreground(cyanColor).
			Bold(true)

	meterBracketStyle = lipgloss.NewStyle().
				Foreground(textMuted)

	infoLabelStyle = lipgloss.NewStyle().
			Foreground(textSecondary)

	infoValueStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Bold(true)

	// table header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(meterGreen)

	// selected row
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(cyanColor)

	// container states
	runningStyle = lipgloss.NewStyle().
			Foreground(meterGreen).
			Bold(true)

	stoppedStyle = lipgloss.NewStyle().
			Foreground(meterRed)

	pausedStyle = lipgloss.NewStyle().
			Foreground(yellowColor)

	normalStyle = lipgloss.NewStyle().
			Foreground(textSecondary)

	// footer
	footerKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(cyanColor).
			Bold(true).
			Padding(0, 0)

	footerDescStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Padding(0, 0)

	footerArrowStyle = lipgloss.NewStyle().
				Foreground(yellowColor).
				Bold(true)

	// message
	messageStyle = lipgloss.NewStyle().
			Foreground(yellowColor).
			Bold(true)

	// divider
	dividerStyle = lipgloss.NewStyle().
			Foreground(borderColor)
)

// debug logger writes snapshots to a file (dockmate-debug.log) by default
var (
	debugLogger *log.Logger
	debugFile   *os.File
)

// init sets up file-backed debug logging. If the file can't be opened,
// debugLogger falls back to discarding output.
func init() {
	// default debug file in working directory
	_ = SetDebugFile("dockmate-debug.log")
}

// SetDebugFile opens (or creates) the given path and directs debug output there.
// It returns an error if the file cannot be opened.
func SetDebugFile(path string) error {
	if debugFile != nil {
		_ = debugFile.Close()
		debugFile = nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		// fallback to discard
		debugLogger = log.New(io.Discard, "DEBUG: ", log.LstdFlags)
		return err
	}
	debugFile = f
	debugLogger = log.New(debugFile, "DEBUG: ", log.LstdFlags)
	return nil
}

// CloseDebug closes the current debug file (if any) and disables logging.
func CloseDebug() error {
	if debugFile == nil {
		return nil
	}
	err := debugFile.Close()
	debugFile = nil
	debugLogger = log.New(io.Discard, "DEBUG: ", log.LstdFlags)
	return err
}

// layout sizing constants
const (
	HEADER_HEIGHT        = 8
	CONTAINER_ROW_HEIGHT = 1
	LOG_PANEL_HEIGHT     = 15
)

// ============================================================================
// App state (bubble tea model)
// ============================================================================

// model holds everything for the TUI
type model struct {
	containers           []docker.Container // all containers (running + stopped)
	cursor               int                // selected container index
	page                 int                // current page
	maxContainersPerPage int                // containers per page (dynamic)
	terminalWidth        int                // terminal width
	terminalHeight       int                // terminal height
	err                  error              // last error
	loading              bool               // fetching data?
	message              string             // page indicator (persistent)
	statusMessage        string             // transient status message
	startTime            time.Time          // when app started
	logsVisible          bool               // logs panel visible?
	logPanelHeight       int                // height of logs panel
	logsLines            []string           // log lines
	logsContainer        string             // container id for logs
	sortBy               sortColumn         // which column to sort by
	sortAsc              bool               // sort direction
	columnMode           bool               // column nav mode (vs row nav)
	selectedColumn       int                // selected column (0-8)
	currentMode          appMode            // current UI mode
}

// which column to sort by
type sortColumn int

const (
	sortByID sortColumn = iota
	sortByName
	sortByMemory
	sortByCPU
	sortByNetIO
	sortByBlockIO
	sortByImage
	sortByStatus
	sortByState
	sortByPIDs
)

// which mode the TUI is in
type appMode int

const (
	modeNormal appMode = iota
	modeColumnSelect
	modeLogs
)

// ============================================================================
// Initialization
// ============================================================================

// set up initial state
func InitialModel() model {
	return model{
		loading:              true,       // start loading
		startTime:            time.Now(), // track uptime
		page:                 0,          // first page
		maxContainersPerPage: 12,         // initial guess until resize event
		terminalWidth:        0,
		terminalHeight:       0,
		logsVisible:          false, // logs hidden by default
		logPanelHeight:       LOG_PANEL_HEIGHT,
		sortBy:               sortByState, // sort by state
		sortAsc:              false,       // descending
		columnMode:           false,       // row nav mode
		selectedColumn:       7,           // state column (adjusted for new NET I/O column)
		currentMode:          modeNormal,  // start in normal mode
	}
}

// called once at startup
// kicks off container fetch and timer
func (m model) Init() tea.Cmd {
	return tea.Batch(fetchContainers(), tickCmd())
}

// ============================================================================
// Message types
// ============================================================================

// sent when docker action finishes
type actionDoneMsg struct {
	err error // nil if ok
}

// sent every 2 seconds for refresh
type tickMsg time.Time

// ============================================================================
// Async commands
// ============================================================================

// grab container list in background
func fetchContainers() tea.Cmd {
	return func() tea.Msg {
		containers, err := docker.ListContainers()
		return docker.ContainersMsg{Containers: containers, Err: err}
	}
}

// fire every 2 seconds for auto-refresh
func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// run docker action in background (start/stop/etc)
func doAction(action, containerID string) tea.Cmd {
	return func() tea.Msg {
		err := docker.DoAction(action, containerID)
		return actionDoneMsg{err: err}
	}
}

// fetch logs for a container
func fetchLogsCmd(id string) tea.Cmd {
	return func() tea.Msg {
		lines, err := docker.GetLogs(id)
		return docker.LogsMsg{ID: id, Lines: lines, Err: err}
	}
}

// ============================================================================
// Sorting
// ============================================================================

// sort containers by current column and direction
func (m *model) sortContainers() {
	sort.Slice(m.containers, func(i, j int) bool {
		less := false

		switch m.sortBy {
		case sortByID:
			// simple string comparison
			less = m.containers[i].ID < m.containers[j].ID

		case sortByName:
			// case-insensitive name sort
			nameI := ""
			if len(m.containers[i].Names) > 0 {
				nameI = m.containers[i].Names[0]
			}
			nameJ := ""
			if len(m.containers[j].Names) > 0 {
				nameJ = m.containers[j].Names[0]
			}
			less = strings.ToLower(nameI) < strings.ToLower(nameJ)

		case sortByMemory:
			// parse percentages and compare
			memI := parsePercent(m.containers[i].Memory)
			memJ := parsePercent(m.containers[j].Memory)
			less = memI < memJ

		case sortByCPU:
			// same for cpu
			cpuI := parsePercent(m.containers[i].CPU)
			cpuJ := parsePercent(m.containers[j].CPU)
			less = cpuI < cpuJ

		case sortByImage:
			// case insensitive
			less = strings.ToLower(m.containers[i].Image) < strings.ToLower(m.containers[j].Image)

		case sortByStatus:
			// case-insensitive
			less = strings.ToLower(m.containers[i].Status) < strings.ToLower(m.containers[j].Status)

		case sortByState:
			// running/exited/etc
			less = strings.ToLower(m.containers[i].State) < strings.ToLower(m.containers[j].State)

		case sortByPIDs:
			// parse pid count as number
			pidsI, _ := strconv.Atoi(m.containers[i].PIDs)
			pidsJ, _ := strconv.Atoi(m.containers[j].PIDs)
			less = pidsI < pidsJ
		case sortByNetIO:
			// compare total network I/O (rx+tx) as bytes
			netI := parseNetIO(m.containers[i].NetIO)
			netJ := parseNetIO(m.containers[j].NetIO)
			less = netI < netJ

		case sortByBlockIO:
			// compare total block I/O (read+write) as bytes
			blockI := parseNetIO(m.containers[i].BlockIO)
			blockJ := parseNetIO(m.containers[j].BlockIO)
			less = blockI < blockJ

		}

		// flip for descending
		if m.sortAsc {
			return less
		}
		return !less
	})
}

// convert "0.48%" to 0.48
func parsePercent(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

// parseNetIO parses a string like "1.2kB / 3.4kB" and returns total bytes
func parseNetIO(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "─" {
		return 0
	}
	parts := strings.Split(s, "/")
	if len(parts) == 0 {
		return 0
	}
	total := 0.0
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v := parseSize(p)
		total += v
	}
	return total
}

// parseSize parses a human-readable size like "1.2kB" or "3MiB" into bytes.
func parseSize(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// remove possible commas
	s = strings.ReplaceAll(s, ",", "")
	// split number and unit
	num := ""
	unit := ""
	for i, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == '-' {
			num += string(r)
		} else {
			unit = strings.TrimSpace(s[i:])
			break
		}
	}
	if num == "" {
		return 0
	}
	val, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0
	}
	unit = strings.ToLower(strings.TrimSpace(unit))
	switch unit {
	case "b", "bytes", "byte", "":
		return val
	case "kb", "kib":
		return val * 1000
	case "mb", "mib":
		return val * 1000 * 1000
	case "gb", "gib":
		return val * 1000 * 1000 * 1000
	default:
		// fallback: if unit ends with b (e.g., kB) treat as *1000
		if strings.HasSuffix(unit, "b") {
			prefix := strings.TrimSuffix(unit, "b")
			if prefix == "k" {
				return val * 1000
			}
			if prefix == "m" {
				return val * 1000 * 1000
			}
			if prefix == "g" {
				return val * 1000 * 1000 * 1000
			}
		}
	}
	return val
}

// calculateMaxContainers determines how many containers fit on screen given current layout state
func (m *model) calculateMaxContainers() int {
	availableHeight := m.terminalHeight - HEADER_HEIGHT
	if m.logsVisible {
		availableHeight -= m.logPanelHeight
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
		}

		// keep cursor in bounds
		if m.cursor >= len(m.containers) {
			m.cursor = max(0, len(m.containers)-1)
		}

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
		// refresh list
		return m, fetchContainers()

	case tickMsg:
		// time to refresh
		// always refresh container list, if logs panel is open, also refresh logs
		if m.logsVisible && m.logsContainer != "" {
			return m, tea.Batch(fetchContainers(), tickCmd(), fetchLogsCmd(m.logsContainer))
		}
		return m, tea.Batch(fetchContainers(), tickCmd())

	case tea.KeyMsg:
		// keyboard input
		m.statusMessage = ""

		// Handle Escape key to return to normal mode
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
		}

		// special keys that work in both modes
		switch msg.String() {
		// for debugging: press backtick (`) to dump a state snapshot to the debug logger
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

		case "L":
			// Toggle logs panel visibility without fetching new logs
			m.logsVisible = !m.logsVisible
			if m.logsVisible {
				m.currentMode = modeLogs
			} else {
				m.currentMode = modeNormal
			}
			m.updatePagination()
			return m, nil

		case "enter":
			// in column mode, sort by selected column
			if m.columnMode {
				// map column index to sort enum
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
					col = sortByState
				case 9:
					col = sortByPIDs
				}

				if canSort {
					// toggle direction if same column, else reset
					if m.sortBy == col {
						m.sortAsc = !m.sortAsc
					} else {
						m.sortBy = col
						m.sortAsc = true
					}
					m.sortContainers()

					// show feedback
					dir := "asc"
					if !m.sortAsc {
						dir = "desc"
					}
					colNames := []string{"ID", "Name", "Memory", "CPU", "NET I/O", "Disk I/O", "Image", "Status", "State", "PIDs"}
					m.statusMessage = fmt.Sprintf("Sorted by %s (%s)", colNames[m.selectedColumn], dir)
				}
			}
			return m, nil

		case "left", "h":
			// In column mode, move selection left
			if m.columnMode {
				if m.selectedColumn > 0 {
					m.selectedColumn--
				}
				return m, nil
			}

		case "right", "l":
			// In column mode, move selection right
			if m.columnMode {
				if m.selectedColumn < 9 { // 0-9 for 10 columns
					m.selectedColumn++
				}
				return m, nil
			}
		}

		// Handle key bindings
		switch {
		case key.Matches(msg, Keys.Quit):
			// Exit application
			return m, tea.Quit

		case key.Matches(msg, Keys.Up):
			// Move cursor up (only in row mode)
			if !m.columnMode && m.cursor > 0 {
				m.cursor--
				// Switch to previous page if needed
				if m.maxContainersPerPage > 0 && m.cursor < m.page*m.maxContainersPerPage {
					m.page--
				}
			}

		case key.Matches(msg, Keys.Down):
			// Move cursor down (only in row mode)
			if !m.columnMode && m.cursor < len(m.containers)-1 {
				m.cursor++
				// Switch to next page if needed
				if m.maxContainersPerPage > 0 && m.cursor >= (m.page+1)*m.maxContainersPerPage {
					m.page++
				}
			}

		case key.Matches(msg, Keys.PageUp):
			// Go to previous page (left arrow)
			if m.page > 0 {
				m.page--
				if m.maxContainersPerPage > 0 {
					m.cursor = m.page * m.maxContainersPerPage
				}
			}
			m.updatePagination()
			// updatePagination will update the persistent page indicator; do not set a transient status here

		case key.Matches(msg, Keys.PageDown):
			// Go to next page (right arrow)
			maxPage := 0
			if m.maxContainersPerPage > 0 {
				maxPage = (len(m.containers) - 1) / m.maxContainersPerPage
			}
			if maxPage < 0 {
				maxPage = 0
			}
			if m.page < maxPage {
				m.page++
				m.cursor = m.page * m.maxContainersPerPage
			}
			m.updatePagination()
			// updatePagination will update the persistent page indicator; do not set a transient status here

		case key.Matches(msg, Keys.Refresh):
			// Manually refresh container list
			m.loading = true
			m.logsVisible = false
			m.updatePagination()
			return m, fetchContainers()

		case key.Matches(msg, Keys.Start):
			// Start selected container
			if len(m.containers) > 0 {
				m.statusMessage = "Starting container..."
				return m, doAction("start", m.containers[m.cursor].ID)
			}

		case key.Matches(msg, Keys.Stop):
			// Stop selected container
			if len(m.containers) > 0 {
				m.statusMessage = "Stopping container..."
				return m, doAction("stop", m.containers[m.cursor].ID)
			}

		case key.Matches(msg, Keys.Logs):
			// Fetch and display logs for selected container
			if len(m.containers) == 0 {
				return m, nil
			}
			m.statusMessage = "Fetching logs..."
			m.currentMode = modeLogs
			// recompute pagination and persistent page indicator
			m.updatePagination()
			return m, fetchLogsCmd(m.containers[m.cursor].ID)

		case key.Matches(msg, Keys.Exec):
			// Open interactive shell in selected container (only if running)
			if len(m.containers) > 0 && m.containers[m.cursor].State == "running" {
				containerID := m.containers[m.cursor].ID
				m.statusMessage = "Opening interactive shell..."
				// Use bash to clear terminal and exec into container shell
				cmdStr := fmt.Sprintf("echo '# you are in interactive shell'; exec docker exec -it %s /bin/sh", containerID)
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
			if len(m.containers) > 0 {
				m.statusMessage = "Restarting container..."
				return m, doAction("restart", m.containers[m.cursor].ID)
			}

		case key.Matches(msg, Keys.Remove):
			// Remove selected container
			if len(m.containers) > 0 {
				m.statusMessage = "Removing container..."
				return m, doAction("rm", m.containers[m.cursor].ID)
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

	// stats section (running/stopped bars)

	// count by state
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

	// table header

	// column widths - compute with smart allocation to prevent overflow
	usableWidth := width - 2 // account for padding and separators (10 columns = ~10 separators)

	// define minimum widths and relative weights for each column
	type colSpec struct {
		min    int
		weight int
	}

	specs := []colSpec{
		{12, 12}, // ID
		{15, 15}, // NAME
		{8, 8},   // MEMORY
		{6, 8},   // CPU
		{10, 13}, // NET I/O
		{12, 13}, // Disk I/O
		{15, 18}, // IMAGE
		{12, 12}, // STATUS
		{7, 13},  // STATE
		{5, 10},  // PIDs
	}

	// compute total weight
	totalWeight := 0
	for _, spec := range specs {
		totalWeight += spec.weight
	}

	// allocate widths proportionally, respecting minimums
	widths := make([]int, len(specs))
	allocated := 0
	for i, spec := range specs {
		// proportional allocation
		desired := (usableWidth * spec.weight) / totalWeight
		widths[i] = max(spec.min, desired)
		allocated += widths[i]
	}

	// if we exceeded usableWidth, shrink columns proportionally (keeping minimums)
	if allocated < usableWidth {
		remaining := usableWidth - allocated
		// spread remaining 1 char at a time across columns with the biggest weight

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
	// debugLogger.Printf(
	// 	"width=%d usableWidth=%d allocated=%d widths=%v specs=%+v",
	// 	width, usableWidth, allocated, widths, specs,
	// )

	// assign to individual variables
	idW := widths[0]
	nameW := widths[1]
	memoryW := widths[2]
	cpuW := widths[3]
	netIOW := widths[4]
	blockIOW := widths[5]
	imageW := widths[6]
	statusW := widths[7]
	stateW := widths[8]
	pidsW := widths[9]

	// debugLogger.Printf(
	// 	"Column widths: ID=%d NAME=%d MEMORY=%d CPU=%d NET I/O=%d Disk I/O=%d IMAGE=%d STATUS=%d STATE=%d PIDs=%d",
	// 	idW, nameW, memoryW, cpuW, netIOW, blockIOW, imageW, statusW, stateW, pidsW,
	// )
	// sort indicator (▲/▼)
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
		// Pad to width (width includes the space before the column)
		paddingNeeded := width - len(text)
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

	// build all 10 columns
	col0 := buildColumn(0, "CONTAINER ID", idW-1, sortIndicator(sortByID))
	col1 := buildColumn(1, "NAME", nameW-1, sortIndicator(sortByName))
	col2 := buildColumn(2, "MEMORY", memoryW-1, sortIndicator(sortByMemory))
	col3 := buildColumn(3, "CPU", cpuW-1, sortIndicator(sortByCPU))
	col4 := buildColumn(4, "NET I/O", netIOW-1, sortIndicator(sortByNetIO))
	col5 := buildColumn(5, "Disk I/O", blockIOW-1, sortIndicator(sortByBlockIO))
	col6 := buildColumn(6, "IMAGE", imageW-1, sortIndicator(sortByImage))
	col7 := buildColumn(7, "STATUS", statusW-1, sortIndicator(sortByStatus))
	col8 := buildColumn(8, "STATE", stateW-1, sortIndicator(sortByState))
	col9 := buildColumn(9, "PIDs", pidsW, sortIndicator(sortByPIDs)) // last column gets full width

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
	hdrBuilder.WriteString(sep)
	hdrBuilder.WriteString(col9)

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

	startIdx := pageStart
	endIdx := pageEnd

	// render rows
	rowsRendered := 0
	for i := startIdx; i < endIdx; i++ {
		c := m.containers[i]
		row := m.renderContainerRow(c, i == m.cursor, idW, nameW, memoryW, cpuW, netIOW, blockIOW, imageW, statusW, stateW, pidsW, width)
		b.WriteString(row)
		b.WriteString("\n")
		rowsRendered++
	}

	// fill empty space
	emptyRow := normalStyle.Render(strings.Repeat(" ", width))
	for i := rowsRendered; i < rowsToShow; i++ {
		b.WriteString(emptyRow)
		b.WriteString("\n")
	}

	// logs panel (if visible)

	if m.logsVisible {
		b.WriteString(m.renderLogsPanel(width))
	}

	// page indicator (persistent) - always render
	pageLine := m.message
	if pageLine == "" {
		pageLine = fmt.Sprintf("Page %d/%d", m.page+1, 1)
	}
	if len(pageLine) < width {
		pageLine += strings.Repeat(" ", width-len(pageLine))
	}
	b.WriteString(messageStyle.Render(pageLine))
	b.WriteString("\n")

	// transient status message (if any)
	if m.statusMessage != "" {
		sm := m.statusMessage
		if len(sm) < width {
			sm += strings.Repeat(" ", width-len(sm))
		}
		b.WriteString(messageStyle.Render(sm))
		b.WriteString("\n")
	}

	// 1-row bottom padding after messages
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

// render running/stopped bars with session info
// two lines: running bar + info, stopped bar + loading
func (m model) renderStatsSection(running, stopped, total int, uptime time.Duration, width int) string {
	var b strings.Builder

	// calculate bar widths
	halfWidth := (width - 6) / 2
	barWidth := halfWidth - 16
	if barWidth < 10 {
		barWidth = 10
	}

	// line 1: running bar + session info on right
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

	// right side: total, uptime, refresh interval
	infoLine := fmt.Sprintf("%s %s  %s %s  %s %s",
		infoLabelStyle.Render("Total:"),
		infoValueStyle.Render(fmt.Sprintf("%d", total)),
		infoLabelStyle.Render("Session:"),
		infoValueStyle.Render(formatDuration(uptime)),
		infoLabelStyle.Render("Refresh:"),
		infoValueStyle.Render("2s"))

	// padding between left and right
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

// draw a progress bar with filled/empty chars
// pct is 0.0 to 1.0
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

// get visible length without ansi codes
// important for width calculations with lipgloss
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

// truncateToWidth truncates a string to fit within the given visible width
// preserving ANSI codes and adding ellipsis if truncated
func truncateToWidth(s string, width int) string {
	if width < 1 {
		return ""
	}

	visLen := visibleLen(s)
	if visLen <= width {
		return s
	}

	// need to truncate - account for ellipsis
	targetWidth := width - 1
	if targetWidth < 1 {
		return "…"
	}

	// walk through string counting visible chars
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
func (m model) renderContainerRow(c docker.Container, selected bool, idW, nameW, memoryW, cpuW, netIOW, blockIOW, imageW, statusW, stateW, pidsW, totalWidth int) string {
	// get name from names array
	name := ""
	if len(c.Names) > 0 {
		name = c.Names[0]
	}

	// truncate fields to fit
	id := c.ID
	if len(id) > idW-2 {
		id = id[:idW-2]
	}

	if len(name) > nameW-2 {
		name = name[:nameW-3] + "…"
	}
	img := c.Image
	if len(img) > imageW-2 {
		img = img[:imageW-3] + "…"
	}
	status := c.Status
	if len(status) > statusW-2 {
		status = status[:statusW-3] + "…"
	}
	state := c.State
	if len(state) > stateW-2 {
		state = state[:stateW-2] + "…"
	}

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
	if visibleLen(cpu) > cpuW-1 {
		// truncate to fit with ellipsis
		cpu = truncateToWidth(cpu, cpuW-1)
	}
	if cpu == "" {
		cpu = "─"
	}
	pids := c.PIDs
	if pids == "" {
		pids = "─"
	}
	if visibleLen(pids) > pidsW-1 {
		pids = truncateToWidth(pids, pidsW-1)
	}

	// Format row (include NET I/O between CPU and IMAGE)
	row := fmt.Sprintf(" %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %-*s",
		idW-1, id,
		nameW-1, name,
		memoryW-1, mem,
		cpuW-1, cpu,
		netIOW-1, netio,
		blockIOW-1, blockio,
		imageW-1, img,
		statusW-1, status,
		stateW-3, state,
		pidsW, pids)

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

// renderLogsPanel prints a fixed-height logs section respecting the configured panel height
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

// render keyboard shortcuts at bottom (mode-aware)
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
			{"Shift + l", "Close Logs"},
			{"↑↓", "Scroll"},
			{"E", "Interactive Shell"},
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
			{"s", "Start"},
			{"x", "Stop"},
			{"r", "Restart"},
			{"l", "Logs"},
			{"e", "Shell"},
			{"d", "Remove"},
			{"q", "Quit"},
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

func min(a, b int) int {
	if a < b {
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
