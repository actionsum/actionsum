package daemon

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

// Daemon manages daemon process operations
type Daemon struct {
	pidFile string
}

// New creates a new daemon manager
func New(pidFile string) *Daemon {
	return &Daemon{pidFile: pidFile}
}

// WritePID writes the current process PID to the PID file
func (d *Daemon) WritePID() error {
	pid := os.Getpid()
	return os.WriteFile(d.pidFile, fmt.Appendf([]byte{}, "%d", pid), 0644)
}

// ReadPID reads the PID from the PID file
func (d *Daemon) ReadPID() (int, error) {
	data, err := os.ReadFile(d.pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

// RemovePID removes the PID file
func (d *Daemon) RemovePID() error {
	if err := os.Remove(d.pidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}

// IsRunning checks if a daemon process is running
func (d *Daemon) IsRunning() (bool, int, error) {
	pid, err := d.ReadPID()
	if err != nil {
		return false, 0, err
	}

	if pid == 0 {
		return false, 0, nil
	}

	// Check if process exists by sending signal 0
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0, nil
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist, clean up stale PID file
		d.RemovePID()
		return false, 0, nil
	}

	return true, pid, nil
}

// Stop stops the daemon process
func (d *Daemon) Stop() error {
	running, pid, err := d.IsRunning()
	if err != nil {
		return err
	}

	if !running {
		return fmt.Errorf("daemon is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Remove PID file
	return d.RemovePID()
}
