// Package logging provides a small rotating file logger.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const maxSize = 2 << 20

// Logger writes human-readable diagnostics without selected text.
type Logger struct {
	mu    sync.Mutex
	path  string
	file  *os.File
	base  *log.Logger
	debug bool
}

// New opens the log file, rotating a previous oversized file.
func New(path string, debug bool) *Logger {
	l := &Logger{path: path, debug: debug}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	if info, err := os.Stat(path); err == nil && info.Size() > maxSize {
		_ = os.Remove(path + ".1")
		_ = os.Rename(path, path+".1")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	var writer io.Writer
	if err != nil {
		writer = os.Stderr
	} else {
		l.file = f
		writer = f
		if debug {
			writer = io.MultiWriter(f, os.Stderr)
		}
	}
	if debug && l.file == nil {
		writer = os.Stderr
	} else if debug {
		writer = io.MultiWriter(l.file, os.Stderr)
	}
	l.base = log.New(writer, "", log.LstdFlags|log.Lmicroseconds)
	return l
}

// Path returns the active log path.
func (l *Logger) Path() string { return l.path }

// Info records an informational event.
func (l *Logger) Info(format string, args ...any) {
	if l == nil || l.base == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.base.Printf("INFO  "+format, args...)
}

// Error records an error.
func (l *Logger) Error(format string, args ...any) {
	if l == nil || l.base == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.base.Printf("ERROR "+format, args...)
}

// Close closes the underlying file.
func (l *Logger) Close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}

// EnsureDir is a helper for callers creating data directories.
func EnsureDir(path string) error {
	if path == "" {
		return fmt.Errorf("empty directory")
	}
	return os.MkdirAll(path, 0o700)
}
