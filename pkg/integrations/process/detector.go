package process

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"actionsum/pkg/integrations/common"
)

// Detector monitors running processes to detect active applications
type Detector struct {
	sessionID      string
	lastScan       time.Time
	knownProcesses map[int]*processInfo
	guiApps        []string
	inputMonitor   *InputMonitor
	initialized    bool
}

type processInfo struct {
	pid         int
	name        string
	cmdline     string
	windowCount int
	lastSeen    time.Time
	cpuTime     uint64
}

// NewDetector creates a new process-based detector
func NewDetector() *Detector {
	return &Detector{
		knownProcesses: make(map[int]*processInfo),
		guiApps:        getCommonGUIApps(),
	}
}

// Initialize sets up the detector
func (d *Detector) Initialize() error {
	if d.initialized {
		return nil
	}

	// Get current session ID
	cmd := exec.Command("loginctl", "show-session", "-p", "Id", "--value")
	if output, err := cmd.Output(); err == nil {
		d.sessionID = strings.TrimSpace(string(output))
	}

	// Initialize input monitor
	d.inputMonitor = NewInputMonitor()
	if err := d.inputMonitor.Initialize(); err != nil {
		// Input monitoring is optional, log but don't fail
		fmt.Printf("Warning: input monitoring unavailable: %v\n", err)
	}

	d.initialized = true
	return nil
}

// GetActiveApp returns the currently active application
func (d *Detector) GetActiveApp() (*common.AppInfo, error) {
	if !d.initialized {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}

	// Scan all processes
	if err := d.scanProcesses(); err != nil {
		return nil, fmt.Errorf("failed to scan processes: %w", err)
	}

	// Get recently active PIDs from input monitor
	activePIDs := d.inputMonitor.GetRecentlyActivePIDs()

	// Score processes based on multiple factors
	candidates := d.scoreProcesses(activePIDs)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no GUI applications detected")
	}

	// Return the highest scoring candidate
	best := candidates[0]
	proc := d.knownProcesses[best.pid]

	return &common.AppInfo{
		AppName:         proc.name,
		WindowTitle:     getWindowTitleForPID(best.pid),
		ProcessName:     proc.name,
		PID:             best.pid,
		LastActivity:    proc.lastSeen,
		Confidence:      best.score,
		DetectionMethod: "process-based",
	}, nil
}

// IsAvailable checks if this detector can run
func (d *Detector) IsAvailable() bool {
	// Process monitoring works on all Linux systems
	_, err := os.Stat("/proc")
	return err == nil
}

// GetPriority returns the priority (lower than window detection)
func (d *Detector) GetPriority() int {
	return 50 // Lower priority than direct window detection (which would be 100)
}

// Close cleans up resources
func (d *Detector) Close() error {
	if d.inputMonitor != nil {
		return d.inputMonitor.Close()
	}
	return nil
}

// scanProcesses scans /proc for running processes
func (d *Detector) scanProcesses() error {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return err
	}

	now := time.Now()
	d.lastScan = now

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Only process numeric directories (PIDs)
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		// Read process info
		info, err := d.readProcessInfo(pid)
		if err != nil {
			continue
		}

		// Check if it's a GUI app
		if d.isGUIApp(info) {
			info.lastSeen = now
			d.knownProcesses[pid] = info
		}
	}

	// Clean up old processes
	for pid, proc := range d.knownProcesses {
		if now.Sub(proc.lastSeen) > 5*time.Second {
			delete(d.knownProcesses, pid)
		}
	}

	return nil
}

// readProcessInfo reads information about a process
func (d *Detector) readProcessInfo(pid int) (*processInfo, error) {
	info := &processInfo{pid: pid}

	// Read /proc/[pid]/stat for basic info
	statPath := filepath.Join("/proc", strconv.Itoa(pid), "stat")
	statData, err := os.ReadFile(statPath)
	if err != nil {
		return nil, err
	}

	// Parse stat file (process name is in parentheses)
	statStr := string(statData)
	startIdx := strings.Index(statStr, "(")
	endIdx := strings.LastIndex(statStr, ")")
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		info.name = statStr[startIdx+1 : endIdx]
	}

	// Read /proc/[pid]/cmdline for full command
	cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
	if cmdData, err := os.ReadFile(cmdlinePath); err == nil {
		info.cmdline = strings.ReplaceAll(string(cmdData), "\x00", " ")
	}

	return info, nil
}

// isGUIApp checks if a process is likely a GUI application
func (d *Detector) isGUIApp(info *processInfo) bool {
	// Check against known GUI apps
	for _, app := range d.guiApps {
		if info.name == app || strings.Contains(info.cmdline, app) {
			return true
		}
	}

	// Check if process has DISPLAY environment variable (indicates X11/XWayland)
	environPath := filepath.Join("/proc", strconv.Itoa(info.pid), "environ")
	if data, err := os.ReadFile(environPath); err == nil {
		environ := string(data)
		if strings.Contains(environ, "DISPLAY=") || strings.Contains(environ, "WAYLAND_DISPLAY=") {
			return true
		}
	}

	return false
}

type scoredProcess struct {
	pid   int
	score float64
}

// scoreProcesses assigns scores to processes based on activity
func (d *Detector) scoreProcesses(activePIDs map[int]time.Time) []scoredProcess {
	var scored []scoredProcess

	// Get current process's parent chain to identify which terminal we're running from
	myTerminalPID := d.findMyTerminal()

	for pid, proc := range d.knownProcesses {
		score := 0.0

		// Base score for being a known GUI app
		score += 0.3

		// CRITICAL: If this is the terminal we're running from, give it highest score
		if pid == myTerminalPID {
			score += 10.0 // Very high score ensures this wins
		}

		// Check if this process is a parent/ancestor of our current process
		if d.isAncestorProcess(pid) {
			score += 5.0 // High score for ancestor processes (e.g., VSCode, terminal)
		}

		// Score based on recent input activity
		if lastActive, ok := activePIDs[pid]; ok {
			timeSinceActive := time.Since(lastActive).Seconds()
			if timeSinceActive < 1.0 {
				score += 0.5
			} else if timeSinceActive < 5.0 {
				score += 0.3
			} else if timeSinceActive < 30.0 {
				score += 0.1
			}
		}

		// Score based on recency
		timeSinceSeen := time.Since(proc.lastSeen).Seconds()
		if timeSinceSeen < 1.0 {
			score += 0.2
		}

		scored = append(scored, scoredProcess{pid: pid, score: score})
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	return scored
}

// findMyTerminal finds the terminal emulator that is an ancestor of the current process
func (d *Detector) findMyTerminal() int {
	// Walk up the process tree to find a known terminal
	pid := os.Getpid()
	terminals := []string{"terminator", "gnome-terminal", "konsole", "alacritty", "kitty", "tilix", "xterm", "rxvt"}

	for pid > 1 {
		// Read parent PID
		statPath := fmt.Sprintf("/proc/%d/stat", pid)
		data, err := os.ReadFile(statPath)
		if err != nil {
			break
		}

		// Parse stat to get ppid (field 4)
		statStr := string(data)
		fields := strings.Fields(statStr)
		if len(fields) < 4 {
			break
		}

		ppid, err := strconv.Atoi(fields[3])
		if err != nil {
			break
		}

		// Check if parent is a known terminal
		cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", ppid)
		if cmdData, err := os.ReadFile(cmdlinePath); err == nil {
			cmdline := strings.ReplaceAll(string(cmdData), "\x00", " ")
			for _, term := range terminals {
				if strings.Contains(strings.ToLower(cmdline), term) {
					return ppid
				}
			}
		}

		pid = ppid
	}

	return 0
}

// isAncestorProcess checks if the given PID is an ancestor of the current process
func (d *Detector) isAncestorProcess(checkPID int) bool {
	pid := os.Getpid()

	for pid > 1 && pid != checkPID {
		// Read parent PID
		statPath := fmt.Sprintf("/proc/%d/stat", pid)
		data, err := os.ReadFile(statPath)
		if err != nil {
			return false
		}

		statStr := string(data)
		fields := strings.Fields(statStr)
		if len(fields) < 4 {
			return false
		}

		ppid, err := strconv.Atoi(fields[3])
		if err != nil {
			return false
		}

		if ppid == checkPID {
			return true
		}

		pid = ppid
	}

	return false
}

// getWindowTitleForPID attempts to get window title for a PID
func getWindowTitleForPID(pid int) string {
	// Try to use xprop to find window for this PID
	cmd := exec.Command("sh", "-c", fmt.Sprintf("xprop -root _NET_CLIENT_LIST | tr ',' '\\n' | while read w; do xprop -id $w _NET_WM_PID | grep -q %d && xprop -id $w WM_NAME; done | head -1", pid))
	if output, err := cmd.Output(); err == nil {
		title := string(output)
		if strings.Contains(title, "=") {
			parts := strings.SplitN(title, "=", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(parts[1]), "\"")
			}
		}
	}

	return "Unknown"
}

// getCommonGUIApps returns a list of common GUI application process names
func getCommonGUIApps() []string {
	return []string{
		// Browsers
		"firefox", "chrome", "chromium", "google-chrome", "brave", "opera", "vivaldi", "microsoft-edge",
		// Editors
		"code", "vscode", "sublime_text", "atom", "gedit", "vim", "nvim", "emacs",
		// Terminals
		"gnome-terminal", "konsole", "terminator", "alacritty", "kitty", "wezterm", "tilix",
		// Communication
		"slack", "discord", "telegram", "signal", "zoom", "teams",
		// Office
		"libreoffice", "soffice.bin", "writer", "calc", "impress",
		// Media
		"vlc", "mpv", "spotify", "rhythmbox", "totem",
		// File managers
		"nautilus", "dolphin", "thunar", "nemo", "caja",
		// IDEs
		"idea", "pycharm", "webstorm", "eclipse", "netbeans",
	}
}

// InputMonitor monitors input device activity
type InputMonitor struct {
	activePIDs map[int]time.Time
	stopChan   chan struct{}
	running    bool
}

// NewInputMonitor creates a new input monitor
func NewInputMonitor() *InputMonitor {
	return &InputMonitor{
		activePIDs: make(map[int]time.Time),
		stopChan:   make(chan struct{}),
	}
}

// Initialize starts monitoring input devices
func (im *InputMonitor) Initialize() error {
	// Check if we have access to /proc/bus/input/devices
	if _, err := os.Stat("/proc/bus/input/devices"); err != nil {
		return fmt.Errorf("cannot access input devices: %w", err)
	}

	// Start monitoring in background (simplified - just tracking timestamp)
	go im.monitor()
	im.running = true

	return nil
}

// monitor runs in background to track activity
func (im *InputMonitor) monitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-im.stopChan:
			return
		case <-ticker.C:
			// In a real implementation, this would monitor /dev/input/event* devices
			// For now, we use a heuristic: check which GUI processes are using CPU
			im.updateActivityFromCPU()
		}
	}
}

// updateActivityFromCPU checks which processes are using CPU (simplified heuristic)
func (im *InputMonitor) updateActivityFromCPU() {
	// Read top processes by CPU usage
	cmd := exec.Command("ps", "aux", "--sort=-pcpu")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	scanner.Scan() // Skip header

	count := 0
	now := time.Now()

	for scanner.Scan() && count < 10 {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 11 {
			continue
		}

		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		cpuStr := fields[2]
		cpu, err := strconv.ParseFloat(cpuStr, 64)
		if err != nil {
			continue
		}

		// If process is using CPU, mark as recently active
		if cpu > 0.5 {
			im.activePIDs[pid] = now
		}

		count++
	}

	// Clean up old entries
	for pid, lastSeen := range im.activePIDs {
		if now.Sub(lastSeen) > 30*time.Second {
			delete(im.activePIDs, pid)
		}
	}
}

// GetRecentlyActivePIDs returns PIDs that have been recently active
func (im *InputMonitor) GetRecentlyActivePIDs() map[int]time.Time {
	result := make(map[int]time.Time)
	for pid, t := range im.activePIDs {
		result[pid] = t
	}
	return result
}

// Close stops the monitor
func (im *InputMonitor) Close() error {
	if im.running {
		close(im.stopChan)
		im.running = false
	}
	return nil
}
