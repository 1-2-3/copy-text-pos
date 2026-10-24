//go:build windows

// Package winapi contains the small subset of Win32, Shell and OLE APIs used by
// Copy Text Pos. It intentionally avoids cgo so releases are a single static
// executable.
package winapi

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type (
	HWND      uintptr
	HICON     uintptr
	HMENU     uintptr
	HINSTANCE uintptr
	HANDLE    uintptr
)

const (
	WmDestroy     = 0x0002
	WmClose       = 0x0010
	WmNull        = 0x0000
	WmCommand     = 0x0111
	WmHotkey      = 0x0312
	WmRButtonUp   = 0x0205
	WmLButtonUp   = 0x0202
	WmLButtonDbl  = 0x0203
	WmContextMenu = 0x007B
	WmApp         = 0x8000
	WmQueryEnd    = 0x0011
	WmEndSession  = 0x0016

	WsOverlapped   = 0x00000000
	WsExToolwindow = 0x00000080

	SwHide = 0

	MfString    = 0x00000000
	MfSeparator = 0x00000800

	TpmRightButton = 0x0002
	TpmBottomAlign = 0x0020
	TpmReturnCmd   = 0x0100

	NimAdd    = 0x00000000
	NimModify = 0x00000001
	NimDelete = 0x00000002

	NifMessage = 0x00000001
	NifIcon    = 0x00000002
	NifTip     = 0x00000004
	NifInfo    = 0x00000010

	NiifInfo    = 0x00000001
	NiifWarning = 0x00000002
	NiifError   = 0x00000003

	ImageIcon      = 1
	LrLoadFromFile = 0x00000010
	LrDefaultSize  = 0x00000040

	WmAppShowSettings = WmApp + 1
	WmAppInvoke       = WmApp + 2
	WmAppNotify       = WmApp + 3

	ProcessQueryLimitedInformation = 0x1000

	CfUnicodeText = 13
	GmemMoveable  = 0x0002

	KeyEventfKeyup = 0x0002

	VkControl = 0x11
	VkShift   = 0x10
	VkMenu    = 0x12
	VkLWin    = 0x5B
	VkRWin    = 0x5C
)

// POINT is a Win32 screen coordinate.
type POINT struct {
	X, Y int32
}

// MSG mirrors the Win32 MSG structure.
type MSG struct {
	HWnd    HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
	Private uint32
}

// WNDCLASSEX mirrors WNDCLASSEXW.
type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

// NOTIFYICONDATA mirrors NOTIFYICONDATAW.
type NOTIFYICONDATA struct {
	CbSize           uint32
	HWnd             HWND
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            HICON
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     HICON
}

func normalizeCallError(err error) error {
	if err == nil {
		return nil
	}
	if errno, ok := err.(syscall.Errno); ok && errno == 0 {
		return nil
	}
	return err
}

func callError(op string, err error) error {
	if e := normalizeCallError(err); e != nil {
		return fmt.Errorf("%s: %w", op, e)
	}
	return fmt.Errorf("%s failed", op)
}

func utf16Ptr(s string) *uint16 {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		// Strings with NUL cannot be represented; return an empty string rather
		// than panicking in a window procedure.
		empty, _ := windows.UTF16PtrFromString("")
		return empty
	}
	return p
}

func copyUTF16(dst []uint16, src string) {
	u := windows.StringToUTF16(src)
	if len(u) > len(dst) {
		u = u[:len(dst)]
		u[len(u)-1] = 0
	}
	copy(dst, u)
}

func hresultError(op string, hr uintptr) error {
	v := int32(uint32(hr))
	if v >= 0 {
		return nil
	}
	return fmt.Errorf("%s failed: HRESULT 0x%08X", op, uint32(hr))
}

func lowWord(v uintptr) uint16 { return uint16(v & 0xffff) }

func boolValue(v uintptr) bool { return v != 0 }

func ptr(v unsafe.Pointer) uintptr { return uintptr(v) }
