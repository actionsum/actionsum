package detector

import (
	"github.com/hugo/actionsum/pkg/integrations/hybrid"
	"github.com/hugo/actionsum/pkg/window"
	"os"
)

// New creates a new hybrid detector that works universally
// This is the V2 detector that combines window detection with process monitoring
func New() (window.Detector, error) {
	return hybrid.NewDetector()
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
