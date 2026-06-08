package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/yhzion/portkey/internal/ssh"
)

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

		idx, err := cfg.FindHostByName(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return ExitRuntime
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

		if err := ssh.Run(h); err != nil {
			fmt.Fprintf(os.Stderr, "SSH error: %v\n", err)
			return ExitRuntime
		}
		return ExitSuccess
	},
}
