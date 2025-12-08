package config_test

import (
	"fmt"
	"time"

	"actionsum/internal/config"
)

// Example of creating a default configuration
func ExampleDefault() {
	cfg := config.Default()
	fmt.Println("Poll Interval:", cfg.Tracker.PollInterval)
	fmt.Println("Web Port:", cfg.Web.Port)
	// Output:
	// Poll Interval: 1m0s
	// Web Port: 8080
}

// Example of creating configuration with environment variables
func ExampleNew() {
	cfg := config.New()
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	fmt.Println("Configuration loaded successfully")
	// Output:
	// Configuration loaded successfully
}

// Example of setting poll interval with validation
func ExampleConfig_SetPollInterval() {
	cfg := config.Default()

	// Valid interval
	if err := cfg.SetPollInterval(30 * time.Second); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Poll interval set to:", cfg.Tracker.PollInterval)
	}

	// Invalid interval (too low)
	if err := cfg.SetPollInterval(5 * time.Second); err != nil {
		fmt.Println("Error:", err)
	}

	// Output:
	// Poll interval set to: 30s
	// Error: poll interval cannot be less than 10s
}

// Example of validating configuration
func ExampleConfig_Validate() {
	cfg := config.Default()

	if err := cfg.Validate(); err != nil {
		fmt.Println("Invalid config:", err)
	} else {
		fmt.Println("Configuration is valid")
	}

	// Output:
	// Configuration is valid
}
