package cli

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
		"  connect    SSH to a host\n" +
		"  update     Update portkey to the latest version\n\n" +
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

func helpUpdate() string {
	return "update - Update portkey to the latest version.\n\n" +
		"USAGE:\n" +
		"  portkey update\n\n" +
		"DESCRIPTION:\n" +
		"  Checks GitHub for the latest release and updates portkey if a newer\n" +
		"  version is available. The running binary is replaced in-place.\n\n" +
		"EXIT CODES:\n" +
		"  0    Updated or already up to date\n" +
		"  1    Update failed (network, permissions, etc.)\n"
}
