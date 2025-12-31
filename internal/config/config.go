package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Database DatabaseConfig

	Tracker TrackerConfig

	Daemon DaemonConfig

	Report ReportConfig

	Web WebConfig
}

type DatabaseConfig struct {
	Path string
}

type TrackerConfig struct {
	PollInterval    time.Duration
	MinPollInterval time.Duration
	MaxPollInterval time.Duration
	IdleThreshold   time.Duration
}

type DaemonConfig struct {
	PIDFile string
}

type ReportConfig struct {
	ExcludeIdle bool
	TimeZone    string
}

type WebConfig struct {
	Host string
	Port int
}

func Default() *Config {
	return &Config{
		Database: DatabaseConfig{
			Path: "",
		},
		Tracker: TrackerConfig{
			PollInterval:    10 * time.Second,
			MinPollInterval: 10 * time.Second,
			MaxPollInterval: 300 * time.Second,
			IdleThreshold:   300 * time.Second,
		},
		Daemon: DaemonConfig{
			PIDFile: fmt.Sprintf("/tmp/actionsum-%d.pid", os.Getuid()),
		},
		Report: ReportConfig{
			ExcludeIdle: true,
			TimeZone:    "Local",
		},
		Web: WebConfig{
			Host: "localhost",
			Port: 10000 + os.Getuid(),
		},
	}
}

func (c *Config) Validate() error {
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

	if c.Web.Port < 1 || c.Web.Port > 65535 {
		return fmt.Errorf("web port must be between 1 and 65535, got %d", c.Web.Port)
	}

	if c.Web.Host == "" {
		return fmt.Errorf("web host cannot be empty")
	}

	if c.Daemon.PIDFile == "" {
		return fmt.Errorf("PID file path cannot be empty")
	}

	return nil
}

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

func (c *Config) SetWebPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	c.Web.Port = port
	return nil
}

func (c *Config) GetPollIntervalSeconds() int64 {
	return int64(c.Tracker.PollInterval.Seconds())
}

func (c *Config) GetIdleThresholdSeconds() int64 {
	return int64(c.Tracker.IdleThreshold.Seconds())
}

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
