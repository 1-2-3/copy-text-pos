// Package assets holds binary resources embedded in the executable.
package assets

import _ "embed"

// Icon is the multi-resolution Windows application icon.
//
//go:embed icon.ico
var Icon []byte
