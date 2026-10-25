// Package textmatch maps a clipboard selection back to a line range in a file.
//
// Editors rarely expose file line numbers through a Windows API. The robust
// common denominator is to copy the selected text and locate that text inside
// the file on disk. Exact matching is tried first, followed by a conservative
// whitespace-insensitive pass and finally a Markdown-aware pass for WYSIWYG
// editors such as Typora and Obsidian.
package textmatch

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrEmptySelection means there was nothing meaningful to locate.
var ErrEmptySelection = errors.New("selection is empty")

// Mode describes how a match was obtained.
type Mode string

const (
	ModeExact      Mode = "exact"
	ModeWhitespace Mode = "whitespace"
	ModeMarkdown   Mode = "markdown"
)

// Options controls fallback matching.
type Options struct {
	// Markdown enables Markdown syntax stripping after exact and whitespace
	// matching both fail. Use it for rendered Markdown editors, not for code.
	Markdown bool
}

// Match is an inclusive 1-based line range.
type Match struct {
	StartLine int  `json:"startLine"`
	EndLine   int  `json:"endLine"`
	Mode      Mode `json:"mode"`
}

const maxMatches = 500

// FindAll returns possible line ranges for selection in content.
// content is expected to use LF newlines, but CRLF is normalized defensively.
func FindAll(content, selection string, opts Options) ([]Match, error) {
	sel := strings.TrimSpace(selection)
	if sel == "" {
		return nil, ErrEmptySelection
	}
	if len(selection) > 8<<20 {
		return nil, fmt.Errorf("selection is too large (%d bytes)", len(selection))
	}

	content = normalizeNewlines(content)
	selection = normalizeNewlines(selection)
	selection = trimOneTrailingNewline(selection)
	if strings.TrimSpace(selection) == "" {
		return nil, ErrEmptySelection
	}

	lines, starts := splitLines(content)
	if matches := exactMatches(content, selection, starts); len(matches) > 0 {
		return matches, nil
	}

	if len(selection) <= 1<<20 {
		selectedLines := strings.Split(selection, "\n")
		if len(selectedLines) <= 1000 {
			if matches := lineMatches(lines, selectedLines, false); len(matches) > 0 {
				return matches, nil
			}
			if opts.Markdown {
				if matches := lineMatches(lines, selectedLines, true); len(matches) > 0 {
					return matches, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("selected text was not found in the file")
}

// Find returns all matches and chooses the one nearest to anchorLine when one
// is supplied. anchorIndex is the returned slice index.
func Find(content, selection string, opts Options, anchorLine int) (Match, int, int, error) {
	matches, err := FindAll(content, selection, opts)
	if err != nil {
		return Match{}, 0, 0, err
	}
	idx := 0
	if anchorLine > 0 {
		idx = chooseNearest(matches, anchorLine)
	}
	return matches[idx], idx, len(matches), nil
}

func chooseNearest(matches []Match, anchor int) int {
	best := 0
	bestDistance := int(^uint(0) >> 1)
	for i, m := range matches {
		distance := 0
		switch {
		case anchor < m.StartLine:
			distance = m.StartLine - anchor
		case anchor > m.EndLine:
			distance = anchor - m.EndLine
		}
		if distance < bestDistance {
			bestDistance = distance
			best = i
			if distance == 0 {
				break
			}
		}
	}
	return best
}

func exactMatches(content, selection string, starts []int) []Match {
	matches := make([]Match, 0, 4)
	offset := 0
	for len(matches) < maxMatches {
		i := strings.Index(content[offset:], selection)
		if i < 0 {
			break
		}
		at := offset + i
		startLine := lineAt(starts, at)
		// The range is inclusive. If the selection ends with a line break it
		// was already trimmed, so counting embedded breaks gives the last line.
		endLine := startLine + strings.Count(selection, "\n")
		matches = append(matches, Match{StartLine: startLine, EndLine: endLine, Mode: ModeExact})
		offset = at + 1
	}
	return dropDuplicates(matches)
}

func lineMatches(fileLines, selectedRaw []string, markdown bool) []Match {
	if len(fileLines) == 0 || len(selectedRaw) == 0 {
		return nil
	}
	selected := make([]string, 0, len(selectedRaw))
	for _, line := range selectedRaw {
		// Rendered Markdown tables may or may not include the separator row in
		// the copied text. It carries no user-visible content, so ignore it on
		// both sides and keep the source line mapping separately.
		if markdown && isMarkdownTableSeparator(line) {
			continue
		}
		normalized := normalizeLine(line, markdown)
		// Typora inserts blank lines between ordered-list items that are
		// adjacent in the source. Blank lines do not carry selectable text, so
		// ignoring them keeps the non-blank source lines aligned.
		if markdown && normalized == "" {
			continue
		}
		selected = append(selected, normalized)
	}
	// Drop leading/trailing blank lines created by the clipboard.
	for len(selected) > 0 && selected[0] == "" {
		selected = selected[1:]
	}
	for len(selected) > 0 && selected[len(selected)-1] == "" {
		selected = selected[:len(selected)-1]
	}
	if len(selected) == 0 {
		return nil
	}

	type sourceLine struct {
		text string
		line int
	}
	normalizedFile := make([]sourceLine, 0, len(fileLines))
	for i, line := range fileLines {
		if markdown && isMarkdownTableSeparator(line) {
			continue
		}
		normalized := normalizeLine(line, markdown)
		if markdown && normalized == "" {
			continue
		}
		normalizedFile = append(normalizedFile, sourceLine{
			text: normalized,
			line: i + 1,
		})
	}
	if len(selected) > len(normalizedFile) {
		return nil
	}

	mode := ModeWhitespace
	if markdown {
		mode = ModeMarkdown
	}
	matches := make([]Match, 0, 2)
	for i := 0; i+len(selected) <= len(normalizedFile) && len(matches) < maxMatches; i++ {
		ok := true
		for j := range selected {
			if normalizedFile[i+j].text != selected[j] {
				ok = false
				break
			}
		}
		if ok {
			matches = append(matches, Match{
				StartLine: normalizedFile[i].line,
				EndLine:   normalizedFile[i+len(selected)-1].line,
				Mode:      mode,
			})
		}
	}
	return matches
}

// isMarkdownTableSeparator identifies rows such as |---|:---:|---|. Typora
// reformats these with padding when it copies a table, so the exact text
// differs from the source and must not participate in line matching.
func isMarkdownTableSeparator(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" || !strings.Contains(line, "|") {
		return false
	}
	hasDash := false
	for _, r := range line {
		switch r {
		case '|', ' ', '\t':
		case '-':
			hasDash = true
		case ':':
		default:
			return false
		}
	}
	return hasDash
}

func normalizeLine(line string, markdown bool) string {
	line = strings.ReplaceAll(line, "\t", " ")
	if markdown {
		line = stripMarkdown(line)
	}
	line = strings.TrimSpace(line)
	line = strings.Join(strings.Fields(line), " ")
	return line
}

var (
	markdownHeadingRE = regexp.MustCompile(`^#{1,6}\s*`)
	markdownQuoteRE   = regexp.MustCompile(`^(?:>\s*)+`)
	markdownListRE    = regexp.MustCompile(`^(?:[-*+]|\d+[.)])\s+`)
	markdownImageRE   = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	markdownLinkRE    = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	htmlTagRE         = regexp.MustCompile(`<[^>]+>`)
)

func stripMarkdown(line string) string {
	line = strings.TrimSpace(line)
	line = markdownQuoteRE.ReplaceAllString(line, "")
	line = markdownHeadingRE.ReplaceAllString(line, "")
	line = markdownListRE.ReplaceAllString(line, "")
	line = markdownImageRE.ReplaceAllString(line, "$1")
	line = markdownLinkRE.ReplaceAllString(line, "$1")
	line = htmlTagRE.ReplaceAllString(line, "")
	for _, marker := range []string{"**", "__", "~~", "`", "*", "_"} {
		line = strings.ReplaceAll(line, marker, "")
	}
	line = strings.ReplaceAll(line, `\`, "")
	return line
}

func splitLines(content string) ([]string, []int) {
	lines := strings.Split(content, "\n")
	starts := make([]int, len(lines))
	offset := 0
	for i, line := range lines {
		starts[i] = offset
		offset += len(line) + 1
	}
	return lines, starts
}

func lineAt(starts []int, offset int) int {
	lo, hi := 0, len(starts)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if starts[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\u00a0", " ")
}

func trimOneTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return strings.TrimSuffix(s, "\n")
	}
	return s
}

func dropDuplicates(matches []Match) []Match {
	if len(matches) < 2 {
		return matches
	}
	out := matches[:0]
	seen := map[[2]int]bool{}
	for _, m := range matches {
		key := [2]int{m.StartLine, m.EndLine}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, m)
	}
	return out
}

// EstimatedEndLine estimates the end line from an anchor and the number of
// visual lines in the selection. It is used only when the file changed on disk
// (for example an unsaved editor buffer) and an anchor is available.
func EstimatedEndLine(anchor int, selection string) int {
	selection = normalizeNewlines(selection)
	selection = trimOneTrailingNewline(selection)
	n := strings.Count(selection, "\n")
	if n < 0 {
		n = 0
	}
	return anchor + n
}
