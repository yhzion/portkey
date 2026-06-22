# Changelog

## [0.4.2] - 2026-06-22

### Fixed
- redirect dead loopback DNS instead of fallback-on-error (#80) — fixes
  `portkey update` on Android/Termux, which the v0.4.1 attempt did not (a UDP
  dial to the dead `[::1]:53` default succeeds, so the fallback never ran).
  Now redirects to the device's own resolvers (`net.dnsN`), then public DNS.
  **Verified working on Termux (android/arm64).**
- install.sh auto-installs `minisign` via `pkg` on Termux so releases are
  signature-verified there instead of silently trusted (#80)


## [0.4.1] - 2026-06-22

### Fixed
- fall back to public DNS on Android/Termux (#79)


## [0.4.0] - 2026-06-22

### Added
- minisign release signature verification (#53) (#77)
- portkey update UX — dev guard, --check-only/--version-target/--force, progress, --yes (#64, #67, #53) (#74)
- update-check robustness — classify errors, context-cancel, cache (#65, #66, #63) (#73)
- track and highlight matched field in fuzzy search (#62)

### Fixed
- install updates on confirm — wire raw client Installer (#41) (#76)
- require confirmation before connecting on suffix host match (#46) (#72)
- memoize fuzzy matcher to avoid exponential blowup on repeated chars (#42) (#71)
- snapshot config before async save to fix race and add rollback (#44) (#70)
- atomic self-replace + tar extraction guards (#51, #52) (#69)
- harden updater/install against MITM and download failures (#60)
- type 'q' into query during active search instead of quitting (#58)
- stamp LastUsed on successful connect (#56)
- semver pre-release precedence + harden CheckLatest (#48, #50) (#55)
- write hosts config atomically
- use literal space in sed for macOS compatibility


## [0.3.13] - 2026-06-08

### Fixed
- use literal space in sed for macOS compatibility


## [0.3.12] - 2026-06-08


## [0.3.11] - 2026-06-08

### Fixed
- mock GitHub API in TestDispatchUpdateAlreadyUpToDate


## [0.3.10] - 2026-06-08

### Added
- add update subcommand for self-update


## [0.3.9] - 2026-06-08


## [0.3.8] - 2026-06-08

### Added
- add download, verify, and install logic for self-update

### Fixed
- pick host and exit, run ssh from main after TUI quits
- exit after SSH session and preserve terminal state


## [0.3.7] - 2026-06-08

### Fixed
- correct palette for WCAG 2.1 AA text readability


## [0.3.6] - 2026-06-08


## [0.3.5] - 2026-06-08

### Fixed
- add android/arm64 build target for Termux compatibility (#14)
- add android/arm64 build target for Termux compatibility


## [0.3.4] - 2026-06-08

### Fixed
- forward huh-internal messages for form field navigation
- enable PIE build mode for Android/Termux compatibility


## [0.3.3] - 2026-06-07

### Fixed
- use gh auth token when GITHUB_TOKEN is not set


## [0.3.2] - 2026-06-07


## [0.3.1] - 2026-06-07

### Fixed
- detect Termux and install to $PREFIX/bin


## [0.3.0] - 2026-06-07

### Added
- add CLI subcommands for non-interactive usage
- sort host list by last used (default ordering)
- add fuzzy search/filter in host list


## [0.2.1] - 2026-06-07

### Added
- MIT LICENSE file
- Improved update notification wording with ✨ emoji and gold accent color

### Fixed
- Goreleaser archives now include LICENSE and README.md

## [0.2.0] - 2026-06-07

### Added
- One-line install script (`install.sh`) with checksum verification
- Auto-detects OS/arch, installs to `/usr/local/bin` or `~/.local/bin`
- Auto-patches shell rc file (`.bashrc`/`.zshrc`/`.profile`) when `~/.local/bin` not in PATH
- Install script uploaded as release asset via goreleaser

## [0.1.1] - 2026-06-07

### Added
- README with install instructions and usage docs

### Fixed
- Goreleaser config updated to v2 schema (was failing on missing README.md)

## [0.1.0] - 2026-06-07

### Added
- Initial release: host add, edit, delete, SSH connect
