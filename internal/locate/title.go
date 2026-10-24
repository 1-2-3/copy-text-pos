//go:build windows

package locate

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// PathHint is a file path inferred from a window title.
type PathHint struct {
	Path       string
	AnchorLine int
	Origin     string
}

var (
	pathLineRE  = regexp.MustCompile(`^(.*\.[A-Za-z0-9]{1,12}):(\d{1,7})(?:-(\d{1,7}))?$`)
	pathParenRE = regexp.MustCompile(`^(.*\.[A-Za-z0-9]{1,12})\s*[\(\[](\d{1,7})[\)\]]$`)
	lineParenRE = regexp.MustCompile(`\s[\(\[](\d{1,7})[\)\]]\s*$`)
	appSuffixes = []string{
		"Visual Studio Code - Insiders", "Visual Studio Code", "VSCodium",
		"Cursor", "Windsurf", "Trae", "Kiro", "Positron", "Void",
		"Typora", "Obsidian", "Sublime Text", "Notepad++", "EmEditor",
	}
)

// TitleSegments splits common editor title separators.
func TitleSegments(title string) []string {
	title = strings.ReplaceAll(title, " – ", " - ")
	title = strings.ReplaceAll(title, " — ", " - ")
	title = strings.ReplaceAll(title, " | ", " - ")
	parts := strings.Split(title, " - ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// StripAppSuffix removes the product name from the end of a title.
func StripAppSuffix(title string) string {
	segments := TitleSegments(title)
	for len(segments) > 0 {
		last := strings.TrimSpace(segments[len(segments)-1])
		if !isAppSuffix(last) {
			break
		}
		segments = segments[:len(segments)-1]
	}
	return strings.Join(segments, " - ")
}

// isAppSuffix also accepts versioned title suffixes such as
// "Obsidian 1.14.5" or "Sublime Text 4".
func isAppSuffix(segment string) bool {
	for _, suffix := range appSuffixes {
		if strings.EqualFold(segment, suffix) {
			return true
		}
		if len(segment) <= len(suffix) || !strings.EqualFold(segment[:len(suffix)], suffix) {
			continue
		}
		if segment[len(suffix)] != ' ' {
			continue
		}
		if looksLikeVersion(strings.TrimSpace(segment[len(suffix):])) {
			return true
		}
	}
	return false
}

func looksLikeVersion(tail string) bool {
	tail = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(tail), "v"))
	if tail == "" || tail[0] < '0' || tail[0] > '9' {
		return false
	}
	for _, r := range tail {
		if (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '(' || r == ')' {
			continue
		}
		return false
	}
	return true
}

// CleanTitlePart removes modification markers and decorative bullets.
func CleanTitlePart(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "●○•* ")
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, " *")
	s = strings.TrimSuffix(s, " (Modified)")
	s = strings.TrimSpace(s)
	return s
}

// AbsolutePathHints extracts existing absolute file paths from a title.
func AbsolutePathHints(title string) []PathHint {
	var candidates []string
	full := CleanTitlePart(title)
	candidates = append(candidates, full)

	stripped := CleanTitlePart(StripAppSuffix(title))
	if stripped != "" {
		candidates = append(candidates, stripped)
	}

	// Quoted paths survive separators inside the path.
	for _, quote := range []byte{'"', '\''} {
		start := -1
		for i := 0; i < len(title); i++ {
			if title[i] == quote {
				if start < 0 {
					start = i + 1
				} else {
					candidates = append(candidates, title[start:i])
					start = -1
				}
			}
		}
	}

	for _, segment := range TitleSegments(title) {
		candidates = append(candidates, CleanTitlePart(segment))
	}

	seen := map[string]bool{}
	hints := make([]PathHint, 0, 2)
	for _, candidate := range candidates {
		for _, variant := range pathVariants(candidate) {
			path, anchor, ok := normalizePathToken(variant)
			if !ok {
				continue
			}
			key := strings.ToLower(path)
			if seen[key] {
				continue
			}
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			seen[key] = true
			hints = append(hints, PathHint{Path: path, AnchorLine: anchor, Origin: "window-title"})
		}
	}
	return hints
}

func pathVariants(candidate string) []string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return nil
	}
	out := []string{candidate}
	// Remove title segments from the right until only the path remains.
	for _, separator := range []string{" - ", " – ", " — ", " | "} {
		searchEnd := len(candidate)
		for {
			idx := strings.LastIndex(candidate[:searchEnd], separator)
			if idx < 0 {
				break
			}
			out = append(out, strings.TrimSpace(candidate[:idx]))
			searchEnd = idx
		}
	}
	out = append(out, strings.Trim(candidate, "\"'"))
	return out
}

func normalizePathToken(token string) (string, int, bool) {
	token = strings.TrimSpace(token)
	token = strings.Trim(token, "\"'()[]{}<>")
	token = strings.TrimSpace(token)
	token = strings.TrimLeft(token, "●○•* ")
	if token == "" {
		return "", 0, false
	}
	anchor := 0
	if m := pathLineRE.FindStringSubmatch(token); m != nil {
		token = m[1]
		anchor, _ = strconv.Atoi(m[2])
	} else if m := pathParenRE.FindStringSubmatch(token); m != nil {
		token = m[1]
		anchor, _ = strconv.Atoi(m[2])
	}
	token = strings.TrimSpace(token)
	token = strings.TrimSuffix(token, " *")
	token = strings.TrimRight(token, ".,;:")
	if !looksAbsolute(token) {
		return "", 0, false
	}
	token = filepath.FromSlash(token)
	token = filepath.Clean(token)
	// Normalize a lowercase drive letter to the conventional uppercase form.
	if len(token) >= 2 && token[1] == ':' {
		token = strings.ToUpper(token[:1]) + token[1:]
	}
	if anchor < 1 {
		anchor = 0
	}
	return token, anchor, true
}

func looksAbsolute(path string) bool {
	if len(path) >= 3 && isDriveLetter(path[0]) && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		return true
	}
	if strings.HasPrefix(path, `\\`) && len(path) > 3 {
		return true
	}
	return false
}

func isDriveLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// TitleAnchor looks for a caret line indicator in a generic title, e.g.
// "file.ts:42" or "file.ts (42) - Editor".
func TitleAnchor(title string) int {
	if hints := AbsolutePathHints(title); len(hints) > 0 && hints[0].AnchorLine > 0 {
		return hints[0].AnchorLine
	}
	stripped := StripAppSuffix(title)
	if m := regexp.MustCompile(`(?i)\.[a-z0-9]{1,12}:(\d{1,7})(?:-\d{1,7})?`).FindStringSubmatch(stripped); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	if m := regexp.MustCompile(`(?i)\.[a-z0-9]{1,12}\s*[\(\[](\d{1,7})[\)\]]`).FindStringSubmatch(stripped); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	if m := lineParenRE.FindStringSubmatch(stripped); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

// VSCodeTitle holds the parts of a VS Code family title.
type VSCodeTitle struct {
	FilePart  string
	Workspace string
	FilePath  string
	Anchor    int
}

// ParseVSCodeTitle parses "file.ts - project - Cursor". Filenames containing
// " - " are handled by joining all segments before the last one.
func ParseVSCodeTitle(title string) VSCodeTitle {
	hints := AbsolutePathHints(title)
	out := VSCodeTitle{}
	if len(hints) > 0 {
		out.FilePath = hints[0].Path
		out.Anchor = hints[0].AnchorLine
		out.FilePart = filepath.Base(hints[0].Path)
	}
	segments := TitleSegments(title)
	if len(segments) == 0 {
		return out
	}
	// Drop a known product suffix.
	if len(segments) > 1 {
		last := strings.TrimSpace(segments[len(segments)-1])
		for _, label := range appSuffixes {
			if strings.EqualFold(last, label) {
				segments = segments[:len(segments)-1]
				break
			}
		}
	}
	segments[0] = CleanTitlePart(segments[0])
	if len(segments) == 1 {
		out.FilePart = segments[0]
		return out
	}
	out.Workspace = CleanTitlePart(segments[len(segments)-1])
	out.FilePart = CleanTitlePart(strings.Join(segments[:len(segments)-1], " - "))
	if out.FilePart == "" {
		out.FilePart = segments[0]
	}
	return out
}

// FileNameFromTitle returns the most likely document name for a generic
// editor. It removes a known product suffix and modification markers.
func FileNameFromTitle(title string) string {
	stripped := CleanTitlePart(StripAppSuffix(title))
	if stripped == "" {
		return ""
	}
	segments := TitleSegments(stripped)
	if len(segments) == 0 {
		return ""
	}
	// Prefer a segment that looks like a file name (has an extension).
	for i := len(segments) - 1; i >= 0; i-- {
		name := CleanTitlePart(segments[i])
		if filepath.Ext(name) != "" {
			return name
		}
	}
	return CleanTitlePart(segments[0])
}
