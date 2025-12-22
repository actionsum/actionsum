# actionsum

[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8.svg?logo=go)](https://go.dev/) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![Linux](https://img.shields.io/badge/Linux-FCC624.svg?logo=linux&logoColor=black)](https://www.kernel.org/) [![Build](https://github.com/actionsum/actionsum/actions/workflows/build.yml/badge.svg)](https://github.com/actionsum/actionsum/actions)

**actionsum** is a lightweight, Linux CLI daemon for tracking and reporting time spent in focused applications. It runs in the background, monitors your current app focus, and generates detailed time reports for productivity insights—accessible via terminal or web browser.

**Note: Currently supports X11 only. Wayland support is planned for a future release.**

Perfect for developers, remote workers, and anyone tracking focus time on Linux desktops.

---

## Installation

### Via Go Install (Recommended)
```bash
go install github.com/actionsum/actionsum@latest
```

### From Source
```bash
git clone https://github.com/actionsum/actionsum.git
cd actionsum
go build -o actionsum .
sudo mv actionsum /usr/local/bin/
```

### From Release
Download the latest binary from [Releases](https://github.com/actionsum/actionsum/releases):
```bash
# Download and install (replace VERSION and ARCH as needed)
wget https://github.com/actionsum/actionsum/releases/download/v0.1.0/actionsum_0.1.0_linux_amd64.tar.gz
tar -xzf actionsum_0.1.0_linux_amd64.tar.gz
sudo mv actionsum /usr/local/bin/
```

### Dependencies
For X11 systems, install one of:
- `xdotool` (recommended)
- `wmctrl`

---

## Features

### Core Functionality
- **Background Daemon**: Runs continuously in the background tracking application focus
- **X11 Support**: Works with X11 display server
- **Smart Polling**: Configurable focus detection interval (default: 1 minute, customizable min/max)
- **Idle/Lock Detection**: Automatically excludes away-from-keyboard time from reports
- **Local Storage**: SQLite database stored in `~/.config/actionsum/`

### Reporting
- **Terminal Reports**: View time summaries directly in your terminal
  - Daily, weekly, and monthly breakdowns
  - JSON export format
- **Web Reports**: Interactive browser-based reports via built-in web server
- **Time Aggregation**: Summarizes total time per application over selected periods

### Commands
```bash
actionsum start         # Start the tracking daemon
actionsum serve         # Start daemon with web API server
actionsum stop          # Stop the daemon
actionsum status        # Check daemon status + current focused app
actionsum report [day|week|month]  # Display terminal report
actionsum clear         # Clear all tracking data
actionsum version       # Show version information
actionsum help          # Show help message
```

---

## Technical Decisions

### Architecture
- **Language**: Go
- **Storage**: SQLite in `~/.config/actionsum/actionsum.db`
- **Data Retention**: Indefinite (no automatic cleanup for now)
- **Configuration**: Default values, no config file needed initially

### Detection Methods

- **X11**: Using `xdotool` or `wmctrl` for window detection

### Data Model
- Track: timestamp, application name, window title, focus duration
- Exclude: idle time, locked screen sessions
- Reports: Aggregated by application with JSON output support

---

## Roadmap

### Completed
- [x] Basic daemon architecture (start/stop/status)
- [x] X11 window focus detection
- [x] SQLite storage layer
- [x] Idle/lock detection
- [x] Terminal report generation (day/week/month)
- [x] Web-based report UI
- [x] Export functionality (JSON)

### Planned
- [ ] Wayland support

---

## Contributing

Contributions welcome! This is an early-stage project.

---

## License

MIT License - See LICENSE file for details