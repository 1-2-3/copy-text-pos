//go:build windows

// Package autostart manages the per-user Run registry entry.
package autostart

import (
	"errors"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName = "CopyTextPos"
)

// Set enables or disables starting the app at user sign-in.
func Set(enabled bool, executable string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if !enabled {
		err = key.DeleteValue(valueName)
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	executable = strings.TrimSpace(executable)
	if executable == "" {
		return errors.New("empty executable path")
	}
	return key.SetStringValue(valueName, `"`+executable+`"`)
}

// IsEnabled reports whether the Run entry exists.
func IsEnabled() (bool, error) {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer key.Close()
	_, _, err = key.GetStringValue(valueName)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
