//go:build windows

package winapi

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ClipboardSequence returns the global clipboard change counter.
func ClipboardSequence() uint32 {
	r, _, _ := procGetClipboardSequenceNum.Call()
	return uint32(r)
}

func openClipboardRetry() error {
	var last error
	for i := 0; i < 12; i++ {
		r, _, err := procOpenClipboard.Call(0)
		if r != 0 {
			return nil
		}
		last = normalizeCallError(err)
		SleepMilliseconds(10 + i*5)
	}
	if last != nil {
		return fmt.Errorf("OpenClipboard: %w", last)
	}
	return fmt.Errorf("OpenClipboard failed; another application is holding the clipboard")
}

// ReadClipboardText reads CF_UNICODETEXT.
func ReadClipboardText() (string, error) {
	if err := openClipboardRetry(); err != nil {
		return "", err
	}
	defer procCloseClipboard.Call()

	h, _, err := procGetClipboardData.Call(CfUnicodeText)
	if h == 0 {
		return "", callError("GetClipboardData(CF_UNICODETEXT)", err)
	}
	p, _, lockErr := procGlobalLock.Call(h)
	if p == 0 {
		return "", callError("GlobalLock", lockErr)
	}
	defer procGlobalUnlock.Call(h)
	text := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(p)))
	return text, nil
}

// WriteClipboardText replaces the clipboard with CF_UNICODETEXT.
func WriteClipboardText(text string) error {
	text = strings.ReplaceAll(text, "\x00", " ")
	utf16, err := windows.UTF16FromString(text)
	if err != nil {
		return err
	}
	if err := openClipboardRetry(); err != nil {
		return err
	}
	defer procCloseClipboard.Call()

	if r, _, emptyErr := procEmptyClipboard.Call(); r == 0 {
		return callError("EmptyClipboard", emptyErr)
	}
	size := uintptr(len(utf16) * 2)
	h, _, allocErr := procGlobalAlloc.Call(GmemMoveable, size)
	if h == 0 {
		return callError("GlobalAlloc", allocErr)
	}
	owned := true
	defer func() {
		if owned {
			procGlobalFree.Call(h)
		}
	}()
	p, _, lockErr := procGlobalLock.Call(h)
	if p == 0 {
		return callError("GlobalLock", lockErr)
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(utf16)), utf16)
	procGlobalUnlock.Call(h)
	if r, _, setErr := procSetClipboardData.Call(CfUnicodeText, h); r == 0 {
		return callError("SetClipboardData", setErr)
	}
	owned = false // clipboard now owns the HGLOBAL
	return nil
}

// ClearClipboard empties the clipboard, e.g. to restore a previously empty
// clipboard after a failed selection copy.
func ClearClipboard() error {
	if err := openClipboardRetry(); err != nil {
		return err
	}
	defer procCloseClipboard.Call()
	if r, _, err := procEmptyClipboard.Call(); r == 0 {
		return callError("EmptyClipboard", err)
	}
	return nil
}

// ClipboardSnapshot preserves all native clipboard formats across a synthetic
// Ctrl+C when OLE is available. It is thread-affine: create, restore and close
// it on the same locked OS thread.
type ClipboardSnapshot struct {
	data        unsafe.Pointer
	text        string
	hasText     bool
	empty       bool
	oleReady    bool
	initialized bool
}

// NewClipboardSnapshot captures the current clipboard. It never fails: if OLE
// is unavailable it falls back to preserving Unicode text only.
func NewClipboardSnapshot() *ClipboardSnapshot {
	s := &ClipboardSnapshot{}
	if count, _, _ := procCountClipboardFormats.Call(); count == 0 {
		s.empty = true
	}
	if text, err := ReadClipboardText(); err == nil {
		s.text = text
		s.hasText = true
	}
	hr, _, _ := procOleInitialize.Call(0)
	if int32(uint32(hr)) < 0 {
		return s
	}
	s.initialized = true
	var data unsafe.Pointer
	hr, _, _ = procOleGetClipboard.Call(uintptr(unsafe.Pointer(&data)))
	if int32(uint32(hr)) < 0 {
		return s
	}
	s.data = data
	s.oleReady = data != nil
	return s
}

// OLEAvailable reports whether rich clipboard formats are preserved.
func (s *ClipboardSnapshot) OLEAvailable() bool {
	return s != nil && s.oleReady
}

// Restore puts the original clipboard back. Call only after a failed capture.
func (s *ClipboardSnapshot) Restore() error {
	if s == nil {
		return nil
	}
	if s.empty {
		return ClearClipboard()
	}
	// Plain text is restored through the Win32 clipboard first: delayed-render
	// OLE objects from a source process can disappear, and a failed hotkey must
	// never destroy what the user had copied. Images and other non-text formats
	// still go through OLE.
	if s.hasText {
		return WriteClipboardText(s.text)
	}
	if s.data != nil {
		hr, _, _ := procOleSetClipboard.Call(uintptr(s.data))
		if int32(uint32(hr)) < 0 {
			return hresultError("OleSetClipboard", hr)
		}
		procOleFlushClipboard.Call()
		return nil
	}
	return nil
}

// Close releases the captured OLE object. It does not restore.
func (s *ClipboardSnapshot) Close() {
	if s == nil {
		return
	}
	if s.data != nil {
		releaseCOM(s.data)
		s.data = nil
	}
	if s.initialized {
		procOleUninitialize.Call()
		s.initialized = false
	}
}

func releaseCOM(p unsafe.Pointer) {
	if p == nil {
		return
	}
	// The first word of a COM object is its vtable pointer; Release is the
	// third slot (after QueryInterface and AddRef). Read it through typed
	// pointers so go vet's unsafe.Pointer checks stay happy.
	type comObject struct {
		vtbl *[3]uintptr
	}
	obj := (*comObject)(p)
	release := obj.vtbl[2]
	syscall.SyscallN(release, uintptr(p))
}

// WaitForKeysReleased blocks until all vks are up or timeout expires.
func WaitForKeysReleased(vks []uint16, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		down := false
		for _, vk := range vks {
			if IsKeyDown(vk) {
				down = true
				break
			}
		}
		if !down {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		SleepMilliseconds(10)
	}
}
