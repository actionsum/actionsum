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

	"github.com/actionsum/actionsum/pkg/integrations/common"
)

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

func NewDetector() *Detector {
	return &Detector{
		knownProcesses: make(map[int]*processInfo),
		guiApps:        getCommonGUIApps(),
	}
}

func (d *Detector) Initialize() error {
	if d.initialized {
		return nil
	}

	cmd := exec.Command("gdbus", "call", "--session", "--dest", "org.gnome.ScreenSaver", "--object-path", "/org/gnome/ScreenSaver", "--method", "org.gnome.ScreenSaver.GetActive")
	if output, err := cmd.Output(); err == nil {
		d.sessionID = strings.TrimSpace(string(output))
	}

	d.inputMonitor = NewInputMonitor()
	if err := d.inputMonitor.Initialize(); err != nil {
		fmt.Printf("Warning: input monitoring unavailable: %v\n", err)
	}

	d.initialized = true
	return nil
}

func (d *Detector) GetActiveApp() (*common.AppInfo, error) {
	if !d.initialized {
		if err := d.Initialize(); err != nil {
			return nil, err
		}
	}

	if err := d.scanProcesses(); err != nil {
		return nil, fmt.Errorf("failed to scan processes: %w", err)
	}

	activePIDs := d.inputMonitor.GetRecentlyActivePIDs()

	candidates := d.scoreProcesses(activePIDs)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no GUI applications detected")
	}

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

func (d *Detector) IsAvailable() bool {
	_, err := os.Stat("/proc")
	return err == nil
}

func (d *Detector) GetPriority() int {
	return 50 // Lower priority than direct window detection (which would be 100)
}

func (d *Detector) Close() error {
	if d.inputMonitor != nil {
		return d.inputMonitor.Close()
	}
	return nil
}

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

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		info, err := d.readProcessInfo(pid)
		if err != nil {
			continue
		}

		if d.isGUIApp(info) {
			info.lastSeen = now
			d.knownProcesses[pid] = info
		}
	}

	for pid, proc := range d.knownProcesses {
		if now.Sub(proc.lastSeen) > 5*time.Second {
			delete(d.knownProcesses, pid)
		}
	}

	return nil
}

func (d *Detector) readProcessInfo(pid int) (*processInfo, error) {
	info := &processInfo{pid: pid}

	statPath := filepath.Join("/proc", strconv.Itoa(pid), "stat")
	statData, err := os.ReadFile(statPath)
	if err != nil {
		return nil, err
	}

	statStr := string(statData)
	startIdx := strings.Index(statStr, "(")
	endIdx := strings.LastIndex(statStr, ")")
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		info.name = statStr[startIdx+1 : endIdx]
	}

	cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
	if cmdData, err := os.ReadFile(cmdlinePath); err == nil {
		info.cmdline = strings.ReplaceAll(string(cmdData), "\x00", " ")
	}

	return info, nil
}

func (d *Detector) isGUIApp(info *processInfo) bool {
	blacklist := []string{
		"bash", "zsh", "fish", "sh", "dash", "tcsh", "ksh",
		"goa-daemon", "goa-identity-service", "gvfs", "dbus-daemon", "systemd",
		"pulseaudio", "pipewire", "wireplumber", "bluetoothd",
		"ssh-agent", "gpg-agent", "dconf-service",
	}

	for _, blocked := range blacklist {
		if info.name == blocked || strings.HasPrefix(info.name, blocked) {
			return false
		}
	}

	for _, app := range d.guiApps {
		if info.name == app || strings.Contains(info.cmdline, app) {
			return true
		}
	}

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

func (d *Detector) scoreProcesses(activePIDs map[int]time.Time) []scoredProcess {
	var scored []scoredProcess

	myTerminalPID := d.findMyTerminal()

	for pid, proc := range d.knownProcesses {
		score := 0.0

		score += 0.3

		if pid == myTerminalPID {
			score += 10.0 // Very high score ensures this wins
		}

		if d.isAncestorProcess(pid) {
			score += 5.0 // High score for ancestor processes (e.g., VSCode, terminal)
		}

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

		timeSinceSeen := time.Since(proc.lastSeen).Seconds()
		if timeSinceSeen < 1.0 {
			score += 0.2
		}

		scored = append(scored, scoredProcess{pid: pid, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	return scored
}

func (d *Detector) findMyTerminal() int {
	pid := os.Getpid()
	terminals := []string{"terminator", "gnome-terminal", "konsole", "alacritty", "kitty", "tilix", "xterm", "rxvt"}

	for pid > 1 {
		statPath := fmt.Sprintf("/proc/%d/stat", pid)
		data, err := os.ReadFile(statPath)
		if err != nil {
			break
		}

		statStr := string(data)
		fields := strings.Fields(statStr)
		if len(fields) < 4 {
			break
		}

		ppid, err := strconv.Atoi(fields[3])
		if err != nil {
			break
		}

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

func (d *Detector) isAncestorProcess(checkPID int) bool {
	pid := os.Getpid()

	for pid > 1 && pid != checkPID {
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

func getWindowTitleForPID(pid int) string {
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

func getCommonGUIApps() []string {
	return []string{
		"firefox", "chrome", "chromium", "google-chrome", "brave", "opera", "vivaldi", "microsoft-edge",
		"code", "vscode", "sublime_text", "atom", "gedit", "vim", "nvim", "emacs",
		"gnome-terminal", "konsole", "terminator", "alacritty", "kitty", "wezterm", "tilix",
		"slack", "discord", "telegram", "signal", "zoom", "teams",
		"libreoffice", "soffice.bin", "writer", "calc", "impress",
		"vlc", "mpv", "spotify", "rhythmbox", "totem",
		"nautilus", "dolphin", "thunar", "nemo", "caja",
		"idea", "pycharm", "webstorm", "eclipse", "netbeans",
	}
}

type InputMonitor struct {
	activePIDs map[int]time.Time
	stopChan   chan struct{}
	running    bool
}

func NewInputMonitor() *InputMonitor {
	return &InputMonitor{
		activePIDs: make(map[int]time.Time),
		stopChan:   make(chan struct{}),
	}
}

func (im *InputMonitor) Initialize() error {
	if _, err := os.Stat("/proc/bus/input/devices"); err != nil {
		return fmt.Errorf("cannot access input devices: %w", err)
	}

	go im.monitor()
	im.running = true

	return nil
}

func (im *InputMonitor) monitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-im.stopChan:
			return
		case <-ticker.C:
			im.updateActivityFromCPU()
		}
	}
}

func (im *InputMonitor) updateActivityFromCPU() {
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

		if cpu > 0.5 {
			im.activePIDs[pid] = now
		}

		count++
	}

	for pid, lastSeen := range im.activePIDs {
		if now.Sub(lastSeen) > 30*time.Second {
			delete(im.activePIDs, pid)
		}
	}
}

func (im *InputMonitor) GetRecentlyActivePIDs() map[int]time.Time {
	result := make(map[int]time.Time)
	for pid, t := range im.activePIDs {
		result[pid] = t
	}
	return result
}

func (im *InputMonitor) Close() error {
	if im.running {
		close(im.stopChan)
		im.running = false
	}
	return nil
}
