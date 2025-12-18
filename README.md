# actionsum

[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8.svg?logo=go)](https://go.dev/) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![Linux](https://img.shields.io/badge/Linux-FCC624.svg?logo=linux&logoColor=black)](https://www.kernel.org/) [![Build](https://actionsum/actions/workflows/build.yml/badge.svg)](https://actionsum/actions)

**actionsum** is a lightweight, Linux CLI daemon for tracking and reporting time spent in focused applications. It runs in the background, monitors your current app focus, and generates detailed time reports for productivity insights—accessible via terminal or web browser.

Perfect for developers, remote workers, and anyone tracking focus time on Linux desktops (X11 and Wayland support for GNOME, KDE, Sway, Hyprland, etc.).

---

## 📦 Installation

### Via Go Install (Recommended)
```bash
go install actionsum@latest
```

### From Source
```bash
git clone https://github.com/actionsum/actionsum.git
cd actionsum
go build -o actionsum ./cmd/actionsum
sudo mv actionsum /usr/local/bin/
```

### From Release
Download the latest binary from [Releases](https://actionsum/releases):
```bash
# Download and install (replace VERSION and ARCH as needed)
wget https://actionsum/releases/download/v0.1.0/actionsum_0.1.0_linux_amd64.tar.gz
tar -xzf actionsum_0.1.0_linux_amd64.tar.gz
sudo mv actionsum /usr/local/bin/
```

### Dependencies
For X11 systems, install one of:
- `xdotool` (recommended)
- `wmctrl`

For Wayland systems with GNOME, install:
- `gdbus` (usually pre-installed)

---

## 🚀 Features

### Core Functionality
- **Background Daemon**: Runs continuously in the background tracking application focus
- **X11 & Wayland Support**: Works across different display servers and window managers
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

## 🎯 Technical Decisions

### Architecture
- **Language**: Go
- **Storage**: SQLite in `~/.config/actionsum/actionsum.db`
- **Data Retention**: Indefinite (no automatic cleanup for now)
- **Configuration**: Default values, no config file needed initially

### Detection Methods (V2 - Universal Hybrid Approach)

actionsum uses a **hybrid detection system** that combines multiple methods for universal compatibility:

#### Primary: Window Detection
- **X11**: Using `xprop` for window detection ✅
- **Wayland Compositors**:
  - **Sway/Hyprland**: Native protocol support via `swaymsg`/`hyprctl` ✅
  - **KDE Plasma**: D-Bus scripting API ✅
  - **GNOME**: XWayland bridge for X11 apps ✅

#### Fallback: Process Monitoring
When window detection fails (e.g., native Wayland apps on GNOME), the system automatically falls back to:
- **Process scanning**: Monitors `/proc` filesystem for running GUI applications
- **Activity tracking**: Uses CPU usage heuristics to identify the active application
- **Smart scoring**: Combines multiple signals to determine the most likely active app

#### Universal Compatibility
- ✅ **All X11 applications** - Full window detection with titles
- ✅ **XWayland applications** (Chrome, VSCode, Firefox, Electron apps, etc.) - Full window detection with titles
- ✅ **Native Wayland applications** (GNOME Terminal, Nautilus, etc.) - Process-based detection (app name only)
- ✅ **Works on all compositors** - No compositor-specific limitations
- ✅ **No failed tracking** - Always falls back to process detection

The hybrid approach ensures tracking works **regardless of display server, compositor, or desktop environment**.

> **Note**: When using process-based detection (fallback for native Wayland apps), window titles may not be available. The system will still accurately track the application name and time spent.

### Data Model
- Track: timestamp, application name, window title, focus duration
- Exclude: idle time, locked screen sessions
- Reports: Aggregated by application with JSON output support

---

## 📋 Roadmap

### Completed ✅
- [x] Basic daemon architecture (start/stop/status)
- [x] X11 window focus detection
- [x] Wayland window focus detection (Sway, Hyprland, KDE)
- [x] SQLite storage layer
- [x] Idle/lock detection
- [x] Terminal report generation (day/week/month)
- [x] Web-based report UI
- [x] Export functionality (JSON)
- [x] **V2 Hybrid Detection** - Universal compatibility with process-based fallback

---

## 🤝 Contributing

Contributions welcome! This is an early-stage project.

---

## 📄 License

MIT License - See LICENSE file for details