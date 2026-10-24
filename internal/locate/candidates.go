//go:build windows

package locate

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type candidateSet struct {
	items []Candidate
	seen  map[string]int
}

func newCandidateSet() *candidateSet {
	return &candidateSet{seen: map[string]int{}}
}

func (s *candidateSet) add(path, origin string, score, anchor int) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || !filepath.IsAbs(path) {
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return
	}
	key := strings.ToLower(path)
	if idx, ok := s.seen[key]; ok {
		if score > s.items[idx].Score {
			s.items[idx].Score = score
			s.items[idx].Origin = origin
		}
		if s.items[idx].Anchor == 0 && anchor > 0 {
			s.items[idx].Anchor = anchor
		}
		return
	}
	s.seen[key] = len(s.items)
	s.items = append(s.items, Candidate{Path: path, Origin: origin, Score: score, Anchor: anchor})
}

func (r *Resolver) collectCandidates(req Request, deadline time.Time) []Candidate {
	opts := r.Options()
	w := req.Window
	set := newCandidateSet()

	for _, hint := range AbsolutePathHints(w.Title) {
		set.add(hint.Path, hint.Origin, 120, hint.AnchorLine)
	}
	// A complete path in the title is authoritative. Do not spend the hotkey
	// budget scanning workspaces or search roots in that case.
	if maxCandidateScore(set) >= 120 {
		return set.items
	}

	if opts.EnableJetBrainsReference && req.ReferenceText != "" {
		var roots []string
		if opts.EnableRecents {
			roots = JetBrainsProjectRootsForTitle(w.Title, JetBrainsProjectRoots())
		}
		if hint, ok := ParseReference(req.ReferenceText, roots, opts.SearchRoots); ok {
			set.add(hint.Path, hint.Origin, 130, hint.AnchorLine)
		}
	}
	if maxCandidateScore(set) >= 130 {
		return set.items
	}

	if opts.EnableRecents && w.Typora() {
		r.addTyporaCandidates(set, w)
	}
	if opts.EnableRecents && w.Obsidian() {
		r.addObsidianCandidates(set, w, deadline)
	}

	if w.VSCodeFamily() {
		r.addVSCodeCandidates(set, w, deadline)
	}

	if maxCandidateScore(set) < 90 {
		r.addGenericCandidates(set, w, deadline)
	}
	return set.items
}

func maxCandidateScore(set *candidateSet) int {
	best := 0
	for _, item := range set.items {
		if item.Score > best {
			best = item.Score
		}
	}
	return best
}

func (r *Resolver) addTyporaCandidates(set *candidateSet, w Window) {
	name := fileNameFromTitleFor(w.Title)
	if name == "" {
		return
	}
	for _, recent := range TyporaRecent() {
		if strings.EqualFold(recent.Name, name) || strings.EqualFold(filepath.Base(recent.Path), name) {
			set.add(recent.Path, "typora-recent", 110, TitleAnchor(w.Title))
		}
	}
}

func (r *Resolver) addObsidianCandidates(set *candidateSet, w Window, deadline time.Time) {
	note, vaultName := obsidianNoteAndVault(w.Title)
	if note == "" {
		return
	}
	if filepath.Ext(note) == "" {
		note += ".md"
	}
	vaults := ObsidianVaults()
	for _, vault := range vaults {
		if time.Now().After(deadline) {
			return
		}
		if vaultName != "" && !strings.EqualFold(filepath.Base(vault.Path), vaultName) {
			continue
		}
		if info, err := os.Stat(vault.Path); err != nil || !info.IsDir() {
			continue
		}
		// Titles often show an alias instead of the on-disk filename. The open
		// workspace state gives the real path without scanning the whole vault.
		for _, path := range ObsidianWorkspaceFiles(vault.Path) {
			set.add(path, "obsidian-workspace", 115, TitleAnchor(w.Title))
		}
		result := findFilesNamed(vault.Path, note, deadline, 8)
		for _, path := range result.Paths {
			set.add(path, "obsidian-vault", 100, TitleAnchor(w.Title))
		}
	}
}

func obsidianNoteAndVault(title string) (note, vault string) {
	stripped := CleanTitlePart(StripAppSuffix(title))
	segments := TitleSegments(stripped)
	if len(segments) == 0 {
		return "", ""
	}
	if len(segments) == 1 {
		return CleanTitlePart(segments[0]), ""
	}
	note = CleanTitlePart(strings.Join(segments[:len(segments)-1], " - "))
	vault = CleanTitlePart(segments[len(segments)-1])
	return note, vault
}

func (r *Resolver) addVSCodeCandidates(set *candidateSet, w Window, deadline time.Time) {
	parsed := ParseVSCodeTitle(w.Title)
	if parsed.FilePath != "" {
		set.add(parsed.FilePath, "window-title", 120, parsed.Anchor)
	}
	name := cleanDocumentName(parsed.FilePart)
	if name == "" {
		return
	}
	workspace := CleanTitlePart(parsed.Workspace)
	entries := VSCodeWorkspaces()
	if len(entries) == 0 {
		return
	}

	// First, exact workspace-name matches. This avoids scanning unrelated
	// project trees and is extremely reliable for the normal editor title.
	var matching []WorkspaceEntry
	for _, entry := range entries {
		if workspace != "" && strings.EqualFold(entry.Label, workspace) {
			matching = append(matching, entry)
		}
	}
	searched := map[string]bool{}
	found := 0
	for i, entry := range matching {
		if i >= 4 || time.Now().After(deadline) {
			break
		}
		key := strings.ToLower(entry.Root)
		if searched[key] {
			continue
		}
		searched[key] = true
		result := findFilesNamed(entry.Root, name, deadline, 8)
		for _, path := range result.Paths {
			set.add(path, "vscode-workspace", 90, parsed.Anchor)
			found++
		}
	}
	if found > 0 {
		return
	}

	// Single-file windows, or workspaces missing from workspaceStorage: look
	// only at the most recent roots to keep the hotkey latency bounded.
	limit := 6
	if workspace != "" {
		limit = 4
	}
	for i, entry := range entries {
		if i >= limit || time.Now().After(deadline) {
			break
		}
		key := strings.ToLower(entry.Root)
		if searched[key] {
			continue
		}
		searched[key] = true
		result := findFilesNamed(entry.Root, name, deadline, 4)
		for _, path := range result.Paths {
			set.add(path, "vscode-recent", 60, parsed.Anchor)
			found++
		}
		if found >= 4 {
			return
		}
	}
}

func (r *Resolver) addGenericCandidates(set *candidateSet, w Window, deadline time.Time) {
	opts := r.Options()
	if len(opts.SearchRoots) == 0 {
		return
	}
	var name, workspace string
	if w.Obsidian() {
		note, vault := obsidianNoteAndVault(w.Title)
		name = note
		workspace = vault
	} else {
		name = fileNameFromTitleFor(w.Title)
		workspace = genericWorkspace(w.Title)
	}
	if name == "" {
		return
	}
	if filepath.Ext(name) == "" {
		if w.Typora() || w.Obsidian() {
			name += ".md"
		} else {
			return
		}
	}
	if isUntitled(name) {
		return
	}
	searched := map[string]bool{}

	if workspace != "" {
		for _, root := range opts.SearchRoots {
			if time.Now().After(deadline) {
				return
			}
			for _, dir := range findDirsNamed(root, workspace, 4, deadline, 3) {
				key := strings.ToLower(dir)
				if searched[key] {
					continue
				}
				searched[key] = true
				result := findFilesNamed(dir, name, deadline, 6)
				for _, path := range result.Paths {
					set.add(path, "workspace-dir", 70, TitleAnchor(w.Title))
				}
			}
		}
	}

	for _, root := range opts.SearchRoots {
		if time.Now().After(deadline) {
			return
		}
		result := findFilesNamed(root, name, deadline, 8)
		for _, path := range result.Paths {
			set.add(path, "search-root", 40, TitleAnchor(w.Title))
		}
	}
}

func fileNameFromTitleFor(title string) string {
	name := CleanTitlePart(FileNameFromTitle(title))
	name = cleanDocumentName(name)
	return name
}

func cleanDocumentName(name string) string {
	name = CleanTitlePart(name)
	name = strings.TrimSpace(name)
	// "file.ts:42" and "file.ts (42)" style titles.
	if idx := strings.LastIndex(name, ":"); idx > 1 {
		suffix := name[idx+1:]
		if isDigits(suffix) {
			name = name[:idx]
		}
	}
	if m := lineParenRE.FindStringSubmatch(name); m != nil {
		name = strings.TrimSpace(name[:len(name)-len(m[0])])
	}
	name = strings.TrimSpace(name)
	if isUntitled(name) {
		return ""
	}
	return name
}

func isUntitled(name string) bool {
	lower := strings.ToLower(name)
	return lower == "" || strings.HasPrefix(lower, "untitled") || strings.HasPrefix(lower, "无标题")
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func genericWorkspace(title string) string {
	segments := TitleSegments(CleanTitlePart(StripAppSuffix(title)))
	if len(segments) < 2 {
		return ""
	}
	return CleanTitlePart(segments[len(segments)-1])
}
