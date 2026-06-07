package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yhzion/portkey/internal/cli"
	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/ssh"
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

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"%s\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render(fmt.Sprintf("Error: %v", err)),
		)
		os.Exit(1)
	}

	// After TUI exits: check if user selected a host to connect to.
	selectedHost, ok := tui.SelectedHost(finalModel)
	if !ok {
		os.Exit(0)
	}

	// Check ssh is available.
	if _, err := exec.LookPath("ssh"); err != nil {
		fmt.Fprintln(os.Stderr, "ssh command not found. Please install OpenSSH client first.")
		os.Exit(1)
	}

	// Update LastUsed timestamp.
	selectedHost.LastUsed = time.Now().Format(time.RFC3339)
	for i, h := range cfg.Hosts {
		if h.Name == selectedHost.Name {
			cfg.Hosts[i].LastUsed = selectedHost.LastUsed
			break
		}
	}
	config.Save(cfg)

	// Run ssh.
	if err := ssh.Run(*selectedHost); err != nil {
		fmt.Fprintf(os.Stderr, "SSH error: %v\n", err)
		os.Exit(1)
	}
}
