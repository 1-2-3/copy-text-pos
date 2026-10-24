// Package render turns a resolved file position into the text copied to the
// clipboard. The template is deliberately small and documented in the README.
package render

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Location is a file and a 1-based inclusive line range.
type Location struct {
	Path  string
	Start int
	End   int
}

// DefaultTemplate wraps the complete path:line reference in double quotes for
// direct pasting into agents and chat boxes.
const DefaultTemplate = `"{path}:{range}"`

// LegacyDefaultTemplate is the pre-0.1.3 default, used for config migration.
const LegacyDefaultTemplate = "{path}:{range}"

var placeholderRE = regexp.MustCompile(`\{([A-Za-z_]+)\}`)

var knownPlaceholders = map[string]bool{
	"path": true, "dir": true, "name": true, "ext": true,
	"start": true, "end": true, "line": true, "range": true,
	"gh_range": true, "start_line": true, "end_line": true,
}

// Validate checks placeholders without needing a real location.
func Validate(template string) error {
	if strings.TrimSpace(template) == "" {
		return fmt.Errorf("output template is empty")
	}
	matches := placeholderRE.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return fmt.Errorf("output template must contain at least one placeholder")
	}
	for _, m := range matches {
		if !knownPlaceholders[m[1]] {
			return fmt.Errorf("unknown placeholder {%s}", m[1])
		}
	}
	return nil
}

// Render expands template. Start and End must be 1-based.
func Render(template string, loc Location) (string, error) {
	if err := Validate(template); err != nil {
		return "", err
	}
	if loc.Path == "" {
		return "", fmt.Errorf("empty path")
	}
	if loc.Start < 1 {
		loc.Start = 1
	}
	if loc.End < loc.Start {
		loc.End = loc.Start
	}

	start := strconv.Itoa(loc.Start)
	end := strconv.Itoa(loc.End)
	lineRange := start
	if loc.End != loc.Start {
		lineRange = start + "-" + end
	}
	ghRange := "L" + start
	if loc.End != loc.Start {
		ghRange = "L" + start + "-L" + end
	}
	name := filepath.Base(loc.Path)
	ext := strings.TrimPrefix(filepath.Ext(name), ".")

	values := map[string]string{
		"path":       loc.Path,
		"dir":        filepath.Dir(loc.Path),
		"name":       name,
		"ext":        ext,
		"start":      start,
		"end":        end,
		"line":       start,
		"range":      lineRange,
		"gh_range":   ghRange,
		"start_line": start,
		"end_line":   end,
	}

	out := placeholderRE.ReplaceAllStringFunc(template, func(token string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(token, "{"), "}")
		return values[key]
	})
	return out, nil
}

// Preview renders a stable example for the settings page and documentation.
func Preview(template string) (string, error) {
	return Render(template, Location{
		Path:  `D:\projects\android\src\composables\useAsync.ts`,
		Start: 13,
		End:   18,
	})
}
