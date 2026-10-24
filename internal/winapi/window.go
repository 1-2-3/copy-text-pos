//go:build windows

package winapi

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	ole32    = windows.NewLazySystemDLL("ole32.dll")

	procRegisterClassExW        = user32.NewProc("RegisterClassExW")
	procCreateWindowExW         = user32.NewProc("CreateWindowExW")
	procDefWindowProcW          = user32.NewProc("DefWindowProcW")
	procDestroyWindow           = user32.NewProc("DestroyWindow")
	procGetMessageW             = user32.NewProc("GetMessageW")
	procTranslateMessage        = user32.NewProc("TranslateMessage")
	procDispatchMessageW        = user32.NewProc("DispatchMessageW")
	procPostQuitMessage         = user32.NewProc("PostQuitMessage")
	procPostMessageW            = user32.NewProc("PostMessageW")
	procSendMessageW            = user32.NewProc("SendMessageW")
	procRegisterWindowMessageW  = user32.NewProc("RegisterWindowMessageW")
	procFindWindowW             = user32.NewProc("FindWindowW")
	procIsWindow                = user32.NewProc("IsWindow")
	procShowWindow              = user32.NewProc("ShowWindow")
	procSetForegroundWindow     = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow     = user32.NewProc("GetForegroundWindow")
	procGetAncestor             = user32.NewProc("GetAncestor")
	procEnumWindows             = user32.NewProc("EnumWindows")
	procIsWindowVisible         = user32.NewProc("IsWindowVisible")
	procGetWindowTextLengthW    = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW          = user32.NewProc("GetWindowTextW")
	procGetClassNameW           = user32.NewProc("GetClassNameW")
	procGetWindowThreadProcess  = user32.NewProc("GetWindowThreadProcessId")
	procRegisterHotKey          = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey        = user32.NewProc("UnregisterHotKey")
	procGetAsyncKeyState        = user32.NewProc("GetAsyncKeyState")
	procKeybdEvent              = user32.NewProc("keybd_event")
	procGetCursorPos            = user32.NewProc("GetCursorPos")
	procCreatePopupMenu         = user32.NewProc("CreatePopupMenu")
	procAppendMenuW             = user32.NewProc("AppendMenuW")
	procDestroyMenu             = user32.NewProc("DestroyMenu")
	procSetMenuDefaultItem      = user32.NewProc("SetMenuDefaultItem")
	procTrackPopupMenu          = user32.NewProc("TrackPopupMenu")
	procLoadIconW               = user32.NewProc("LoadIconW")
	procLoadImageW              = user32.NewProc("LoadImageW")
	procDestroyIcon             = user32.NewProc("DestroyIcon")
	procOpenClipboard           = user32.NewProc("OpenClipboard")
	procCloseClipboard          = user32.NewProc("CloseClipboard")
	procEmptyClipboard          = user32.NewProc("EmptyClipboard")
	procGetClipboardData        = user32.NewProc("GetClipboardData")
	procSetClipboardData        = user32.NewProc("SetClipboardData")
	procGetClipboardSequenceNum = user32.NewProc("GetClipboardSequenceNumber")
	procCountClipboardFormats   = user32.NewProc("CountClipboardFormats")
	procSetProcessDPIAware      = user32.NewProc("SetProcessDPIAware")
	procSetProcessDPIAwareness  = user32.NewProc("SetProcessDpiAwarenessContext")

	procGetModuleHandleW          = kernel32.NewProc("GetModuleHandleW")
	procGetModuleFileNameW        = kernel32.NewProc("GetModuleFileNameW")
	procGetUserDefaultUILanguage  = kernel32.NewProc("GetUserDefaultUILanguage")
	procGlobalAlloc               = kernel32.NewProc("GlobalAlloc")
	procGlobalLock                = kernel32.NewProc("GlobalLock")
	procGlobalUnlock              = kernel32.NewProc("GlobalUnlock")
	procGlobalFree                = kernel32.NewProc("GlobalFree")
	procSleep                     = kernel32.NewProc("Sleep")
	procOpenProcess               = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle               = kernel32.NewProc("CloseHandle")
	procCreateMutexW              = kernel32.NewProc("CreateMutexW")
	procGetCurrentProcessId       = kernel32.NewProc("GetCurrentProcessId")

	procShellExecuteW     = shell32.NewProc("ShellExecuteW")
	procShellNotifyIconW  = shell32.NewProc("Shell_NotifyIconW")
	procShellExecuteExW   = shell32.NewProc("ShellExecuteExW")
	procMessageBoxW       = user32.NewProc("MessageBoxW")
	procOleInitialize     = ole32.NewProc("OleInitialize")
	procOleUninitialize   = ole32.NewProc("OleUninitialize")
	procOleGetClipboard   = ole32.NewProc("OleGetClipboard")
	procOleSetClipboard   = ole32.NewProc("OleSetClipboard")
	procOleFlushClipboard = ole32.NewProc("OleFlushClipboard")
)

// RegisterClass registers a window class and returns its atom.
func RegisterClass(className string, wndProc uintptr) (uint16, error) {
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   wndProc,
		HInstance:     windows.Handle(hInstance),
		LpszClassName: utf16Ptr(className),
	}
	r, _, err := procRegisterClassExW.Call(ptr(unsafe.Pointer(&wc)))
	if r == 0 {
		return 0, callError("RegisterClassExW", err)
	}
	return uint16(r), nil
}

// CreateMessageWindow creates a hidden top-level window suitable for tray
// notifications and hotkey messages.
func CreateMessageWindow(className, title string) (HWND, error) {
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	r, _, err := procCreateWindowExW.Call(
		WsExToolwindow,
		uintptr(unsafe.Pointer(utf16Ptr(className))),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		WsOverlapped,
		0, 0, 0, 0,
		0, 0, hInstance, 0,
	)
	if r == 0 {
		return 0, callError("CreateWindowExW", err)
	}
	return HWND(r), nil
}

// DestroyWindow destroys hwnd.
func DestroyWindow(hwnd HWND) {
	procDestroyWindow.Call(uintptr(hwnd))
}

// WindowProcCallback converts a Go function into a C-compatible callback.
func WindowProcCallback(fn func(HWND, uint32, uintptr, uintptr) uintptr) uintptr {
	return syscall.NewCallback(func(hwnd, msg, wparam, lparam uintptr) uintptr {
		return fn(HWND(hwnd), uint32(msg), wparam, lparam)
	})
}

// DefWindowProc invokes the default window procedure.
func DefWindowProc(hwnd HWND, msg uint32, wparam, lparam uintptr) uintptr {
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
	return r
}

// MessageLoop runs a standard Windows message loop until WM_QUIT.
func MessageLoop() error {
	var msg MSG
	for {
		r, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		ret := int32(r)
		if ret == -1 {
			return callError("GetMessageW", err)
		}
		if ret == 0 {
			return nil
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// PostQuitMessage asks the message loop to exit.
func PostQuitMessage(code int32) {
	procPostQuitMessage.Call(uintptr(uint32(code)))
}

// PostMessage posts a message to a window.
func PostMessage(hwnd HWND, msg uint32, wparam, lparam uintptr) error {
	r, _, err := procPostMessageW.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
	if r == 0 {
		return callError("PostMessageW", err)
	}
	return nil
}

// SendMessage sends a message synchronously.
func SendMessage(hwnd HWND, msg uint32, wparam, lparam uintptr) uintptr {
	r, _, _ := procSendMessageW.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
	return r
}

// RegisterWindowMessage registers a system-wide message.
func RegisterWindowMessage(name string) uint32 {
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(utf16Ptr(name))))
	return uint32(r)
}

// FindWindow finds a top-level window by class and title.
func FindWindow(className, title string) HWND {
	var classPtr, titlePtr uintptr
	if className != "" {
		classPtr = uintptr(unsafe.Pointer(utf16Ptr(className)))
	}
	if title != "" {
		titlePtr = uintptr(unsafe.Pointer(utf16Ptr(title)))
	}
	r, _, _ := procFindWindowW.Call(classPtr, titlePtr)
	return HWND(r)
}

// IsWindow reports whether hwnd is a valid window.
func IsWindow(hwnd HWND) bool {
	r, _, _ := procIsWindow.Call(uintptr(hwnd))
	return r != 0
}

// ShowWindow changes a window's visibility.
func ShowWindow(hwnd HWND, cmd int32) {
	procShowWindow.Call(uintptr(hwnd), uintptr(cmd))
}

// SetForegroundWindow brings a window to the foreground.
func SetForegroundWindow(hwnd HWND) bool {
	r, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	return r != 0
}

// ForegroundWindow returns the current foreground window.
func ForegroundWindow() HWND {
	r, _, _ := procGetForegroundWindow.Call()
	return HWND(r)
}

// RootAncestor returns the top-level ancestor of hwnd (GA_ROOT).
func RootAncestor(hwnd HWND) HWND {
	r, _, _ := procGetAncestor.Call(uintptr(hwnd), 2)
	if r == 0 {
		return hwnd
	}
	return HWND(r)
}

// TopLevelWindowForProcess finds a visible, titled top-level window belonging
// to pid. Editors occasionally move focus into a child window; this recovers
// the document title in that case.
func TopLevelWindowForProcess(pid uint32) HWND {
	if pid == 0 {
		return 0
	}
	var found HWND
	callback := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		if !isWindowVisible(HWND(hwnd)) {
			return 1
		}
		if WindowProcessID(HWND(hwnd)) != pid {
			return 1
		}
		if WindowTitle(HWND(hwnd)) == "" {
			return 1
		}
		found = HWND(hwnd)
		return 0
	})
	procEnumWindows.Call(callback, 0)
	return found
}

func isWindowVisible(hwnd HWND) bool {
	r, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return r != 0
}

// WindowTitle returns the window's title text.
func WindowTitle(hwnd HWND) string {
	length, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if length == 0 {
		return ""
	}
	buf := make([]uint16, length+1)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), length+1)
	return windows.UTF16ToString(buf)
}

// WindowClass returns the window class name.
func WindowClass(hwnd HWND) string {
	buf := make([]uint16, 256)
	procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf)
}

// WindowProcessID returns the process that owns hwnd.
func WindowProcessID(hwnd HWND) uint32 {
	var pid uint32
	procGetWindowThreadProcess.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	return pid
}

// RegisterHotKey registers a global hotkey for hwnd.
func RegisterHotKey(hwnd HWND, id int, mods, vk uint32) error {
	r, _, err := procRegisterHotKey.Call(uintptr(hwnd), uintptr(id), uintptr(mods), uintptr(vk))
	if r == 0 {
		return callError("RegisterHotKey", err)
	}
	return nil
}

// UnregisterHotKey removes a global hotkey.
func UnregisterHotKey(hwnd HWND, id int) error {
	r, _, err := procUnregisterHotKey.Call(uintptr(hwnd), uintptr(id))
	if r == 0 {
		return callError("UnregisterHotKey", err)
	}
	return nil
}

// IsKeyDown reports whether a virtual key is currently down.
func IsKeyDown(vk uint16) bool {
	r, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	return int16(uint16(r)) < 0
}

// KeyDown injects a key-down event.
func KeyDown(vk uint16) {
	procKeybdEvent.Call(uintptr(vk), 0, 0, 0)
}

// KeyUp injects a key-up event.
func KeyUp(vk uint16) {
	procKeybdEvent.Call(uintptr(vk), 0, KeyEventfKeyup, 0)
}

// CursorPos returns the mouse position.
func CursorPos() POINT {
	var p POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p
}

// CreatePopupMenu creates an empty popup menu.
func CreatePopupMenu() HMENU {
	r, _, _ := procCreatePopupMenu.Call()
	return HMENU(r)
}

// AppendMenuString appends a text item.
func AppendMenuString(menu HMENU, id uint32, text string) {
	procAppendMenuW.Call(uintptr(menu), MfString, uintptr(id), uintptr(unsafe.Pointer(utf16Ptr(text))))
}

// AppendMenuSeparator appends a separator.
func AppendMenuSeparator(menu HMENU) {
	procAppendMenuW.Call(uintptr(menu), MfSeparator, 0, 0)
}

// DestroyMenu frees a menu.
func DestroyMenu(menu HMENU) {
	procDestroyMenu.Call(uintptr(menu))
}

// SetMenuDefaultItem marks the default menu item.
func SetMenuDefaultItem(menu HMENU, id uint32) {
	procSetMenuDefaultItem.Call(uintptr(menu), uintptr(id), 0)
}

// TrackPopupMenu shows a popup menu at the cursor and returns the chosen ID.
func TrackPopupMenu(menu HMENU, hwnd HWND) uint32 {
	p := CursorPos()
	r, _, _ := procTrackPopupMenu.Call(
		uintptr(menu),
		TpmRightButton|TpmBottomAlign|TpmReturnCmd,
		uintptr(p.X), uintptr(p.Y), 0,
		uintptr(hwnd), 0,
	)
	return uint32(r)
}

// LoadIcon loads an icon resource from the current executable.
func LoadIcon(resourceID uintptr) HICON {
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	r, _, _ := procLoadIconW.Call(hInstance, resourceID)
	return HICON(r)
}

// LoadIconFromFile loads an icon from an .ico file.
func LoadIconFromFile(path string) (HICON, error) {
	r, _, err := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr(path))),
		ImageIcon,
		32, 32,
		LrLoadFromFile|LrDefaultSize,
	)
	if r == 0 {
		return 0, callError("LoadImageW", err)
	}
	return HICON(r), nil
}

// DestroyIcon frees an icon created from a file.
func DestroyIcon(icon HICON) {
	if icon != 0 {
		procDestroyIcon.Call(uintptr(icon))
	}
}

// ShellNotifyIcon adds, modifies or deletes a notification-area icon.
func ShellNotifyIcon(action uint32, data *NOTIFYICONDATA) error {
	r, _, err := procShellNotifyIconW.Call(uintptr(action), uintptr(unsafe.Pointer(data)))
	if r == 0 {
		return callError("Shell_NotifyIconW", err)
	}
	return nil
}

// MessageBox shows a native message box.
func MessageBox(title, text string, flags uint32) int32 {
	r, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		uintptr(flags),
	)
	return int32(r)
}

// SetDPIAware enables per-monitor v2 DPI awareness, falling back gracefully.
func SetDPIAware() {
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is (HANDLE)-4.
	if r, _, _ := procSetProcessDPIAwareness.Call(^uintptr(3)); r != 0 {
		return
	}
	procSetProcessDPIAware.Call()
}

// UserDefaultUILanguage returns the Windows UI language identifier.
func UserDefaultUILanguage() uint16 {
	r, _, _ := procGetUserDefaultUILanguage.Call()
	return lowWord(r)
}

// ModulePath returns the current executable path.
func ModulePath() (string, error) {
	buf := make([]uint16, windows.MAX_PATH)
	for {
		n, _, err := procGetModuleFileNameW.Call(0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n == 0 {
			return "", callError("GetModuleFileNameW", err)
		}
		if int(n) < len(buf) {
			return windows.UTF16ToString(buf[:n]), nil
		}
		if len(buf) >= 32768 {
			return "", syscall.ENAMETOOLONG
		}
		buf = make([]uint16, len(buf)*2)
	}
}

// SleepMilliseconds suspends the calling thread.
func SleepMilliseconds(ms int) {
	if ms > 0 {
		procSleep.Call(uintptr(ms))
	}
}

// CurrentProcessID returns the current process identifier.
func CurrentProcessID() uint32 {
	r, _, _ := procGetCurrentProcessId.Call()
	return uint32(r)
}
