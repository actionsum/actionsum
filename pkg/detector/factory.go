package detector

import (
	"os"

	"github.com/actionsum/actionsum/pkg/integrations/hybrid"
	"github.com/actionsum/actionsum/pkg/window"
)

func New() (window.Detector, error) {
	return hybrid.NewDetector()
}

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
