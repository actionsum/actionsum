package window

// WindowInfo represents information about the currently focused window
type WindowInfo struct {
	AppName       string
	WindowTitle   string
	ProcessName   string
	DisplayServer string // "x11" or "wayland"
}

// IdleInfo represents system idle/lock state
type IdleInfo struct {
	IsIdle   bool
	IsLocked bool
	IdleTime int64 // Idle time in seconds
}

// Detector is the interface that all window detection implementations must satisfy
type Detector interface {
	// GetFocusedWindow returns information about the currently focused window
	GetFocusedWindow() (*WindowInfo, error)

	// GetIdleInfo returns information about system idle/lock state
	GetIdleInfo() (*IdleInfo, error)

	// IsAvailable checks if this detector can run on the current system
	IsAvailable() bool

	// GetDisplayServer returns the display server type ("x11" or "wayland")
	GetDisplayServer() string

	// Close cleans up any resources used by the detector
	Close() error
}
