package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shubh-io/dockmate/internal/check"
	"github.com/shubh-io/dockmate/internal/tui"
	"github.com/shubh-io/dockmate/internal/update"
	"github.com/shubh-io/dockmate/pkg/version"
)

// ============================================================================
// Main
// ============================================================================

func main() {

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Printf("DockMate version: %s\n", version.Dockmate_Version)
			return
		case "update":
			update.UpdateCommand()
			return
		}
	}

	result := check.RunPreChecks()

	if !result.Passed {
		fmt.Fprintf(os.Stderr, "%s\n\n%s\n", result.ErrorMessage, result.SuggestedAction)
		os.Stderr.Sync()
		os.Exit(1)
	}

	// start the TUI with alternate screen mode
	// (alternate screen = your terminal history stays clean)

	p := tea.NewProgram(tui.InitialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
