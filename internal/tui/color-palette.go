package tui

import (
	"github.com/charmbracelet/lipgloss"
)

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
