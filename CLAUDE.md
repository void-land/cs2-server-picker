# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

YASP (Yet Another Server Picker) is a Go TUI and CLI for picking CS2 servers. It fetches Valve SDR relay data from `https://api.steampowered.com/ISteamApps/GetSDRConfig/v1?appid=730` and blocks the relay IPs you don't want in the OS firewall. It uses iptables on Linux and `netsh advfirewall` on Windows. The Go module path is `github.com/dom1torii/yet-another-server-picker`, which is the upstream repo. This checkout is a fork.

## Commands

```bash
go build ./cmd/yasp/            # build the binary into the repo root
make                            # build into ./build/ (Makefile; alternative to taskfile.yml)
make && sudo make install       # build as your user, then install the binary into /usr/bin (PREFIX/DESTDIR supported)
sudo make uninstall
go-task build                   # build into ./build/
sudo go-task install            # build, then install binary + icon + .desktop file into /usr (Linux)
sudo go-task uninstall
go vet ./...
GOOS=windows go build ./cmd/yasp/   # check that the Windows build tags still compile
goreleaser release --snapshot --clean   # local release build (linux + windows amd64, CGO_ENABLED=0)
```

There are no tests in the repo.

The binary refuses to start without root: `sudo` on Linux, Administrator on Windows (`internal/platform/sudo`). Pinging and firewall changes both need it. To run the app locally, use `sudo ./yasp`. CLI-mode flags are defined in `internal/config/config.go`, for example `sudo ./yasp --listrelays` and `--selectpreset eu --blockrelays`.

## Architecture

**Startup** (`cmd/yasp/main.go`): `sudo.CheckIfSudo()` → `config.Init()` → either `cli.HandleFlags` or the Bubble Tea program. `config.Init()` does several things:
- It loads `~/.config/yasp/config.toml` and writes defaults if the file is empty.
- It parses **all** pflag flags. Config and CLI flags live in one `Config` struct, and the CLI-mode fields have no TOML tags.
- It sets up logging. When logging is disabled, `log` output goes to `io.Discard`.

`cli.IsCLIMode` decides between CLI and TUI. It returns true if any action flag is set.

**The IPs file is the central handoff.** Selecting relays (through the TUI, `--selectrelays`, or `--selectpreset`) only writes the IPs *to block* to a text file, one IP per line (`ips.WriteIpsToFile`, default `~/yasp_ips.txt`). Blocking is a separate step: `firewall.BlockIps` reads that file. "Selected" relays are the ones to *keep*, so the file holds the IPs of every relay that was **not** selected. In the TUI, the "block" mode (toggled with `t`) inverts this.

**Platform code** lives in `internal/platform/{firewall,sudo,perms}`. Each package has a `_linux.go` and a `_windows.go` file with `//go:build` tags and identical exported function signatures. Any new platform function needs both variants, or the other OS's build breaks.
- Firewall state goes under the name `CS2_BLOCKLIST`. On Linux it is a custom iptables chain attached to `OUTPUT`, with one DROP rule per IP. On Windows it is a single netsh rule with comma-joined `remoteip`. `BlockIps` always calls `UnBlockIps` first. `GetBlockedIps` parses the output of `iptables -S` or `netsh show rule`.
- Running under sudo has some side effects. `fs.GetHomeDir` resolves `SUDO_USER`'s home, not root's. `fs.EnsureDirectory` chowns created files back to `SUDO_UID`/`SUDO_GID` through `perms.FixPermissions`.

**Relays and presets**:
- `api.FetchRelays` drops pops with no relays.
- It also drops Perfect World (China) pops when `show_perfectworld` is false. The `-g/--onlyglobal` flag sets that option too.
- PW pops are identified by the `cn-pw` entry in `internal/presets/presets.go`.
- Presets are a hardcoded map from a preset key to a set of pop codes.

**TUI** (`internal/tui`, Bubble Tea + Lipgloss). It is one root `model` with a `sessionState` (start / relays / presets / confirm) and a sub-model per screen.
- Each screen lives in `view_<screen>.go`, which holds both its `update<Screen>Selection` key handler and its `<screen>View` renderer.
- `update.go` dispatches async messages (types in `messages.go`) and routes key presses by state.
- `commands.go` holds the `tea.Cmd` constructors for I/O: API fetch, pings, firewall, and status refresh.
- Relays are pinged in batches of 20. Each `pingMsg` advances a counter, and the next batch starts when the counter reaches a multiple of 20. A ping of `-1` means the IP is blocked.
- The relay list is laid out in two columns (`getRows`), so pressing left or right moves the selection index by `rows`.
- Most errors anywhere in the codebase are handled with `log.Fatalln`.
