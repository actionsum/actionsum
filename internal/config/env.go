package config

import (
	"os"
	"strconv"
	"time"
)

// LoadFromEnv loads configuration from environment variables
// Environment variables override default values
func LoadFromEnv(cfg *Config) {
	// Database configuration
	if dbPath := os.Getenv("ACTIONSUM_DB_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}

	// Tracker configuration
	if pollInterval := os.Getenv("ACTIONSUM_POLL_INTERVAL"); pollInterval != "" {
		if seconds, err := strconv.Atoi(pollInterval); err == nil && seconds > 0 {
			interval := time.Duration(seconds) * time.Second
			if interval >= cfg.Tracker.MinPollInterval && interval <= cfg.Tracker.MaxPollInterval {
				cfg.Tracker.PollInterval = interval
			}
		}
	}

	if idleThreshold := os.Getenv("ACTIONSUM_IDLE_THRESHOLD"); idleThreshold != "" {
		if seconds, err := strconv.Atoi(idleThreshold); err == nil && seconds > 0 {
			cfg.Tracker.IdleThreshold = time.Duration(seconds) * time.Second
		}
	}

	// Daemon configuration
	if pidFile := os.Getenv("ACTIONSUM_PID_FILE"); pidFile != "" {
		cfg.Daemon.PIDFile = pidFile
	}

	// Report configuration
	if excludeIdle := os.Getenv("ACTIONSUM_EXCLUDE_IDLE"); excludeIdle != "" {
		if val, err := strconv.ParseBool(excludeIdle); err == nil {
			cfg.Report.ExcludeIdle = val
		}
	}

	if timeZone := os.Getenv("ACTIONSUM_TIMEZONE"); timeZone != "" {
		cfg.Report.TimeZone = timeZone
	}

	// Web configuration
	if webHost := os.Getenv("ACTIONSUM_WEB_HOST"); webHost != "" {
		cfg.Web.Host = webHost
	}

	if webPort := os.Getenv("ACTIONSUM_WEB_PORT"); webPort != "" {
		if port, err := strconv.Atoi(webPort); err == nil && port > 0 && port <= 65535 {
			cfg.Web.Port = port
		}
	}
}

// New creates a new Config with default values and loads from environment
func New() *Config {
	cfg := Default()
	LoadFromEnv(cfg)
	return cfg
}
