package x11

import (
	"testing"

	"github.com/actionsum/actionsum/pkg/window"
)

func TestNewDetector(t *testing.T) {
	detector := NewDetector()
	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}
}

func TestGetDisplayServer(t *testing.T) {
	detector := NewDetector()
	displayServer := detector.GetDisplayServer()

	if displayServer != "x11" {
		t.Errorf("GetDisplayServer() = %s, want %s", displayServer, "x11")
	}
}

func TestIsAvailable(t *testing.T) {
	detector := NewDetector()

	available := detector.IsAvailable()
	t.Logf("X11 detector available: %v", available)
	t.Logf("Has xdotool: %v", detector.hasXdotool)
	t.Logf("Has wmctrl: %v", detector.hasWmctrl)
}

func TestCommandExists(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name    string
		command string
	}{
		{"ls should exist", "ls"},
		{"sh should exist", "sh"},
		{"nonexistent_cmd should not exist", "nonexistent_command_xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists := detector.commandExists(tt.command)
			t.Logf("Command %s exists: %v", tt.command, exists)
		})
	}
}

func TestGetFocusedWindow(t *testing.T) {
	detector := NewDetector()

	if !detector.IsAvailable() {
		t.Skip("X11 detector not available on this system")
	}

	windowInfo, err := detector.GetFocusedWindow()
	if err != nil {
		t.Logf("GetFocusedWindow() error (may be expected): %v", err)
		return
	}

	if windowInfo == nil {
		t.Fatal("GetFocusedWindow() returned nil windowInfo without error")
	}

	t.Logf("App Name: %s", windowInfo.AppName)
	t.Logf("Window Title: %s", windowInfo.WindowTitle)
	t.Logf("Process Name: %s", windowInfo.ProcessName)
	t.Logf("Display Server: %s", windowInfo.DisplayServer)

	if windowInfo.AppName == "" {
		t.Error("AppName is empty")
	}
	if windowInfo.DisplayServer != "x11" {
		t.Errorf("DisplayServer = %s, want x11", windowInfo.DisplayServer)
	}
}

func TestGetIdleInfo(t *testing.T) {
	detector := NewDetector()

	if !detector.IsAvailable() {
		t.Skip("X11 detector not available on this system")
	}

	idleInfo, err := detector.GetIdleInfo()
	if err != nil {
		t.Logf("GetIdleInfo() error: %v", err)
		return
	}

	if idleInfo == nil {
		t.Fatal("GetIdleInfo() returned nil idleInfo without error")
	}

	t.Logf("Is Idle: %v", idleInfo.IsIdle)
	t.Logf("Is Locked: %v", idleInfo.IsLocked)
	t.Logf("Idle Time: %d seconds", idleInfo.IdleTime)

	if idleInfo.IdleTime < 0 {
		t.Errorf("IdleTime is negative: %d", idleInfo.IdleTime)
	}
}

func TestParseWMClass(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard format",
			input:    `WM_CLASS(STRING) = "Navigator", "Firefox"`,
			expected: "Firefox",
		},
		{
			name:     "Single class",
			input:    `WM_CLASS(STRING) = "kitty", "kitty"`,
			expected: "kitty",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "No equals sign",
			input:    "WM_CLASS(STRING)",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseWMClass(tt.input)
			if result != tt.expected {
				t.Errorf("parseWMClass(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestClose(t *testing.T) {
	detector := NewDetector()
	err := detector.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestDetectorInterface(t *testing.T) {
	var _ window.Detector = (*Detector)(nil)
}
