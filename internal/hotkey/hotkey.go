// Package hotkey parses and formats Windows global-hotkey combinations.
//
// Only a compact, portable subset of virtual-key names is accepted. Keeping the
// grammar small makes the value safe to round-trip through config.json and the
// settings page.
package hotkey

import (
	"fmt"
	"sort"
	"strings"
)

// Windows modifier bits used by RegisterHotKey.
const (
	ModAlt      uint32 = 0x0001
	ModControl  uint32 = 0x0002
	ModShift    uint32 = 0x0004
	ModWin      uint32 = 0x0008
	ModNoRepeat uint32 = 0x4000
)

// Spec is a parsed global shortcut.
type Spec struct {
	Mods uint32
	VK   uint32
	Key  string
}

// Default is the shortcut advertised by the project.
func Default() Spec {
	return Spec{Mods: ModControl | ModShift, VK: 'C', Key: "C"}
}

// String returns a canonical label such as "Ctrl+Shift+C".
func (s Spec) String() string {
	parts := make([]string, 0, 5)
	if s.Mods&ModControl != 0 {
		parts = append(parts, "Ctrl")
	}
	if s.Mods&ModShift != 0 {
		parts = append(parts, "Shift")
	}
	if s.Mods&ModAlt != 0 {
		parts = append(parts, "Alt")
	}
	if s.Mods&ModWin != 0 {
		parts = append(parts, "Win")
	}
	key := s.Key
	if key == "" {
		key = KeyName(s.VK)
	}
	parts = append(parts, key)
	return strings.Join(parts, "+")
}

// Parse accepts values such as "ctrl + shift + c", "Alt+F12" or "Win+Shift+S".
func Parse(value string) (Spec, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return Spec{}, fmt.Errorf("shortcut is empty")
	}
	pieces := strings.Split(raw, "+")
	if len(pieces) < 2 {
		return Spec{}, fmt.Errorf("shortcut %q needs at least one modifier and one key", value)
	}

	var mods uint32
	var keyToken string
	for _, piece := range pieces {
		token := strings.TrimSpace(piece)
		if token == "" {
			return Spec{}, fmt.Errorf("shortcut %q has an empty component", value)
		}
		switch strings.ToLower(token) {
		case "ctrl", "control":
			mods |= ModControl
		case "shift":
			mods |= ModShift
		case "alt", "menu":
			mods |= ModAlt
		case "win", "windows", "meta", "super":
			mods |= ModWin
		default:
			if keyToken != "" {
				return Spec{}, fmt.Errorf("shortcut %q has more than one non-modifier key", value)
			}
			keyToken = token
		}
	}
	if mods == 0 {
		return Spec{}, fmt.Errorf("shortcut %q needs Ctrl, Shift, Alt or Win", value)
	}
	if keyToken == "" {
		return Spec{}, fmt.Errorf("shortcut %q has no key", value)
	}
	vk, canonical, ok := lookupKey(keyToken)
	if !ok {
		return Spec{}, fmt.Errorf("unsupported key %q", keyToken)
	}
	return Spec{Mods: mods, VK: vk, Key: canonical}, nil
}

// ModifierVirtualKeys returns the virtual keys that must be released before a
// synthetic Ctrl+C can be sent safely.
func (s Spec) ModifierVirtualKeys() []uint16 {
	keys := make([]uint16, 0, 5)
	if s.Mods&ModControl != 0 {
		keys = append(keys, 0x11)
	}
	if s.Mods&ModShift != 0 {
		keys = append(keys, 0x10)
	}
	if s.Mods&ModAlt != 0 {
		keys = append(keys, 0x12)
	}
	if s.Mods&ModWin != 0 {
		keys = append(keys, 0x5B, 0x5C)
	}
	return keys
}

var namedKeys = map[string]struct {
	vk   uint32
	name string
}{
	"space":        {0x20, "Space"},
	"enter":        {0x0D, "Enter"},
	"return":       {0x0D, "Enter"},
	"tab":          {0x09, "Tab"},
	"backspace":    {0x08, "Backspace"},
	"delete":       {0x2E, "Delete"},
	"del":          {0x2E, "Delete"},
	"insert":       {0x2D, "Insert"},
	"ins":          {0x2D, "Insert"},
	"home":         {0x24, "Home"},
	"end":          {0x23, "End"},
	"pageup":       {0x21, "PageUp"},
	"pagedown":     {0x22, "PageDown"},
	"up":           {0x26, "Up"},
	"down":         {0x28, "Down"},
	"left":         {0x25, "Left"},
	"right":        {0x27, "Right"},
	"pause":        {0x13, "Pause"},
	"printscreen":  {0x2C, "PrintScreen"},
	"scrolllock":   {0x91, "ScrollLock"},
	"capslock":     {0x14, "CapsLock"},
	"numlock":      {0x90, "NumLock"},
	"minus":        {0xBD, "Minus"},
	"equal":        {0xBB, "Equal"},
	"comma":        {0xBC, "Comma"},
	"period":       {0xBE, "Period"},
	"slash":        {0xBF, "Slash"},
	"backslash":    {0xDC, "Backslash"},
	"semicolon":    {0xBA, "Semicolon"},
	"quote":        {0xDE, "Quote"},
	"leftbracket":  {0xDB, "LeftBracket"},
	"rightbracket": {0xDD, "RightBracket"},
	"backtick":     {0xC0, "Backtick"},
	"grave":        {0xC0, "Backtick"},
}

func lookupKey(token string) (vk uint32, canonical string, ok bool) {
	upper := strings.ToUpper(strings.TrimSpace(token))
	if len(upper) == 1 {
		c := upper[0]
		if c >= 'A' && c <= 'Z' {
			return uint32(c), upper, true
		}
		if c >= '0' && c <= '9' {
			return uint32(c), upper, true
		}
	}
	if strings.HasPrefix(upper, "F") {
		var n int
		if _, err := fmt.Sscanf(upper, "F%d", &n); err == nil && n >= 1 && n <= 24 {
			return uint32(0x70 + n - 1), fmt.Sprintf("F%d", n), true
		}
	}
	if strings.HasPrefix(upper, "NUM") {
		var n int
		if _, err := fmt.Sscanf(upper, "NUM%d", &n); err == nil && n >= 0 && n <= 9 {
			return uint32(0x60 + n), fmt.Sprintf("Num%d", n), true
		}
	}
	if entry, found := namedKeys[strings.ToLower(token)]; found {
		return entry.vk, entry.name, true
	}
	return 0, "", false
}

// KeyName returns a display name for a virtual key, or its hex value.
func KeyName(vk uint32) string {
	if vk >= 'A' && vk <= 'Z' {
		return string(rune(vk))
	}
	if vk >= '0' && vk <= '9' {
		return string(rune(vk))
	}
	if vk >= 0x70 && vk <= 0x87 {
		return fmt.Sprintf("F%d", vk-0x70+1)
	}
	if vk >= 0x60 && vk <= 0x69 {
		return fmt.Sprintf("Num%d", vk-0x60)
	}
	for _, entry := range namedKeys {
		if entry.vk == vk {
			return entry.name
		}
	}
	return fmt.Sprintf("0x%02X", vk)
}

// KnownKeys is used by the settings UI documentation and tests.
func KnownKeys() []string {
	keys := []string{"A-Z", "0-9", "F1-F24"}
	names := make([]string, 0, len(namedKeys))
	seen := map[string]bool{}
	for _, entry := range namedKeys {
		if !seen[entry.name] {
			seen[entry.name] = true
			names = append(names, entry.name)
		}
	}
	sort.Strings(names)
	return append(keys, names...)
}
