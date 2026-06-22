package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	upd := updater.DefaultClient()

	if len(os.Args) > 1 {
		// CLI dispatch always uses the raw client — explicit `portkey update` must
		// never be served from cache; it is infrequent and user-initiated.
		code := cli.Dispatch(os.Args, version, "", upd)
		if code >= 0 {
			os.Exit(code)
		}
	}

	store, err := config.GetStore("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config store: %v\n", err)
		os.Exit(1)
	}

	cfg, err := store.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// The TUI background check runs on every launch; cache it to stay within
	// GitHub's unauthenticated rate limit (60 req/hr/IP). If ConfigDir() fails
	// we fall back to the raw client rather than aborting startup.
	tuiChecker := buildTUIChecker(upd)
	m := tui.InitialModel(cfg, version, tuiChecker, upd, store)

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
	if err := store.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
	}

	// Run ssh.
	if err := ssh.Run(*selectedHost); err != nil {
		fmt.Fprintf(os.Stderr, "SSH error: %v\n", err)
		os.Exit(1)
	}
}

// buildTUIChecker returns a CachingChecker for the TUI background update
// check. If the config directory cannot be determined, it falls back to the
// raw client so startup is never blocked by a config-path failure.
func buildTUIChecker(raw *updater.Client) tui.UpdateChecker {
	dir, err := config.ConfigDir()
	if err != nil {
		// Cannot determine cache path; use raw client (no caching) rather than
		// failing startup. This is expected to be extremely rare in practice.
		return raw
	}
	cachePath := filepath.Join(dir, "update-check.json")
	return updater.NewCachingChecker(raw, cachePath, updater.UpdateCheckTTL)
}
