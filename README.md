# Madori

A Windows system-tray utility that remembers window positions and restores them
when display configurations change — monitor connect/disconnect, sleep/resume,
resolution changes, RDP connections — any event that alters the desktop geometry.

Madori is a **Go port** of the core engine of
[PersistentWindows](https://github.com/kangyu-california/PersistentWindows),
a long-standing C# project that addresses a [known Windows
issue](https://answers.microsoft.com/en-us/windows/forum/windows_10-hardware/windows-10-multiple-display-windows-are-moved-and/2b9d5a18-45cc-4c50-b16e-fd95dbf27ff3)
where windows get scrambled after display events. The port drops the GUI
framework (WPF), the legacy XML persistence format, and the webpage-commander
feature, while keeping the battle-tested capture/restore engine and adding a
modern Go-native architecture.

## What's in the name?

The name is a blend of **mado** (窓), meaning "window," and **modori** (戻り),
meaning "return." It can also be read as **madori** (間取り), meaning "floor
plan" or "room layout" — a hopefully fitting name for a tool that restores the
layout of your workspace.

## Key Features

- **Automatic restore** — Detects display configuration changes (monitor
  connect/disconnect, sleep/resume, resolution changes, and RDP
  connections — all of which alter the desktop geometry) and restores
  windows to their previous positions for the matching setup.
- **Manual snapshots** — Save and restore desktop layouts on demand via the
  tray menu (or single/double-click on the tray icon for snapshot `0`).
  Up to 37 snapshots (keys `0`–`9`, `a`–`z`, `` ` ``) per display configuration.
- **Window parking** — Hold **Shift** while minimizing a window to park it in
  the system tray. Each parked window gets its own tray icon; click to restore.
- **Z-order preservation** — Optionally restores window stacking order
  alongside position and size.
- **Crash recovery** — On startup, restores any windows that were left parked
  if a previous session ended abnormally.
- **One-shot CLI commands** — Capture or restore snapshots from the command
  line for scripting (`-capture_snapshot`, `-restore_snapshot`,
  `-restore_parked_windows`). These run, do their job, and exit — no tray icon.
- **Process filtering** — Ignore specific processes or track only an exclusive
  list of processes.
- **Portable mode** — Store all data under a `user_data/` directory next to
  the executable instead of `%LocalAppData%`.
- **Single-instance lock** — Prevents multiple copies from running
  simultaneously, with automatic stale-lock detection.
- **Per-monitor DPI awareness** — Uses physical pixel coordinates on
  mixed-DPI multi-monitor systems so coordinates don't drift.

## Requirements

- **Windows 7, 10, or 11**
- **Administrator privileges** are required to manage windows owned by
  elevated processes (Task Manager, administrative consoles, etc.). Without
  elevation, those windows are skipped during capture and restore.

## Installation

1. Download the latest `madori.exe` from the
   [Releases](../../releases)
   page (Madori is distributed alongside PersistentWindows releases).
2. Place `madori.exe` in any directory.
3. Run it — preferably as Administrator.

### Auto-start at login

Use **Task Scheduler** (recommended) or the **Startup Folder**. See the
[PersistentWindows README](https://github.com/kangyu-california/PersistentWindows#installation)
for detailed instructions — the same methods work for Madori (substitute
`madori.exe` for `PersistentWindows.exe`).

## Quick Start

For a thorough walkthrough of the tray interface, command-line options,
logging, and files — see **[QUICKSTART.md](QUICKSTART.md)**.

### The short version

| Action | Result |
| --- | --- |
| Double-click tray icon | Capture snapshot `0` |
| Single-click tray icon | Restore snapshot `0` |
| Right-click tray icon | Full context menu |
| **Shift** + minimize a window | Park it to the tray |
| Click a parked window's tray icon | Restore that window |

### One-shot (scripting) mode

```
madori.exe -capture_snapshot 3      # capture snapshot 3 and exit
madori.exe -restore_snapshot 3      # restore snapshot 3 and exit
madori.exe -restore_parked_windows  # restore any orphaned parked windows and exit
```

### Common options

```
madori.exe -portable_mode                    # store data in ./user_data/
madori.exe -log all -log_level debug         # verbose logging
madori.exe -ignore_process "teams;slack"     # skip these processes
madori.exe -disable_window_parking           # turn off Shift+minimize-to-tray
```

## Architecture

```
cmd/madori/          Entry point, CLI flag parsing, one-shot commands
internal/
  engine/            Core capture/restore engine (event processing, timers,
                     snapshots, window parking, display-change handling)
  winapi/            Windows API bindings (user32, kernel32, gdi32, dwmapi,
                     shell32, virtual-desktop, WinEvents, WTS)
  tray/              System-tray icon, context menu, notification balloons,
                     parked-window icons
  storage/           BoltDB persistence layer (window positions, snapshots,
                     parked windows, display-key metadata)
  models/            Shared data types (WindowMetrics)
  logger/            Category-filtered, level-gated logging
```

### How it works

1. On startup, Madori registers **WinEvent hooks** to listen for window
   creation, destruction, moves, minimizes, and foreground changes.
2. Events from the hooks are sent to a channel and processed on a single
   goroutine — the same pattern as the original C# main-thread dispatch.
3. A **debounced capture timer** fires after a quiet period (default 3 s),
   enumerating all visible top-level windows and recording their positions,
   sizes, states, and z-order.
4. When a display change is detected (via `WM_DISPLAYCHANGE`), a
   **debounced restore timer** fires and repositions windows to match the
   last capture for the current monitor configuration.
5. All state is persisted to **BoltDB** (replacing the original LiteDB) so it
   survives application restarts and system reboots.

### Key differences from PersistentWindows

| Aspect | PersistentWindows (C#) | Madori (Go) |
| --- | --- | --- |
| Database | LiteDB | BoltDB (bbolt) |
| GUI | WPF window + tray | Tray only (no main window) |
| Webpage commander | Built-in | Removed |
| XML persistence | Yes (parallel to LiteDB) | No (BoltDB only) |
| Logging | Event Log (Event ID 9990/9999) | Category-filtered stderr output |
| Build | Visual Studio / MSBuild | `go build` (cross-compile from WSL) |
| Dependencies | .NET Framework 4.7.2+ | Single static binary (~6 MB) |

## Building from source

```bash
# Requires Go 1.25+ and cross-compilation for Windows

# Development build (console window visible, useful for log output):
make build

# Release build (no console window, gui subsystem):
make build-release

# Run tests (compilation only — execution requires Windows):
make test

# Clean build artifacts:
make clean
```

See the [Makefile](Makefile) for details. Resource files (icon, version info)
are embedded via `go-winres`.

## Privacy

Madori collects the following information to do its job:
- Window position, size, and state
- Window title text and class name
- Process name, ID, and executable path
- Window z-order (stacking rank)
- Keystroke state (**Shift** key) for window-parking interception

Window information is kept in RAM and persisted to a local BoltDB file
(`%LocalAppData%\Madori\` or `user_data\` in portable mode). Keystroke
state is ephemeral (tracked via a low-level keyboard hook with a 300 ms
grace period and never written to disk).

No data is sent over the network. There is no telemetry, no update checking,
and no analytics.

## Known Issues

- During a restore, if a window becomes unresponsive, Madori may appear stuck
  with a busy icon. Use Task Manager's "Analyze wait chain" to identify the
  culprit window, then kill or wait for that application to recover.

## License

[GPLv3](LICENSE) — inherited from the original PersistentWindows project.

## Related Projects

- [PersistentWindows](https://github.com/kangyu-california/PersistentWindows) —
  The original C# project from which Madori is derived.
