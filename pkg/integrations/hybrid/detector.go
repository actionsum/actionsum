package hybrid

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/hugo/actionsum/pkg/integrations/common"
	"github.com/hugo/actionsum/pkg/integrations/process"
	"github.com/hugo/actionsum/pkg/integrations/wayland"
	"github.com/hugo/actionsum/pkg/integrations/x11"
	"github.com/hugo/actionsum/pkg/window"
)

// Detector combines multiple detection methods for universal application tracking
type Detector struct {
	// Window-based detector (X11/Wayland compositor-specific)
	windowDetector window.Detector

	// Process-based detector (universal fallback)
	processDetector *process.Detector

	// Track which method worked last time
	lastSuccessfulMethod string

	// Cache for process-to-window mapping
	windowCache map[int]string // PID -> window title

	initialized bool
}

// NewDetector creates a new hybrid detector
func NewDetector() (*Detector, error) {
	d := &Detector{
		windowCache: make(map[int]string),
	}

	// Try to initialize window detector (may fail on some systems)
	windowDet := detectWindowDetector()
	if windowDet != nil {
		d.windowDetector = windowDet
		log.Printf("Window detector initialized: %s", windowDet.GetDisplayServer())
	} else {
		log.Printf("Window detector unavailable, using process-based detection only")
	}

	// Initialize process detector (should always work)
	d.processDetector = process.NewDetector()
	if err := d.processDetector.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize process detector: %w", err)
	}

	d.initialized = true
	return d, nil
}

// detectWindowDetector tries to create the appropriate window detector
func detectWindowDetector() window.Detector {
	// Check if we're on Wayland
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	xdgSessionType := os.Getenv("XDG_SESSION_TYPE")

	if waylandDisplay != "" || xdgSessionType == "wayland" {
		// Try Wayland detector
		det := wayland.NewDetector()
		if det.IsAvailable() {
			return det
		}
	}

	// Check if we're on X11
	display := os.Getenv("DISPLAY")
	if display != "" {
		// Try X11 detector
		det := x11.NewDetector()
		if det.IsAvailable() {
			return det
		}
	}

	return nil
}

// GetActiveApp returns the currently active application using the best available method
func (d *Detector) GetActiveApp() (*common.AppInfo, error) {
	if !d.initialized {
		return nil, fmt.Errorf("detector not initialized")
	}

	var windowErr error

	// Try window detection first (most accurate when available)
	if d.windowDetector != nil && d.windowDetector.IsAvailable() {
		if appInfo, err := d.getActiveAppFromWindow(); err == nil {
			d.lastSuccessfulMethod = "window"
			return appInfo, nil
		} else {
			windowErr = err
			// Don't log here - only log if all methods fail
		}
	}

	// Fall back to process detection
	if appInfo, err := d.processDetector.GetActiveApp(); err == nil {
		d.lastSuccessfulMethod = "process"

		// Try to enhance with window title if available
		if d.windowDetector != nil {
			if windowInfo, err := d.windowDetector.GetFocusedWindow(); err == nil && windowInfo != nil {
				if windowInfo.AppName == appInfo.AppName || windowInfo.ProcessName == appInfo.ProcessName {
					appInfo.WindowTitle = windowInfo.WindowTitle
					appInfo.Confidence = 0.9
					appInfo.DetectionMethod = "hybrid"
				}
			}
		}

		return appInfo, nil
	} else {
		// Both methods failed - now log the details
		if windowErr != nil {
			log.Printf("All detection methods failed - Window: %v, Process: %v", windowErr, err)
		} else {
			log.Printf("Process detection failed: %v", err)
		}
	}

	return nil, fmt.Errorf("all detection methods failed")
}

// getActiveAppFromWindow uses window detection
func (d *Detector) getActiveAppFromWindow() (*common.AppInfo, error) {
	windowInfo, err := d.windowDetector.GetFocusedWindow()
	if err != nil {
		return nil, err
	}

	if windowInfo == nil || windowInfo.AppName == "" || windowInfo.AppName == "Unknown" {
		return nil, fmt.Errorf("no valid window information")
	}

	return &common.AppInfo{
		AppName:         windowInfo.AppName,
		WindowTitle:     windowInfo.WindowTitle,
		ProcessName:     windowInfo.ProcessName,
		LastActivity:    time.Now(),
		Confidence:      1.0, // Window detection is most accurate
		DetectionMethod: "window",
	}, nil
}

// IsAvailable checks if any detection method is available
func (d *Detector) IsAvailable() bool {
	if d.windowDetector != nil && d.windowDetector.IsAvailable() {
		return true
	}
	if d.processDetector != nil && d.processDetector.IsAvailable() {
		return true
	}
	return false
}

// GetPriority returns priority (highest since it combines methods)
func (d *Detector) GetPriority() int {
	return 100
}

// Initialize is a no-op (initialization happens in NewDetector)
func (d *Detector) Initialize() error {
	return nil
}

// Close cleans up resources
func (d *Detector) Close() error {
	if d.windowDetector != nil {
		if err := d.windowDetector.Close(); err != nil {
			log.Printf("Error closing window detector: %v", err)
		}
	}
	if d.processDetector != nil {
		if err := d.processDetector.Close(); err != nil {
			log.Printf("Error closing process detector: %v", err)
		}
	}
	return nil
}

// GetIdleInfo returns system idle information
func (d *Detector) GetIdleInfo() (*window.IdleInfo, error) {
	// Try window detector first
	if d.windowDetector != nil && d.windowDetector.IsAvailable() {
		if info, err := d.windowDetector.GetIdleInfo(); err == nil {
			return info, nil
		}
	}

	// Fallback: basic idle detection
	return &window.IdleInfo{
		IsIdle:   false,
		IsLocked: d.isScreenLocked(),
		IdleTime: 0,
	}, nil
}

// isScreenLocked checks if screen is locked using common methods
func (d *Detector) isScreenLocked() bool {
	// Check loginctl
	cmd := exec.Command("loginctl", "show-session", "-p", "LockedHint")
	if output, err := cmd.Output(); err == nil {
		if strings.Contains(string(output), "LockedHint=yes") {
			return true
		}
	}

	// Check for lock screen processes
	lockers := []string{"swaylock", "waylock", "gtklock", "hyprlock", "gnome-screensaver-dialog", "kscreenlocker"}
	for _, locker := range lockers {
		cmd := exec.Command("pgrep", "-x", locker)
		if err := cmd.Run(); err == nil {
			return true
		}
	}

	return false
}

// GetAllDetectors returns information about all available detectors
func (d *Detector) GetAllDetectors() []DetectorInfo {
	var detectors []DetectorInfo

	if d.windowDetector != nil {
		detectors = append(detectors, DetectorInfo{
			Name:      "Window Detector",
			Type:      "window",
			Available: d.windowDetector.IsAvailable(),
			Priority:  100,
			Method:    d.windowDetector.GetDisplayServer(),
		})
	}

	if d.processDetector != nil {
		detectors = append(detectors, DetectorInfo{
			Name:      "Process Detector",
			Type:      "process",
			Available: d.processDetector.IsAvailable(),
			Priority:  d.processDetector.GetPriority(),
			Method:    "process-based",
		})
	}

	// Sort by priority descending
	sort.Slice(detectors, func(i, j int) bool {
		return detectors[i].Priority > detectors[j].Priority
	})

	return detectors
}

// DetectorInfo provides information about a detector
type DetectorInfo struct {
	Name      string
	Type      string
	Available bool
	Priority  int
	Method    string
}

// GetStatus returns a status string for debugging
func (d *Detector) GetStatus() string {
	status := "Hybrid Detector Status:\n"

	if d.windowDetector != nil {
		status += fmt.Sprintf("  Window Detector: %s (available: %v)\n",
			d.windowDetector.GetDisplayServer(),
			d.windowDetector.IsAvailable())
	} else {
		status += "  Window Detector: unavailable\n"
	}

	if d.processDetector != nil {
		status += " Process Detector: available\n"
	} else {
		status += "  Process Detector: unavailable\n"
	}

	status += fmt.Sprintf("  Last successful method: %s\n", d.lastSuccessfulMethod)

	return status
}

// GetDisplayServer returns the display server name
func (d *Detector) GetDisplayServer() string {
	if d.windowDetector != nil {
		return d.windowDetector.GetDisplayServer()
	}
	return "process-based"
}

// GetFocusedWindow returns window information (compatible with window.Detector interface)
func (d *Detector) GetFocusedWindow() (*window.WindowInfo, error) {
	appInfo, err := d.GetActiveApp()
	if err != nil {
		return nil, err
	}

	// Convert AppInfo to WindowInfo
	return &window.WindowInfo{
		AppName:       appInfo.AppName,
		WindowTitle:   appInfo.WindowTitle,
		ProcessName:   appInfo.ProcessName,
		DisplayServer: d.GetDisplayServer(),
	}, nil
}
