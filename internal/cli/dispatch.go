package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/updater"
)

const (
	ExitSuccess = 0
	ExitRuntime = 1
	ExitUsage   = 2

	// ExitUpdateAvailable is returned by `portkey update --check-only` (or
	// --dry-run) when a newer version is available. Scripts and CI can branch
	// on this value to decide whether to trigger an automated update.
	ExitUpdateAvailable = 10
)

// Command represents a CLI subcommand.
type Command struct {
	Name      string
	ShortDesc string
	Flags     func(fs *flag.FlagSet)    // register flags
	Run       func(ctx *RunContext) int // business logic
}

// RunContext carries parsed flags and shared data into Command.Run.
type RunContext struct {
	Flags      *flag.FlagSet
	ConfigPath string
	Args       []string
}

// Commands holds all registered subcommands in display order. Exported for tests.
var Commands = []*Command{
	listCmd, addCmd, editCmd, deleteCmd, connectCmd, updateCmd,
}

// commands maps subcommand names to their Command definitions.
var commands = map[string]*Command{
	"list":    listCmd,
	"add":     addCmd,
	"edit":    editCmd,
	"delete":  deleteCmd,
	"connect": connectCmd,
	"update":  updateCmd,
}

// Dispatch interprets os.Args and dispatches to the appropriate subcommand.
// Returns exit code, or -1 to signal the caller to launch the TUI.
func Dispatch(osArgs []string, version string, configPath string, upd *updater.Client) int {
	if len(osArgs) < 2 {
		return -1
	}

	name := osArgs[1]

	switch name {
	case "--help", "-h":
		fmt.Print(rootHelp())
		return ExitSuccess
	case "--version", "-v":
		fmt.Printf("portkey %s\n", version)
		return ExitSuccess
	}

	cmd, ok := commands[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", name)
		fmt.Fprintf(os.Stderr, "Run 'portkey --help' for usage.\n")
		return ExitUsage
	}

	// Update needs client and version injected at call time (must happen
	// before hasHelp so the flags appear in --help output).
	if name == "update" {
		cmd = newUpdateCmd(upd, version)
	}

	if hasHelp(osArgs[2:]) {
		fmt.Print(cmd.Help())
		return ExitSuccess
	}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() { fmt.Print(cmd.Help()) }
	if cmd.Flags != nil {
		cmd.Flags(fs)
	}
	if err := fs.Parse(osArgs[2:]); err != nil {
		return ExitUsage
	}

	return cmd.Run(&RunContext{
		Flags:      fs,
		ConfigPath: configPath,
		Args:       fs.Args(),
	})
}

// Help generates usage text from Command metadata + flag definitions.
func (c *Command) Help() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Usage: portkey %s [flags]\n\n", c.Name)
	fmt.Fprintf(&b, "%s\n", c.ShortDesc)

	if c.Flags != nil {
		fs := flag.NewFlagSet(c.Name, flag.ContinueOnError)
		c.Flags(fs)
		hasFlags := false
		fs.VisitAll(func(f *flag.Flag) { hasFlags = true })
		if hasFlags {
			fmt.Fprintf(&b, "\nFlags:\n")
			fs.VisitAll(func(f *flag.Flag) {
				fmt.Fprintf(&b, "  --%-12s %s", f.Name, f.Usage)
				if f.DefValue != "" && f.DefValue != "false" {
					fmt.Fprintf(&b, " (default: %s)", f.DefValue)
				}
				b.WriteByte('\n')
			})
		}
	}

	return b.String()
}

// rootHelp generates the top-level help.
func rootHelp() string {
	var b strings.Builder
	b.WriteString("portkey - Pick a host and jump in.\n\n")
	b.WriteString("USAGE:\n")
	b.WriteString("  portkey                              Launch interactive TUI\n")
	b.WriteString("  portkey <subcommand> [flags]\n\n")
	b.WriteString("SUBCOMMANDS:\n")
	for _, cmd := range Commands {
		fmt.Fprintf(&b, "  %-10s %s\n", cmd.Name, cmd.ShortDesc)
	}
	b.WriteString("\nRun 'portkey <subcommand> --help' for details.\n")
	return b.String()
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
	s, err := config.GetStore(configPath)
	if err != nil {
		return nil, err
	}
	return s.Load()
}

// saveConfig saves to configPath, or falls back to default location.
func saveConfig(configPath string, cfg *config.Config) error {
	s, err := config.GetStore(configPath)
	if err != nil {
		return err
	}
	return s.Save(cfg)
}

// getString returns the string value of a named flag.
func getString(fs *flag.FlagSet, name string) string {
	f := fs.Lookup(name)
	if f == nil {
		return ""
	}
	return f.Value.String()
}

// getBool returns the bool value of a named flag.
func getBool(fs *flag.FlagSet, name string) bool {
	f := fs.Lookup(name)
	if f == nil {
		return false
	}
	b, _ := strconv.ParseBool(f.Value.String())
	return b
}

// parsePort validates a port string. Returns (port, exitCode).
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
