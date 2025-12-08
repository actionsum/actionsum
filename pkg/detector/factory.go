package detector

import (
	"fmt"
	"os"

	"actionsum/pkg/integrations/wayland"
	"actionsum/pkg/integrations/x11"
	"actionsum/pkg/window"
)

// New creates and returns the appropriate window detector for the current system
func New() (window.Detector, error) {
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	x11Display := os.Getenv("DISPLAY")

	var detector window.Detector

	if sessionType == "wayland" || waylandDisplay != "" {
		detector = wayland.NewDetector()
		if detector.IsAvailable() {
			return detector, nil
		}
	}

	if sessionType == "x11" || x11Display != "" {
		detector = x11.NewDetector()
		if detector.IsAvailable() {
			return detector, nil
		}
	}

	detector = wayland.NewDetector()
	if detector.IsAvailable() {
		return detector, nil
	}

	detector = x11.NewDetector()
	if detector.IsAvailable() {
		return detector, nil
	}

	return nil, fmt.Errorf("no supported display server found (tried X11 and Wayland)")
}

// DetectDisplayServer returns the detected display server type
func DetectDisplayServer() string {
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	x11Display := os.Getenv("DISPLAY")

	if sessionType == "wayland" || waylandDisplay != "" {
		return "wayland"
	}

	if sessionType == "x11" || x11Display != "" {
		return "x11"
	}

	return "unknown"
}
