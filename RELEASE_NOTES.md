# Release Notes

## v0.8.14 (2026-11-17)

- **Ctrl+C works like Esc** in every view: it goes back one level (closes the search, the selection, the detail view, the log or single view, the full key help) and quits ctop from the container list. It used to quit ctop from any view
- **Mouse wheel on Windows**: the Windows console does not turn the wheel into arrow keys like xterm-like terminals do, so on Windows ctop reads the mouse and the wheel works like up / down in the container list, the panels and menus, and the single view (scrolls the environment or the detail view, moves the panel selection). Text is selected with Shift + drag while ctop runs. Linux and macOS are unchanged

## v0.8.13 (2026-11-17)

### Single view
- **New layout that fits the terminal**: a header with the container details in up to three columns and the state in color (health shows N/A when the container has no healthcheck); CPU and MEM, then NET and IO, side by side from 110 columns, one above the other on smaller terminals; the panels get their preferred height and the environment variables take the rest of the screen
- **CPU** shows the current, average and highest usage of the period shown, on a round scale (5, 10, 25, 50, 100%) so that a low usage does not fill the chart
- **MEM** shows the used memory and the limit, a bar with the percentage (green, yellow from 70%, red from 90%) and the usage history
- **NET and IO** show the current and highest rate of each direction with its history
- **Charts fill the whole width** with the newest value on the right edge: NET and IO kept 60 values drawn from the left, so on a wide terminal they stopped halfway
- **Environment variables** are sorted by name (VAR_2 before VAR_10), scroll with up / down and Home / End, and can be filtered with `/` or Ctrl+F
- **Panel selection and detail views**: Enter selects a panel (colored border), the arrows move the selection and Enter opens the panel's detail view: the chart in large and a list that scrolls and can be filtered. Header: command, entrypoint, restart policy, exit code, healthcheck and its last result, security options, logging, labels. CPU and MEM: usage over the last minute, 5 and 15 minutes, and the limits. NET: IP, gateway, MAC and aliases of each network, ports, DNS, totals. IO: mounts with source, type and mode, totals. Env: every variable, long values wrapped. Left and right show the previous or next panel's details, Esc goes back one level
- **Navigation**: PgUp / PgDn show the previous or next container; with nothing selected, left and right go back to the list and up / down scroll the environment; `l` opens the logs, `m` the container menu, `e` a shell, Esc or `q` go back to the list. Other keys no longer close the view
- **Host network**: IPs shows `host` instead of `host:`, and NET says that the traffic is not counted per container
- **Uptime** is updated every second instead of showing the value from when the container was last inspected
- **Key help** at the bottom, like in the log view

### Log view
- **Base search finds the text as typed**, spaces included
- **The Whole word mode is now All words**: words separated by spaces find the lines that contain all of them, in any order and also inside longer words (`warn power main` finds `WARNING (MainThread) ... power state`), each one highlighted

### Container list
- **Stopped and paused containers** are marked with `■` and `‖` instead of `⏹` and `⏸`, which some fonts (e.g. MobaXterm's) draw as tall emoji-like boxes

### Install
- **`install.ps1` for Windows**: `irm https://raw.githubusercontent.com/maxeo/ctop/master/install.ps1 | iex` installs or updates `ctop.exe` in PowerShell, without administrator rights: it verifies the SHA256 checksum, installs into `%LOCALAPPDATA%\Programs\ctop` (or where `ctop.exe` already is), adds the folder to the user's PATH and replaces a running `ctop.exe`. Options for version, folder, uninstall

### Other
- The Go module path is now `github.com/maxeo/ctop`

## v0.8.12 (2026-11-17)

### Container list
- **Esc quits ctop** from the container list, like `q`. When something is open (full key help, sort or columns panel, filter, menu, log view), Esc closes it first, one level at a time. ctop now runs in termbox's Esc input mode everywhere: the Alt mode never reported a lone Esc
- **Ctrl+C quits ctop from any view** and restores the terminal; in the log view's selection mode it works like Esc, since it is often a copy shortcut there
- **Home / End sent as `\e[H`, `\e[1~` and similar** (PuTTY, MobaXterm) are ignored on the list: the `H` of `\e[H` used to hide the header

### Log view
- **Large logs use little memory**: the view keeps at most 50,000 lines and drops the ones furthest from the screen, loading them again from Docker when you scroll back to them. Searching the whole log no longer loads it into memory: the log is read as a stream and only the lines around the match are kept (with 200,000 lines, about 21 MB instead of about 105 MB)
- **Search in selection mode**: `Ctrl+F` (or `/`), `n` and `N` also work in text selection mode, and a click while typing a search switches to selection without closing the prompt
- **Search for several words**: in the Base and Whole word modes, words separated by spaces find the lines that contain all of them, in any order, and each word is highlighted (`warn power main` finds `WARNING (MainThread) ... power state`)
- **Search modes**: `Tab` (in the search prompt, or while a search is shown) switches between Base (the text anywhere), Whole word (letters with accents count as letters) and Regex (Go syntax; while it is not a valid expression, e.g. half typed, its error is shown). All modes ignore case
- **Enter goes to the next match in the direction you last scrolled**: up (or `n`, `End`) finds the previous match, down (or `N`, `Home`) the next one
- **Ctrl+S** switches the text selection mode on and off like `s`, also while typing a search
- **Key help on three rows** when the keys of the current mode do not fit in two
- **Scroll while typing a search**: the arrows, `Home`, `End` and the mouse wheel move the log while the search prompt is open (they were ignored), the next letter searches from the new position, and the key help lists the keys of the prompt

### Fixes
- **Long lines with accents or wide characters**: lines were wrapped by bytes, so a row could end in the middle of a character (`è`, `日`) and show broken symbols. They are now wrapped by terminal cells, and a wide character that does not fit goes to the next row
- **Lines with the same timestamp**: loading older or newer pages could skip or repeat lines that share their timestamp with the first or last loaded line
- **Loading older lines after a search or the start of the log** read from the end of the log and could find nothing

### Install
- **`install.sh` stops if the checksum cannot be checked** (file missing from `sha256sums.txt`, or no `sha256sum`/`shasum`), unless `--no-verify` is given
- **ARMv6** (Raspberry Pi Zero / 1): the `linux-arm` binary is now built for ARMv6, so it runs on both ARMv6 and ARMv7
- **README and Homebrew formula** pointed to the upstream `eqms/ctop` (Docker image and release downloads); they now point to `maxeo/ctop`. There is no Homebrew tap for this fork yet

## v0.8.11 (2026-11-17)

### Container list
- **Key help** on the last two rows, like nano (Enter, l, o, e, f, s, a, c, h, q)
- **Help, sort field and columns stay on the list page**: `h` shows every key below the list instead of opening a separate page; `s` and `c` open a panel below the list, which keeps updating. The list is sorted while you move through the sort fields (`Enter` keeps the field, `Esc` goes back, `r` reverses), and shown, hidden or moved columns appear at once

### Fixes
- **Filter (`f`)**: Esc and Ctrl+C did not close it (Esc is never reported in termbox's Alt input mode), so ctop looked frozen; Backspace sent as `^H` was ignored; the input box now sits above the key help. The input no longer feeds a channel from the event loop
- **Commit as image**: same Esc / Ctrl+C / Backspace fixes
- **Open in browser (`w`)**: the browser's own error output (e.g. Chromium's) was written over the container list; the port opened changed at random between refreshes and could be an IPv6 or UDP binding; over SSH, where no local browser can open, the URL is now shown in the status line, with the address of the SSH server instead of `localhost`; a container with no published port shows a message instead of doing nothing
- **Status messages no longer take the next key**: messages (open in browser, config saved, errors) were shown in a dialog that the next key only closed, so that key was lost (e.g. `f` did not open the filter, and the letters typed next ran list commands). They now show for 10 seconds above the key help while the list keeps working
- **Sort by uptime** compared the text, so "3 weeks" came before "25 weeks" and "2 hours"; it now compares the duration
- **Columns panel**: a piece of the panel border could stay near the top of the list
- **Raw Docker timestamps inside log lines**: the 8-byte frame headers Docker adds to the logs of containers without a TTY were left in the text, and a header byte equal to a space or newline (a 32 or 10 byte line) put Docker's timestamp inside the message. The headers are now removed before splitting lines

### Single view
- **Full terminal width**: panels are no longer limited to 65 columns, so long values (image, env vars) are not cut and the memory bars spread over the width

### Log view
- **End** shows the newest lines as if scrolled past the end: the last 20 lines on top, room for new ones below
- **Timestamps** at the start of log lines are shown in cyan (ISO 8601 with space or `T`, optional fraction, time zone and square brackets), including ctop's own with `t`
- **`t` timestamp in a compact format**: `2026-11-17 08:49:09.250` in local time, instead of `2026-11-17T08:49:09.25Z`

## v0.8.10 (2026-11-17)

### Log view
- **Home shows the start of the log**: it reads the first 200 lines instead of loading the whole log; scrolling down loads the next pages until the view joins the newest lines and follows new output again. End goes back to the newest lines
- **Search** with Ctrl+F or `/`: loaded lines are searched while you type, matches are highlighted, `n` / `N` move to the older / newer match, loading more of the log when no loaded line matches
- **tail -f layout**: log lines use the whole width with no border, with a title bar on top and two rows of key help for the current mode at the bottom, like nano
- **Scroll past the newest line**: blank rows below it make room for new lines, keeping at least 20 lines on screen
- **Text selection mode**: a click or `s` leaves the mouse to the terminal, so click and drag selects and copies log lines; Esc or `s` goes back to the wheel scrolling the log
- **Left / right arrows** close the log view, also in selection mode

### Fixes
- Esc did nothing in the log view: termbox's Alt input mode never reports a lone Esc
- Backspace (0x7F) was ignored in the search prompt
- Changing the mouse mode could race with a redraw and lose its escape sequence

## v0.8.9 (2026-11-17)

### Security
- **runc 1.4.4** (CVE-2026-41579 / GO-2026-5761), plus `klauspost/compress` 1.20.2 and `moby/go-archive` 0.3.0 (GO-2026-6253); `govulncheck` reports no vulnerabilities

### Log view
- **Mouse wheel, arrows, PgUp/PgDn** scroll the log; at the bottom the view follows new lines again
- **Paged history**: opens with the last 20 lines, 200 older lines are loaded each time you reach the top
- **Home / End**: Home loads the whole log and jumps to the start, End jumps back to the newest line; also with sequences termbox does not decode (PuTTY, MobaXterm, tmux)
- **ANSI color codes** are stripped from log lines

### Fixes
- Log view froze permanently when scrolling (Ctrl+C included)
- Several fast keys in the log view quit ctop
- Quadratic layout blocked the log view on logs with thousands of lines
- Flicker and lag: `ui.TermWidth/TermHeight` forced a full repaint on every tick
- Containers recreated by `docker compose` kept the temporary `<id>_<name>` name (`rename` event now handled)
- Goroutine leak in the Docker log collector

## v0.8.8 (2026-07-03)

### Security
- **Env redaction for debug dump** (`redact/`, `debug.go`): new shared `redact` package — the container state dump (key `D`) now masks credential-like env variables as `[REDACTED]` too; previously they were logged unredacted to the log ring buffer, the debug file (`CTOP_DEBUG_FILE`) and the debug socket. The pattern was extended with `AUTH`, `DSN`, `CERT`, `COOKIE`, `JWT` and additionally detects credentials embedded in values (connection strings like `postgres://user:pass@host/db`) regardless of the variable name
- **CI supply-chain hardening** (`.github/workflows/ci.yml`): all GitHub Actions pinned to full commit SHAs (with version comments), `govulncheck` pinned to v1.5.0 instead of `@latest`, `timeout-minutes` on all six jobs
- **Dependabot** (`.github/dependabot.yml`): weekly automated updates for Go modules, GitHub Actions (incl. SHA pins) and the Docker base image
- **Docker hardening**: base image `golang:1.26-alpine` pinned by SHA256 digest; root-by-design for `docker.sock` access documented in the Dockerfile; new `.dockerignore` slims down the build context

### Internal
- `cwidgets/single/env.go` now uses the shared `redact` package (duplicate logic removed); new unit tests in `redact/redact_test.go`
- `.gitignore` extended with `*.sock`, `.env`, `*.pem`, `*.key`

---

## v0.8.7 (2026-06-11)

### Security
- **runc upgrade v1.4.0 → v1.4.2**: Patches three more high-severity container-escape vulnerabilities (GO-2025-4096, GO-2025-4097, GO-2025-4098), published November 2025 and only fixed from runc v1.4.1 onwards
- **CI hardening** (`.github/workflows/ci.yml`): fixed shell injection via the `workflow_dispatch` tag input (passed through `env:` instead of direct `${{ }}` interpolation); workflow permissions scoped per job (top level reduced to `contents: read`); `govulncheck` now blocks the build instead of only warning
- **Debug socket secured** (`logging/server.go`): the Unix socket now lives in a private runtime directory (`$XDG_RUNTIME_DIR/ctop.sock` or `$TMPDIR/ctop-<uid>/`, directory 0700, socket 0600) instead of the working directory; stale sockets are removed on startup
- **Env variable redaction** (`cwidgets/single/env.go`): values of credential-like variables (`SECRET`, `PASSWORD`, `TOKEN`, `API_KEY`, …) are masked as `[REDACTED]` in the single-container view
- **RUNC_ROOT validation** (`connector/runc.go`): paths outside `/run` and `/var/run` are rejected — a poisoned env variable can no longer point the connector at `/etc` or `/proc`

### Internal
- **Dependency bumps**: `BurntSushi/toml` v1.5.0 → v1.6.0, `fsouza/go-dockerclient` v1.13.1 → v1.13.2, Go directive 1.26.0 → 1.26.4

---

## v0.8.6 (2026-04-21)

### Security
- **runc upgrade v1.1.14 → v1.4.0**: Patches three container-escape vulnerabilities (CVE-2025-31133, CVE-2025-52565, CVE-2025-52881) that remain unpatched in the runc 1.1.x line — 1.1.x is end-of-life

### Bugfixes
- **Race conditions in the runC connector** (`connector/runc.go`): map accesses on `libContainers` and `containers` are now correctly guarded with `RLock`; `refreshAll` snapshots IDs under the lock before sending on the channel
- **Nil-channel deadlock**: `needsRefresh` channel was never initialised in the original code — any send or receive would have silently blocked
- **libcontainer API migration**: replaced Factory pattern with direct `libcontainer.Load(root, id)`; `cgroups` package now pulled from `opencontainers/cgroups` (runc v1.2+ split)

### Internal
- **Go 1.24 → 1.26**: Go 1.24 reached EOL on 2026-02-11
- **golangci-lint v1 → v2**: config migrated and CI action v6 → v8
- **Dependency cleanup**: `cilium/ebpf` unpinned v0.7.0 → v0.17.3 (the 1.1.14-compat pin is now obsolete); `fsouza/go-dockerclient` v1.7.0 → v1.13.1

### Documentation
- **Bubble Tea migration plan** (`_docs/bubbletea-migration-plan.md`): 6-phase plan to replace the dead TUI stack (`gizak/termui` + `nsf/termbox-go`) with the Charm stack

---

## v0.8.5 (2026-03-29)

### Bugfixes
- **Menu redraw fix**: After returning from container shell, the background grid is now correctly redrawn and the container menu is refocused

---

## v0.8.4 (2026-02-28)

### Bugfixes
- **Input widget width fix**: Commit dialog input field was too narrow (28 instead of 68 characters) — `SetMaxLen()` now correctly recalculates widget width
- **Exec shell fix**: Shell execution inside container overlapped with the TUI — terminal is now released before shell start (`ui.Close()`) and cleanly restored after exit (`ui.Init()`)

---

## v0.8.3 (2026-02-28)

### New Features
- **Flicker fix**: Eliminated screen flicker during 1-second refresh — header and grid are now drawn in a single render call. Screen clear only triggered when container count shrinks
- **Stop + Remove**: New combined menu entry `[X] stop + remove` in container menu for running containers — stops and removes in one step with confirmation dialog
- **Docker Commit**: Save containers as local images via `[C] commit to image` in container menu — input dialog with name suggestion (`<name>-snapshot`) and optional tag (default: `latest`)

---

## v0.8.2 (2026-02-28)

### New Features
- **Version display in header**: Version is shown in the top-right corner of the TUI header (e.g. `v0.8.2`)
- **Configurable shell**: Shell for `exec` is configurable via `--shell` flag, `CTOP_SHELL` environment variable, or config file (priority: CLI > Env > Config > Default `/bin/sh`)
- **Health status column**: New optional column shows health check status (`healthy`/`unhealthy`/`starting`) with color coding
- **Restart count column**: New optional column shows container restart count with color warnings (>0 yellow, >5 red)

### Bugfixes
- Fixed header version positioning (no longer at the very edge of the terminal)

### CI/CD
- **Homebrew tap auto-update**: CI job automatically updates the Homebrew formula after a release
- New Makefile target `make update-homebrew` to sync the Homebrew tap repo

---

## v0.8.0 (2025-06-24)

First release of the maintained fork of [bcicen/ctop](https://github.com/bcicen/ctop).

### Security Fixes
- Fixed shell injection vulnerability in `container.Exec()` — using `exec.Command()` instead of `sh -c`
- Set config file permissions to `0600`
- Debug server now binds to `localhost` instead of `0.0.0.0`

### Modernization
- Updated to **Go 1.22** with modern stdlib packages
- Migrated from `op/go-logging` to `log/slog` (structured logging)
- Replaced `nu7hatch/gouuid` with `google/uuid`
- Replaced `pkg/errors` with `fmt.Errorf` using `%w` wrapping
- Replaced `io/ioutil` with `os`/`io` functions
- Using `os.ReadDir` instead of `ioutil.ReadDir`

### CI/CD & Infrastructure
- GitHub Actions workflow for automatic multi-platform builds (Linux, macOS, Windows — AMD64 + ARM64)
- Automatic GitHub Releases with binaries and checksums
- Docker image via `ghcr.io/eqms/ctop`

### Documentation
- Bilingual README (German/English)
- Fork motivation and change overview documented
