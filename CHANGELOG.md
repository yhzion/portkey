# Changelog

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
