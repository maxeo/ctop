# Release Notes

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
