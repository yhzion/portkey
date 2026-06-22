package cli

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// confirmSuffix asks the user to confirm an action on a suffix (non-exact)
// host match. It returns true only on an explicit affirmative response.
//
// When stdin is NOT a terminal (piped/scripted, e.g. CI), it refuses with an
// error naming the matched host so a suffix match can never silently connect
// or edit in non-interactive contexts (issue #46). hostName is the matched
// host's full name; query is the user's --name argument; verb is "Connect" or
// "Edit".
//
// confirmSuffix is a package-level variable so tests can stub it without
// touching real stdin.
var confirmSuffix = defaultConfirmSuffix

func defaultConfirmSuffix(hostName, query, verb string) (bool, error) {
	if !isTerminal(os.Stdin) {
		return false, fmt.Errorf(
			"non-interactive suffix match %q for %q; %s aborted. "+
				"supply the exact host name to proceed",
			hostName, query, verb,
		)
	}

	fmt.Printf("Found suffix match %q for %q. %s? [y/N] ", hostName, query, verb)
	var response string
	fmt.Scanln(&response)
	switch response {
	case "y", "Y", "yes", "YES":
		return true, nil
	default:
		return false, nil
	}
}

// isTerminal reports whether f is a terminal (character device). It uses the
// stdlib only (no external deps) so the safety check works on macOS and Linux.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

var connectCmd = &Command{
	Name:      "connect",
	ShortDesc: "SSH to a configured host",
	Flags: func(fs *flag.FlagSet) {
		fs.String("name", "", "host to connect to (required)")
		fs.String("user", "", "override username for this session")
		fs.String("port", "", "override port for this session (1-65535)")
	},
	Run: func(ctx *RunContext) int {
		name := getString(ctx.Flags, "name")
		user := getString(ctx.Flags, "user")
		portStr := getString(ctx.Flags, "port")

		if name == "" {
			fmt.Fprintf(os.Stderr, "Error: --name is required.\n")
			return ExitUsage
		}

		cfg, err := loadConfig(ctx.ConfigPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			return ExitRuntime
		}

		idx, exact, err := cfg.FindHostByNameMatch(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return ExitRuntime
		}

		// A suffix (not exact) match can land on a host the user did not
		// intend. Require explicit confirmation before connecting; in a
		// non-interactive context the confirm seam aborts with a safety error.
		if !exact {
			ok, cerr := confirmSuffix(cfg.Hosts[idx].Name, name, "Connect")
			if cerr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", cerr)
				return ExitRuntime
			}
			if !ok {
				fmt.Println("Canceled.")
				return ExitRuntime
			}
		}

		h := cfg.Hosts[idx]
		if user != "" {
			h.Username = user
		}
		if portStr != "" {
			port, code := parsePort(portStr)
			if code != ExitSuccess {
				return code
			}
			h.Port = port
		}

		if err := defaultSSHRunner.Run(h); err != nil {
			fmt.Fprintf(os.Stderr, "SSH error: %v\n", err)
			return ExitRuntime
		}

		// Stamp LastUsed so this host bubbles up the recency-sorted list,
		// mirroring the TUI path. A save failure here is reported but does
		// not fail the already-successful SSH session.
		cfg.Hosts[idx].LastUsed = time.Now().Format(time.RFC3339)
		if err := saveConfig(ctx.ConfigPath, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		}
		return ExitSuccess
	},
}
