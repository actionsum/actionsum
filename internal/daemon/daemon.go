package daemon

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

type Daemon struct {
	pidFile string
}

func New(pidFile string) *Daemon {
	return &Daemon{pidFile: pidFile}
}

func (d *Daemon) WritePID() error {
	pid := os.Getpid()
	return os.WriteFile(d.pidFile, fmt.Appendf([]byte{}, "%d", pid), 0644)
}

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

func (d *Daemon) RemovePID() error {
	if err := os.Remove(d.pidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}

func (d *Daemon) IsRunning() (bool, int, error) {
	pid, err := d.ReadPID()
	if err != nil {
		return false, 0, err
	}

	if pid == 0 {
		return false, 0, nil
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0, nil
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		d.RemovePID()
		return false, 0, nil
	}

	return true, pid, nil
}

func (d *Daemon) Stop() error {
	running, pid, err := d.IsRunning()
	if err != nil {
		return fmt.Errorf("error checking daemon status: %w", err)
	}

	if !running {
		return fmt.Errorf("daemon is not running or PID file is stale")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		if err.Error() == "os: process already finished" {
			_ = d.RemovePID()
			return fmt.Errorf("daemon process already terminated")
		}
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	if err := d.RemovePID(); err != nil {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	return nil
}
