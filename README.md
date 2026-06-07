# portkey

**Pick a host and jump in.**

Portkey is an interactive SSH host picker written in Go.
It reads your saved hosts from a JSON config and presents a terminal UI
where you can connect, add, edit, or delete hosts — all without leaving
the keyboard.

![portkey](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-blue)
![go](https://img.shields.io/badge/Go-1.26-00ADD8)

---

## Install

### One-liner (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/yhzion/portkey/main/install.sh | bash
```

Or with `wget`:

```bash
wget -qO- https://raw.githubusercontent.com/yhzion/portkey/main/install.sh | bash
```

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

Extract and move the binary into your `$PATH`:

```bash
tar xzf portkey_*_*.tar.gz
mv portkey /usr/local/bin/
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

## Uninstall

```bash
rm "$(which portkey)"
rm -rf "${XDG_CONFIG_HOME:-$HOME/.config}/portkey"
```

---

## License

[MIT](LICENSE)
