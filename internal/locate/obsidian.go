//go:build windows

package locate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// obsidianWorkspaceNode is the subset of .obsidian/workspace.json needed to
// find files open in the main editor area.
type obsidianWorkspaceNode struct {
	Type       string                  `json:"type"`
	CurrentTab int                     `json:"currentTab"`
	Children   []obsidianWorkspaceNode `json:"children"`
	State      struct {
		Type  string `json:"type"`
		State struct {
			File string `json:"file"`
		} `json:"state"`
	} `json:"state"`
}

// ObsidianWorkspaceFiles returns the Markdown files currently open in an
// Obsidian vault's main editor area. Obsidian's window title may show a note's
// frontmatter alias rather than its filename, so workspace.json is the reliable
// mapping from the visible title to a path.
func ObsidianWorkspaceFiles(vaultRoot string) []string {
	data, err := os.ReadFile(filepath.Join(vaultRoot, ".obsidian", "workspace.json"))
	if err != nil {
		return nil
	}
	return parseObsidianWorkspace(data, vaultRoot)
}

func parseObsidianWorkspace(data []byte, vaultRoot string) []string {
	var doc struct {
		Main obsidianWorkspaceNode `json:"main"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return nil
	}
	var paths []string
	seen := map[string]bool{}
	var walk func(node obsidianWorkspaceNode)
	walk = func(node obsidianWorkspaceNode) {
		if node.Type == "tabs" && len(node.Children) > 0 {
			idx := node.CurrentTab
			if idx >= 0 && idx < len(node.Children) {
				walk(node.Children[idx])
			}
			for i, child := range node.Children {
				if i != idx {
					walk(child)
				}
			}
			return
		}
		if node.Type == "leaf" {
			file := strings.TrimSpace(node.State.State.File)
			if file != "" {
				rel := filepath.Clean(filepath.FromSlash(file))
				if rel != "." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..") {
					full := filepath.Join(vaultRoot, rel)
					key := strings.ToLower(full)
					if !seen[key] {
						seen[key] = true
						paths = append(paths, full)
					}
				}
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(doc.Main)
	return paths
}
