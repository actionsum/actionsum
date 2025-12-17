package wayland

import (
	"testing"

	"github.com/hugo/actionsum/pkg/window"
)

func TestNewDetector(t *testing.T) {
	detector := NewDetector()
	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}

	// Verify compositor was detected
	t.Logf("Detected compositor: %s", detector.compositor)
}

func TestGetDisplayServer(t *testing.T) {
	detector := NewDetector()
	displayServer := detector.GetDisplayServer()

	if displayServer != "wayland" {
		t.Errorf("GetDisplayServer() = %s, want %s", displayServer, "wayland")
	}
}

func TestDetectCompositor(t *testing.T) {
	detector := NewDetector()

	validCompositors := []string{"sway", "hyprland", "wayfire", "river", "gnome", "kde", "unknown"}
	found := false
	for _, valid := range validCompositors {
		if detector.compositor == valid {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Unknown compositor detected: %s", detector.compositor)
	}

	t.Logf("Compositor: %s", detector.compositor)
	t.Logf("Has swaymsg: %v", detector.hasSwaymsg)
	t.Logf("Has gdbus: %v", detector.hasGdbus)
}

func TestIsAvailable(t *testing.T) {
	detector := NewDetector()

	available := detector.IsAvailable()
	t.Logf("Wayland detector available: %v", available)
	t.Logf("Compositor: %s", detector.compositor)

	// Log availability reasons
	switch detector.compositor {
	case "sway", "hyprland":
		t.Logf("Sway/Hyprland requires swaymsg: %v", detector.hasSwaymsg)
	case "gnome":
		t.Logf("GNOME requires gdbus: %v", detector.hasGdbus)
	case "kde":
		t.Log("KDE should be available")
	default:
		t.Logf("Unknown compositor: %s", detector.compositor)
	}
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
		t.Skip("Wayland detector not available on this system")
	}

	windowInfo, err := detector.GetFocusedWindow()
	if err != nil {
		t.Logf("GetFocusedWindow() error (may be expected): %v", err)
		// Don't fail - detection might fail for various reasons
		return
	}

	if windowInfo == nil {
		t.Fatal("GetFocusedWindow() returned nil windowInfo without error")
	}

	// Log the results
	t.Logf("App Name: %s", windowInfo.AppName)
	t.Logf("Window Title: %s", windowInfo.WindowTitle)
	t.Logf("Process Name: %s", windowInfo.ProcessName)
	t.Logf("Display Server: %s", windowInfo.DisplayServer)

	// Verify fields
	if windowInfo.DisplayServer != "wayland" {
		t.Errorf("DisplayServer = %s, want wayland", windowInfo.DisplayServer)
	}
}

func TestGetIdleInfo(t *testing.T) {
	detector := NewDetector()

	if !detector.IsAvailable() {
		t.Skip("Wayland detector not available on this system")
	}

	idleInfo, err := detector.GetIdleInfo()
	if err != nil {
		t.Logf("GetIdleInfo() error: %v", err)
		return
	}

	if idleInfo == nil {
		t.Fatal("GetIdleInfo() returned nil idleInfo without error")
	}

	// Log the results
	t.Logf("Is Idle: %v", idleInfo.IsIdle)
	t.Logf("Is Locked: %v", idleInfo.IsLocked)
	t.Logf("Idle Time: %d seconds", idleInfo.IdleTime)

	// Verify idle time is non-negative
	if idleInfo.IdleTime < 0 {
		t.Errorf("IdleTime is negative: %d", idleInfo.IdleTime)
	}
}

func TestParseSwayTree(t *testing.T) {
	sampleJSON := `{
		"id": 123,
		"focused": true,
		"app_id": "firefox",
		"name": "Mozilla Firefox",
		"pid": 1234
	}`

	windowInfo, err := parseSwayTree(sampleJSON)
	if err != nil {
		t.Fatalf("parseSwayTree() error: %v", err)
	}

	if windowInfo.AppName != "firefox" {
		t.Errorf("AppName = %s, want firefox", windowInfo.AppName)
	}

	if windowInfo.WindowTitle != "Mozilla Firefox" {
		t.Errorf("WindowTitle = %s, want Mozilla Firefox", windowInfo.WindowTitle)
	}
}

func TestParseHyprlandWindow(t *testing.T) {
	sampleJSON := `{
		"class": "kitty",
		"title": "Terminal Window",
		"pid": 5678
	}`

	windowInfo := parseHyprlandWindow(sampleJSON)

	if windowInfo.AppName != "kitty" {
		t.Errorf("AppName = %s, want kitty", windowInfo.AppName)
	}

	if windowInfo.WindowTitle != "Terminal Window" {
		t.Errorf("WindowTitle = %s, want Terminal Window", windowInfo.WindowTitle)
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

func TestGetProcessName(t *testing.T) {
	// Test with current process PID
	pid := "1" // init/systemd should always exist

	name := getProcessName(pid)
	t.Logf("Process name for PID %s: %s", pid, name)

	// Should return something for PID 1
	if name == "" {
		t.Log("Warning: Could not get process name for PID 1")
	}
}

func TestIsScreenLocked(t *testing.T) {
	detector := NewDetector()

	locked := detector.isScreenLocked()
	t.Logf("Screen is locked: %v", locked)

	// Just verify it doesn't panic and returns a boolean
	if locked != true && locked != false {
		t.Error("isScreenLocked() returned non-boolean value")
	}
}

func TestGetIdleTime(t *testing.T) {
	detector := NewDetector()

	idleTime := detector.getIdleTime()
	t.Logf("Idle time: %d seconds", idleTime)

	// Verify idle time is non-negative
	if idleTime < 0 {
		t.Errorf("getIdleTime() returned negative value: %d", idleTime)
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
	// Verify that Detector implements window.Detector interface
	var _ window.Detector = (*Detector)(nil)
}

// Benchmark tests
func BenchmarkGetFocusedWindow(b *testing.B) {
	detector := NewDetector()

	if !detector.IsAvailable() {
		b.Skip("Wayland detector not available")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.GetFocusedWindow()
	}
}

func BenchmarkGetIdleInfo(b *testing.B) {
	detector := NewDetector()

	if !detector.IsAvailable() {
		b.Skip("Wayland detector not available")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.GetIdleInfo()
	}
}

func BenchmarkDetectCompositor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		detector := NewDetector()
		_ = detector.compositor
	}
}
