//go:build windows

// Package capture talks to the focused editor through the clipboard. A Session
// owns a snapshot of the clipboard and restores it if anything goes wrong.
package capture

import (
	"fmt"
	"strings"
	"time"

	"copytextpos/internal/hotkey"
	"copytextpos/internal/winapi"
)

// Session must be used on one locked OS thread because OLE clipboard snapshots
// are apartment-thread affine. Begin and Close bracket the whole operation.
type Session struct {
	hwnd      winapi.HWND
	spec      hotkey.Spec
	snapshot  *winapi.ClipboardSnapshot
	committed bool
	closed    bool
}

// Begin captures the old clipboard. Call runtime.LockOSThread before Begin and
// runtime.UnlockOSThread after Close.
func Begin(hwnd winapi.HWND, spec hotkey.Spec) *Session {
	return &Session{
		hwnd:     hwnd,
		spec:     spec,
		snapshot: winapi.NewClipboardSnapshot(),
	}
}

// OLEAvailable reports whether all clipboard formats can be restored on error.
func (s *Session) OLEAvailable() bool {
	return s.snapshot != nil && s.snapshot.OLEAvailable()
}

// CopySelection asks the editor to copy the current selection.
func (s *Session) CopySelection() (string, error) {
	if winapi.ForegroundWindow() != s.hwnd {
		return "", fmt.Errorf("focus moved away from the editor")
	}
	vks := append(s.spec.ModifierVirtualKeys(), uint16(s.spec.VK))
	if !winapi.WaitForKeysReleased(vks, 1500*time.Millisecond) {
		return "", fmt.Errorf("release the shortcut keys and try again")
	}
	if winapi.ForegroundWindow() != s.hwnd {
		return "", fmt.Errorf("focus moved away from the editor")
	}

	text, changed, err := s.copyWith(winapi.SendCopySelection, 1400*time.Millisecond)
	if err == nil && changed && strings.TrimSpace(text) != "" {
		return text, nil
	}
	// A few editors/extensions remap copy or are more compatible with the
	// traditional Ctrl+Insert. Try it once, but only if Ctrl+C produced nothing.
	altText, altChanged, altErr := s.copyWith(winapi.SendCopySelectionAlt, 700*time.Millisecond)
	if altErr == nil && altChanged && strings.TrimSpace(altText) != "" {
		return altText, nil
	}
	if err != nil {
		return "", err
	}
	if altErr != nil {
		return "", altErr
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("nothing was selected (or the editor copied no text)")
	}
	return "", fmt.Errorf("no text appeared on the clipboard")
}

func (s *Session) copyWith(send func(), timeout time.Duration) (string, bool, error) {
	before := winapi.ClipboardSequence()
	send()
	deadline := time.Now().Add(timeout)
	changed := false
	for {
		seq := winapi.ClipboardSequence()
		if seq != before {
			changed = true
			// Give delayed renderers a moment to publish CF_UNICODETEXT.
			winapi.SleepMilliseconds(25)
			text, err := winapi.ReadClipboardText()
			if err == nil {
				return text, true, nil
			}
		}
		if time.Now().After(deadline) {
			return "", changed, fmt.Errorf("the editor did not put text on the clipboard; select text in its editor pane and try again")
		}
		winapi.SleepMilliseconds(15)
	}
}

// CopyJetBrainsReference sends the stock JetBrains "Copy Reference" shortcut.
// The returned text is normally path:line.
func (s *Session) CopyJetBrainsReference() (string, error) {
	if winapi.ForegroundWindow() != s.hwnd {
		return "", fmt.Errorf("focus moved away from the IDE")
	}
	if !winapi.WaitForKeysReleased([]uint16{winapi.VkControl, winapi.VkShift, winapi.VkMenu, uint16(s.spec.VK)}, 1200*time.Millisecond) {
		return "", fmt.Errorf("release the shortcut keys and try again")
	}
	before := winapi.ClipboardSequence()
	winapi.SendJetBrainsCopyReference()
	deadline := time.Now().Add(900 * time.Millisecond)
	for {
		if winapi.ClipboardSequence() != before {
			winapi.SleepMilliseconds(30)
			text, err := winapi.ReadClipboardText()
			if err == nil && strings.TrimSpace(text) != "" {
				return text, nil
			}
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("JetBrains Copy Reference produced no text")
		}
		winapi.SleepMilliseconds(15)
	}
}

// WriteResult replaces the clipboard with the final path reference and marks
// the session successful.
func (s *Session) WriteResult(text string) error {
	if err := winapi.WriteClipboardText(text); err != nil {
		return err
	}
	s.committed = true
	return nil
}

// Close restores the old clipboard on failure, then releases OLE resources.
func (s *Session) Close() {
	if s == nil || s.closed {
		return
	}
	s.closed = true
	if !s.committed && s.snapshot != nil {
		_ = s.snapshot.Restore()
	}
	if s.snapshot != nil {
		s.snapshot.Close()
		s.snapshot = nil
	}
}
