package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/ssh"
)

const (
	ExitSuccess = 0
	ExitRuntime = 1
	ExitUsage   = 2
)

// Dispatch interprets os.Args and dispatches to the appropriate subcommand.
// Returns exit code, or -1 to signal the caller to launch the TUI.
func Dispatch(osArgs []string, version string, configPath string) int {
	if len(osArgs) < 2 {
		return -1
	}

	switch osArgs[1] {
	case "--help", "-h":
		fmt.Print(helpRoot(version))
		return ExitSuccess
	case "--version", "-v":
		fmt.Printf("portkey %s\n", version)
		return ExitSuccess
	case "list":
		return runList(osArgs[2:], configPath)
	case "add":
		return runAdd(osArgs[2:], configPath)
	case "edit":
		return runEdit(osArgs[2:], configPath)
	case "delete":
		return runDelete(osArgs[2:], configPath)
	case "connect":
		return runConnect(osArgs[2:], configPath)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", osArgs[1])
		fmt.Fprintf(os.Stderr, "Run 'portkey --help' for usage.\n")
		return ExitUsage
	}
}

// hasHelp checks args for --help or -h flags.
func hasHelp(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}
	return false
}

// loadConfig loads from configPath, or falls back to default location.
func loadConfig(configPath string) (*config.Config, error) {
	if configPath == "" {
		return config.Load()
	}
	return config.NewStore(configPath).Load()
}

// saveConfig saves to configPath, or falls back to default location.
func saveConfig(configPath string, cfg *config.Config) error {
	if configPath == "" {
		return config.Save(cfg)
	}
	return config.NewStore(configPath).Save(cfg)
}

// parsePort validates a port string. Returns (port, exitCode). SSOT.
func parsePort(s string) (int, int) {
	port, err := strconv.Atoi(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: port must be a number\n")
		return 0, ExitUsage
	}
	if port < 1 || port > 65535 {
		fmt.Fprintf(os.Stderr, "Error: port must be between 1 and 65535\n")
		return 0, ExitUsage
	}
	return port, ExitSuccess
}

// --- subcommands ---

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

// --- help text ---

func helpRoot(version string) string {
	return "portkey - Pick a host and jump in.\n\n" +
		"USAGE:\n" +
		"  portkey                              Launch interactive TUI\n" +
		"  portkey <subcommand> [flags]\n\n" +
		"SUBCOMMANDS:\n" +
		"  NAME       DESCRIPTION\n" +
		"  list       List configured hosts\n" +
		"  add        Add a new host\n" +
		"  edit       Edit an existing host\n" +
		"  delete     Delete a host\n" +
		"  connect    SSH to a host\n\n" +
		"GLOBAL FLAGS:\n" +
		"  --help       Print help and exit\n" +
		"  --version    Print version and exit\n\n" +
		"EXIT CODES:\n" +
		"  0    Success\n" +
		"  1    Runtime error\n" +
		"  2    Usage error\n\n" +
		"MORE HELP:\n" +
		"  portkey <subcommand> --help\n"
}

func helpList() string {
	return "list - List configured hosts.\n\n" +
		"USAGE:\n" +
		"  portkey list [flags]\n\n" +
		"FLAGS:\n" +
		"  --json      Output as JSON (default: false)\n\n" +
		"EXIT CODES:\n" +
		"  0    Success\n" +
		"  2    Usage error\n\n" +
		"EXAMPLES:\n" +
		"  portkey list\n" +
		"  portkey list --json\n"
}

func helpAdd() string {
	return "add - Add a new host to the config.\n\n" +
		"USAGE:\n" +
		"  portkey add --name <string> --user <string> --host <string> [--port <int>]\n\n" +
		"FLAGS:\n" +
		"  --name      string    (required)    Host name. Chars: [a-z0-9_-]. Must be unique.\n" +
		"  --user      string    (required)    SSH username.\n" +
		"  --host      string    (required)    Hostname or IP address.\n" +
		"  --port      int       (default 22)  SSH port. Range: 1-65535.\n\n" +
		"EXIT CODES:\n" +
		"  0    Host added\n" +
		"  1    Config write error\n" +
		"  2    Usage error\n\n" +
		"EXAMPLES:\n" +
		"  portkey add --name prod --user admin --host 10.0.0.1\n" +
		"  portkey add --name staging --user deploy --host 10.0.0.2 --port 2222\n"
}

func helpEdit() string {
	return "edit - Edit an existing host. Only specified flags are updated.\n\n" +
		"USAGE:\n" +
		"  portkey edit --name <string> [--new-name <string>] [--user <string>] [--host <string>] [--port <int>]\n\n" +
		"FLAGS:\n" +
		"  --name        string    (required)    Host to edit. Exact or suffix match.\n" +
		"  --new-name    string    (optional)    Rename host. Chars: [a-z0-9_-]. Must be unique.\n" +
		"  --user        string    (optional)    New SSH username.\n" +
		"  --host        string    (optional)    New hostname or IP address.\n" +
		"  --port        int       (optional)    New SSH port. Range: 1-65535.\n\n" +
		"EXIT CODES:\n" +
		"  0    Host updated\n" +
		"  1    Host not found or ambiguous match\n" +
		"  2    Usage error\n\n" +
		"EXAMPLES:\n" +
		"  portkey edit --name prod --user root\n" +
		"  portkey edit --name staging --new-name staging-v2 --host 10.0.1.2\n"
}

func helpDelete() string {
	return "delete - Delete a host from the config.\n\n" +
		"USAGE:\n" +
		"  portkey delete --name <string> [--force]\n\n" +
		"FLAGS:\n" +
		"  --name      string    (required)    Host to delete. Exact or suffix match.\n" +
		"  --force     bool      (default false)    Skip confirmation prompt.\n\n" +
		"EXIT CODES:\n" +
		"  0    Host deleted\n" +
		"  1    Host not found or ambiguous match\n" +
		"  2    Usage error\n\n" +
		"EXAMPLES:\n" +
		"  portkey delete --name staging\n" +
		"  portkey delete --name staging --force\n"
}

func helpConnect() string {
	return "connect - SSH to a configured host.\n\n" +
		"USAGE:\n" +
		"  portkey connect --name <string> [--user <string>] [--port <int>]\n\n" +
		"FLAGS:\n" +
		"  --name      string    (required)    Host to connect to. Exact or suffix match.\n" +
		"  --user      string    (optional)    Override username for this session.\n" +
		"  --port      int       (optional)    Override port for this session. Range: 1-65535.\n\n" +
		"EXIT CODES:\n" +
		"  0    SSH session exited cleanly\n" +
		"  1    Host not found or SSH failed\n" +
		"  2    Usage error\n\n" +
		"EXAMPLES:\n" +
		"  portkey connect --name prod\n" +
		"  portkey connect --name prod --user root --port 2222\n"
}
