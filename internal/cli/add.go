package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/yhzion/portkey/internal/config"
)

func runAdd(args []string, configPath string) int {
	if hasHelp(args) {
		fmt.Print(helpAdd())
		return ExitSuccess
	}
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	name := fs.String("name", "", "host name (required, [a-z0-9_-])")
	user := fs.String("user", "", "SSH username (required)")
	host := fs.String("host", "", "hostname or IP (required)")
	portStr := fs.String("port", "22", "SSH port (1-65535)")
	fs.Usage = func() { fmt.Print(helpAdd()) }
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	if *name == "" || *user == "" || *host == "" {
		fmt.Fprintf(os.Stderr, "Error: --name, --user, and --host are required.\n")
		return ExitUsage
	}
	if err := config.ValidateName(*name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitUsage
	}
	port, code := parsePort(*portStr)
	if code != ExitSuccess {
		return code
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return ExitRuntime
	}
	if cfg.IsDuplicateName(*name, -1) {
		fmt.Fprintf(os.Stderr, "Error: duplicate host name %q\n", *name)
		return ExitUsage
	}

	cfg.AddHost(config.Host{
		Name: *name, Username: *user, Host: *host, Port: port,
	})
	if err := saveConfig(configPath, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return ExitRuntime
	}

	fmt.Printf("Host %q added.\n", *name)
	return ExitSuccess
}
