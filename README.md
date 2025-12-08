# actionsum

[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8.svg?logo=go)](https://go.dev/) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![Linux](https://img.shields.io/badge/Linux-FCC624.svg?logo=linux&logoColor=black)](https://www.kernel.org/)

**actionsum** is a lightweight, Linux CLI daemon for tracking and reporting time spent in focused applications. It runs in the background, monitors your current app focus, and generates detailed time reports for productivity insights—accessible via terminal or web browser.

Perfect for developers, remote workers, and anyone tracking focus time on Linux desktops (X11 and Wayland support for GNOME, KDE, Sway, Hyprland, etc.).

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

### Commands (Planned)
```bash
actionsum start         # Start the tracking daemon
actionsum stop          # Stop the daemon
actionsum status        # Check daemon status + current focused app
actionsum report        # Display terminal report (--day/--week/--month)
actionsum webreport     # Launch web UI in browser
```

---

## 🎯 Technical Decisions

### Architecture
- **Language**: Go
- **Storage**: SQLite in `~/.config/actionsum/actionsum.db`
- **Data Retention**: Indefinite (no automatic cleanup for now)
- **Configuration**: Default values, no config file needed initially

### Detection Methods
- **X11**: Using `xdotool`, `wmctrl`, or X11 libraries
- **Wayland**: Compositor-specific protocols (GNOME Shell extensions, KDE protocols, wlr-foreign-toplevel for wlroots-based compositors)
- **Idle Detection**: X11 screensaver extension, logind D-Bus, or lock screen process detection

### Data Model
- Track: timestamp, application name, window title, focus duration
- Exclude: idle time, locked screen sessions
- Reports: Aggregated by application with JSON output support

---

## 📋 Roadmap

- [ ] Basic daemon architecture (start/stop/status)
- [ ] X11 window focus detection
- [ ] Wayland window focus detection
- [ ] SQLite storage layer
- [ ] Idle/lock detection
- [ ] Terminal report generation (day/week/month)
- [ ] Web-based report UI
- [ ] Export functionality (CSV/JSON)
- [ ] systemd service integration (optional)

---

## 🔧 Installation

*Coming soon - build from source instructions*

---

## 📖 Usage

*Coming soon - detailed usage examples*

---

## 🤝 Contributing

Contributions welcome! This is an early-stage project.

---

## 📄 License

MIT License - See LICENSE file for details