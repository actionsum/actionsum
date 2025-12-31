package hybrid

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/actionsum/actionsum/pkg/integrations/common"
	"github.com/actionsum/actionsum/pkg/integrations/process"
	"github.com/actionsum/actionsum/pkg/integrations/wayland"
	"github.com/actionsum/actionsum/pkg/integrations/x11"
	"github.com/actionsum/actionsum/pkg/window"
)

type Detector struct {
	windowDetector window.Detector

	processDetector *process.Detector

	lastSuccessfulMethod string

	windowCache map[int]string // PID -> window title

	initialized bool
}

func NewDetector() (*Detector, error) {
	d := &Detector{
		windowCache: make(map[int]string),
	}

	windowDet := detectWindowDetector()
	if windowDet != nil {
		d.windowDetector = windowDet
		log.Printf("Window detector initialized: %s", windowDet.GetDisplayServer())
	} else {
		log.Printf("Window detector unavailable, using process-based detection only")
	}

	d.processDetector = process.NewDetector()
	if err := d.processDetector.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize process detector: %w", err)
	}

	d.initialized = true
	return d, nil
}

func detectWindowDetector() window.Detector {
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	xdgSessionType := os.Getenv("XDG_SESSION_TYPE")

	if waylandDisplay != "" || xdgSessionType == "wayland" {
		det := wayland.NewDetector()
		if det.IsAvailable() {
			return det
		}
	}

	display := os.Getenv("DISPLAY")
	if display != "" {
		det := x11.NewDetector()
		if det.IsAvailable() {
			return det
		}
	}

	return nil
}

func (d *Detector) GetActiveApp() (*common.AppInfo, error) {
	if !d.initialized {
		return nil, fmt.Errorf("detector not initialized")
	}

	var windowErr error

	if d.windowDetector != nil && d.windowDetector.IsAvailable() {
		if appInfo, err := d.getActiveAppFromWindow(); err == nil {
			d.lastSuccessfulMethod = "window"
			return appInfo, nil
		} else {
			windowErr = err
		}
	}

	if appInfo, err := d.processDetector.GetActiveApp(); err == nil {
		d.lastSuccessfulMethod = "process"

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
		if windowErr != nil {
			log.Printf("All detection methods failed - Window: %v, Process: %v", windowErr, err)
		} else {
			log.Printf("Process detection failed: %v", err)
		}
	}

	return nil, fmt.Errorf("all detection methods failed")
}

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

func (d *Detector) IsAvailable() bool {
	if d.windowDetector != nil && d.windowDetector.IsAvailable() {
		return true
	}
	if d.processDetector != nil && d.processDetector.IsAvailable() {
		return true
	}
	return false
}

func (d *Detector) GetPriority() int {
	return 100
}

func (d *Detector) Initialize() error {
	return nil
}

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

func (d *Detector) GetIdleInfo() (*window.IdleInfo, error) {
	if d.windowDetector != nil && d.windowDetector.IsAvailable() {
		if info, err := d.windowDetector.GetIdleInfo(); err == nil {
			return info, nil
		}
	}

	return &window.IdleInfo{
		IsIdle:   false,
		IsLocked: d.isScreenLocked(),
		IdleTime: 0,
	}, nil
}

func (d *Detector) isScreenLocked() bool {
	cmd := exec.Command("gdbus", "call", "--session", "--dest", "org.gnome.ScreenSaver", "--object-path", "/org/gnome/ScreenSaver", "--method", "org.gnome.ScreenSaver.GetActive")
	if output, err := cmd.Output(); err == nil {
		if strings.Contains(string(output), "true") {
			return true
		}
	}

	cmd = exec.Command("loginctl", "show-session", "-p", "LockedHint")
	if output, err := cmd.Output(); err == nil {
		if strings.Contains(string(output), "LockedHint=yes") {
			return true
		}
	}

	return false
}

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

	sort.Slice(detectors, func(i, j int) bool {
		return detectors[i].Priority > detectors[j].Priority
	})

	return detectors
}

type DetectorInfo struct {
	Name      string
	Type      string
	Available bool
	Priority  int
	Method    string
}

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

func (d *Detector) GetDisplayServer() string {
	if d.windowDetector != nil {
		return d.windowDetector.GetDisplayServer()
	}
	return "process-based"
}

func (d *Detector) GetFocusedWindow() (*window.WindowInfo, error) {
	appInfo, err := d.GetActiveApp()
	if err != nil {
		return nil, err
	}

	return &window.WindowInfo{
		AppName:       appInfo.AppName,
		WindowTitle:   appInfo.WindowTitle,
		ProcessName:   appInfo.ProcessName,
		DisplayServer: d.GetDisplayServer(),
	}, nil
}
