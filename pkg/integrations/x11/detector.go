package x11

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/actionsum/actionsum/pkg/window"
)

// Detector implements window.Detector for X11
type Detector struct {
	hasXdotool bool
	hasWmctrl  bool
}

// NewDetector creates a new X11 detector
func NewDetector() *Detector {
	d := &Detector{}
	d.hasXdotool = d.commandExists("xdotool")
	d.hasWmctrl = d.commandExists("wmctrl")
	return d
}

// commandExists checks if a command is available in PATH
func (d *Detector) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// IsAvailable checks if X11 detection is available
func (d *Detector) IsAvailable() bool {
	if d.hasXdotool {
		return true
	}
	if d.hasWmctrl {
		return true
	}
	return false
}

// GetDisplayServer returns "x11"
func (d *Detector) GetDisplayServer() string {
	return "x11"
}

// GetFocusedWindow returns information about the currently focused window
func (d *Detector) GetFocusedWindow() (*window.WindowInfo, error) {
	if d.hasXdotool {
		return d.getFocusedWindowXdotool()
	}
	if d.hasWmctrl {
		return d.getFocusedWindowWmctrl()
	}
	return nil, fmt.Errorf("no X11 detection tool available (xdotool or wmctrl required)")
}

// getFocusedWindowXdotool uses xdotool to get focused window info
func (d *Detector) getFocusedWindowXdotool() (*window.WindowInfo, error) {
	windowIDCmd := exec.Command("xdotool", "getactivewindow")
	windowIDOutput, err := windowIDCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get active x11 window ID: %w", err)
	}

	windowID := strings.TrimSpace(string(windowIDOutput))

	windowNameCmd := exec.Command("xdotool", "getwindowname", windowID)
	windowNameOutput, err := windowNameCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get window name: %w", err)
	}

	windowTitle := strings.TrimSpace(string(windowNameOutput))

	// Try to get WM_CLASS first (works for Flatpak apps)
	appName := "Unknown"
	processName := ""

	classCmd := exec.Command("xprop", "-id", windowID, "WM_CLASS")
	if classOutput, err := classCmd.Output(); err == nil {
		if class := parseWMClass(string(classOutput)); class != "" {
			appName = class
		}
	}

	// Try to get PID and process name (may fail for Flatpak/sandboxed apps)
	pidCmd := exec.Command("xdotool", "getwindowpid", windowID)
	if pidOutput, err := pidCmd.Output(); err == nil {
		pid := strings.TrimSpace(string(pidOutput))

		psCmd := exec.Command("ps", "-p", pid, "-o", "comm=")
		if psOutput, err := psCmd.Output(); err == nil {
			processName = strings.TrimSpace(string(psOutput))
			// Only use process name if we didn't get WM_CLASS
			if appName == "Unknown" && processName != "" {
				appName = processName
			}
		}
	}

	return &window.WindowInfo{
		AppName:       appName,
		WindowTitle:   windowTitle,
		ProcessName:   processName,
		DisplayServer: "x11",
	}, nil
}

// getFocusedWindowWmctrl uses wmctrl to get focused window info
func (d *Detector) getFocusedWindowWmctrl() (*window.WindowInfo, error) {
	cmd := exec.Command("wmctrl", "-l", "-p")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute wmctrl: %w", err)
	}

	activeWindowCmd := exec.Command("xdotool", "getactivewindow")
	activeWindowOutput, err := activeWindowCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get active window: %w", err)
	}

	activeWindowID := strings.TrimSpace(string(activeWindowOutput))

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, activeWindowID) {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}

			pid := fields[2]
			windowTitle := strings.Join(fields[4:], " ")

			psCmd := exec.Command("ps", "-p", pid, "-o", "comm=")
			psOutput, err := psCmd.Output()
			processName := "Unknown"
			if err == nil {
				processName = strings.TrimSpace(string(psOutput))
			}

			return &window.WindowInfo{
				AppName:       processName,
				WindowTitle:   windowTitle,
				ProcessName:   processName,
				DisplayServer: "x11",
			}, nil
		}
	}

	return nil, fmt.Errorf("could not find active window")
}

// parseWMClass extracts the class name from WM_CLASS property
func parseWMClass(output string) string {
	parts := strings.Split(output, "=")
	if len(parts) < 2 {
		return ""
	}

	classInfo := strings.TrimSpace(parts[1])
	classInfo = strings.Trim(classInfo, "\"")

	classes := strings.Split(classInfo, ",")
	if len(classes) > 0 {
		className := strings.TrimSpace(classes[len(classes)-1])
		className = strings.Trim(className, "\" ")
		return className
	}

	return ""
}

// GetIdleInfo returns system idle/lock information
func (d *Detector) GetIdleInfo() (*window.IdleInfo, error) {
	idleTime, err := d.getIdleTime()
	if err != nil {
		return nil, err
	}

	isLocked := d.isScreenLocked()

	const idleThreshold = 300
	isIdle := idleTime > idleThreshold

	return &window.IdleInfo{
		IsIdle:   isIdle,
		IsLocked: isLocked,
		IdleTime: idleTime,
	}, nil
}

// getIdleTime returns the system idle time in seconds
func (d *Detector) getIdleTime() (int64, error) {
	if d.hasXdotool {
		cmd := exec.Command("xprintidle")
		output, err := cmd.Output()
		if err != nil {
			return 0, nil
		}

		idleMs := strings.TrimSpace(string(output))
		idleMilliseconds, err := strconv.ParseInt(idleMs, 10, 64)
		if err != nil {
			return 0, nil
		}

		return idleMilliseconds / 1000, nil
	}

	return 0, nil
}

// isScreenLocked checks if the screen is locked
func (d *Detector) isScreenLocked() bool {
	lockers := []string{
		"gnome-screensaver-dialog",
		"kscreenlocker",
		"i3lock",
		"slock",
		"xscreensaver",
		"xsecurelock",
	}

	for _, locker := range lockers {
		cmd := exec.Command("pgrep", "-x", locker)
		if err := cmd.Run(); err == nil {
			return true
		}
	}

	return false
}

// Close cleans up resources
func (d *Detector) Close() error {
	return nil
}
