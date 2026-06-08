# Design: CLI Shared Command Helpers & Unified Help Text

**Date:** 2026-06-08
**Issue:** #23
**Status:** Approved

## Summary

Extract repeated boilerplate from 5 CLI subcommand files into a `Command` struct-based framework, and replace hand-authored help text with auto-generation from flag definitions.

## Problem

### 1. Boilerplate duplication (5 files, 7-step pattern repeated)

Every subcommand that touches config repeats:

1. Help check (`hasHelp` + `fmt.Print(helpXxx())`)
2. `flag.NewFlagSet` + flag declarations
3. `fs.Usage` override
4. `fs.Parse` + error return
5. Required-flag validation
6. Config load + error
7. Config save + error (where applicable)

Over 6 command files, this produces ~120 lines of pure boilerplate.

### 2. Help text synchronization risk

- `help.go` contains 7 hand-authored help functions (~130 lines)
- Flag descriptions are registered via `fs.String(...)` but never displayed (custom `fs.Usage` overrides them)
- Root help subcommand list must be manually synced with `dispatch.go` switch cases
- Adding a new subcommand requires edits in 3 places: command file, dispatch.go, help.go

## Design

### Approach: `Command` struct + shared dispatch framework

Rather than extracting helper functions that each command still calls individually, define a `Command` struct that each subcommand populates. The dispatch framework handles the common lifecycle (help check, flag parsing), and each command's `Run` function contains only its unique business logic.

### Core types

```go
// dispatch.go

const (
    ExitSuccess = 0
    ExitUsage   = 1
    ExitRuntime = 2
)

type Command struct {
    Name      string
    ShortDesc string
    Flags     func(fs *flag.FlagSet)   // register flags only
    Run       func(ctx *RunContext) int // business logic only
}

type RunContext struct {
    Flags      *flag.FlagSet
    ConfigPath string
    Args       []string
}
```

**Design decisions:**

- `RunContext` does not include a loaded `Config`. Commands like `list` and `update` don't need config; `connect` is read-only; `add`/`edit`/`delete` read and write. Each command loads config as needed via existing `loadConfig()` / `saveConfig()` helpers.
- `Flags` is optional (`nil` for `version`). The framework skips flag setup when `Flags` is nil.

### Dispatch flow

```go
var commands = map[string]*Command{
    "list":    listCmd,
    "add":     addCmd,
    "edit":    editCmd,
    "delete":  deleteCmd,
    "connect": connectCmd,
    "update":  updateCmd,
}

var commandOrder = []*Command{
    listCmd, addCmd, editCmd, deleteCmd, connectCmd, updateCmd,
}

func Dispatch(args []string, version string, configPath string, upd *updater.Client) int {
    if len(args) < 2 {
        fmt.Print(rootHelp())
        return ExitSuccess
    }
    name := args[1]

    if name == "version" {
        fmt.Printf("portkey %s\n", version)
        return ExitSuccess
    }

    cmd, ok := commands[name]
    if !ok {
        fmt.Fprintf(os.Stderr, "Unknown command: %s\n", name)
        return ExitUsage
    }

    if hasHelp(args[2:]) {
        fmt.Print(cmd.Help())
        return ExitSuccess
    }

    fs := flag.NewFlagSet(name, flag.ContinueOnError)
    fs.Usage = func() { fmt.Print(cmd.Help()) }
    if cmd.Flags != nil {
        cmd.Flags(fs)
    }
    if err := fs.Parse(args[2:]); err != nil {
        return ExitUsage
    }

    return cmd.Run(&RunContext{
        Flags:      fs,
        ConfigPath: configPath,
        Args:       fs.Args(),
    })
}
```

**Boilerplate eliminated by the framework:**

- Help check + `fmt.Print(helpXxx())` + early return (6 commands)
- `flag.NewFlagSet` + `fs.Usage` binding + `fs.Parse` + error handling (5 commands)
- `hasHelp()` direct calls (all commands)

**Intentionally left to each command:**

- Config load/save and name lookup — error handling varies slightly per command, and some commands don't need them

### Command example: delete

```go
var deleteCmd = &Command{
    Name:      "delete",
    ShortDesc: "Remove a saved host",
    Flags: func(fs *flag.FlagSet) {
        fs.String("name", "", "host name (required)")
        fs.Bool("force", false, "skip confirmation prompt")
    },
    Run: func(ctx *RunContext) int {
        name := ctx.Flags.Lookup("name").Value.String()
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

        force := getBool(ctx.Flags, "force")
        if !force {
            // existing stdin confirmation logic, unchanged
        }

        cfg.RemoveHost(idx)
        return saveAndExit(ctx.ConfigPath, cfg)
    },
}
```

### Help text auto-generation

Replace `help.go` (7 hand-authored functions, ~130 lines) with two auto-generating functions:

```go
// Command.Help generates usage text from Command metadata + FlagSet
func (c *Command) Help() string {
    var b strings.Builder
    fmt.Fprintf(&b, "Usage: portkey %s [flags]\n\n", c.Name)
    fmt.Fprintf(&b, "%s\n\n", c.ShortDesc)

    if c.Flags != nil {
        fs := flag.NewFlagSet(c.Name, flag.ContinueOnError)
        c.Flags(fs)

        b.WriteString("Flags:\n")
        fs.VisitAll(func(f *flag.Flag) {
            fmt.Fprintf(&b, "  --%-12s %s", f.Name, f.Usage)
            if f.DefValue != "" && f.DefValue != "false" {
                fmt.Fprintf(&b, " (default: %s)", f.DefValue)
            }
            b.WriteByte('\n')
        })
    }
    return b.String()
}

// rootHelp generates the top-level help from commandOrder
func rootHelp() string {
    var b strings.Builder
    b.WriteString("Usage: portkey <command> [flags]\n\n")
    b.WriteString("Commands:\n")
    for _, cmd := range commandOrder {
        fmt.Fprintf(&b, "  %-10s %s\n", cmd.Name, cmd.ShortDesc)
    }
    b.WriteString("\nRun 'portkey <command> --help' for details.\n")
    return b.String()
}
```

**Result:** Flag definitions are the single source of truth. Adding a flag to `Flags` automatically updates help text. Adding a new command to `commands` + `commandOrder` automatically updates root help.

### Update command: factory pattern

The `update` command needs an `*updater.Client` not available at init time. Use a factory:

```go
func newUpdateCmd(upd *updater.Client) *Command {
    return &Command{
        Name:      "update",
        ShortDesc: "Check and install the latest version",
        Run: func(ctx *RunContext) int {
            // uses captured upd directly
        },
    }
}
```

In `Dispatch`, register dynamically: `commands["update"] = newUpdateCmd(upd)`.

### Helper functions

Small helpers for repeated patterns within `Run` functions:

```go
func saveAndExit(configPath string, cfg *config.Config) int {
    if err := saveConfig(configPath, cfg); err != nil {
        fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
        return ExitRuntime
    }
    return ExitSuccess
}

func getString(fs *flag.FlagSet, name string) string {
    return fs.Lookup(name).Value.String()
}

func getBool(fs *flag.FlagSet, name string) bool {
    b, _ := strconv.ParseBool(fs.Lookup(name).Value.String())
    return b
}
```

## Testing

### Existing tests: no changes needed

All 22 tests call `cli.Dispatch()` as a black box. The public API (`Dispatch` signature) is unchanged. Tests should pass without modification.

### New test: help-flag synchronization

```go
func TestCommandHelpContainsAllFlags(t *testing.T) {
    for _, cmd := range commandOrder {
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

This test guarantees that any registered flag appears in the generated help text — the synchronization that was previously manual is now verified automatically.

## File changes

| File | Change |
|------|--------|
| `dispatch.go` | Add `Command`, `RunContext`, `commandOrder`, rewrite `Dispatch`, add `rootHelp()`, `Command.Help()`, helper functions |
| `add.go` | `runAdd` function → `addCmd` struct literal |
| `edit.go` | `runEdit` function → `editCmd` struct literal |
| `delete.go` | `runDelete` function → `deleteCmd` struct literal |
| `connect.go` | `runConnect` function → `connectCmd` struct literal |
| `list.go` | `runList` function → `listCmd` struct literal |
| `update.go` | `runUpdate` function → `newUpdateCmd` factory + `updateCmd` |
| `help.go` | **Delete** — replaced by `rootHelp()` + `Command.Help()` |
| `cli_test.go` | No changes to existing tests; add `TestCommandHelpContainsAllFlags` |

## Dependency

- Requires Issue 2 (ConfigStore interface) — already completed and merged in PR #28.

## Risk

Medium — 6 subcommand files modified, help text format changes slightly. Mitigated by:
- Public API (`Dispatch` signature) unchanged
- All existing tests pass without modification
- Help text format is visually similar to current output
