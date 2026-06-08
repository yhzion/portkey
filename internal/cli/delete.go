package cli

import (
	"flag"
	"fmt"
	"os"
)

var deleteCmd = &Command{
	Name:      "delete",
	ShortDesc: "Remove a saved host",
	Flags: func(fs *flag.FlagSet) {
		fs.String("name", "", "host to delete (required)")
		fs.Bool("force", false, "skip confirmation prompt")
	},
	Run: func(ctx *RunContext) int {
		name := getString(ctx.Flags, "name")
		force := getBool(ctx.Flags, "force")

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

		if !force {
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
		if err := saveConfig(ctx.ConfigPath, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			return ExitRuntime
		}

		fmt.Printf("Host %q deleted.\n", hostName)
		return ExitSuccess
	},
}
