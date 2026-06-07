package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/yhzion/portkey/internal/config"
)

func runList(args []string, configPath string) int {
	if hasHelp(args) {
		fmt.Print(helpList())
		return ExitSuccess
	}
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "output as JSON")
	fs.Usage = func() { fmt.Print(helpList()) }
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return ExitRuntime
	}

	if *jsonOutput {
		return printHostsJSON(cfg.Hosts)
	}
	return printHostsTable(cfg.Hosts)
}

func printHostsJSON(hosts []config.Host) int {
	data, err := json.MarshalIndent(hosts, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		return ExitRuntime
	}
	fmt.Println(string(data))
	return ExitSuccess
}

func printHostsTable(hosts []config.Host) int {
	if len(hosts) == 0 {
		fmt.Println("No hosts configured.")
		return ExitSuccess
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "#\tName\tUser\tHost\tPort")
	for i, h := range hosts {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%d\n", i+1, h.Name, h.Username, h.Host, h.Port)
	}
	tw.Flush()
	return ExitSuccess
}
