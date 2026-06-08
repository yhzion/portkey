package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/yhzion/portkey/internal/config"
)

var editCmd = &Command{
	Name:      "edit",
	ShortDesc: "Edit an existing host; only specified flags are updated",
	Flags: func(fs *flag.FlagSet) {
		fs.String("name", "", "host to edit (required)")
		fs.String("new-name", "", "rename host")
		fs.String("user", "", "new SSH username")
		fs.String("host", "", "new hostname or IP")
		fs.String("port", "", "new SSH port (1-65535)")
	},
	Run: func(ctx *RunContext) int {
		name := getString(ctx.Flags, "name")
		newName := getString(ctx.Flags, "new-name")
		user := getString(ctx.Flags, "user")
		host := getString(ctx.Flags, "host")
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
		if newName != "" {
			if err := config.ValidateName(newName); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return ExitUsage
			}
			if cfg.IsDuplicateName(newName, idx) {
				fmt.Fprintf(os.Stderr, "Error: duplicate host name %q\n", newName)
				return ExitUsage
			}
			h.Name = newName
		}
		if user != "" {
			h.Username = user
		}
		if host != "" {
			h.Host = host
		}
		if portStr != "" {
			port, code := parsePort(portStr)
			if code != ExitSuccess {
				return code
			}
			h.Port = port
		}

		cfg.UpdateHost(idx, h)
		if err := saveConfig(ctx.ConfigPath, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			return ExitRuntime
		}

		fmt.Printf("Host %q updated.\n", name)
		return ExitSuccess
	},
}
