package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yhzion/portkey/internal/cli"
	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/tui"
	"github.com/yhzion/portkey/internal/updater"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		code := cli.Dispatch(os.Args, version, "")
		if code >= 0 {
			os.Exit(code)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	upd := updater.DefaultClient()
	m := tui.InitialModel(cfg, version, upd)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"%s\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render(fmt.Sprintf("Error: %v", err)),
		)
		os.Exit(1)
	}
}
