//go:build windows

// Package tray implements the native notification-area icon, its menu and the
// Win32 message loop. It deliberately has no third-party UI dependency.
package tray

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"copytextpos/internal/hotkey"
	"copytextpos/internal/winapi"
)

const (
	className  = "CopyTextPos.HostWindow"
	windowText = "Copy Text Pos"
	hotkeyID   = 1
	trayID     = 1
)

// Item is one tray context-menu entry.
type Item struct {
	ID        int
	Label     string
	Default   bool
	Separator bool
}

// Config initializes the tray.
type Config struct {
	Tooltip    string
	IconData   []byte
	Items      []Item
	OnCommand  func(id int)
	OnHotkey   func()
	OnSettings func()
}

// Tray owns the hidden message window and notification icon.
type Tray struct {
	cfg            Config
	hwnd           winapi.HWND
	icon           winapi.HICON
	ownIcon        bool
	nid            winapi.NOTIFYICONDATA
	taskbarCreated uint32

	wndProcCallback uintptr
	currentHotkey   *hotkey.Spec
	desiredHotkey   *hotkey.Spec
	hotkeyErr       error

	started   atomic.Bool
	suspended atomic.Bool
	uiQueue   chan func()

	hotkeyMu sync.Mutex
}

// New creates the hidden window and adds the notification icon.
func New(cfg Config) (*Tray, error) {
	t := &Tray{cfg: cfg, uiQueue: make(chan func(), 32)}
	icon, own, err := loadIcon(cfg.IconData)
	if err != nil {
		return nil, err
	}
	t.icon = icon
	t.ownIcon = own

	t.wndProcCallback = winapi.WindowProcCallback(t.windowProc)
	if _, err := winapi.RegisterClass(className, t.wndProcCallback); err != nil {
		t.freeIcon()
		return nil, err
	}
	hwnd, err := winapi.CreateMessageWindow(className, windowText)
	if err != nil {
		t.freeIcon()
		return nil, err
	}
	t.hwnd = hwnd
	t.taskbarCreated = winapi.RegisterWindowMessage("TaskbarCreated")

	t.nid = winapi.NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(winapi.NOTIFYICONDATA{})),
		HWnd:             hwnd,
		UID:              trayID,
		UFlags:           winapi.NifMessage | winapi.NifIcon | winapi.NifTip,
		UCallbackMessage: winapi.WmApp,
		HIcon:            icon,
	}
	t.setTip(cfg.Tooltip)
	if err := t.addIcon(); err != nil {
		winapi.DestroyWindow(hwnd)
		t.freeIcon()
		return nil, err
	}
	return t, nil
}

func loadIcon(data []byte) (winapi.HICON, bool, error) {
	if icon := winapi.LoadIcon(1); icon != 0 {
		return icon, false, nil
	}
	if len(data) > 0 {
		path := filepath.Join(os.TempDir(), fmt.Sprintf("copy-text-pos-%d.ico", os.Getpid()))
		if err := os.WriteFile(path, data, 0o600); err == nil {
			icon, loadErr := winapi.LoadIconFromFile(path)
			_ = os.Remove(path)
			if loadErr == nil && icon != 0 {
				return icon, true, nil
			}
		}
	}
	if icon := winapi.LoadIcon(32512); icon != 0 { // IDI_APPLICATION
		return icon, false, nil
	}
	return 0, false, fmt.Errorf("could not load an application icon")
}

func (t *Tray) freeIcon() {
	if t.ownIcon {
		winapi.DestroyIcon(t.icon)
		t.ownIcon = false
	}
	t.icon = 0
}

func (t *Tray) setTip(tip string) {
	if tip == "" {
		return
	}
	for i := range t.nid.SzTip {
		t.nid.SzTip[i] = 0
	}
	copyUTF16(t.nid.SzTip[:], tip)
}

func copyUTF16(dst []uint16, s string) {
	u := syscall.StringToUTF16(s)
	n := copy(dst, u)
	if n == len(dst) {
		dst[len(dst)-1] = 0
	}
}

func (t *Tray) addIcon() error {
	return winapi.ShellNotifyIcon(winapi.NimAdd, &t.nid)
}

func (t *Tray) removeIcon() {
	if t.hwnd != 0 {
		_ = winapi.ShellNotifyIcon(winapi.NimDelete, &t.nid)
	}
}

// Run starts the blocking Windows message loop.
func (t *Tray) Run() error {
	t.started.Store(true)
	err := winapi.MessageLoop()
	t.started.Store(false)
	t.removeIcon()
	if winapi.IsWindow(t.hwnd) {
		winapi.DestroyWindow(t.hwnd)
	}
	t.freeIcon()
	return err
}

// Quit asks the message loop to exit.
func (t *Tray) Quit() {
	if t.hwnd != 0 {
		_ = winapi.PostMessage(t.hwnd, winapi.WmClose, 0, 0)
	}
}

// ShowSettings asks the owner to open the settings page.
func (t *Tray) ShowSettings() {
	if t.hwnd != 0 {
		_ = winapi.PostMessage(t.hwnd, winapi.WmAppShowSettings, 0, 0)
	}
}

func (t *Tray) windowProc(hwnd winapi.HWND, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case winapi.WmApp:
		event := uint32(lparam) & 0xffff
		switch event {
		case winapi.WmRButtonUp, winapi.WmContextMenu, winapi.WmLButtonUp:
			t.showMenu()
		}
		return 0
	case winapi.WmHotkey:
		if uint32(wparam) == hotkeyID && t.cfg.OnHotkey != nil {
			t.cfg.OnHotkey()
		}
		return 0
	case winapi.WmAppInvoke:
		t.drainQueue()
		return 0
	case winapi.WmAppShowSettings:
		if t.cfg.OnSettings != nil {
			t.cfg.OnSettings()
		}
		return 0
	case winapi.WmClose:
		if t.currentHotkey != nil {
			_ = winapi.UnregisterHotKey(t.hwnd, hotkeyID)
			t.currentHotkey = nil
		}
		t.removeIcon()
		winapi.DestroyWindow(hwnd)
		return 0
	case winapi.WmDestroy:
		winapi.PostQuitMessage(0)
		return 0
	case winapi.WmQueryEnd:
		return 1
	case winapi.WmEndSession:
		if wparam != 0 {
			t.removeIcon()
		}
		return 0
	}
	if msg == t.taskbarCreated && msg != 0 {
		_ = t.addIcon()
		return 0
	}
	return winapi.DefWindowProc(hwnd, msg, wparam, lparam)
}

func (t *Tray) showMenu() {
	menu := winapi.CreatePopupMenu()
	if menu == 0 {
		return
	}
	defer winapi.DestroyMenu(menu)
	for _, item := range t.cfg.Items {
		if item.Separator {
			winapi.AppendMenuSeparator(menu)
			continue
		}
		winapi.AppendMenuString(menu, uint32(item.ID), item.Label)
		if item.Default {
			winapi.SetMenuDefaultItem(menu, uint32(item.ID))
		}
	}
	previous := winapi.ForegroundWindow()
	winapi.SetForegroundWindow(t.hwnd)
	id := winapi.TrackPopupMenu(menu, t.hwnd)
	_ = winapi.PostMessage(t.hwnd, winapi.WmNull, 0, 0)
	// Give focus back to the editor before a menu command such as "Copy path
	// at cursor" runs; otherwise it would target our hidden message window.
	if previous != 0 && previous != t.hwnd && winapi.IsWindow(previous) {
		winapi.SetForegroundWindow(previous)
	}
	if id != 0 && t.cfg.OnCommand != nil {
		t.cfg.OnCommand(int(id))
	}
}

// ApplyHotkey registers (or clears) the global shortcut on the UI thread.
// It rolls back to the previous shortcut if Windows rejects the new one.
// While the tray is suspended, the new shortcut is remembered and registered
// after suspension ends.
func (t *Tray) ApplyHotkey(spec *hotkey.Spec) error {
	return t.invokeSync(func() error {
		var next *hotkey.Spec
		if spec != nil {
			copySpec := *spec
			next = &copySpec
		}
		if t.suspended.Load() {
			t.desiredHotkey = next
			t.hotkeyMu.Lock()
			t.hotkeyErr = nil
			t.hotkeyMu.Unlock()
			return nil
		}
		var prev *hotkey.Spec
		if t.currentHotkey != nil {
			prevCopy := *t.currentHotkey
			prev = &prevCopy
		}
		if err := t.replaceHotkey(next); err != nil {
			if prev != nil {
				_ = t.replaceHotkey(prev)
			}
			t.hotkeyMu.Lock()
			t.hotkeyErr = err
			t.hotkeyMu.Unlock()
			return err
		}
		t.desiredHotkey = next
		t.hotkeyMu.Lock()
		t.hotkeyErr = nil
		t.hotkeyMu.Unlock()
		return nil
	})
}

// SetHotkeySuspended temporarily releases or re-registers the configured
// shortcut. It is used while a File Explorer window is focused so Explorer's
// own Ctrl+Shift+C "Copy as path" action still works.
func (t *Tray) SetHotkeySuspended(suspend bool) error {
	return t.invokeSync(func() error {
		if t.suspended.Load() == suspend {
			return nil
		}
		if suspend {
			if t.currentHotkey != nil {
				if err := winapi.UnregisterHotKey(t.hwnd, hotkeyID); err != nil {
					return err
				}
				t.currentHotkey = nil
			}
			t.suspended.Store(true)
			return nil
		}

		t.suspended.Store(false)
		if t.desiredHotkey == nil {
			return nil
		}
		desired := *t.desiredHotkey
		if err := winapi.RegisterHotKey(t.hwnd, hotkeyID, desired.Mods|hotkey.ModNoRepeat, desired.VK); err != nil {
			err = fmt.Errorf("cannot re-register %s: %w", desired.String(), err)
			t.hotkeyMu.Lock()
			t.hotkeyErr = err
			t.hotkeyMu.Unlock()
			return err
		}
		t.currentHotkey = &desired
		t.hotkeyMu.Lock()
		t.hotkeyErr = nil
		t.hotkeyMu.Unlock()
		return nil
	})
}

// HotkeySuspended reports whether the shortcut is currently released.
func (t *Tray) HotkeySuspended() bool {
	return t.suspended.Load()
}

func (t *Tray) replaceHotkey(next *hotkey.Spec) error {
	if t.currentHotkey != nil {
		if err := winapi.UnregisterHotKey(t.hwnd, hotkeyID); err != nil {
			return err
		}
		t.currentHotkey = nil
	}
	if next == nil {
		return nil
	}
	if err := winapi.RegisterHotKey(t.hwnd, hotkeyID, next.Mods|hotkey.ModNoRepeat, next.VK); err != nil {
		return fmt.Errorf("cannot register %s (it may already be used by another app): %w", next.String(), err)
	}
	t.currentHotkey = next
	return nil
}

// HotkeyError returns the last registration error.
func (t *Tray) HotkeyError() error {
	t.hotkeyMu.Lock()
	defer t.hotkeyMu.Unlock()
	return t.hotkeyErr
}

// Notify shows a tray balloon. It is safe to call from any goroutine.
func (t *Tray) Notify(title, message string, warning bool) {
	t.invoke(func() {
		nid := t.nid
		nid.UFlags = winapi.NifInfo
		nid.DwInfoFlags = winapi.NiifInfo
		if warning {
			nid.DwInfoFlags = winapi.NiifWarning
		}
		copyUTF16(nid.SzInfoTitle[:], title)
		copyUTF16(nid.SzInfo[:], message)
		_ = winapi.ShellNotifyIcon(winapi.NimModify, &nid)
	})
}

// SetTooltip updates the hover text.
func (t *Tray) SetTooltip(tip string) {
	t.invoke(func() {
		t.setTip(tip)
		nid := t.nid
		nid.UFlags = winapi.NifTip
		_ = winapi.ShellNotifyIcon(winapi.NimModify, &nid)
	})
}

func (t *Tray) invoke(fn func()) {
	if !t.started.Load() {
		fn()
		return
	}
	select {
	case t.uiQueue <- fn:
	case <-time.After(2 * time.Second):
		return
	}
	_ = winapi.PostMessage(t.hwnd, winapi.WmAppInvoke, 0, 0)
}

func (t *Tray) invokeSync(fn func() error) error {
	if !t.started.Load() {
		return fn()
	}
	ch := make(chan error, 1)
	select {
	case t.uiQueue <- func() { ch <- fn() }:
	case <-time.After(2 * time.Second):
		return fmt.Errorf("tray UI is not responding")
	}
	if err := winapi.PostMessage(t.hwnd, winapi.WmAppInvoke, 0, 0); err != nil {
		return err
	}
	select {
	case err := <-ch:
		return err
	case <-time.After(3 * time.Second):
		return fmt.Errorf("tray UI did not answer")
	}
}

func (t *Tray) drainQueue() {
	for {
		select {
		case fn := <-t.uiQueue:
			fn()
		default:
			return
		}
	}
}
