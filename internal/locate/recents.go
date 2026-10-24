//go:build windows

package locate

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// WorkspaceEntry is a recent VS Code family workspace.
type WorkspaceEntry struct {
	Root    string
	Label   string
	App     string
	Updated time.Time
}

type recentCache struct {
	mu           sync.Mutex
	workspaces   []WorkspaceEntry
	workspacesAt time.Time
	typora       []RecentFile
	typoraAt     time.Time
	obsidian     []Vault
	obsidianAt   time.Time
	jetbrains    []string
	jetbrainsAt  time.Time
}

var recent recentCache

// RecentFile is a document from an editor's recent-file list.
type RecentFile struct {
	Name    string
	Path    string
	Updated time.Time
	Source  string
}

// Vault is an Obsidian vault.
type Vault struct {
	Path    string
	Open    bool
	Updated time.Time
}

func appData() string {
	if v := os.Getenv("APPDATA"); v != "" {
		return v
	}
	v, _ := os.UserConfigDir()
	return v
}

// VSCodeWorkspaces returns recent workspace roots from installed VS Code
// derivatives. Results are cached briefly because several small JSON files may
// be involved.
func VSCodeWorkspaces() []WorkspaceEntry {
	recent.mu.Lock()
	defer recent.mu.Unlock()
	if time.Since(recent.workspacesAt) < 15*time.Second {
		return append([]WorkspaceEntry(nil), recent.workspaces...)
	}
	base := appData()
	apps := []string{"Code", "Code - Insiders", "Cursor", "Windsurf", "VSCodium", "Trae", "Kiro", "Positron"}
	entries := make([]WorkspaceEntry, 0, 32)
	for _, app := range apps {
		storage := filepath.Join(base, app, "User", "workspaceStorage")
		dirs, err := os.ReadDir(storage)
		if err != nil {
			continue
		}
		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}
			path := filepath.Join(storage, dir.Name(), "workspace.json")
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var doc struct {
				Folder    string `json:"folder"`
				Workspace string `json:"workspace"`
			}
			if json.Unmarshal(data, &doc) != nil {
				continue
			}
			updated := time.Time{}
			if info, err := dir.Info(); err == nil {
				updated = info.ModTime()
			}
			if doc.Workspace != "" {
				wsPath, err := fileURIToPath(doc.Workspace)
				if err == nil {
					if strings.HasSuffix(strings.ToLower(wsPath), ".code-workspace") {
						for _, root := range workspaceFolders(wsPath) {
							entries = append(entries, WorkspaceEntry{Root: root, Label: filepath.Base(root), App: app, Updated: updated})
						}
					}
				}
			}
			if doc.Folder != "" {
				root, err := fileURIToPath(doc.Folder)
				if err == nil {
					if info, statErr := os.Stat(root); statErr == nil && info.IsDir() {
						entries = append(entries, WorkspaceEntry{Root: root, Label: filepath.Base(root), App: app, Updated: updated})
					}
				}
			}
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Updated.After(entries[j].Updated) })
	recent.workspaces = entries
	recent.workspacesAt = time.Now()
	return append([]WorkspaceEntry(nil), entries...)
}

func fileURIToPath(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(u.Scheme, "file") {
		return "", os.ErrInvalid
	}
	if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
		return `\\` + u.Host + filepath.FromSlash(u.Path), nil
	}
	p := filepath.FromSlash(u.Path)
	if len(p) >= 3 && p[0] == '\\' && isDriveLetter(p[1]) && p[2] == ':' {
		p = p[1:]
	}
	if len(p) >= 2 && isDriveLetter(p[0]) && p[1] == ':' {
		p = strings.ToUpper(p[:1]) + p[1:]
	}
	return filepath.Clean(p), nil
}

func workspaceFolders(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Folders []struct {
			Path string `json:"path"`
		} `json:"folders"`
	}
	if json.Unmarshal([]byte(stripJSONComments(string(data))), &doc) != nil {
		return nil
	}
	base := filepath.Dir(path)
	var roots []string
	for _, folder := range doc.Folders {
		root := folder.Path
		if root == "" {
			continue
		}
		if !filepath.IsAbs(root) {
			root = filepath.Join(base, root)
		}
		root = filepath.Clean(root)
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			roots = append(roots, root)
		}
	}
	return roots
}

var jsonCommentRE = regexp.MustCompile(`(?s)/\*.*?\*/|(?m)//.*$`)

func stripJSONComments(s string) string {
	return jsonCommentRE.ReplaceAllString(s, "")
}

// TyporaRecent returns Typora's recent document list.
//
// Depending on version, Typora writes history.data either as plain JSON or as
// a hex-encoded JSON blob (the common format on Windows). Both are accepted.
func TyporaRecent() []RecentFile {
	recent.mu.Lock()
	defer recent.mu.Unlock()
	if time.Since(recent.typoraAt) < 10*time.Second {
		return append([]RecentFile(nil), recent.typora...)
	}
	path := filepath.Join(appData(), "Typora", "history.data")
	data, err := os.ReadFile(path)
	if err != nil {
		recent.typora = nil
		recent.typoraAt = time.Now()
		return nil
	}
	files, err := parseTyporaHistory(data)
	if err != nil {
		recent.typora = nil
		recent.typoraAt = time.Now()
		return nil
	}
	recent.typora = files
	recent.typoraAt = time.Now()
	return append([]RecentFile(nil), files...)
}

func parseTyporaHistory(data []byte) ([]RecentFile, error) {
	jsonBytes, err := decodeTyporaHistory(data)
	if err != nil {
		return nil, err
	}
	var doc struct {
		RecentDocument []struct {
			Name string          `json:"name"`
			Path string          `json:"path"`
			Data json.RawMessage `json:"data"`
		} `json:"recentDocument"`
	}
	if err := json.Unmarshal(jsonBytes, &doc); err != nil {
		return nil, err
	}
	files := make([]RecentFile, 0, len(doc.RecentDocument))
	for _, item := range doc.RecentDocument {
		p := filepath.Clean(item.Path)
		if !filepath.IsAbs(p) {
			continue
		}
		updated := parseRecentTimestamp(item.Data)
		files = append(files, RecentFile{Name: item.Name, Path: p, Updated: updated, Source: "Typora recent"})
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].Updated.After(files[j].Updated) })
	return files, nil
}

// decodeTyporaHistory accepts raw JSON, a UTF-8 BOM, or hex-encoded JSON.
func decodeTyporaHistory(data []byte) ([]byte, error) {
	s := bytesTrimBOM(strings.TrimSpace(string(data)))
	if looksLikeJSON(s) {
		return []byte(s), nil
	}
	if len(s)%2 == 0 && len(s) > 0 && isHexString(s) {
		decoded, err := hex.DecodeString(s)
		if err != nil {
			return nil, err
		}
		decodedString := bytesTrimBOM(strings.TrimSpace(string(decoded)))
		if looksLikeJSON(decodedString) {
			return []byte(decodedString), nil
		}
	}
	return nil, fmt.Errorf("unrecognized Typora history format")
}

func bytesTrimBOM(s string) string {
	return strings.TrimPrefix(s, "\ufeff")
}

func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func isHexString(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func parseRecentTimestamp(raw json.RawMessage) time.Time {
	s := strings.Trim(string(raw), `"`)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	if n > 1e12 {
		return time.UnixMilli(n)
	}
	if n > 1e9 {
		return time.Unix(n, 0)
	}
	return time.Time{}
}

// ObsidianVaults returns locally registered vaults.
func ObsidianVaults() []Vault {
	recent.mu.Lock()
	defer recent.mu.Unlock()
	if time.Since(recent.obsidianAt) < 15*time.Second {
		return append([]Vault(nil), recent.obsidian...)
	}
	path := filepath.Join(appData(), "obsidian", "obsidian.json")
	data, err := os.ReadFile(path)
	if err != nil {
		recent.obsidian = nil
		recent.obsidianAt = time.Now()
		return nil
	}
	var doc struct {
		Vaults map[string]struct {
			Path string `json:"path"`
			TS   int64  `json:"ts"`
			Open bool   `json:"open"`
		} `json:"vaults"`
	}
	if json.Unmarshal(data, &doc) != nil {
		recent.obsidian = nil
		recent.obsidianAt = time.Now()
		return nil
	}
	vaults := make([]Vault, 0, len(doc.Vaults))
	for _, v := range doc.Vaults {
		p := filepath.Clean(v.Path)
		if !filepath.IsAbs(p) {
			continue
		}
		vaults = append(vaults, Vault{Path: p, Open: v.Open, Updated: time.UnixMilli(v.TS)})
	}
	sort.SliceStable(vaults, func(i, j int) bool { return vaults[i].Updated.After(vaults[j].Updated) })
	recent.obsidian = vaults
	recent.obsidianAt = time.Now()
	return append([]Vault(nil), vaults...)
}

// JetBrainsProjectRoots returns project roots from recentProjects.xml in both
// JetBrains and Android Studio configuration directories.
func JetBrainsProjectRoots() []string {
	recent.mu.Lock()
	defer recent.mu.Unlock()
	if time.Since(recent.jetbrainsAt) < 20*time.Second {
		return append([]string(nil), recent.jetbrains...)
	}
	base := appData()
	patterns := []string{
		filepath.Join(base, "JetBrains", "*", "options", "recentProjects.xml"),
		filepath.Join(base, "Google", "*", "options", "recentProjects.xml"),
	}
	var files []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		files = append(files, matches...)
	}
	sort.SliceStable(files, func(i, j int) bool {
		a, _ := os.Stat(files[i])
		b, _ := os.Stat(files[j])
		return a != nil && b != nil && a.ModTime().After(b.ModTime())
	})

	entryRE := regexp.MustCompile(`<entry key="([^"]+)"`)
	lastRE := regexp.MustCompile(`<option name="lastOpenedProject" value="([^"]+)"`)
	seen := map[string]bool{}
	var roots []string
	add := func(raw string) {
		raw = strings.ReplaceAll(raw, "&amp;", "&")
		raw = strings.ReplaceAll(raw, "$USER_HOME$", os.Getenv("USERPROFILE"))
		if strings.Contains(raw, "$") {
			return
		}
		p := filepath.Clean(filepath.FromSlash(raw))
		if !filepath.IsAbs(p) {
			return
		}
		key := strings.ToLower(p)
		if seen[key] {
			return
		}
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			seen[key] = true
			roots = append(roots, p)
		}
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		for _, m := range lastRE.FindAllStringSubmatch(string(data), -1) {
			add(m[1])
		}
		for _, m := range entryRE.FindAllStringSubmatch(string(data), -1) {
			add(m[1])
		}
	}
	recent.jetbrains = roots
	recent.jetbrainsAt = time.Now()
	return append([]string(nil), roots...)
}
