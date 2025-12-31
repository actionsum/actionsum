package window

type WindowInfo struct {
	AppName       string
	WindowTitle   string
	ProcessName   string
	DisplayServer string // "x11" or "wayland"
}

type IdleInfo struct {
	IsIdle   bool
	IsLocked bool
	IdleTime int64 // Idle time in seconds
}

type Detector interface {
	GetFocusedWindow() (*WindowInfo, error)
	GetIdleInfo() (*IdleInfo, error)
	IsAvailable() bool
	GetDisplayServer() string
	Close() error
}
