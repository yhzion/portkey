package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/yhzion/portkey/internal/config"
)

var addCmd = &Command{
	Name:      "add",
	ShortDesc: "Add a new host to the config",
	Flags: func(fs *flag.FlagSet) {
		fs.String("name", "", "host name (required, [a-z0-9_-])")
		fs.String("user", "", "SSH username (required)")
		fs.String("host", "", "hostname or IP (required)")
		fs.String("port", "22", "SSH port (1-65535)")
	},
	Run: func(ctx *RunContext) int {
		name := getString(ctx.Flags, "name")
		user := getString(ctx.Flags, "user")
		host := getString(ctx.Flags, "host")
		portStr := getString(ctx.Flags, "port")

		if name == "" || user == "" || host == "" {
			fmt.Fprintf(os.Stderr, "Error: --name, --user, and --host are required.\n")
			return ExitUsage
		}
		if err := config.ValidateName(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return ExitUsage
		}
		port, code := parsePort(portStr)
		if code != ExitSuccess {
			return code
		}

		cfg, err := loadConfig(ctx.ConfigPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			return ExitRuntime
		}
		if cfg.IsDuplicateName(name, -1) {
			fmt.Fprintf(os.Stderr, "Error: duplicate host name %q\n", name)
			return ExitUsage
		}

		cfg.AddHost(config.Host{
			Name: name, Username: user, Host: host, Port: port,
		})
		if err := saveConfig(ctx.ConfigPath, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			return ExitRuntime
		}

		fmt.Printf("Host %q added.\n", name)
		return ExitSuccess
	},
}
