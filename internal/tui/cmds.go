package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shubh-io/dockmate/internal/docker"
)

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

// fetch compose projects asynchronously
func fetchComposeProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := docker.FetchComposeProjects()
		return composeProjectsMsg{Projects: projects, Err: err}
	}
}

// fire every 2 seconds for auto-refresh
func tickCmd(d time.Duration) tea.Cmd {
	if d < time.Second {
		d = 1 * time.Second
	}
	return tea.Tick(d, func(t time.Time) tea.Msg {
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
