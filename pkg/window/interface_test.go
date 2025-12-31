package window

import (
	"testing"
	"time"
)

type MockDetector struct {
	windowInfo    *WindowInfo
	idleInfo      *IdleInfo
	isAvailable   bool
	displayServer string
	closeError    error
}

func (m *MockDetector) GetFocusedWindow() (*WindowInfo, error) {
	return m.windowInfo, nil
}

func (m *MockDetector) GetIdleInfo() (*IdleInfo, error) {
	return m.idleInfo, nil
}

func (m *MockDetector) IsAvailable() bool {
	return m.isAvailable
}

func (m *MockDetector) GetDisplayServer() string {
	return m.displayServer
}

func (m *MockDetector) Close() error {
	return m.closeError
}

func TestMockDetector(t *testing.T) {
	var _ Detector = (*MockDetector)(nil)

	mock := &MockDetector{
		windowInfo: &WindowInfo{
			AppName:       "TestApp",
			WindowTitle:   "Test Window",
			ProcessName:   "test",
			DisplayServer: "x11",
		},
		idleInfo: &IdleInfo{
			IsIdle:   false,
			IsLocked: false,
			IdleTime: 0,
		},
		isAvailable:   true,
		displayServer: "x11",
	}

	windowInfo, err := mock.GetFocusedWindow()
	if err != nil {
		t.Errorf("GetFocusedWindow() error: %v", err)
	}
	if windowInfo.AppName != "TestApp" {
		t.Errorf("AppName = %s, want TestApp", windowInfo.AppName)
	}

	idleInfo, err := mock.GetIdleInfo()
	if err != nil {
		t.Errorf("GetIdleInfo() error: %v", err)
	}
	if idleInfo.IsIdle {
		t.Error("IsIdle = true, want false")
	}

	if !mock.IsAvailable() {
		t.Error("IsAvailable() = false, want true")
	}

	if mock.GetDisplayServer() != "x11" {
		t.Errorf("GetDisplayServer() = %s, want x11", mock.GetDisplayServer())
	}

	if err := mock.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestWindowInfo(t *testing.T) {
	info := WindowInfo{
		AppName:       "Firefox",
		WindowTitle:   "Mozilla Firefox",
		ProcessName:   "firefox",
		DisplayServer: "wayland",
	}

	if info.AppName != "Firefox" {
		t.Errorf("AppName = %s, want Firefox", info.AppName)
	}
	if info.WindowTitle != "Mozilla Firefox" {
		t.Errorf("WindowTitle = %s, want Mozilla Firefox", info.WindowTitle)
	}
	if info.ProcessName != "firefox" {
		t.Errorf("ProcessName = %s, want firefox", info.ProcessName)
	}
	if info.DisplayServer != "wayland" {
		t.Errorf("DisplayServer = %s, want wayland", info.DisplayServer)
	}
}

func TestIdleInfo(t *testing.T) {
	tests := []struct {
		name     string
		info     IdleInfo
		wantIdle bool
	}{
		{
			name: "Not idle",
			info: IdleInfo{
				IsIdle:   false,
				IsLocked: false,
				IdleTime: 30,
			},
			wantIdle: false,
		},
		{
			name: "Idle",
			info: IdleInfo{
				IsIdle:   true,
				IsLocked: false,
				IdleTime: 600,
			},
			wantIdle: true,
		},
		{
			name: "Locked",
			info: IdleInfo{
				IsIdle:   false,
				IsLocked: true,
				IdleTime: 0,
			},
			wantIdle: false,
		},
		{
			name: "Idle and locked",
			info: IdleInfo{
				IsIdle:   true,
				IsLocked: true,
				IdleTime: 900,
			},
			wantIdle: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.info.IsIdle != tt.wantIdle {
				t.Errorf("IsIdle = %v, want %v", tt.info.IsIdle, tt.wantIdle)
			}

			if tt.info.IdleTime < 0 {
				t.Errorf("IdleTime is negative: %d", tt.info.IdleTime)
			}
		})
	}
}

func TestIdleThresholds(t *testing.T) {
	tests := []struct {
		idleTime  int64
		threshold int64
		wantIdle  bool
	}{
		{idleTime: 0, threshold: 300, wantIdle: false},
		{idleTime: 100, threshold: 300, wantIdle: false},
		{idleTime: 299, threshold: 300, wantIdle: false},
		{idleTime: 300, threshold: 300, wantIdle: false}, // Equal to threshold
		{idleTime: 301, threshold: 300, wantIdle: true},
		{idleTime: 600, threshold: 300, wantIdle: true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			isIdle := tt.idleTime > tt.threshold
			if isIdle != tt.wantIdle {
				t.Errorf("idleTime %d > threshold %d = %v, want %v",
					tt.idleTime, tt.threshold, isIdle, tt.wantIdle)
			}
		})
	}
}

func BenchmarkWindowInfoCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = WindowInfo{
			AppName:       "TestApp",
			WindowTitle:   "Test Window",
			ProcessName:   "test",
			DisplayServer: "x11",
		}
	}
}

func BenchmarkIdleInfoCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = IdleInfo{
			IsIdle:   false,
			IsLocked: false,
			IdleTime: 0,
		}
	}
}

func ExampleDetector() {
	mock := &MockDetector{
		windowInfo: &WindowInfo{
			AppName:       "Firefox",
			WindowTitle:   "Example Page",
			ProcessName:   "firefox",
			DisplayServer: "x11",
		},
		idleInfo: &IdleInfo{
			IsIdle:   false,
			IsLocked: false,
			IdleTime: 30,
		},
		isAvailable:   true,
		displayServer: "x11",
	}

	if mock.IsAvailable() {
		windowInfo, _ := mock.GetFocusedWindow()
		println("Current app:", windowInfo.AppName)
	}
}

func TestDetectorLifecycle(t *testing.T) {
	mock := &MockDetector{
		windowInfo: &WindowInfo{
			AppName:       "TestApp",
			WindowTitle:   "Test",
			ProcessName:   "test",
			DisplayServer: "x11",
		},
		idleInfo: &IdleInfo{
			IsIdle:   false,
			IsLocked: false,
			IdleTime: 0,
		},
		isAvailable:   true,
		displayServer: "x11",
	}

	if !mock.IsAvailable() {
		t.Fatal("Detector should be available")
	}

	for i := 0; i < 5; i++ {
		_, err := mock.GetFocusedWindow()
		if err != nil {
			t.Errorf("Iteration %d: GetFocusedWindow() error: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := mock.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}
