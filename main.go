package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/tui"
	"github.com/yhzion/portkey/internal/updater"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("portkey %s\n", version)
			os.Exit(0)
		case "--help", "-h":
			fmt.Println("Portkey — pick a host and jump in.")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  portkey          Start interactive SSH host picker")
			fmt.Println("  portkey -v       Print version")
			fmt.Println("  portkey -h       Print help")
			os.Exit(0)
		default:
			if !strings.HasPrefix(os.Args[1], "-") {
				break
			}
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", os.Args[1])
			os.Exit(1)
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
