package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/yhzion/portkey/internal/ssh"
)

func runConnect(args []string, configPath string) int {
	if hasHelp(args) {
		fmt.Print(helpConnect())
		return ExitSuccess
	}
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	name := fs.String("name", "", "host to connect to (required)")
	user := fs.String("user", "", "override username for this session")
	portStr := fs.String("port", "", "override port for this session (1-65535)")
	fs.Usage = func() { fmt.Print(helpConnect()) }
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	if *name == "" {
		fmt.Fprintf(os.Stderr, "Error: --name is required.\n")
		return ExitUsage
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return ExitRuntime
	}

	idx, err := cfg.FindHostByName(*name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitRuntime
	}

	h := cfg.Hosts[idx]
	if *user != "" {
		h.Username = *user
	}
	if *portStr != "" {
		port, code := parsePort(*portStr)
		if code != ExitSuccess {
			return code
		}
		h.Port = port
	}

	if err := ssh.Run(h); err != nil {
		fmt.Fprintf(os.Stderr, "SSH error: %v\n", err)
		return ExitRuntime
	}
	return ExitSuccess
}
