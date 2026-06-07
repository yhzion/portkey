package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/updater"
)

const (
	ExitSuccess = 0
	ExitRuntime = 1
	ExitUsage   = 2
)

// Dispatch interprets os.Args and dispatches to the appropriate subcommand.
// Returns exit code, or -1 to signal the caller to launch the TUI.
func Dispatch(osArgs []string, version string, configPath string, upd *updater.Client) int {
	if len(osArgs) < 2 {
		return -1
	}

	switch osArgs[1] {
	case "--help", "-h":
		fmt.Print(helpRoot(version))
		return ExitSuccess
	case "--version", "-v":
		fmt.Printf("portkey %s\n", version)
		return ExitSuccess
	case "list":
		return runList(osArgs[2:], configPath)
	case "add":
		return runAdd(osArgs[2:], configPath)
	case "edit":
		return runEdit(osArgs[2:], configPath)
	case "delete":
		return runDelete(osArgs[2:], configPath)
	case "connect":
		return runConnect(osArgs[2:], configPath)
	case "update":
		return runUpdate(osArgs[2:], version, upd)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", osArgs[1])
		fmt.Fprintf(os.Stderr, "Run 'portkey --help' for usage.\n")
		return ExitUsage
	}
}

// hasHelp checks args for --help or -h flags.
func hasHelp(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}
	return false
}

// loadConfig loads from configPath, or falls back to default location.
func loadConfig(configPath string) (*config.Config, error) {
	if configPath == "" {
		return config.Load()
	}
	return config.NewStore(configPath).Load()
}

// saveConfig saves to configPath, or falls back to default location.
func saveConfig(configPath string, cfg *config.Config) error {
	if configPath == "" {
		return config.Save(cfg)
	}
	return config.NewStore(configPath).Save(cfg)
}

// parsePort validates a port string. Returns (port, exitCode). SSOT.
func parsePort(s string) (int, int) {
	port, err := strconv.Atoi(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: port must be a number\n")
		return 0, ExitUsage
	}
	if port < 1 || port > 65535 {
		fmt.Fprintf(os.Stderr, "Error: port must be between 1 and 65535\n")
		return 0, ExitUsage
	}
	return port, ExitSuccess
}
