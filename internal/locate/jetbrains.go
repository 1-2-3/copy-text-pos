//go:build windows

package locate

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	referenceLineRE = regexp.MustCompile(`^(.*?\.\w{1,12}):(\d{1,7})(?::\d{1,7})?$`)
	referenceHashRE = regexp.MustCompile(`^(.*?\.\w{1,12})#L(\d{1,7})(?:-L\d{1,7})?$`)
)

// ParseReference parses JetBrains "Copy Reference" text such as
// "src/main/java/Foo.java:42" and resolves it against known project roots.
func ParseReference(text string, projectRoots, searchRoots []string) (PathHint, bool) {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "`\"'")
	if text == "" || strings.Contains(text, "\n") {
		return PathHint{}, false
	}
	anchor := 0
	rawPath := ""
	if m := referenceLineRE.FindStringSubmatch(text); m != nil {
		rawPath = m[1]
		anchor, _ = strconv.Atoi(m[2])
	} else if m := referenceHashRE.FindStringSubmatch(text); m != nil {
		rawPath = m[1]
		anchor, _ = strconv.Atoi(m[2])
	} else if looksAbsolute(filepath.FromSlash(text)) {
		rawPath = text
	} else {
		return PathHint{}, false
	}
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return PathHint{}, false
	}
	if !looksAbsolute(filepath.FromSlash(rawPath)) && filepath.Ext(rawPath) == "" {
		return PathHint{}, false
	}
	rawPath = filepath.FromSlash(rawPath)

	if looksAbsolute(rawPath) {
		candidate := filepath.Clean(rawPath)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return PathHint{Path: candidate, AnchorLine: anchor, Origin: "jetbrains-reference"}, true
		}
		return PathHint{}, false
	}

	rawPath = strings.TrimLeft(rawPath, `.\/`)
	roots := append([]string(nil), projectRoots...)
	roots = append(roots, searchRoots...)
	seen := map[string]bool{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		candidate := filepath.Clean(filepath.Join(root, rawPath))
		key := strings.ToLower(candidate)
		if seen[key] {
			continue
		}
		seen[key] = true
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return PathHint{Path: candidate, AnchorLine: anchor, Origin: "jetbrains-reference"}, true
		}
	}
	return PathHint{}, false
}

// JetBrainsTitleProject returns the project name from titles such as
// "MyProject – MainActivity.java".
func JetBrainsTitleProject(title string) string {
	segments := TitleSegments(title)
	if len(segments) == 0 {
		return ""
	}
	return CleanTitlePart(segments[0])
}

// JetBrainsProjectRootsForTitle chooses recent project roots whose basename
// matches the title. If nothing matches, every recent root is returned.
func JetBrainsProjectRootsForTitle(title string, all []string) []string {
	project := JetBrainsTitleProject(title)
	if project == "" {
		return all
	}
	var matches []string
	for _, root := range all {
		if strings.EqualFold(filepath.Base(root), project) {
			matches = append(matches, root)
		}
	}
	if len(matches) > 0 {
		return matches
	}
	return all
}
