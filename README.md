# portkey

**Pick a host and jump in.**

Portkey is an interactive SSH host picker written in Go.
It reads your saved hosts from a JSON config and presents a terminal UI
where you can connect, add, edit, or delete hosts — all without leaving
the keyboard.

![portkey](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Android-blue)
![go](https://img.shields.io/badge/Go-1.26-00ADD8)

---

## Install

### One-liner (macOS / Linux / Android Termux)

```bash
curl -fsSL https://raw.githubusercontent.com/yhzion/portkey/main/install.sh | bash
```

Or with `wget`:

```bash
wget -qO- https://raw.githubusercontent.com/yhzion/portkey/main/install.sh | bash
```

> **Termux users:** The installer auto-detects Android/Termux and downloads
> the correct binary. No manual steps needed.

### From source

```bash
go install github.com/yhzion/portkey@latest
```

### Download a binary

Grab the latest archive from the
[Releases](https://github.com/yhzion/portkey/releases) page:

| OS      | Arch  | File                                  |
|---------|-------|---------------------------------------|
| macOS   | amd64 | `portkey_<version>_darwin_amd64.tar.gz`  |
| macOS   | arm64 | `portkey_<version>_darwin_arm64.tar.gz`  |
| Linux   | amd64 | `portkey_<version>_linux_amd64.tar.gz`   |
| Linux   | arm64 | `portkey_<version>_linux_arm64.tar.gz`   |
| Android | arm64 | `portkey_<version>_android_arm64.tar.gz` |

Extract and move the binary into your `$PATH`:

```bash
tar xzf portkey_*_*.tar.gz
mv portkey /usr/local/bin/
```

---

## Upgrade

Run `portkey update` to check for and install the latest release:

```bash
portkey update
```

Flags:

```
--check-only / --dry-run     Report whether an update is available without installing.
                             Exits with code 10 when an update exists (useful in scripts).
--version-target <tag>       Install a specific release (e.g. v1.2.0). Use to pin,
                             downgrade, or reinstall a known version.
--force                      Reinstall the latest release even if already up to date.
--yes / -y                   Skip the interactive confirmation prompt.
                             Piped or CI runs skip it automatically.
```

---

## Usage

Just run:

```bash
portkey
```

### Key Bindings

| Key            | Action              |
|----------------|---------------------|
| `↑` / `k`     | Move selection up   |
| `↓` / `j`     | Move selection down |
| `Enter`/`Space`| Connect to host    |
| `1`–`9`       | Quick-connect       |
| `a`            | Add a new host      |
| `e`            | Edit selected host  |
| `d`            | Delete host         |
| `q` / `Ctrl+C`| Quit                |
| `Esc`          | Cancel / go back    |

### Flags

```
portkey -v       Print version
portkey -h       Print help
```

---

## Configuration

Hosts are stored as JSON at:

```
$XDG_CONFIG_HOME/portkey/hosts.json
```

(falls back to `~/.config/portkey/hosts.json` on most systems).

Portkey assumes key-based SSH authentication (`ssh-copy-id`).
No passwords or private keys are stored.

---

## Verifying releases

Every release ships a `checksums.txt.minisig` alongside `checksums.txt`.
The signature is produced with [minisign](https://jedisct1.github.io/minisign/)
and covers `checksums.txt`, which in turn covers every release archive via
SHA-256.

- **`install.sh`** verifies the signature automatically when `minisign` is
  present on your `PATH`. If `minisign` is absent it prints a warning and falls
  back to checksum-only verification.
- **`portkey update`** (the in-app updater) always verifies the signature before
  installing. It is fail-closed: an update is refused if `checksums.txt.minisig`
  is missing or invalid.

To verify a release manually:

```bash
minisign -V -P <public-key> -m checksums.txt -x checksums.txt.minisig
```

The current public key is embedded in `install.sh` (`MINISIGN_PUBKEY`) and in
`internal/updater/pubkey.go` (`MinisignPublicKey`).

---

## Uninstall

```bash
rm "$(which portkey)"
rm -rf "${XDG_CONFIG_HOME:-$HOME/.config}/portkey"
```

---

## License

[MIT](LICENSE)
