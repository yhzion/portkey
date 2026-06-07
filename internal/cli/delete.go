package cli

import (
	"flag"
	"fmt"
	"os"
)

func runDelete(args []string, configPath string) int {
	if hasHelp(args) {
		fmt.Print(helpDelete())
		return ExitSuccess
	}
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	name := fs.String("name", "", "host to delete (required)")
	force := fs.Bool("force", false, "skip confirmation prompt")
	fs.Usage = func() { fmt.Print(helpDelete()) }
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

	if !*force {
		fmt.Printf("Delete %q? [y/n] ", cfg.Hosts[idx].Name)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Canceled.")
			return ExitSuccess
		}
	}

	hostName := cfg.Hosts[idx].Name
	cfg.RemoveHost(idx)
	if err := saveConfig(configPath, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		return ExitRuntime
	}

	fmt.Printf("Host %q deleted.\n", hostName)
	return ExitSuccess
}
