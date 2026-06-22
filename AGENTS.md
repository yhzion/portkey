# AGENTS.md

> **For AI coding agents and human contributors.** This file is the single source of truth
> for all rules, conventions, and workflows in this project. Read it before writing any code.

---

## Project Identity

- **Name:** portkey
- **Tagline:** Pick a host and jump in.
- **Description:** Portkey is an interactive SSH host picker written in Go.
- **Module:** `github.com/yhzion/portkey`
- **Binary:** `portkey`
- **Platforms:** macOS, Linux (no Windows)

---

## Architecture

### Directory Structure

```
.
├── AGENTS.md                  # THIS FILE — rules and conventions
├── CHANGELOG.md               # Auto-generated from conventional commits
├── README.md                  # User-facing documentation
├── release.sh                 # Semantic release script (svu-based)
├── .goreleaser.yml            # Cross-platform build config (macOS, Linux)
├── .github/
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── workflows/
│       └── release.yml        # CI: test, lint, release on push to main
├── .git/
│   └── hooks/
│       ├── prepare-commit-msg # Injects commit message template
│       ├── pre-commit         # Auto-fix: goimports, gofumpt, golines, go vet
│       └── pre-push           # Gate: build, test, staticcheck, gocritic, deadcode, dupl
├── go.mod
├── go.sum
├── main.go                    # Entry point: load config → launch TUI
└── internal/
    ├── config/
    │   ├── config.go          # Host/Config types, Store (load/save), path helpers
    │   └── config_test.go     # Unit tests for all config operations
    ├── ssh/
    │   ├── ssh.go             # BuildArgs, Run (exec.Command ssh)
    │   └── ssh_test.go        # Unit tests for arg building, injection safety
    └── tui/
        ├── model.go           # Screen states, hostForm, form builders, screen transitions
        ├── update.go          # Key handling per screen, tea.Update dispatcher
        ├── view.go            # Render functions per screen
        └── styles.go          # Lip Gloss style definitions (colors, widths)
```

### Package Responsibilities

| Package | Role |
|---------|------|
| `main` | Load config, create tea.Program, run. Nothing else. |
| `internal/config` | `Host`/`Config` structs. `Store` interface for file I/O. Path resolution via `os.UserConfigDir`. Name validation (`ValidateName`), lookup (`FindHostByName`). |
| `internal/ssh` | `BuildArgs` converts `Host` → `[]string`. `Run` executes `exec.Command("ssh", ...)`. No shell interpolation. |
| `internal/tui` | All Bubble Tea Model/Update/View. Screen state machine. Lip Gloss styling. Huh forms for add/edit. |

### Key Design Decisions

- **No Cobra/Viper.** This is an interactive TUI, not a command-line tool. Bubble Tea is the framework.
- **No password management.** Assumes `ssh-copy-id` has already been used. Portkey only runs `ssh` commands.
- **SSH execution uses `exec.Command`.** Never `os/exec` with a shell string. Arguments are always separated.
- **Port 22 is the default.** Only adds `-p <port>` when port ≠ 22.
- **Name must be valid** per `config.ValidateName`: lowercase `[a-z0-9_-]` only, unique, non-empty.
- **Config stored as JSON** at `$XDG_CONFIG_HOME/portkey/hosts.json` (platform-respectful via `os.UserConfigDir`).
- **Schema uses `name` (not `displayName`).**

---

## Tech Stack

### Direct Dependencies

| Library | Version | Purpose |
|---------|---------|---------|
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | v1.3.10 | TUI framework (Model/Update/View) |
| [Bubbles](https://github.com/charmbracelet/bubbles) | v1.0.0 | Key bindings (`key.Binding`, `key.Matches`) |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss) | v1.1.0 | Terminal styling |
| [Huh](https://github.com/charmbracelet/huh) | v1.0.0 | Forms for add/edit host |

### Dev Toolchain

| Tool | Purpose | Install |
|------|---------|---------|
| `goimports` | Import sorting | `go install golang.org/x/tools/cmd/goimports@latest` |
| `gofumpt` | Strict `gofmt` | `go install mvdan.cc/gofumpt@latest` |
| `golines` | Long line wrapping (120 col) | `go install github.com/segmentio/golines@latest` |
| `staticcheck` | SA/ST/S/Q checks (blocking) | `go install honnef.co/go/tools/cmd/staticcheck@latest` |
| `gocritic` | Opinionated checks (non-blocking) | `go install github.com/go-critic/go-critic/cmd/gocritic@latest` |
| `deadcode` | Unused code detection (non-blocking) | `go install golang.org/x/tools/cmd/deadcode@latest` |
| `dupl` | Code duplication (non-blocking) | `go install github.com/mibk/dupl@latest` |
| `svu` | Semantic version calculation | `go install github.com/caarlos0/svu@latest` |
| `goreleaser` | Cross-platform builds | `go install github.com/goreleaser/goreleaser/v2@latest` |

> All tools live in `$GOPATH/bin`. Git hooks verify availability before running.

---

## Coding Rules

### TDD (Test-Driven Development)

This project follows strict TDD:

1. **Write tests first.** Red → Green → Refactor.
2. **Every package has a `*_test.go` file.**
3. **Tests are in package `*_test` (external) unless testing unexported internals.**
4. **Use `t.TempDir()` for filesystem tests.** Never write to real config paths in tests.
5. **Dependency injection for testability.** `config.Store` accepts a file path; `ssh.BuildArgs` is pure function. `ssh.Run` uses `exec.Command` but can be swapped via interface.
6. **Run tests continuously:** `go test ./...`

### Go Conventions

- **Imports:** Sorted by `goimports`. Standard library → external → internal.
- **Formatting:** `gofumpt` (stricter than `gofmt`). Max line length: 120 chars (`golines`).
- **No unused imports, variables, or functions.** `deadcode` will flag them.
- **No duplicated code.** `dupl` with threshold 80.
- **Error wrapping:** Use `fmt.Errorf("context: %w", err)`. Never swallow errors.
- **Interface compliance:** `var _ tea.Model = (*model)(nil)` in `view.go`.
- **Small functions.** Each function does one thing. View helpers are separate from logic.
- **No `// TODO` or `// FIXME` comments in committed code.** Open an issue instead.
- **Comments:** Only when the code doesn't explain itself. Keep them short and meaningful.
- **No global mutable state.** All state lives in the `model` struct.

### Security

- **Never store passwords or private keys.** Portkey only stores display name, username, host, port.
- **Never use shell interpolation for SSH.** Always `exec.Command("ssh", args...)`.
- **Validate all inputs.** Username and host required. Port must be 1–65535 if provided, default 22. Name must match `[a-z0-9_-]` and be unique. Use `config.ValidateName` as SSOT.

---

## Screen State Machine

```
┌─────────────┐    a/enter    ┌───────────┐
│  HostList    │──────────────→│  AddHost  │
│              │←──────────────│           │
│              │  esc/save     └───────────┘
│              │    e           ┌───────────┐
│              │──────────────→│  EditHost │
│              │←──────────────│           │
│              │  esc/save     └───────────┘
│              │    d           ┌───────────┐
│              │──────────────→│  Delete   │
│              │←──────────────│  Confirm  │
└─────────────┘  y/n/esc       └───────────┘
       │
       │  1-9/enter/space
       ↓
┌─────────────┐
│  SSH Exec   │────→ back to HostList (or Error)
└─────────────┘
       │
       │  (on error)
       ↓
┌─────────────┐
│  Error      │──── any key ──→ HostList
└─────────────┘
```

### Key Bindings (Host List)

| Key | Action |
|-----|--------|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `Enter` / `Space` | Select current item |
| `1`–`9` | Quick-connect to host by number |
| `a` | Open add host form |
| `e` | Edit selected host |
| `d` | Delete selected host (with confirmation) |
| `q` / `Ctrl+C` | Quit |
| `Esc` | Cancel / go back |

### Key Bindings (Form / Delete Confirm / Error)

| Screen | Key | Action |
|--------|-----|--------|
| Add/Edit form | `Esc` | Cancel, return to host list |
| Delete confirm | `y` | Confirm deletion |
| Delete confirm | `n` / `Esc` | Cancel, return to host list |
| Error | any key | Return to host list |

---

## Git Workflow

### Branch Model

- `main` — stable, release-worthy. Protected.
- Feature branches: `feat/short-description`, `fix/short-description`, `chore/...`
- Always rebase on `main` before pushing.

### Commit Message Convention

Conventional Commits 1.0 compliant. Git hooks enforce this.

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**

| Type | Purpose | Bump |
|------|---------|------|
| `feat` | New feature | minor |
| `fix` | Bug fix | patch |
| `docs` | Documentation only | none |
| `style` | Formatting (no logic change) | none |
| `refactor` | Code restructuring | none |
| `perf` | Performance improvement | patch |
| `test` | Adding/updating tests | none |
| `chore` | Tooling, deps, build | none |
| `ci` | CI/CD changes | none |

**Scopes:** `config`, `ssh`, `tui`, `main`, `hooks`, `release`

**Examples:**
```
feat(tui): add host search filter
fix(config): handle corrupted JSON gracefully
test(ssh): verify BuildArgs with edge-case ports
chore(deps): update bubbletea to v1.3.10
```

### Commit Template

Git hooks inject this template via `prepare-commit-msg`:

```
# <type>(<scope>): <description>
#
# Types: feat | fix | docs | style | refactor | perf | test | chore | ci
# Scopes: config | ssh | tui | main | hooks | release
#
# Body: explain what and why (not how).
# Footer: Breaking change: <description>
```

### Pull Request Template

```
## Summary
> One-line description of the change.

## Type
- [ ] feat  - [ ] fix  - [ ] docs  - [ ] refactor  - [ ] test  - [ ] chore

## Changes
-

## Testing
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] `staticcheck ./...` passes
- [ ] Manual test: portkey runs, add/edit/delete hosts work

## Breaking Changes
- [ ] None  - [ ] Yes (describe):
```

### Push Template

No separate push template. Pre-push hook enforces quality gates.

---

## Git Hooks

### pre-commit (auto-fix + gate)

Runs on `git commit`. Operates on staged `.go` files only.

| Step | Tool | Action | Blocking |
|------|------|--------|----------|
| 1 | `goimports` | Sort and organize imports | — (auto-fix) |
| 2 | `gofumpt` | Strict formatting | — (auto-fix) |
| 3 | `golines --max-len=120` | Wrap long lines | — (auto-fix) |
| 4 | `go vet ./...` | Compiler-level checks | **Yes** |

After auto-fix, modified files are re-staged to the commit automatically.

### pre-push (analysis gate)

Runs on `git push`. Analyzes the full codebase.

| Step | Tool | Action | Blocking |
|------|------|--------|----------|
| 1 | `go build ./...` | Compilation check | **Yes** |
| 2 | `go test -count=1 -race ./...` | All tests | **Yes** |
| 3 | `staticcheck ./...` | SA/ST/S/Q analysis | **Yes** |
| 4 | `gocritic check ./...` | Opinionated checks | Warn only |
| 5 | `deadcode -test ./...` | Unused code detection | Warn only |
| 6 | `dupl -threshold 80 ./...` | Code duplication | Warn only |

If any **blocking** step fails, the push is rejected. Fix and push again.

### prepare-commit-msg (template injection)

Injects the conventional commit template into the editor when no message is provided.

---

## Semantic Versioning & Release

### Version Calculation

Uses [`svu`](https://github.com/caarlos0/svu) to calculate the next version from git history:

| Current | Latest commit type | Next version |
|---------|-------------------|--------------|
| `v0.1.0` | `feat: ...` | `v0.2.0` |
| `v0.1.0` | `fix: ...` | `v0.1.1` |
| `v0.1.0` | `docs: ...` | `v0.1.0` (no bump) |

### release.sh

```bash
./release.sh [patch|minor|major]
```

- Calculates next version with `svu`.
- Updates `CHANGELOG.md` with entries since last tag.
- Commits and tags with `v<VERSION>`.
- Runs `goreleaser` to build for macOS (amd64, arm64) and Linux (amd64, arm64).

### CHANGELOG.md Format

```markdown
# Changelog

## [0.2.0] - 2026-06-07

### Added
- Host search filter in TUI

### Fixed
- Corrupted JSON handled gracefully

## [0.1.0] - 2026-06-07

### Added
- Initial release: host add, edit, delete, SSH connect
```

### Goreleaser Build Targets

| OS | Arch | Format |
|----|------|--------|
| darwin (macOS) | amd64 | tar.gz |
| darwin (macOS) | arm64 | tar.gz |
| linux | amd64 | tar.gz |
| linux | arm64 | tar.gz |

---

## Release Signing

Every release must be signed with minisign. The in-app updater (`install.go`) is
**fail-closed**: it refuses to install if `checksums.txt.minisig` is missing or
invalid. `install.sh` is **best-effort**: it verifies when `minisign` is present
and warns otherwise.

### One-time keypair generation (maintainer only)

```bash
minisign -G -p portkey.pub -s ~/.minisign/portkey.key
```

- Keep `$HOME/.minisign/portkey.key` **secret** — never commit it.
- The second line of `portkey.pub` (the `RW...` base64 line) is the public key.

### Public key locations (must be kept in sync)

The public key line is embedded in **two** places:

| File | Symbol |
|------|--------|
| `internal/updater/pubkey.go` | `const MinisignPublicKey` |
| `install.sh` | `MINISIGN_PUBKEY="..."` |

`TestInstallScriptPublicKeyMatches` in `internal/updater/install_sh_key_test.go`
enforces that both values are identical. Update both files whenever the keypair
rotates, and keep the test green.

### Signing environment variables

`release.sh` requires these two variables to be set before running:

| Variable | Value |
|----------|-------|
| `MINISIGN_KEY_FILE` | Absolute path to the secret key (e.g. `$HOME/.minisign/portkey.key`) |
| `MINISIGN_PASSWORD` | Password for the secret key |

```bash
export MINISIGN_KEY_FILE="$HOME/.minisign/portkey.key"
export MINISIGN_PASSWORD="your-key-password"
./release.sh patch
```

`release.sh` will fail fast with a clear error if either variable is unset.

### Goreleaser signing config

`.goreleaser.yml` uses the `signs:` block to invoke `minisign -S` on
`checksums.txt`, producing `checksums.txt.minisig`, which is uploaded as a
release asset (`output: true`). The key password is piped via `stdin` so the
release is non-interactive.

---

## Testing Strategy

### Unit Tests

Every package has dedicated tests:

- **`internal/config/config_test.go`** — Host CRUD, Store load/save, edge cases (missing file, corrupt JSON, empty file, directory creation, round-trip).
- **`internal/ssh/ssh_test.go`** — `BuildArgs` for default port, custom port, boundary ports (1, 65535), shell injection prevention.
- **`internal/tui/`** — (Future) model update cycle tests using Bubble Tea's `NewProgram` test mode.

### Test Principles

1. **No filesystem side effects.** Use `t.TempDir()`.
2. **Test boundary values.** Port 1, 22, 65535. Empty strings. Out-of-range indices.
3. **Test error paths.** Missing config file, corrupt JSON, invalid inputs.
4. **Test security.** Verify shell metacharacters are never passed as combined strings.
5. **Table-driven tests** for input/output mappings.

### Running Tests

```bash
go test ./...                    # All tests
go test -count=1 -race ./...     # With race detector (used in pre-push)
go test -v ./internal/config/    # Verbose for one package
```

---

## What NOT to Do

- **Do not** use `os/exec` with a combined shell string (e.g., `exec.Command("sh", "-c", "ssh user@host")`).
- **Do not** store passwords, private keys, or any sensitive authentication material.
- **Do not** add Cobra, Viper, or any CLI framework. Bubble Tea is the framework.
- **Do not** implement SSH password authentication. Assume key-based auth.
- **Do not** write all code in one file. Separate by responsibility.
- **Do not** commit code that fails `go vet`, `staticcheck`, or tests.
- **Do not** add Windows support. macOS and Linux only.
- **Do not** skip writing tests. TDD is mandatory.
- **Do not** leave `// TODO` comments in production code.
- **Do not** introduce new direct dependencies without updating `go.mod` and `AGENTS.md`.

---

## Future Extensibility

These are NOT in scope for the current MVP but the architecture is designed to accommodate them:

- Recent host sorting (last-connected timestamp in Host struct)
- Host groups (tag field + filter)
- Host search (filter bar in host list)
- Connection confirmation dialog (optional prompt before SSH)
- Command presets (run commands on connect, stored per-host)
- Connection log (separate file, append-only)
- CLI subcommands (`portkey add`, `portkey list`, `portkey remove`)
- SSH config import (`~/.ssh/config` parser)
- Tags and labels for hosts

---

## Quick Reference

```bash
# Build
go build -o portkey .

# Run
./portkey

# Test
go test ./...

# Full quality check (what pre-push does)
go vet ./... && go test -count=1 -race ./... && staticcheck ./...

# Release
./release.sh

# Install dev tools
go install golang.org/x/tools/cmd/goimports@latest
go install mvdan.cc/gofumpt@latest
go install github.com/segmentio/golines@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/go-critic/go-critic/cmd/gocritic@latest
go install golang.org/x/tools/cmd/deadcode@latest
go install github.com/mibk/dupl@latest
go install github.com/caarlos0/svu@latest
go install github.com/goreleaser/goreleaser/v2@latest
```
