//go:build windows

package winapi

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Process describes the executable that owns a window.
type Process struct {
	PID  uint32
	Path string
	Name string
}

// ProcessInfo opens pid and resolves its executable path.
func ProcessInfo(pid uint32) (Process, error) {
	p := Process{PID: pid}
	if pid == 0 {
		return p, fmt.Errorf("no process id")
	}
	h, _, openErr := procOpenProcess.Call(ProcessQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return p, callError("OpenProcess", openErr)
	}
	defer procCloseHandle.Call(h)

	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	r, _, queryErr := procQueryFullProcessImageName.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r == 0 {
		return p, callError("QueryFullProcessImageNameW", queryErr)
	}
	p.Path = windows.UTF16ToString(buf[:size])
	p.Name = filepath.Base(p.Path)
	return p, nil
}

// SendKeyCombo injects modifier keys plus one key. It is used to ask the
// foreground editor to copy its selection or reference.
func SendKeyCombo(modifiers []uint16, key uint16) {
	for _, mod := range modifiers {
		KeyDown(mod)
		SleepMilliseconds(4)
	}
	KeyDown(key)
	SleepMilliseconds(18)
	KeyUp(key)
	SleepMilliseconds(4)
	for i := len(modifiers) - 1; i >= 0; i-- {
		KeyUp(modifiers[i])
		SleepMilliseconds(2)
	}
}

// SendCopySelection asks the focused application to copy using Ctrl+C.
func SendCopySelection() {
	SendKeyCombo([]uint16{VkControl}, 'C')
}

// SendCopySelectionAlt asks the focused application to copy using Ctrl+Insert.
func SendCopySelectionAlt() {
	SendKeyCombo([]uint16{VkControl}, 0x2D)
}

// SendJetBrainsCopyReference sends the default JetBrains "Copy Reference"
// shortcut (Ctrl+Alt+Shift+C).
func SendJetBrainsCopyReference() {
	SendKeyCombo([]uint16{VkControl, VkMenu, VkShift}, 'C')
}

// ShellExecute opens a file, folder or URL with the default handler.
func ShellExecute(target string) error {
	verb := utf16Ptr("open")
	file := utf16Ptr(target)
	r, _, err := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0, 0, 1, // SW_SHOWNORMAL
	)
	if r <= 32 {
		return callError("ShellExecuteW", err)
	}
	return nil
}

// SingleInstanceMutex owns a named mutex for the lifetime of the process.
type SingleInstanceMutex struct {
	Handle  HANDLE
	Already bool
}

// CreateSingleInstanceMutex creates (or opens) a Local\ named mutex.
func CreateSingleInstanceMutex(name string) (*SingleInstanceMutex, error) {
	r, _, err := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(utf16Ptr(name))))
	if r == 0 {
		return nil, callError("CreateMutexW", err)
	}
	already := false
	if e, ok := err.(syscall.Errno); ok && e == 183 { // ERROR_ALREADY_EXISTS
		already = true
	}
	return &SingleInstanceMutex{Handle: HANDLE(r), Already: already}, nil
}

// Close closes the mutex handle.
func (m *SingleInstanceMutex) Close() {
	if m != nil && m.Handle != 0 {
		procCloseHandle.Call(uintptr(m.Handle))
		m.Handle = 0
	}
}

// WaitForModifierRelease waits until the shortcut modifiers are no longer held.
// This avoids sending Ctrl+C while the user still holds Shift from the hotkey.
func WaitForModifierRelease(vks []uint16) bool {
	return WaitForKeysReleased(vks, 1500*time.Millisecond)
}

// IsTerminalClass identifies native console windows where Ctrl+C is SIGINT.
func IsTerminalClass(class string) bool {
	switch strings.ToLower(class) {
	case "consolewindowclass", "cascadia_hosting_window_class", "mintty", "putty":
		return true
	}
	return false
}
