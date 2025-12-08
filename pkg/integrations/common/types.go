package common

import "time"

// AppInfo represents information about a running application
type AppInfo struct {
	// AppName is the application identifier (e.g., "firefox", "code")
	AppName string

	// WindowTitle is the title of the focused window
	WindowTitle string

	// ProcessName is the actual process name
	ProcessName string

	// PID is the process ID
	PID int

	// LastActivity is when this app was last active
	LastActivity time.Time

	// Confidence is how confident we are this is the active app (0.0-1.0)
	Confidence float64

	// DetectionMethod describes how this was detected
	DetectionMethod string
}

// Detector is the universal interface for detecting active applications
type Detector interface {
	// GetActiveApp returns the currently active application
	GetActiveApp() (*AppInfo, error)

	// IsAvailable checks if this detector can run on the current system
	IsAvailable() bool

	// GetPriority returns the priority of this detector (higher = preferred)
	GetPriority() int

	// Initialize sets up the detector
	Initialize() error

	// Close cleans up resources
	Close() error
}
