// Package config owns the on-disk configuration and its validation.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"copytextpos/internal/hotkey"
	"copytextpos/internal/render"
)

// CurrentVersion is the config schema version.
const CurrentVersion = 3

// ErrNotFound means no config file exists yet.
var ErrNotFound = errors.New("config file not found")

// Config is serialized to %APPDATA%\CopyTextPos\config.json.
type Config struct {
	Version                  int      `json:"version"`
	Hotkey                   string   `json:"hotkey"`
	Format                   string   `json:"format"`
	SearchRoots              []string `json:"searchRoots"`
	EnableVSCodeBridge       bool     `json:"enableVSCodeBridge"`
	EnableJetBrainsReference bool     `json:"enableJetBrainsReference"`
	EnableEditorRecents      bool     `json:"enableEditorRecents"`
	Notifications            bool     `json:"notifications"`
	StartWithWindows         bool     `json:"startWithWindows"`
	ReleaseHotkeyProcesses   []string `json:"releaseHotkeyProcesses"`
	MaxSearchMs              int      `json:"maxSearchMs"`

	// Migrated is transient: true when Load upgraded an older schema.
	Migrated bool `json:"-"`
}

// Default returns a ready-to-run configuration.
func Default() Config {
	return Config{
		Version:                  CurrentVersion,
		Hotkey:                   hotkey.Default().String(),
		Format:                   render.DefaultTemplate,
		SearchRoots:              []string{},
		EnableVSCodeBridge:       true,
		EnableJetBrainsReference: true,
		EnableEditorRecents:      true,
		Notifications:            true,
		StartWithWindows:         false,
		ReleaseHotkeyProcesses:   []string{"explorer.exe"},
		MaxSearchMs:              1800,
	}
}

// AppDataDir is the roaming data directory.
func AppDataDir() (string, error) {
	base := os.Getenv("APPDATA")
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, "CopyTextPos"), nil
}

// LocalDataDir is the local (non-roaming) data directory.
func LocalDataDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		var err error
		base, err = os.UserCacheDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, "CopyTextPos"), nil
}

// DefaultPath returns the standard config path.
func DefaultPath() (string, error) {
	dir, err := AppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LogPath returns the standard log path.
func LogPath() (string, error) {
	dir, err := LocalDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "copy-text-pos.log"), nil
}

// BridgePath is shared with the optional VS Code/Cursor extension.
func BridgePath() (string, error) {
	dir, err := LocalDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vscode-bridge.json"), nil
}

// Load reads path, preserving defaults for omitted fields.
// If the file does not exist, defaults and ErrNotFound are returned.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, ErrNotFound
		}
		return cfg, err
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("parse %s: %w", path, err)
	}
	incomingVersion := cfg.Version
	if incomingVersion < CurrentVersion {
		cfg.Migrated = true
	}
	// v1 configs stored the unquoted default explicitly. Upgrade that one
	// default to the new quoted form while leaving custom templates alone.
	if incomingVersion < 2 && cfg.Format == render.LegacyDefaultTemplate {
		cfg.Format = render.DefaultTemplate
	}
	// v2 used a single "release in Explorer" boolean. Carry a disabled setting
	// over to the new process list; missing or true keeps the default Explorer.
	var rawFields map[string]json.RawMessage
	if json.Unmarshal(data, &rawFields) == nil {
		if _, hasList := rawFields["releaseHotkeyProcesses"]; !hasList {
			if rawPause, hasPause := rawFields["pauseHotkeyInExplorer"]; hasPause {
				var pause bool
				if json.Unmarshal(rawPause, &pause) == nil && !pause {
					cfg.ReleaseHotkeyProcesses = []string{}
				}
			}
		}
	}
	cfg.Normalize()
	if cfg.Hotkey == "" {
		cfg.Hotkey = hotkey.Default().String()
	}
	if spec, err := hotkey.Parse(cfg.Hotkey); err == nil {
		cfg.Hotkey = spec.String()
	}
	if cfg.Format == "" {
		cfg.Format = render.DefaultTemplate
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Save writes atomically, creating the parent directory on first run.
func Save(path string, cfg Config) error {
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return err
	}
	if spec, err := hotkey.Parse(cfg.Hotkey); err == nil {
		cfg.Hotkey = spec.String()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Normalize cleans paths and clamps numeric values. It never fails.
func (c *Config) Normalize() {
	c.Version = CurrentVersion
	if c.MaxSearchMs <= 0 {
		c.MaxSearchMs = Default().MaxSearchMs
	}
	if c.MaxSearchMs < 300 {
		c.MaxSearchMs = 300
	}
	if c.MaxSearchMs > 10000 {
		c.MaxSearchMs = 10000
	}
	if c.SearchRoots == nil {
		c.SearchRoots = []string{}
	}
	seen := make(map[string]bool, len(c.SearchRoots))
	roots := make([]string, 0, len(c.SearchRoots))
	for _, root := range c.SearchRoots {
		root = strings.TrimSpace(os.ExpandEnv(root))
		if root == "" {
			continue
		}
		root = filepath.Clean(root)
		if !filepath.IsAbs(root) {
			continue
		}
		key := strings.ToLower(root)
		if seen[key] {
			continue
		}
		seen[key] = true
		roots = append(roots, root)
	}
	c.SearchRoots = roots

	if c.ReleaseHotkeyProcesses == nil {
		c.ReleaseHotkeyProcesses = []string{"explorer.exe"}
	}
	processSeen := make(map[string]bool, len(c.ReleaseHotkeyProcesses))
	processes := make([]string, 0, len(c.ReleaseHotkeyProcesses))
	for _, name := range c.ReleaseHotkeyProcesses {
		name = NormalizeProcessName(name)
		if name == "" || processSeen[name] {
			continue
		}
		processSeen[name] = true
		processes = append(processes, name)
	}
	c.ReleaseHotkeyProcesses = processes
}

// NormalizeProcessName converts a user-entered executable path or bare name to
// the lowercase basename used for foreground matching.
func NormalizeProcessName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.Trim(name, `"'`)
	name = filepath.Base(name)
	if name == "" || name == "." {
		return ""
	}
	if !strings.HasSuffix(name, ".exe") {
		name += ".exe"
	}
	return name
}

// ReleasesHotkeyFor reports whether the shortcut should be released while the
// given process executable is in the foreground.
func (c Config) ReleasesHotkeyFor(processName string) bool {
	processName = NormalizeProcessName(processName)
	if processName == "" {
		return false
	}
	for _, configured := range c.ReleaseHotkeyProcesses {
		if configured == processName {
			return true
		}
	}
	return false
}

// Validate returns a fatal error for bad shortcuts/templates and warnings for
// search roots that do not currently exist.
func (c Config) Validate() error {
	if _, err := hotkey.Parse(c.Hotkey); err != nil {
		return fmt.Errorf("hotkey: %w", err)
	}
	if err := render.Validate(c.Format); err != nil {
		return fmt.Errorf("format: %w", err)
	}
	if c.MaxSearchMs < 300 || c.MaxSearchMs > 10000 {
		return fmt.Errorf("maxSearchMs must be between 300 and 10000")
	}
	return nil
}

// Warnings reports non-fatal configuration issues.
func (c Config) Warnings() []string {
	var warnings []string
	for _, root := range c.SearchRoots {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			warnings = append(warnings, fmt.Sprintf("Search root does not exist: %s", root))
		}
	}
	return warnings
}

// ParseHotkey is a convenience wrapper for callers that only need the spec.
func (c Config) ParseHotkey() (hotkey.Spec, error) {
	return hotkey.Parse(c.Hotkey)
}

// LastModified returns the config file modification time, or zero.
func LastModified(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
