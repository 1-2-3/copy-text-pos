//go:build windows

package locate

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"copytextpos/internal/textmatch"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// Options configures path discovery.
type Options struct {
	SearchRoots              []string
	EnableRecents            bool
	EnableBridge             bool
	EnableJetBrainsReference bool
	MaxSearch                time.Duration
	BridgePath               string
}

// Request is one resolution attempt.
type Request struct {
	Window        Window
	Selection     string
	ReferenceText string
}

// Candidate is one possible source file.
type Candidate struct {
	Path   string
	Origin string
	Score  int
	Anchor int
}

// Result is a confirmed file position.
type Result struct {
	Path       string
	StartLine  int
	EndLine    int
	Origin     string
	Mode       string
	Matches    int
	Candidates int
	Note       string
}

type cachedFile struct {
	modTime time.Time
	size    int64
	text    string
}

// Resolver owns caches and configuration for path discovery.
type Resolver struct {
	mu    sync.Mutex
	opts  Options
	cache map[string]cachedFile
}

// NewResolver creates a resolver with the supplied options.
func NewResolver(opts Options) *Resolver {
	if opts.MaxSearch <= 0 {
		opts.MaxSearch = 1800 * time.Millisecond
	}
	return &Resolver{
		opts:  opts,
		cache: make(map[string]cachedFile),
	}
}

// SetOptions updates configuration.
func (r *Resolver) SetOptions(opts Options) {
	if opts.MaxSearch <= 0 {
		opts.MaxSearch = 1800 * time.Millisecond
	}
	r.mu.Lock()
	r.opts = opts
	r.mu.Unlock()
}

// Options returns a copy of the current options.
func (r *Resolver) Options() Options {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.opts
}

// BridgeResult returns an exact result from the VS Code companion extension.
func (r *Resolver) BridgeResult(w Window) (Result, bool) {
	opts := r.Options()
	if !opts.EnableBridge || opts.BridgePath == "" {
		return Result{}, false
	}
	ctx, ok := BridgeContextForWindow(opts.BridgePath, w)
	if !ok {
		return Result{}, false
	}
	return Result{
		Path:      ctx.File,
		StartLine: ctx.StartLine,
		EndLine:   ctx.EndLine,
		Origin:    "vscode-bridge",
		Mode:      "editor",
		Matches:   1,
		Note:      "exact selection range reported by the editor extension",
	}, true
}

// Resolve locates the request's selection inside a source file.
func (r *Resolver) Resolve(req Request) (Result, error) {
	if strings.TrimSpace(req.Selection) == "" {
		return Result{}, errors.New("no text was selected")
	}
	opts := r.Options()
	deadline := time.Now().Add(opts.MaxSearch)
	candidates := r.collectCandidates(req, deadline)
	if len(candidates) == 0 {
		return Result{}, r.noCandidatesError(req)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Path < candidates[j].Path
		}
		return candidates[i].Score > candidates[j].Score
	})

	markdown := req.Window.Typora() || req.Window.Obsidian()
	titleAnchor := TitleAnchor(req.Window.Title)
	type scored struct {
		result Result
		score  int
	}
	var best *scored
	var fallback *Candidate
	fallbackAnchor := titleAnchor
	var foundFiles int
	var lastReadErr error

	for _, candidate := range candidates {
		if time.Now().After(deadline) {
			break
		}
		info, err := os.Stat(candidate.Path)
		if err != nil || info.IsDir() {
			continue
		}
		content, err := r.readTextFile(candidate.Path)
		if err != nil {
			lastReadErr = fmt.Errorf("%s: %w", candidate.Path, err)
			continue
		}
		foundFiles++
		anchor := candidate.Anchor
		if anchor == 0 {
			anchor = titleAnchor
		}
		if anchor > 0 && (fallback == nil || candidate.Score > fallback.Score) {
			copyCandidate := candidate
			copyCandidate.Anchor = anchor
			fallback = &copyCandidate
			fallbackAnchor = anchor
		}
		useMarkdown := markdown || candidate.Origin == "typora-recent" || candidate.Origin == "obsidian-vault"
		match, _, count, err := textmatch.Find(content, req.Selection, textmatch.Options{Markdown: useMarkdown}, anchor)
		if err != nil {
			continue
		}
		score := candidate.Score
		switch match.Mode {
		case textmatch.ModeExact:
			score += 20
		case textmatch.ModeWhitespace:
			score += 8
		case textmatch.ModeMarkdown:
			score += 3
		}
		if count == 1 {
			score += 3
		}
		if anchor > 0 && anchor >= match.StartLine && anchor <= match.EndLine {
			score += 25
		}
		note := ""
		if count > 1 && anchor == 0 {
			note = fmt.Sprintf("selected text occurs %d times; picked the first occurrence", count)
		}
		result := Result{
			Path:       candidate.Path,
			StartLine:  match.StartLine,
			EndLine:    match.EndLine,
			Origin:     candidate.Origin,
			Mode:       string(match.Mode),
			Matches:    count,
			Candidates: len(candidates),
			Note:       note,
		}
		if best == nil || score > best.score {
			best = &scored{result: result, score: score}
		}
	}

	if best != nil {
		return best.result, nil
	}

	// If the file exists but the selected text does not match (an unsaved
	// buffer, for example), a known anchor still lets us infer the end line.
	if fallback != nil && foundFiles > 0 && fallbackAnchor > 0 {
		content, err := r.readTextFile(fallback.Path)
		if err == nil {
			lineCount := strings.Count(content, "\n") + 1
			start := fallbackAnchor
			end := textmatch.EstimatedEndLine(start, req.Selection)
			if end > lineCount {
				end = lineCount
			}
			if start <= lineCount {
				return Result{
					Path:       fallback.Path,
					StartLine:  start,
					EndLine:    end,
					Origin:     fallback.Origin,
					Mode:       "anchor-estimate",
					Candidates: len(candidates),
					Note:       "selected text was not found on disk (unsaved changes?); range estimated from the editor anchor",
				}, nil
			}
		}
	}

	if foundFiles > 0 {
		msg := "found the file but not the selected text on disk"
		if lastReadErr != nil {
			msg += " (" + lastReadErr.Error() + ")"
		}
		if markdown {
			msg += "; rendered Markdown can differ from the source, especially for formatted text"
		} else {
			msg += "; save the file first, or install the VS Code bridge for exact ranges"
		}
		return Result{}, errors.New(msg)
	}
	if lastReadErr != nil {
		return Result{}, fmt.Errorf("could not read candidate files: %w", lastReadErr)
	}
	return Result{}, errors.New("candidate files disappeared or are unreadable")
}

func (r *Resolver) noCandidatesError(req Request) error {
	opts := r.Options()
	name := FileNameFromTitle(req.Window.Title)
	if name == "" {
		name = "the current document"
	}
	msg := fmt.Sprintf("could not determine the file path for %s", name)
	if len(opts.SearchRoots) == 0 {
		msg += "; add your projects folder under Settings > Path discovery"
	} else {
		msg += "; add the project folder to Settings > Path discovery"
	}
	if req.Window.VSCodeFamily() {
		msg += ", install the VS Code bridge, or set window.title to include ${activeEditorLong}"
	}
	if req.Window.JetBrains() && opts.EnableJetBrainsReference {
		msg += "; JetBrains Copy Reference did not return a resolvable path"
	}
	return errors.New(msg)
}

func (r *Resolver) readTextFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Size() > 24<<20 {
		return "", fmt.Errorf("file is larger than 24 MB")
	}
	r.mu.Lock()
	if cached, ok := r.cache[path]; ok && cached.size == info.Size() && cached.modTime.Equal(info.ModTime()) {
		text := cached.text
		r.mu.Unlock()
		return text, nil
	}
	r.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text, err := decodeText(data)
	if err != nil {
		return "", err
	}
	r.mu.Lock()
	if len(r.cache) >= 8 {
		for key := range r.cache {
			delete(r.cache, key)
			break
		}
	}
	r.cache[path] = cachedFile{modTime: info.ModTime(), size: info.Size(), text: text}
	r.mu.Unlock()
	return text, nil
}

func decodeText(data []byte) (string, error) {
	switch {
	case len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF:
		data = data[3:]
	case len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE:
		return decodeUTF16(data[2:], true)
	case len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF:
		return decodeUTF16(data[2:], false)
	}
	if utf8.Valid(data) {
		if looksBinary(data) {
			return "", fmt.Errorf("file appears to be binary")
		}
		return string(data), nil
	}
	decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(data)
	if err == nil && len(decoded) > 0 {
		if looksBinary(decoded) {
			return "", fmt.Errorf("file appears to be binary")
		}
		return string(decoded), nil
	}
	return "", fmt.Errorf("unsupported or invalid text encoding")
}

func decodeUTF16(data []byte, littleEndian bool) (string, error) {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	units := make([]uint16, len(data)/2)
	for i := 0; i < len(units); i++ {
		if littleEndian {
			units[i] = uint16(data[2*i]) | uint16(data[2*i+1])<<8
		} else {
			units[i] = uint16(data[2*i])<<8 | uint16(data[2*i+1])
		}
	}
	return string(utf16.Decode(units)), nil
}

func looksBinary(data []byte) bool {
	limit := len(data)
	if limit > 8192 {
		limit = 8192
	}
	for _, b := range data[:limit] {
		if b == 0 {
			return true
		}
	}
	return false
}
