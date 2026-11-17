<p align="center"><img width="200px" src="/_docs/img/logo.png" alt="ctop"/></p>

# ctop

Top-like interface for container metrics

<p align="center"><img src="_docs/img/grid.gif" alt="ctop"/></p>

---

## Project Overview

`ctop` is a terminal-based monitoring tool for containers. It displays real-time metrics (CPU, memory, network, I/O) for multiple containers in a compact overview as well as a [single container view](_docs/single.md) for detailed inspection.

Supported container runtimes: **Docker** and **runC**.

### Fork Information

This is a maintained fork of [bcicen/ctop](https://github.com/bcicen/ctop), originally created by [VektorLab](https://github.com/vektorlab).

`ctop` is an indispensable tool in my daily work with containers — compact, fast and to the point. Unfortunately, the original project is no longer actively maintained. Since I use this tool on a regular basis and don't want to let it go, I maintain it here as an independent fork: with up-to-date dependencies, security fixes and modern Go.

Changes in this fork:
- Security fixes (shell injection, file permissions, debug server binding, secret redaction in UI and debug logs)
- Supply-chain hardening (SHA-pinned CI actions, Dependabot, digest-pinned Docker base image)
- Updated to Go 1.22 with modern stdlib packages (`log/slog`, `os.ReadDir`)
- Replaced unmaintained dependencies (`op/go-logging`, `nu7hatch/gouuid`, `pkg/errors`)
- GitHub Actions CI/CD with multi-platform builds (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
- Docker images published to GHCR (`ghcr.io/eqms/ctop`)

## Installation

### Install or update with `install.sh` (Linux & macOS)

Downloads the latest release for the system and architecture (amd64, arm64, armv7), verifies its SHA256 checksum and replaces an existing `ctop` in place (otherwise `/usr/local/bin`). `sudo` is used only when the target directory is not writable. Works with `curl` or `wget` and POSIX `sh` (Debian/Ubuntu, Fedora/RHEL, Alpine, Arch, Raspberry Pi OS, macOS).

```bash
curl -fsSL https://raw.githubusercontent.com/maxeo/ctop/master/install.sh | sh
```

Options (after `sh -s --`):

```bash
# specific version, into a user directory without sudo
curl -fsSL https://raw.githubusercontent.com/maxeo/ctop/master/install.sh | sh -s -- --version 0.8.9 --dir ~/.local/bin

# without curl
wget -qO- https://raw.githubusercontent.com/maxeo/ctop/master/install.sh | sh

# uninstall
curl -fsSL https://raw.githubusercontent.com/maxeo/ctop/master/install.sh | sh -s -- --uninstall
```

| Option | Meaning |
|--------|---------|
| `-v`, `--version X` | install version X (default: latest release) |
| `-d`, `--dir DIR` | target directory (default: where `ctop` already is, otherwise `/usr/local/bin`) |
| `-a`, `--arch ARCH` | `amd64`, `arm64` or `arm` (default: from `uname -m`) |
| `-r`, `--repo O/N` | GitHub repository (default: `maxeo/ctop`) |
| `--no-verify` | skip the SHA256 check |
| `--uninstall` | remove `ctop` |

> If `ctop` lives in `/usr/bin` and comes from a package manager (apt, snap, brew), uninstall it there instead: the next package update would overwrite the file again.

### Download binaries manually

Get the latest binaries from the [Releases page](https://github.com/maxeo/ctop/releases):

| Platform | Architecture | File |
|----------|-------------|------|
| Linux | amd64 | `ctop-<version>-linux-amd64` |
| Linux | arm64 | `ctop-<version>-linux-arm64` |
| Linux | armv7 | `ctop-<version>-linux-arm` |
| macOS | Intel | `ctop-<version>-darwin-amd64` |
| macOS | Apple Silicon | `ctop-<version>-darwin-arm64` |
| Windows | amd64 | `ctop-<version>-windows-amd64.exe` |

> **macOS Note**: When downloading via a browser, macOS sets the quarantine flag and Gatekeeper blocks execution. Fix:
> ```bash
> # Remove quarantine flag
> xattr -d com.apple.quarantine ./ctop-0.8.9-darwin-arm64
> chmod +x ./ctop-0.8.9-darwin-arm64
> ```
> When installing via `install.sh` (above), no quarantine flag is set.

### Homebrew (macOS & Linux)

```bash
brew install eqms/ctop/ctop
```

### Docker

```bash
docker run --rm -ti \
  --name=ctop \
  --volume /var/run/docker.sock:/var/run/docker.sock:ro \
  ghcr.io/eqms/ctop:latest
```

## Usage

`ctop` requires no arguments and uses Docker host variables by default. See [connectors](_docs/connectors.md) for further configuration options.

### Options

| Option | Description |
|--------|------------|
| `-a` | Show active containers only |
| `-f <string>` | Set an initial filter string |
| `-h` | Display help dialog |
| `-i` | Invert default colors |
| `-r` | Reverse container sort order |
| `-s` | Select initial container sort field |
| `-v` | Output version information and exit |
| `--connector` | Select container connector (`docker`, `runc`) |
| `--shell` | Shell for container exec (default: `/bin/sh`, also via `CTOP_SHELL` env) |

### Keybindings

| Key | Action |
|:---:|--------|
| <kbd>Enter</kbd> | Open container menu |
| <kbd>a</kbd> | Toggle display of all containers |
| <kbd>f</kbd> | Filter displayed containers (<kbd>Esc</kbd> to clear) |
| <kbd>H</kbd> | Toggle ctop header |
| <kbd>h</kbd> | Open help dialog |
| <kbd>s</kbd> | Select container sort field |
| <kbd>r</kbd> | Reverse container sort order |
| <kbd>o</kbd> | Open single view |
| <kbd>l</kbd> | View container logs (<kbd>t</kbd> to toggle timestamp) |
| <kbd>e</kbd> | Exec shell |
| <kbd>c</kbd> | Configure columns |
| <kbd>S</kbd> | Save current configuration to file |
| <kbd>q</kbd> | Quit ctop |

#### Log view

| Key | Action |
|:---:|--------|
| Mouse wheel, <kbd>↑</kbd> <kbd>↓</kbd> | Scroll (older lines are loaded at the top; past the newest line, blank rows make room for new lines, keeping 20 lines on screen) |
| <kbd>PgUp</kbd> <kbd>PgDn</kbd> | Previous / next screen |
| <kbd>Home</kbd> | Start of the log (later pages load as you scroll down) |
| <kbd>End</kbd> | Newest lines, following new output |
| <kbd>Ctrl</kbd>+<kbd>F</kbd> or <kbd>/</kbd> | Search: loaded lines are searched while you type, <kbd>Enter</kbd> keeps the search (and searches the whole log if no loaded line matches), <kbd>n</kbd> older match, <kbd>N</kbd> newer match, <kbd>Esc</kbd> to end |
| Click or <kbd>s</kbd> | Text selection mode: the mouse is left to the terminal, so click and drag selects text (<kbd>s</kbd> or <kbd>Esc</kbd> to leave) |
| <kbd>t</kbd> | Toggle timestamps |
| <kbd>Esc</kbd> | End the search or the selection mode, otherwise close the log view |
| <kbd>q</kbd>, <kbd>←</kbd> <kbd>→</kbd>, other keys | Close the log view |

### Config File

While running, press <kbd>S</kbd> to save the current configuration:
- XDG systems: `~/.config/ctop/config`
- Fallback: `~/.ctop`

## Building from Source

```bash
git clone https://github.com/maxeo/ctop.git
cd ctop
make build
```

### All Platforms

```bash
make build-all    # Binaries in _build/
```

### Docker Image

```bash
make image        # Builds local ctop:latest image
```

## Architecture

```
ctop/
├── main.go                  # Entry point, CLI flags
├── grid.go                  # Main UI grid controller
├── cursor.go                # Grid cursor / navigation
├── menus.go                 # TUI menus (container, filter, sort)
├── config/                  # Configuration management (TOML)
├── connector/               # Container runtime abstractions
│   ├── docker.go            # Docker connector
│   ├── runc.go              # runC connector (Linux only)
│   ├── mock.go              # Mock connector (development)
│   ├── collector/           # Metrics collectors
│   └── manager/             # Container lifecycle management
├── container/               # Container model and sorting
├── cwidgets/                # TUI widgets
│   ├── compact/             # Compact grid view
│   └── single/              # Single view (CPU, Mem, Net, I/O)
├── logging/                 # Logging (log/slog + ring buffer)
├── widgets/                 # Base UI widgets
├── Makefile                 # Build targets
├── Dockerfile               # Multi-stage Docker build
└── .github/workflows/ci.yml # GitHub Actions CI/CD
```

## Development

**Prerequisites**: Go 1.26+

```bash
# Development mode with debug server
make run-dev

# Debug options (environment variables)
CTOP_DEBUG=1          # Enable debug logging and Unix socket server
                      # Socket: $XDG_RUNTIME_DIR/ctop.sock or $TMPDIR/ctop-<uid>/ctop.sock
CTOP_DEBUG_TCP=1      # Use TCP debug server (127.0.0.1:9000, unauthenticated) instead of Unix socket
CTOP_DEBUG_FILE=/path # Additionally write log output to file
```

### Connectors

| Connector | Environment Variable | Default |
|-----------|---------------------|---------|
| Docker | `DOCKER_HOST` | `unix://var/run/docker.sock` |
| runC | `RUNC_ROOT` | `/run/runc` (must be under `/run` or `/var/run`) |
| runC | `RUNC_SYSTEMD_CGROUP` | (disabled) |

## CI/CD

GitHub Actions pipeline with 4 jobs:

1. **test** — `go test ./...`
2. **build** — Cross-compile for 5 platforms
3. **release** — GitHub Release with binaries + SHA256 checksums (on tag push `v*`)
4. **docker** — Multi-arch Docker image to GHCR (on tag push `v*`)

## License

MIT License — Copyright (c) 2017 VektorLab. See [LICENSE](LICENSE).
