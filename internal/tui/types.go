package tui

import (
	"time"

	"github.com/shubh-io/dockmate/internal/docker"
)

type model struct {
	containers           []docker.Container                // all containers (running + stopped)
	projects             map[string]*docker.ComposeProject // compose projects
	expandedProjects     map[string]bool                   // track which projects are expanded
	flatList             []treeRow                         // flattened tree for rendering
	cursor               int                               // selected container index
	page                 int                               // current page
	maxContainersPerPage int                               // containers per page (dynamic)
	terminalWidth        int                               // terminal width
	terminalHeight       int                               // terminal height
	err                  error                             // last error
	loading              bool                              // fetching data?
	message              string                            // page indicator (persistent)
	statusMessage        string                            // transient status message
	startTime            time.Time                         // when app started
	logsVisible          bool                              // logs panel visible?
	logPanelHeight       int                               // height of logs panel
	logsLines            []string                          // log lines
	logsContainer        string                            // container id for logs
	infoVisible          bool                              // info panel visible?
	infoPanelHeight      int                               // height of info panel
	infoContainer        *docker.Container                 // container for info display
	sortBy               sortColumn                        // which column to sort by
	sortAsc              bool                              // sort direction
	columnMode           bool                              // column nav mode (vs row nav)
	selectedColumn       int                               // selected column (0-8)
	currentMode          appMode                           // current UI mode

	// settings
	settings         Settings
	composeViewMode  bool
	suspendRefresh   bool
	settingsSelected int
}

// treeRow represents a row in the flattened tree
type treeRow struct {
	isProject   bool
	projectName string
	container   *docker.Container
	indent      int
	running     int
	total       int
}

// runtime
type ContainerRuntime string

const (
	RuntimeDocker ContainerRuntime = "docker"
	RuntimePodman ContainerRuntime = "podman"
)

// available shell options for container exec
var ShellOptions = []string{"/bin/sh", "/bin/bash", "/bin/zsh", "/bin/ash"}

// app settings
type Settings struct {
	ColumnPercents  []int
	RefreshInterval int
	Runtime         ContainerRuntime
	Shell           string
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
	sortByPorts
)

// which mode the TUI is in
type appMode int

const (
	modeNormal appMode = iota
	modeColumnSelect
	modeLogs
	modeInfo
	modeSettings
	modeComposeView
	modeHelp
)

type actionDoneMsg struct {
	err error // nil if ok
}
type tickMsg time.Time

type composeProjectsMsg struct {
	Projects map[string]*docker.ComposeProject
	Err      error
}
