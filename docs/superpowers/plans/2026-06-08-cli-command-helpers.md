# CLI Shared Command Helpers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace repeated CLI subcommand boilerplate with a `Command` struct framework and auto-generate help text from flag definitions.

**Architecture:** Define `Command`/`RunContext` types in `dispatch.go`. Each subcommand file exports a `*Command` variable. `Dispatch` handles the common lifecycle (help check, flag parsing) and delegates business logic to each command's `Run` closure. Help text is auto-generated from `Command` metadata + `flag.FlagSet` definitions.

**Tech Stack:** Go stdlib (`flag`, `strings`, `strconv`, `fmt`)

**Design spec:** `docs/superpowers/specs/2026-06-08-cli-command-helpers-design.md`

---

## ⚠️ Compilation Note

Tasks 1–8 modify all files in the `cli` package. The code **will not compile** until all tasks are complete. Task 9 runs tests after everything is in place.

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/cli/dispatch.go` | Rewrite | `Command`, `RunContext`, `Dispatch`, helpers, help generation |
| `internal/cli/add.go` | Rewrite | `addCmd` variable |
| `internal/cli/edit.go` | Rewrite | `editCmd` variable |
| `internal/cli/delete.go` | Rewrite | `deleteCmd` variable |
| `internal/cli/connect.go` | Rewrite | `connectCmd` variable |
| `internal/cli/list.go` | Rewrite | `listCmd` variable + output helpers |
| `internal/cli/update.go` | Rewrite | `updateCmd` placeholder + `newUpdateCmd` factory |
| `internal/cli/help.go` | Delete | Replaced by auto-generation |
| `internal/cli/cli_test.go` | Append | Add `TestCommandHelpContainsAllFlags` |

---

### Task 1: Rewrite dispatch.go

**Files:**
- Rewrite: `internal/cli/dispatch.go`

- [ ] **Step 1: Replace dispatch.go with the new framework**

```go
package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/updater"
)

const (
	ExitSuccess = 0
	ExitRuntime = 1
	ExitUsage   = 2
)

// Command represents a CLI subcommand.
type Command struct {
	Name      string
	ShortDesc string
	Flags     func(fs *flag.FlagSet)   // register flags
	Run       func(ctx *RunContext) int // business logic
}

// RunContext carries parsed flags and shared data into Command.Run.
type RunContext struct {
	Flags      *flag.FlagSet
	ConfigPath string
	Args       []string
}

// Commands holds all registered subcommands in display order. Exported for tests.
var Commands = []*Command{
	listCmd, addCmd, editCmd, deleteCmd, connectCmd, updateCmd,
}

// commands maps subcommand names to their Command definitions.
var commands = map[string]*Command{
	"list":    listCmd,
	"add":     addCmd,
	"edit":    editCmd,
	"delete":  deleteCmd,
	"connect": connectCmd,
	"update":  updateCmd,
}

// Dispatch interprets os.Args and dispatches to the appropriate subcommand.
// Returns exit code, or -1 to signal the caller to launch the TUI.
func Dispatch(osArgs []string, version string, configPath string, upd *updater.Client) int {
	if len(osArgs) < 2 {
		return -1
	}

	name := osArgs[1]

	switch name {
	case "--help", "-h":
		fmt.Print(rootHelp())
		return ExitSuccess
	case "--version", "-v":
		fmt.Printf("portkey %s\n", version)
		return ExitSuccess
	}

	cmd, ok := commands[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", name)
		fmt.Fprintf(os.Stderr, "Run 'portkey --help' for usage.\n")
		return ExitUsage
	}

	if hasHelp(osArgs[2:]) {
		fmt.Print(cmd.Help())
		return ExitSuccess
	}

	// Update needs client and version injected at call time.
	if name == "update" {
		cmd = newUpdateCmd(upd, version)
	}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() { fmt.Print(cmd.Help()) }
	if cmd.Flags != nil {
		cmd.Flags(fs)
	}
	if err := fs.Parse(osArgs[2:]); err != nil {
		return ExitUsage
	}

	return cmd.Run(&RunContext{
		Flags:      fs,
		ConfigPath: configPath,
		Args:       fs.Args(),
	})
}

// Help generates usage text from Command metadata + flag definitions.
func (c *Command) Help() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Usage: portkey %s [flags]\n\n", c.Name)
	fmt.Fprintf(&b, "%s\n", c.ShortDesc)

	if c.Flags != nil {
		fs := flag.NewFlagSet(c.Name, flag.ContinueOnError)
		c.Flags(fs)
		hasFlags := false
		fs.VisitAll(func(f *flag.Flag) { hasFlags = true })
		if hasFlags {
			fmt.Fprintf(&b, "\nFlags:\n")
			fs.VisitAll(func(f *flag.Flag) {
				fmt.Fprintf(&b, "  --%-12s %s", f.Name, f.Usage)
				if f.DefValue != "" && f.DefValue != "false" {
					fmt.Fprintf(&b, " (default: %s)", f.DefValue)
				}
				b.WriteByte('\n')
			})
		}
	}

	return b.String()
}

// rootHelp generates the top-level help.
func rootHelp() string {
	var b strings.Builder
	b.WriteString("portkey - Pick a host and jump in.\n\n")
	b.WriteString("USAGE:\n")
	b.WriteString("  portkey                              Launch interactive TUI\n")
	b.WriteString("  portkey <subcommand> [flags]\n\n")
	b.WriteString("SUBCOMMANDS:\n")
	for _, cmd := range Commands {
		fmt.Fprintf(&b, "  %-10s %s\n", cmd.Name, cmd.ShortDesc)
	}
	b.WriteString("\nRun 'portkey <subcommand> --help' for details.\n")
	return b.String()
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

// getString returns the string value of a named flag.
func getString(fs *flag.FlagSet, name string) string {
	return fs.Lookup(name).Value.String()
}

// getBool returns the bool value of a named flag.
func getBool(fs *flag.FlagSet, name string) bool {
	b, _ := strconv.ParseBool(fs.Lookup(name).Value.String())
	return b
}

// parsePort validates a port string. Returns (port, exitCode).
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

// Compile guard: ensure updater.Client is referenced.
var _ *updater.Client
```

> **Note:** The `var _ *updater.Client` at the bottom is a compile guard to keep the `updater` import. It will be naturally used by `newUpdateCmd` in `update.go`. Remove this line when update.go is converted in Task 7.

---

### Task 2: Convert add.go

**Files:**
- Rewrite: `internal/cli/add.go`

- [ ] **Step 1: Replace add.go with Command struct**

```go
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
```

---

### Task 3: Convert edit.go

**Files:**
- Rewrite: `internal/cli/edit.go`

- [ ] **Step 1: Replace edit.go with Command struct**

```go
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
```

---

### Task 4: Convert delete.go

**Files:**
- Rewrite: `internal/cli/delete.go`

- [ ] **Step 1: Replace delete.go with Command struct**

```go
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
```

---

### Task 5: Convert connect.go

**Files:**
- Rewrite: `internal/cli/connect.go`

- [ ] **Step 1: Replace connect.go with Command struct**

```go
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
```

---

### Task 6: Convert list.go

**Files:**
- Rewrite: `internal/cli/list.go`

- [ ] **Step 1: Replace list.go with Command struct + output helpers**

```go
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/yhzion/portkey/internal/config"
)

var listCmd = &Command{
	Name:      "list",
	ShortDesc: "List configured hosts",
	Flags: func(fs *flag.FlagSet) {
		fs.Bool("json", false, "output as JSON")
	},
	Run: func(ctx *RunContext) int {
		cfg, err := loadConfig(ctx.ConfigPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			return ExitRuntime
		}

		if getBool(ctx.Flags, "json") {
			return printHostsJSON(cfg.Hosts)
		}
		return printHostsTable(cfg.Hosts)
	},
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
```

---

### Task 7: Convert update.go

**Files:**
- Rewrite: `internal/cli/update.go`

- [ ] **Step 1: Replace update.go with placeholder + factory**

```go
package cli

import (
	"fmt"
	"os"

	"github.com/yhzion/portkey/internal/updater"
)

// updateCmd is a placeholder used for root help display and command lookup.
// Dispatch replaces it with a fresh command from newUpdateCmd at runtime.
var updateCmd = &Command{
	Name:      "update",
	ShortDesc: "Check and install the latest version",
}

func newUpdateCmd(upd *updater.Client, version string) *Command {
	return &Command{
		Name:      "update",
		ShortDesc: "Check and install the latest version",
		Run: func(ctx *RunContext) int {
			if upd == nil {
				upd = updater.DefaultClient()
			}

			rel, err := upd.CheckLatest()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
				return ExitRuntime
			}

			if !updater.IsNewer(version, rel.Tag) {
				fmt.Printf("Already up to date (%s).\n", version)
				return ExitSuccess
			}

			fmt.Printf("Updating %s → %s ...\n", version, rel.Tag)

			if err := upd.DownloadAndInstall(rel); err != nil {
				fmt.Fprintf(os.Stderr, "Error updating: %v\n", err)
				return ExitRuntime
			}

			fmt.Printf("Updated to %s.\n", rel.Tag)
			return ExitSuccess
		},
	}
}
```

- [ ] **Step 2: Remove compile guard from dispatch.go**

In `internal/cli/dispatch.go`, delete the last two lines:

```go
// Compile guard: ensure updater.Client is referenced.
var _ *updater.Client
```

These are no longer needed since `updater` is now used by `newUpdateCmd` in `update.go`.

---

### Task 8: Delete help.go

**Files:**
- Delete: `internal/cli/help.go`

- [ ] **Step 1: Delete the file**

Run: `rm internal/cli/help.go`

All help functions (`helpRoot`, `helpList`, `helpAdd`, `helpEdit`, `helpDelete`, `helpConnect`, `helpUpdate`) are now replaced by `rootHelp()` and `Command.Help()` in `dispatch.go`.

---

### Task 9: Run all existing tests

**Files:**
- None (validation only)

- [ ] **Step 1: Compile and run tests**

Run: `go build ./... && go test ./internal/cli/ -v -count=1`

Expected: All 22 existing tests pass. The public API (`Dispatch` signature) is unchanged.

If any test fails, compare the actual behavior with the original:
- Help text format may differ (simplified) — only tests checking exit code are affected
- Flag parsing behavior must be identical
- Exit codes must match: `ExitSuccess=0`, `ExitRuntime=1`, `ExitUsage=2`

- [ ] **Step 2: Commit the refactoring**

```bash
git add -A
git commit -m "refactor(cli): extract Command framework and auto-generate help text (#23)

- Introduce Command/RunContext types for subcommand registration
- Replace per-command boilerplate (help check, flag parse) with shared Dispatch framework
- Auto-generate help text from Command metadata + flag definitions
- Delete help.go — replaced by rootHelp() + Command.Help()
- All 22 existing tests pass without modification"
```

---

### Task 10: Add help-flag synchronization test

**Files:**
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Add TestCommandHelpContainsAllFlags**

Append to `internal/cli/cli_test.go`:

```go
func TestCommandHelpContainsAllFlags(t *testing.T) {
	for _, cmd := range cli.Commands {
		if cmd.Flags == nil {
			continue
		}
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		cmd.Flags(fs)
		help := cmd.Help()

		fs.VisitAll(func(f *flag.Flag) {
			if !strings.Contains(help, "--"+f.Name) {
				t.Errorf("%s: help missing flag --%s", cmd.Name, f.Name)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify**

Run: `go test ./internal/cli/ -v -run TestCommandHelpContainsAllFlags`

Expected: PASS — all registered flags appear in their command's generated help text.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/cli_test.go
git commit -m "test(cli): add help-flag synchronization test (#23)"
```
