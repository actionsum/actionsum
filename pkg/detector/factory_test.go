package detector

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	detector, err := New()
	if err != nil {
		t.Logf("New() returned error (may be expected): %v", err)
		return
	}

	if detector == nil {
		t.Fatal("New() returned nil detector without error")
	}

	displayServer := detector.GetDisplayServer()
	t.Logf("Detected display server: %s", displayServer)

	if displayServer != "x11" && displayServer != "wayland" {
		t.Errorf("GetDisplayServer() = %s, want x11 or wayland", displayServer)
	}

	windowInfo, err := detector.GetFocusedWindow()
	if err != nil {
		t.Logf("GetFocusedWindow() error: %v", err)
	} else if windowInfo != nil {
		t.Logf("Current window: %s - %s", windowInfo.AppName, windowInfo.WindowTitle)
	}

	idleInfo, err := detector.GetIdleInfo()
	if err != nil {
		t.Logf("GetIdleInfo() error: %v", err)
	} else if idleInfo != nil {
		t.Logf("Idle state: idle=%v, locked=%v", idleInfo.IsIdle, idleInfo.IsLocked)
	}

	if err := detector.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestDetectDisplayServer(t *testing.T) {
	tests := []struct {
		name             string
		sessionType      string
		waylandDisplay   string
		x11Display       string
		expectedContains string
	}{
		{
			name:             "Wayland session",
			sessionType:      "wayland",
			waylandDisplay:   "wayland-0",
			x11Display:       "",
			expectedContains: "wayland",
		},
		{
			name:             "X11 session",
			sessionType:      "x11",
			waylandDisplay:   "",
			x11Display:       ":0",
			expectedContains: "x11",
		},
		{
			name:             "Unknown session",
			sessionType:      "",
			waylandDisplay:   "",
			x11Display:       "",
			expectedContains: "unknown",
		},
		{
			name:             "Wayland display set",
			sessionType:      "",
			waylandDisplay:   "wayland-1",
			x11Display:       "",
			expectedContains: "wayland",
		},
		{
			name:             "X11 display set",
			sessionType:      "",
			waylandDisplay:   "",
			x11Display:       ":1",
			expectedContains: "x11",
		},
	}

	origSessionType := os.Getenv("XDG_SESSION_TYPE")
	origWaylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	origX11Display := os.Getenv("DISPLAY")

	defer func() {
		os.Setenv("XDG_SESSION_TYPE", origSessionType)
		os.Setenv("WAYLAND_DISPLAY", origWaylandDisplay)
		os.Setenv("DISPLAY", origX11Display)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("XDG_SESSION_TYPE", tt.sessionType)
			os.Setenv("WAYLAND_DISPLAY", tt.waylandDisplay)
			os.Setenv("DISPLAY", tt.x11Display)

			result := DetectDisplayServer()
			if result != tt.expectedContains {
				t.Errorf("DetectDisplayServer() = %s, want %s", result, tt.expectedContains)
			}
		})
	}
}

func TestNewWithUnsupportedSystem(t *testing.T) {
	origSessionType := os.Getenv("XDG_SESSION_TYPE")
	origWaylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	origX11Display := os.Getenv("DISPLAY")

	defer func() {
		os.Setenv("XDG_SESSION_TYPE", origSessionType)
		os.Setenv("WAYLAND_DISPLAY", origWaylandDisplay)
		os.Setenv("DISPLAY", origX11Display)
	}()

	os.Unsetenv("XDG_SESSION_TYPE")
	os.Unsetenv("WAYLAND_DISPLAY")
	os.Unsetenv("DISPLAY")

	detector, err := New()

	if err != nil {
		t.Logf("New() correctly returned error when no display server detected: %v", err)
	} else if detector != nil {
		t.Logf("New() succeeded even without display server env vars (tools available)")
		detector.Close()
	}
}

func TestMultipleDetectorInstances(t *testing.T) {
	detector1, err := New()
	if err != nil {
		t.Skip("Display server not available")
	}
	defer detector1.Close()

	detector2, err := New()
	if err != nil {
		t.Skip("Display server not available")
	}
	defer detector2.Close()

	ds1 := detector1.GetDisplayServer()
	ds2 := detector2.GetDisplayServer()

	if ds1 != ds2 {
		t.Errorf("Display servers don't match: %s vs %s", ds1, ds2)
	}

	t.Logf("Successfully created multiple detector instances")
}
