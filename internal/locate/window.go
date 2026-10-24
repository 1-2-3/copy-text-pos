//go:build windows

package locate

import (
	"fmt"
	"path/filepath"
	"strings"

	"copytextpos/internal/winapi"
)

// Window is the foreground window and the process that owns it.
type Window struct {
	Handle  winapi.HWND
	PID     uint32
	Process winapi.Process
	Class   string
	Title   string
}

// Foreground returns details about the current foreground window.
func Foreground() (Window, error) {
	raw := winapi.ForegroundWindow()
	if raw == 0 {
		return Window{}, fmt.Errorf("no foreground window")
	}
	info := winapi.RootAncestor(raw)
	pid := winapi.WindowProcessID(info)
	if pid == 0 {
		pid = winapi.WindowProcessID(raw)
	}
	if winapi.WindowTitle(info) == "" && pid != 0 {
		if top := winapi.TopLevelWindowForProcess(pid); top != 0 {
			info = top
		}
	}
	if pid == 0 {
		pid = winapi.WindowProcessID(info)
	}
	w := Window{
		Handle: raw,
		PID:    pid,
		Class:  winapi.WindowClass(info),
		Title:  winapi.WindowTitle(info),
	}
	if p, err := winapi.ProcessInfo(pid); err == nil {
		w.Process = p
	}
	return w, nil
}

// Executable returns the lowercase executable basename.
func (w Window) Executable() string {
	return strings.ToLower(w.Process.Name)
}

// VSCodeFamily reports whether the process is a VS Code derivative.
func (w Window) VSCodeFamily() bool {
	name := strings.TrimSuffix(w.Executable(), ".exe")
	switch name {
	case "code", "code - insiders", "cursor", "windsurf", "vscodium",
		"trae", "kiro", "positron", "void", "theia":
		return true
	}
	return false
}

// JetBrains reports whether the process is a JetBrains IDE.
func (w Window) JetBrains() bool {
	name := strings.TrimSuffix(w.Executable(), ".exe")
	name = strings.TrimSuffix(name, "64")
	prefixes := []string{
		"idea", "studio", "pycharm", "webstorm", "goland", "clion",
		"rider", "rustrover", "phpstorm", "rubymine", "datagrip",
		"aqua", "fleet",
	}
	for _, prefix := range prefixes {
		if name == prefix || strings.HasPrefix(name, prefix+"-") || strings.HasPrefix(name, prefix+"_") {
			return true
		}
	}
	return false
}

// Obsidian reports whether the process looks like Obsidian.
func (w Window) Obsidian() bool {
	return strings.EqualFold(strings.TrimSuffix(w.Executable(), ".exe"), "obsidian") ||
		strings.Contains(strings.ToLower(w.Title), " - obsidian")
}

// Typora reports whether the process looks like Typora.
func (w Window) Typora() bool {
	return strings.EqualFold(strings.TrimSuffix(w.Executable(), ".exe"), "typora") ||
		strings.Contains(strings.ToLower(w.Title), " - typora")
}

// VSCodeAppLabel returns the product label for a VS Code window.
func (w Window) VSCodeAppLabel() string {
	segments := TitleSegments(w.Title)
	if len(segments) > 0 {
		last := strings.TrimSpace(segments[len(segments)-1])
		for _, label := range []string{
			"Visual Studio Code - Insiders", "Visual Studio Code", "VSCodium",
			"Cursor", "Windsurf", "Trae", "Kiro", "Positron", "Void",
		} {
			if strings.EqualFold(last, label) {
				return label
			}
		}
	}
	return filepath.Base(strings.TrimSuffix(w.Process.Name, filepath.Ext(w.Process.Name)))
}

// DisplayName is a short process name used in notifications and logs.
func (w Window) DisplayName() string {
	if w.Process.Name != "" {
		return strings.TrimSuffix(w.Process.Name, filepath.Ext(w.Process.Name))
	}
	return "editor"
}
