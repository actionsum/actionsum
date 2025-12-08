package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type FocusedWindow struct {
	Title    string
	AppName  string
	Class    string
	PID      uint32
	WindowID uint32
	Source   string
}

func GetFocusedWindow() (*FocusedWindow, error) {
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))

	if sessionType == "wayland" {
		if strings.Contains(desktop, "gnome") || strings.Contains(desktop, "ubuntu") {
			fw, err := getGnomeFocusedWindow()
			if err == nil && fw.Title != "" {
				return fw, nil
			}
		}
	}

	fw, err := getX11FocusedWindow()
	if err == nil && fw.Title != "" {
		return fw, nil
	}

	return nil, errors.New("no active window found")
}

func getGnomeFocusedWindow() (*FocusedWindow, error) {
	script := `
		const start = Date.now();
		let fw = global.get_window_actors()
			.map(a => a.meta_window)
			.find(w => w.has_focus());
		if (!fw) {
			fw = global.display.get_focus_window();
		}
		if (fw) {
			JSON.stringify({
				wm_class: fw.get_wm_class() || '',
				title: fw.get_title() || '',
				pid: fw.get_pid() || 0
			});
		} else {
			'null';
		}
	`

	cmd := exec.Command("gdbus", "call", "--session",
		"--dest", "org.gnome.Shell",
		"--object-path", "/org/gnome/Shell",
		"--method", "org.gnome.Shell.Eval",
		script)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	result := string(output)

	start := strings.Index(result, "{")
	end := strings.LastIndex(result, "}")
	if start == -1 || end == -1 {
		return nil, errors.New("invalid response")
	}

	jsonStr := result[start : end+1]
	jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")
	jsonStr = strings.ReplaceAll(jsonStr, "\\'", "'")

	fw := &FocusedWindow{Source: "gnome-wayland"}

	if title := extractJSONValue(jsonStr, "title"); title != "" {
		fw.Title = title
	}
	if wmClass := extractJSONValue(jsonStr, "wm_class"); wmClass != "" {
		fw.Class = wmClass
		fw.AppName = wmClass
	}
	if pid := extractJSONValue(jsonStr, "pid"); pid != "" {
		fmt.Sscanf(pid, "%d", &fw.PID)
	}

	return fw, nil
}

func extractJSONValue(json, key string) string {
	search := fmt.Sprintf(`"%s":`, key)
	idx := strings.Index(json, search)
	if idx == -1 {
		search = fmt.Sprintf(`'%s':`, key)
		idx = strings.Index(json, search)
		if idx == -1 {
			return ""
		}
	}

	start := idx + len(search)
	rest := strings.TrimSpace(json[start:])

	if strings.HasPrefix(rest, "\"") || strings.HasPrefix(rest, "'") {
		quote := rest[0]
		end := strings.Index(rest[1:], string(quote))
		if end == -1 {
			return ""
		}
		return rest[1 : end+1]
	}

	end := strings.IndexAny(rest, ",}")
	if end == -1 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}

type X11Client struct {
	conn  *xgb.Conn
	root  xproto.Window
	atoms map[string]xproto.Atom
}

func newX11Client() (*X11Client, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}

	setup := xproto.Setup(conn)
	root := setup.DefaultScreen(conn).Root

	client := &X11Client{
		conn:  conn,
		root:  root,
		atoms: make(map[string]xproto.Atom),
	}

	atomNames := []string{
		"_NET_ACTIVE_WINDOW",
		"_NET_WM_NAME",
		"_NET_WM_PID",
		"WM_NAME",
		"WM_CLASS",
		"UTF8_STRING",
	}

	for _, name := range atomNames {
		reply, err := xproto.InternAtom(conn, false, uint16(len(name)), name).Reply()
		if err != nil {
			conn.Close()
			return nil, err
		}
		client.atoms[name] = reply.Atom
	}

	return client, nil
}

func (c *X11Client) close() {
	c.conn.Close()
}

func (c *X11Client) getProperty(window xproto.Window, atom xproto.Atom, atomType xproto.Atom, length uint32) ([]byte, error) {
	reply, err := xproto.GetProperty(c.conn, false, window, atom, atomType, 0, length).Reply()
	if err != nil {
		return nil, err
	}
	return reply.Value, nil
}

func (c *X11Client) getActiveWindowFromProperty() xproto.Window {
	data, err := c.getProperty(c.root, c.atoms["_NET_ACTIVE_WINDOW"], xproto.AtomWindow, 1)
	if err != nil || len(data) < 4 {
		return 0
	}
	return xproto.Window(binary.LittleEndian.Uint32(data))
}

func (c *X11Client) getActiveWindowFromInputFocus() xproto.Window {
	reply, err := xproto.GetInputFocus(c.conn).Reply()
	if err != nil {
		return 0
	}
	return reply.Focus
}

func (c *X11Client) getTopLevelParent(window xproto.Window) xproto.Window {
	for {
		reply, err := xproto.QueryTree(c.conn, window).Reply()
		if err != nil || reply.Parent == c.root || reply.Parent == 0 {
			return window
		}
		window = reply.Parent
	}
}

func (c *X11Client) hasValidName(window xproto.Window) bool {
	data, _ := c.getProperty(window, c.atoms["_NET_WM_NAME"], c.atoms["UTF8_STRING"], 1)
	if len(data) > 0 {
		return true
	}
	data, _ = c.getProperty(window, c.atoms["WM_NAME"], xproto.AtomString, 1)
	return len(data) > 0
}

func (c *X11Client) getActiveWindow() (xproto.Window, error) {
	for i := 0; i < 5; i++ {
		windowID := c.getActiveWindowFromProperty()
		if windowID != 0 && c.hasValidName(windowID) {
			return windowID, nil
		}

		windowID = c.getActiveWindowFromInputFocus()
		if windowID != 0 && windowID != c.root {
			topLevel := c.getTopLevelParent(windowID)
			if topLevel != 0 && c.hasValidName(topLevel) {
				return topLevel, nil
			}
		}

		time.Sleep(20 * time.Millisecond)
	}

	return 0, errors.New("no active window found")
}

func (c *X11Client) getWindowName(window xproto.Window) string {
	data, err := c.getProperty(window, c.atoms["_NET_WM_NAME"], c.atoms["UTF8_STRING"], 256)
	if err == nil && len(data) > 0 {
		return strings.TrimRight(string(data), "\x00")
	}

	data, err = c.getProperty(window, c.atoms["WM_NAME"], xproto.AtomString, 256)
	if err == nil && len(data) > 0 {
		return strings.TrimRight(string(data), "\x00")
	}

	return ""
}

func (c *X11Client) getWindowClass(window xproto.Window) (instance, class string) {
	data, err := c.getProperty(window, c.atoms["WM_CLASS"], xproto.AtomString, 256)
	if err != nil || len(data) == 0 {
		return "", ""
	}

	parts := strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
	if len(parts) >= 1 {
		instance = parts[0]
	}
	if len(parts) >= 2 {
		class = parts[1]
	}
	return instance, class
}

func (c *X11Client) getWindowPID(window xproto.Window) uint32 {
	data, err := c.getProperty(window, c.atoms["_NET_WM_PID"], xproto.AtomCardinal, 1)
	if err != nil || len(data) < 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(data)
}

func getX11FocusedWindow() (*FocusedWindow, error) {
	client, err := newX11Client()
	if err != nil {
		return nil, err
	}
	defer client.close()

	windowID, err := client.getActiveWindow()
	if err != nil {
		return nil, err
	}

	instance, class := client.getWindowClass(windowID)

	return &FocusedWindow{
		Title:    client.getWindowName(windowID),
		AppName:  instance,
		Class:    class,
		PID:      client.getWindowPID(windowID),
		WindowID: uint32(windowID),
		Source:   "x11",
	}, nil
}

func main() {
	fw, err := GetFocusedWindow()
	if err != nil {
		fmt.Printf("Failed to get focused window: %v\n", err)
		return
	}

	fmt.Printf("Source:    %s\n", fw.Source)
	fmt.Printf("Window ID: 0x%x\n", fw.WindowID)
	fmt.Printf("Title:     %s\n", fw.Title)
	fmt.Printf("App Name:  %s\n", fw.AppName)
	fmt.Printf("Class:     %s\n", fw.Class)
	fmt.Printf("PID:       %d\n", fw.PID)
}
