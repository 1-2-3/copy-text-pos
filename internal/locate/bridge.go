//go:build windows

package locate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BridgeContext is written by the optional VS Code/Cursor extension.
type BridgeContext struct {
	Version      int    `json:"version"`
	Timestamp    int64  `json:"timestamp"`
	AppName      string `json:"appName"`
	Workspace    string `json:"workspace"`
	File         string `json:"file"`
	StartLine    int    `json:"startLine"`
	EndLine      int    `json:"endLine"`
	HasSelection bool   `json:"hasSelection"`
	PID          int    `json:"pid"`
}

// BridgeStatus is diagnostic information for the settings page.
type BridgeStatus struct {
	Path         string `json:"path"`
	Present      bool   `json:"present"`
	Fresh        bool   `json:"fresh"`
	HasSelection bool   `json:"hasSelection"`
	File         string `json:"file"`
	App          string `json:"app"`
	AgeSeconds   int64  `json:"ageSeconds"`
	Error        string `json:"error,omitempty"`
}

// ReadBridgeContext loads the extension snapshot.
func ReadBridgeContext(path string) (BridgeContext, time.Time, error) {
	var ctx BridgeContext
	info, err := os.Stat(path)
	if err != nil {
		return ctx, time.Time{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ctx, info.ModTime(), err
	}
	if err := json.Unmarshal(data, &ctx); err != nil {
		return BridgeContext{}, info.ModTime(), err
	}
	return ctx, info.ModTime(), nil
}

// BridgeContextForWindow returns a fresh, trustworthy context for w.
func BridgeContextForWindow(path string, w Window) (BridgeContext, bool) {
	if !w.VSCodeFamily() {
		return BridgeContext{}, false
	}
	ctx, mtime, err := ReadBridgeContext(path)
	if err != nil || ctx.Version < 1 {
		return BridgeContext{}, false
	}
	when := time.UnixMilli(ctx.Timestamp)
	if ctx.Timestamp <= 0 {
		when = mtime
	}
	if time.Since(when) > 3*time.Minute {
		return BridgeContext{}, false
	}
	if !ctx.HasSelection || ctx.StartLine < 1 || ctx.EndLine < ctx.StartLine {
		return BridgeContext{}, false
	}
	if !filepath.IsAbs(ctx.File) {
		return BridgeContext{}, false
	}
	base := filepath.Base(ctx.File)
	if base == "" || !strings.Contains(strings.ToLower(w.Title), strings.ToLower(base)) {
		return BridgeContext{}, false
	}
	info, err := os.Stat(ctx.File)
	if err != nil || info.IsDir() {
		return BridgeContext{}, false
	}
	return ctx, true
}

// BridgeStatus returns diagnostics for the settings page.
func BridgeStatusForWindow(path string, w Window) BridgeStatus {
	status := BridgeStatus{Path: path}
	ctx, mtime, err := ReadBridgeContext(path)
	if err != nil {
		if !os.IsNotExist(err) {
			status.Error = err.Error()
		}
		return status
	}
	status.Present = true
	status.File = ctx.File
	status.App = ctx.AppName
	status.HasSelection = ctx.HasSelection
	when := time.UnixMilli(ctx.Timestamp)
	if ctx.Timestamp <= 0 {
		when = mtime
	}
	status.AgeSeconds = int64(time.Since(when).Seconds())
	status.Fresh = time.Since(when) <= 3*time.Minute
	if w.Handle == 0 {
		return status
	}
	if !w.VSCodeFamily() {
		status.Error = "foreground window is not a VS Code-family editor"
		return status
	}
	if base := filepath.Base(ctx.File); base != "" && ctx.File != "" &&
		!strings.Contains(strings.ToLower(w.Title), strings.ToLower(base)) {
		status.Error = fmt.Sprintf("snapshot is for %s, not the focused window", base)
		return status
	}
	return status
}
