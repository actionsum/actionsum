package common

import "time"

type AppInfo struct {
	AppName string

	WindowTitle string

	ProcessName string

	PID int

	LastActivity time.Time

	Confidence float64

	DetectionMethod string
}

type Detector interface {
	GetActiveApp() (*AppInfo, error)

	IsAvailable() bool

	GetPriority() int

	Initialize() error

	Close() error
}
