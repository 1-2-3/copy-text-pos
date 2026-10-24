// Package version exposes build metadata. Values are injected with -ldflags.
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version, e.g. "0.1.0".
	Version = "dev"
	// Commit is the short git commit hash.
	Commit = "unknown"
	// BuildTime is an RFC 3339 build timestamp.
	BuildTime = "unknown"
)

// String returns a compact human-readable version string.
func String() string {
	if Commit == "" || Commit == "unknown" {
		return Version
	}
	return fmt.Sprintf("%s (%s)", Version, Commit)
}

// Full returns all build metadata.
func Full() string {
	return fmt.Sprintf("Copy Text Pos %s\ncommit: %s\nbuilt: %s\n%s/%s",
		Version, Commit, BuildTime, runtime.GOOS, runtime.GOARCH)
}
