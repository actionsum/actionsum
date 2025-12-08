package wayland

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"actionsum/pkg/window"
)

// Detector implements window.Detector for Wayland
type Detector struct {
	compositor string
	hasSwaymsg bool
	hasGdbus   bool
}

// NewDetector creates a new Wayland detector
func NewDetector() *Detector {
	d := &Detector{}
	d.hasSwaymsg = d.commandExists("swaymsg")
	d.hasGdbus = d.commandExists("gdbus")
	d.detectCompositor()
	return d
}

// commandExists checks if a command is available in PATH
func (d *Detector) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// detectCompositor attempts to detect the Wayland compositor
func (d *Detector) detectCompositor() {
	compositors := map[string]string{
		"sway":         "sway",
		"Hyprland":     "hyprland",
		"wayfire":      "wayfire",
		"river":        "river",
		"gnome-shell":  "gnome",
		"kwin_wayland": "kde",
	}

	for process, name := range compositors {
		cmd := exec.Command("pgrep", "-x", process)
		if err := cmd.Run(); err == nil {
			d.compositor = name
			return
		}
	}

	d.compositor = "unknown"
}

// IsAvailable checks if Wayland detection is available
func (d *Detector) IsAvailable() bool {
	switch d.compositor {
	case "sway", "hyprland":
		return d.hasSwaymsg
	case "gnome":
		return d.hasGdbus
	case "kde":
		return true
	default:
		return false
	}
}

// GetDisplayServer returns "wayland"
func (d *Detector) GetDisplayServer() string {
	return "wayland"
}

// GetFocusedWindow returns information about the currently focused window
func (d *Detector) GetFocusedWindow() (*window.WindowInfo, error) {
	switch d.compositor {
	case "sway":
		return d.getFocusedWindowSway()
	case "hyprland":
		return d.getFocusedWindowHyprland()
	case "gnome":
		return d.getFocusedWindowGnome()
	case "kde":
		return d.getFocusedWindowKDE()
	default:
		return nil, fmt.Errorf("unsupported wayland compositor: %s", d.compositor)
	}
}

// getFocusedWindowSway gets focused window info from Sway
func (d *Detector) getFocusedWindowSway() (*window.WindowInfo, error) {
	cmd := exec.Command("swaymsg", "-t", "get_tree")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute swaymsg: %w", err)
	}

	info, err := parseSwayTree(string(output))
	if err != nil {
		return nil, err
	}

	info.DisplayServer = "wayland"
	return info, nil
}

// parseSwayTree parses sway tree JSON output (simplified parsing)
func parseSwayTree(jsonOutput string) (*window.WindowInfo, error) {
	lines := strings.Split(jsonOutput, "\n")

	var appName, windowTitle, pid string
	inFocusedNode := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, `"focused": true`) {
			inFocusedNode = true
		}

		if inFocusedNode {
			if strings.HasPrefix(line, `"app_id":`) || strings.HasPrefix(line, `"class":`) {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					appName = strings.Trim(strings.TrimRight(parts[1], ","), `" `)
				}
			}

			if strings.HasPrefix(line, `"name":`) {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					windowTitle = strings.Trim(strings.TrimRight(parts[1], ","), `" `)
				}
			}

			if strings.HasPrefix(line, `"pid":`) {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					pid = strings.Trim(strings.TrimRight(parts[1], ","), " ")
				}
			}

			if appName != "" && windowTitle != "" && pid != "" {
				break
			}
		}
	}

	if appName == "" {
		appName = "Unknown"
	}
	if windowTitle == "" {
		windowTitle = "Unknown"
	}

	processName := appName
	if pid != "" {
		if name := getProcessName(pid); name != "" {
			processName = name
		}
	}

	return &window.WindowInfo{
		AppName:     appName,
		WindowTitle: windowTitle,
		ProcessName: processName,
	}, nil
}

// getFocusedWindowHyprland gets focused window info from Hyprland
func (d *Detector) getFocusedWindowHyprland() (*window.WindowInfo, error) {
	cmd := exec.Command("hyprctl", "activewindow", "-j")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute hyprctl: %w", err)
	}

	info := parseHyprlandWindow(string(output))
	info.DisplayServer = "wayland"
	return info, nil
}

// parseHyprlandWindow parses Hyprland active window JSON (simplified)
func parseHyprlandWindow(jsonOutput string) *window.WindowInfo {
	lines := strings.Split(jsonOutput, "\n")

	var appName, windowTitle, pid string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, `"class":`) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				appName = strings.Trim(strings.TrimRight(parts[1], ","), `" `)
			}
		}

		if strings.HasPrefix(line, `"title":`) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				windowTitle = strings.Trim(strings.TrimRight(parts[1], ","), `" `)
			}
		}

		if strings.HasPrefix(line, `"pid":`) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				pid = strings.Trim(strings.TrimRight(parts[1], ","), " ")
			}
		}
	}

	if appName == "" {
		appName = "Unknown"
	}
	if windowTitle == "" {
		windowTitle = "Unknown"
	}

	processName := appName
	if pid != "" {
		if name := getProcessName(pid); name != "" {
			processName = name
		}
	}

	return &window.WindowInfo{
		AppName:     appName,
		WindowTitle: windowTitle,
		ProcessName: processName,
	}
}

// getFocusedWindowGnome gets focused window info from GNOME Shell via D-Bus
func (d *Detector) getFocusedWindowGnome() (*window.WindowInfo, error) {
	// Try gdbus method first (works without X11 authorization)
	script := `
	try {
		let win = global.get_window_actors().find(w => w.meta_window && w.meta_window.has_focus());
		if (win && win.meta_window) {
			let wm_class = win.meta_window.get_wm_class() || 'Unknown';
			let title = win.meta_window.get_title() || 'Unknown';
			wm_class + '|||' + title;
		} else {
			'Unknown|||Unknown';
		}
	} catch(e) {
		'Unknown|||Unknown';
	}
	`

	cmd := exec.Command("gdbus", "call", "--session",
		"--dest", "org.gnome.Shell",
		"--object-path", "/org/gnome/Shell",
		"--method", "org.gnome.Shell.Eval",
		script)

	output, err := cmd.Output()

	// Parse output: (true, 'AppName|||WindowTitle') or (false, '')
	if err == nil {
		result := strings.TrimSpace(string(output))

		// Check if Shell.Eval succeeded
		if strings.HasPrefix(result, "(true,") {
			result = strings.TrimPrefix(result, "(true, '")
			result = strings.TrimSuffix(result, "')")
			result = strings.Trim(result, "'\"")

			parts := strings.Split(result, "|||")
			appName := "Unknown"
			windowTitle := "Unknown"

			if len(parts) >= 1 && parts[0] != "" && parts[0] != "Unknown" {
				appName = parts[0]
			}
			if len(parts) >= 2 && parts[1] != "" && parts[1] != "Unknown" {
				windowTitle = parts[1]
			}

			// Only return if we got valid info
			if appName != "Unknown" {
				return &window.WindowInfo{
					AppName:       appName,
					WindowTitle:   windowTitle,
					ProcessName:   appName,
					DisplayServer: "wayland",
				}, nil
			}
		}
	}

	// Fallback to XWayland using xprop (only needs xprop, not xdotool)
	if d.commandExists("xprop") {
		info, xErr := d.getFocusedWindowXWayland()
		if xErr == nil {
			return info, nil
		}
		// Return detailed error from xprop
		return nil, fmt.Errorf("GNOME window detection failed: gdbus Shell.Eval blocked, xprop failed: %v", xErr)
	}

	// Last resort: return error
	return nil, fmt.Errorf("GNOME window detection failed: gdbus Shell.Eval blocked and xprop unavailable")
}

// getFocusedWindowXWayland uses XWayland bridge (fallback for Wayland)
func (d *Detector) getFocusedWindowXWayland() (*window.WindowInfo, error) {
	// Check if DISPLAY is set - required for xprop
	display := os.Getenv("DISPLAY")
	if display == "" {
		return nil, fmt.Errorf("DISPLAY environment variable not set (XWayland not available)")
	}

	// Get active window ID from root window property
	rootCmd := exec.Command("xprop", "-root", "_NET_ACTIVE_WINDOW")
	rootOutput, err := rootCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get active window from root: %w (output: %s)", err, string(rootOutput))
	}

	// Parse: _NET_ACTIVE_WINDOW(WINDOW): window id # 0x80032b
	windowID := ""
	output := string(rootOutput)
	if strings.Contains(output, "# 0x") {
		parts := strings.Split(output, "# ")
		if len(parts) >= 2 {
			windowID = strings.TrimSpace(parts[1])
		}
	}

	if windowID == "" || windowID == "0x0" {
		return nil, fmt.Errorf("no active window found (focused window may be native Wayland)")
	}

	// Get window title
	nameCmd := exec.Command("xprop", "-id", windowID, "WM_NAME")
	nameOutput, _ := nameCmd.Output()
	windowTitle := parseXPropString(string(nameOutput))
	if windowTitle == "" {
		windowTitle = "Unknown"
	}

	// Get window class
	classCmd := exec.Command("xprop", "-id", windowID, "WM_CLASS")
	classOutput, _ := classCmd.Output()
	appName := parseWMClass(string(classOutput))
	if appName == "" {
		appName = "Unknown"
	}

	return &window.WindowInfo{
		AppName:       appName,
		WindowTitle:   windowTitle,
		ProcessName:   appName,
		DisplayServer: "wayland",
	}, nil
}

// parseXPropString parses xprop string output like: WM_NAME(STRING) = "title"
func parseXPropString(output string) string {
	if strings.Contains(output, "=") {
		parts := strings.SplitN(output, "=", 2)
		if len(parts) == 2 {
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, "\"")
			return value
		}
	}
	return ""
}

// parseWMClass extracts class from WM_CLASS output
func parseWMClass(output string) string {
	if strings.Contains(output, "=") {
		parts := strings.Split(output, "=")
		if len(parts) >= 2 {
			classInfo := strings.TrimSpace(parts[1])
			classInfo = strings.Trim(classInfo, "\"")
			classes := strings.Split(classInfo, ",")
			if len(classes) > 0 {
				return strings.Trim(classes[len(classes)-1], "\" ")
			}
		}
	}
	return ""
}

// getFocusedWindowKDE gets focused window info from KDE Plasma
func (d *Detector) getFocusedWindowKDE() (*window.WindowInfo, error) {
	script := `
	var clients = workspace.clientList();
	for (var i = 0; i < clients.length; i++) {
		if (clients[i].active) {
			print(clients[i].resourceClass + "|" + clients[i].caption);
		}
	}
	`

	cmd := exec.Command("qdbus", "org.kde.KWin", "/Scripting", "org.kde.kwin.Scripting.loadScript", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query KDE window: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), "|")
	appName := "Unknown"
	windowTitle := "Unknown"

	if len(parts) >= 1 && parts[0] != "" {
		appName = parts[0]
	}
	if len(parts) >= 2 && parts[1] != "" {
		windowTitle = parts[1]
	}

	return &window.WindowInfo{
		AppName:       appName,
		WindowTitle:   windowTitle,
		ProcessName:   appName,
		DisplayServer: "wayland",
	}, nil
}

// getProcessName retrieves process name from PID
func getProcessName(pid string) string {
	cmd := exec.Command("ps", "-p", pid, "-o", "comm=")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetIdleInfo returns system idle/lock information for Wayland
func (d *Detector) GetIdleInfo() (*window.IdleInfo, error) {
	idleTime := d.getIdleTime()
	isLocked := d.isScreenLocked()

	const idleThreshold = 300
	isIdle := idleTime > idleThreshold

	return &window.IdleInfo{
		IsIdle:   isIdle,
		IsLocked: isLocked,
		IdleTime: idleTime,
	}, nil
}

// getIdleTime attempts to get idle time (limited support in Wayland)
func (d *Detector) getIdleTime() int64 {
	switch d.compositor {
	case "sway", "hyprland":
		cmd := exec.Command("swaymsg", "-t", "get_idle_inhibitors")
		if err := cmd.Run(); err == nil {
			return 0
		}
	}

	return 0
}

// isScreenLocked checks if screen is locked
func (d *Detector) isScreenLocked() bool {
	lockers := []string{
		"swaylock",
		"waylock",
		"gtklock",
		"hyprlock",
		"gnome-screensaver-dialog",
	}

	for _, locker := range lockers {
		cmd := exec.Command("pgrep", "-x", locker)
		if err := cmd.Run(); err == nil {
			return true
		}
	}

	cmd := exec.Command("loginctl", "show-session", "-p", "LockedHint")
	if output, err := cmd.Output(); err == nil {
		if strings.Contains(string(output), "LockedHint=yes") {
			return true
		}
	}

	return false
}

// Close cleans up resources
func (d *Detector) Close() error {
	return nil
}
