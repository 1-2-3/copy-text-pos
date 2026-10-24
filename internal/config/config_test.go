package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.json")
	cfg := Default()
	cfg.Hotkey = "ctrl+alt+p"
	cfg.SearchRoots = []string{dir, dir, "relative", "  "}
	cfg.Notifications = false
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hotkey != "Ctrl+Alt+P" || got.Notifications {
		t.Fatalf("unexpected config: %+v", got)
	}
	if got.Format != `"{path}:{range}"` || got.Version != CurrentVersion {
		t.Fatalf("unexpected format/version: %+v", got)
	}
	if len(got.SearchRoots) != 1 || got.SearchRoots[0] != filepath.Clean(dir) {
		t.Fatalf("roots not normalized: %#v", got.SearchRoots)
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "none.json"))
	if err != ErrNotFound {
		t.Fatalf("err=%v want ErrNotFound", err)
	}
	if cfg.Hotkey != "Ctrl+Shift+C" || cfg.Format != `"{path}:{range}"` {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestMigrateQuotedDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := []byte(`{"version":1,"hotkey":"Ctrl+Shift+C","format":"{path}:{range}"}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Migrated || cfg.Version != CurrentVersion {
		t.Fatalf("not migrated: %+v", cfg)
	}
	if cfg.Format != `"{path}:{range}"` {
		t.Fatalf("format=%q want quoted default", cfg.Format)
	}

	// A custom template from v1 must survive migration untouched.
	custom := []byte(`{"version":1,"hotkey":"Ctrl+Shift+C","format":"{path}:{start}"}`)
	if err := os.WriteFile(path, custom, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Format != "{path}:{start}" {
		t.Fatalf("custom format changed: %q", cfg.Format)
	}
}

func TestInvalidConfigRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"hotkey":"C","format":"{path}"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected hotkey validation error")
	}
}

func TestLoadUTF8BOM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"hotkey":"Ctrl+Alt+P","format":"{path}:{range}"}`)...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hotkey != "Ctrl+Alt+P" {
		t.Fatalf("hotkey=%q", cfg.Hotkey)
	}
}
