package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/yhzion/portkey/internal/config"
)

func runEdit(args []string, configPath string) int {
	if hasHelp(args) {
		fmt.Print(helpEdit())
		return ExitSuccess
	}
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	name := fs.String("name", "", "host to edit (required)")
	newName := fs.String("new-name", "", "rename host")
	user := fs.String("user", "", "new SSH username")
	host := fs.String("host", "", "new hostname or IP")
	portStr := fs.String("port", "", "new SSH port (1-65535)")
	fs.Usage = func() { fmt.Print(helpEdit()) }
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
	if *newName != "" {
		if err := config.ValidateName(*newName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return ExitUsage
		}
		if cfg.IsDuplicateName(*newName, idx) {
			fmt.Fprintf(os.Stderr, "Error: duplicate host name %q\n", *newName)
			return ExitUsage
		}
		h.Name = *newName
	}
	if *user != "" {
		h.Username = *user
	}
	if *host != "" {
		h.Host = *host
	}
	if *portStr != "" {
		port, code := parsePort(*portStr)
		if code != ExitSuccess {
			return code
		}
		h.Port = port
	}

	cfg.UpdateHost(idx, h)
	if err := saveConfig(configPath, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return ExitRuntime
	}

	fmt.Printf("Host %q updated.\n", *name)
	return ExitSuccess
}
