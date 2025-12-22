package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Database configuration
	Database DatabaseConfig

	// Tracker configuration
	Tracker TrackerConfig

	// Daemon configuration
	Daemon DaemonConfig

	// Report configuration
	Report ReportConfig

	// Web server configuration
	Web WebConfig
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Path string // Path to SQLite database file
}

// TrackerConfig holds tracking behavior configuration
type TrackerConfig struct {
	PollInterval    time.Duration // How often to check focused window
	MinPollInterval time.Duration // Minimum allowed poll interval
	MaxPollInterval time.Duration // Maximum allowed poll interval
	IdleThreshold   time.Duration // Time before considering user idle
}

// DaemonConfig holds daemon process configuration
type DaemonConfig struct {
	PIDFile string // Path to PID file for daemon management
}

// ReportConfig holds report generation configuration
type ReportConfig struct {
	ExcludeIdle bool // Whether to exclude idle/locked time from reports
	TimeZone    string
}

// WebConfig holds web server configuration
type WebConfig struct {
	Host string // Host to bind web server to
	Port int    // Port for web server
}

// Default returns a Config with sensible default values
func Default() *Config {
	return &Config{
		Database: DatabaseConfig{
			Path: "", // Empty means use default ~/.config/actionsum/actionsum.db
		},
		Tracker: TrackerConfig{
			PollInterval:    10 * time.Second,  // 10 seconds default
			MinPollInterval: 10 * time.Second,  // Minimum 10 seconds
			MaxPollInterval: 300 * time.Second, // Maximum allowed poll interval
			IdleThreshold:   300 * time.Second, // 5 minutes idle threshold
		},
		Daemon: DaemonConfig{
			PIDFile: fmt.Sprintf("/tmp/actionsum-%d.pid", os.Getuid()),
		},
		Report: ReportConfig{
			ExcludeIdle: true, // Exclude idle time by default
			TimeZone:    "Local",
		},
		Web: WebConfig{
			Host: "localhost",
			Port: 10000 + os.Getuid(), // Default port based on user PID
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate tracker intervals
	if c.Tracker.PollInterval < c.Tracker.MinPollInterval {
		return fmt.Errorf("poll interval (%v) cannot be less than minimum (%v)",
			c.Tracker.PollInterval, c.Tracker.MinPollInterval)
	}

	if c.Tracker.PollInterval > c.Tracker.MaxPollInterval {
		return fmt.Errorf("poll interval (%v) cannot be greater than maximum (%v)",
			c.Tracker.PollInterval, c.Tracker.MaxPollInterval)
	}

	if c.Tracker.IdleThreshold < 0 {
		return fmt.Errorf("idle threshold cannot be negative")
	}

	// Validate web config
	if c.Web.Port < 1 || c.Web.Port > 65535 {
		return fmt.Errorf("web port must be between 1 and 65535, got %d", c.Web.Port)
	}

	if c.Web.Host == "" {
		return fmt.Errorf("web host cannot be empty")
	}

	// Validate daemon config
	if c.Daemon.PIDFile == "" {
		return fmt.Errorf("PID file path cannot be empty")
	}

	return nil
}

// SetPollInterval sets the poll interval with validation
func (c *Config) SetPollInterval(interval time.Duration) error {
	if interval < c.Tracker.MinPollInterval {
		return fmt.Errorf("poll interval cannot be less than %v", c.Tracker.MinPollInterval)
	}
	if interval > c.Tracker.MaxPollInterval {
		return fmt.Errorf("poll interval cannot be greater than %v", c.Tracker.MaxPollInterval)
	}
	c.Tracker.PollInterval = interval
	return nil
}

// SetWebPort sets the web server port with validation
func (c *Config) SetWebPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	c.Web.Port = port
	return nil
}

// GetPollIntervalSeconds returns the poll interval in seconds
func (c *Config) GetPollIntervalSeconds() int64 {
	return int64(c.Tracker.PollInterval.Seconds())
}

// GetIdleThresholdSeconds returns the idle threshold in seconds
func (c *Config) GetIdleThresholdSeconds() int64 {
	return int64(c.Tracker.IdleThreshold.Seconds())
}

// String returns a string representation of the config
func (c *Config) String() string {
	return fmt.Sprintf(`Configuration:
  Database:
    Path: %s
  Tracker:
    Poll Interval: %v
    Min Interval: %v
    Max Interval: %v
    Idle Threshold: %v
  Daemon:
    PID File: %s
  Report:
    Exclude Idle: %v
    Time Zone: %s
  Web:
    Host: %s
    Port: %d`,
		c.Database.Path,
		c.Tracker.PollInterval,
		c.Tracker.MinPollInterval,
		c.Tracker.MaxPollInterval,
		c.Tracker.IdleThreshold,
		c.Daemon.PIDFile,
		c.Report.ExcludeIdle,
		c.Report.TimeZone,
		c.Web.Host,
		c.Web.Port,
	)
}
