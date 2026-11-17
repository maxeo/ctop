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
- GitHub Actions CI/CD with multi-platform builds (Linux amd64/arm64/arm, macOS amd64/arm64, Windows amd64)
- Docker images published to GHCR (`ghcr.io/maxeo/ctop`)

## Installation

### Install or update with `install.sh` (Linux & macOS)

Downloads the latest release for the system and architecture (amd64, arm64, armv6/armv7), verifies its SHA256 checksum (it stops if the checksum cannot be checked, unless `--no-verify`) and replaces an existing `ctop` in place (otherwise `/usr/local/bin`). `sudo` is used only when the target directory is not writable. Works with `curl` or `wget` and POSIX `sh` (Debian/Ubuntu, Fedora/RHEL, Alpine, Arch, Raspberry Pi OS, macOS).

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

### Install or update with `install.ps1` (Windows)

The same for Windows, in PowerShell (Windows PowerShell 5.1 or PowerShell 7), without administrator rights: downloads `ctop-<version>-windows-amd64.exe` (Windows on ARM runs it through emulation), verifies its SHA256 checksum, installs it as `ctop.exe` where it already is (otherwise `%LOCALAPPDATA%\Programs\ctop`) and adds that folder to the user's `PATH`. A running `ctop.exe` is replaced too.

```powershell
irm https://raw.githubusercontent.com/maxeo/ctop/master/install.ps1 | iex
```

Options:

```powershell
# specific version, into another folder
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/maxeo/ctop/master/install.ps1))) -Version 0.8.12 -Dir C:	ools

# uninstall (also removes the folder from the user's PATH)
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/maxeo/ctop/master/install.ps1))) -Uninstall
```

| Option | Environment variable | Meaning |
|--------|----------------------|---------|
| `-Version X` | `CTOP_VERSION` | install version X (default: latest release) |
| `-Dir DIR` | `CTOP_DIR` | target folder (default: where `ctop.exe` already is, otherwise `%LOCALAPPDATA%\Programs\ctop`) |
| `-Repo O/N` | `CTOP_REPO` | GitHub repository (default: `maxeo/ctop`) |
| `-NoVerify` | `CTOP_NO_VERIFY=1` | skip the SHA256 check |
| `-NoPath` | `CTOP_NO_PATH=1` | do not add the folder to the user's `PATH` |
| `-Uninstall` | `CTOP_UNINSTALL=1` | remove `ctop.exe` |

> `curl ... | sh` does not work in PowerShell, which has no `sh`: use `install.ps1` there.

### Download binaries manually

Get the latest binaries from the [Releases page](https://github.com/maxeo/ctop/releases):

| Platform | Architecture | File |
|----------|-------------|------|
| Linux | amd64 | `ctop-<version>-linux-amd64` |
| Linux | arm64 | `ctop-<version>-linux-arm64` |
| Linux | armv6 / armv7 (Raspberry Pi) | `ctop-<version>-linux-arm` |
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

There is no Homebrew tap for this fork yet: `brew install eqms/ctop/ctop` installs the upstream `eqms/ctop` build, not this one. Use `install.sh` (above) instead, also on macOS.

### Docker

```bash
docker run --rm -ti \
  --name=ctop \
  --volume /var/run/docker.sock:/var/run/docker.sock:ro \
  ghcr.io/maxeo/ctop:latest
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
| <kbd>h</kbd> | Show / hide every key below the container list |
| <kbd>s</kbd> | Select container sort field below the list (the list is sorted while you move, <kbd>Enter</kbd> keeps it, <kbd>Esc</kbd> goes back) |
| <kbd>r</kbd> | Reverse container sort order |
| <kbd>o</kbd> | Open single view |
| <kbd>l</kbd> | View container logs (<kbd>t</kbd> to show Docker's timestamp) |
| <kbd>e</kbd> | Exec shell |
| <kbd>w</kbd> | Open the lowest published TCP port in a browser (over SSH, the URL is shown instead) |
| <kbd>c</kbd> | Configure columns below the list (changes show at once) |
| <kbd>S</kbd> | Save current configuration to file |
| <kbd>q</kbd>, <kbd>Esc</kbd>, <kbd>Ctrl</kbd>+<kbd>C</kbd> | Quit ctop. When something is open (full key help, sort or columns panel, filter, menu, log view, single view), <kbd>Esc</kbd> and <kbd>Ctrl</kbd>+<kbd>C</kbd> close it first, one level at a time |

#### Log view

| Key | Action |
|:---:|--------|
| Mouse wheel, <kbd>↑</kbd> <kbd>↓</kbd> | Scroll (older lines are loaded at the top; past the newest line, blank rows make room for new lines, keeping 20 lines on screen) |
| <kbd>PgUp</kbd> <kbd>PgDn</kbd> | Previous / next screen |
| <kbd>Home</kbd> | Start of the log (later pages load as you scroll down) |
| <kbd>End</kbd> | Newest lines, following new output (shown as if scrolled past the end: the last 20 lines on top, room for new ones below) |
| <kbd>Ctrl</kbd>+<kbd>F</kbd> or <kbd>/</kbd> | Search, ignoring case: loaded lines are searched while you type, <kbd>Enter</kbd> keeps the search (and searches the whole log if no loaded line matches), <kbd>n</kbd> older match, <kbd>N</kbd> newer match, <kbd>Enter</kbd> next match in the direction you last scrolled, <kbd>Esc</kbd> to end. <kbd>Tab</kbd> (in the prompt or while a search is shown) changes the mode: Base (the text as typed, anywhere), All words (words separated by spaces, all of them, in any order, also inside longer words: `warn power main` finds `WARNING (MainThread) ... power state`), Regex (Go syntax). The arrows, <kbd>PgUp</kbd> <kbd>PgDn</kbd>, <kbd>Home</kbd>, <kbd>End</kbd> and the wheel scroll the log while you type |
| Click, <kbd>s</kbd> or <kbd>Ctrl</kbd>+<kbd>S</kbd> | Text selection mode: the mouse is left to the terminal, so click and drag selects text (<kbd>s</kbd>, <kbd>Ctrl</kbd>+<kbd>S</kbd> or <kbd>Esc</kbd> to leave; <kbd>Ctrl</kbd>+<kbd>S</kbd> also works while typing a search). The search keys work here too and a click while typing a search keeps the prompt open |
| <kbd>t</kbd> | Toggle timestamps |
| <kbd>Esc</kbd>, <kbd>Ctrl</kbd>+<kbd>C</kbd> | End the search or the selection mode, otherwise close the log view |
| <kbd>q</kbd>, <kbd>←</kbd> <kbd>→</kbd>, other keys | Close the log view |

#### Single view

The single view fits the terminal: a header with the container details (in up to three columns), the CPU, MEM, NET and IO panels (side by side from 110 columns, one above the other on smaller terminals) and the environment variables in the rest of the screen. CPU shows the current, average and highest usage of the period shown; MEM a bar with the percentage of the limit and the usage history; NET and IO the current and highest rate with their history. The key help is at the bottom.

<kbd>Enter</kbd> selects a panel (its border is colored) and the arrows move the selection; <kbd>Enter</kbd> again opens the panel's detail view: the chart in large and everything known about it, in a list that scrolls and can be filtered:

| Panel | Details |
|-------|---------|
| Header | name, id, image, state, exit code, restart policy, command, entrypoint, working dir, user, healthcheck and its last result, security options, logging, labels |
| CPU | usage now and over the last minute, 5 and 15 minutes (average, lowest, highest), host CPUs, CPU limit, CPU set, shares, pids limit |
| MEM | usage now and over the last periods, limit, reservation, swap |
| NET | rates and totals, network mode, IP, gateway, MAC, IPv6 and aliases of each network, ports, DNS |
| IO | rates and totals, mounts with source, type (bind or volume) and read-only / read-write |
| Env | every variable, long values wrapped |

| Key | Nothing selected | Panel selected | Detail view |
|:---:|------------------|----------------|-------------|
| <kbd>Enter</kbd> | Select the header panel | Open its detail view | |
| <kbd>←</kbd> <kbd>→</kbd> | Back to the container list | Select the panel on the left / right | Previous / next panel's details |
| <kbd>↑</kbd> <kbd>↓</kbd> | Scroll the environment | Select the panel above / below | Scroll |
| <kbd>PgUp</kbd> <kbd>PgDn</kbd> | Previous / next container | Previous / next container | Scroll a page |
| <kbd>Home</kbd> <kbd>End</kbd> | Top / end of the environment | | Top / end |
| <kbd>/</kbd> or <kbd>Ctrl</kbd>+<kbd>F</kbd> | Filter the environment | | Filter the list (rows, or whole sections by title) |
| <kbd>Esc</kbd>, <kbd>Ctrl</kbd>+<kbd>C</kbd> | Back to the container list | Unselect | Back to the panels |
| <kbd>l</kbd> / <kbd>m</kbd> / <kbd>e</kbd> | Logs / container menu / shell | same | same |
| <kbd>q</kbd> | Back to the container list | same | same |

In a filter, <kbd>Enter</kbd> keeps it and <kbd>Esc</kbd> clears it. Details come from `docker inspect` (the runC connector shows the header fields only).

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
│   └── single/              # Single view (header, CPU, Mem, Net, I/O, Env)
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
