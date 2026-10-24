//go:build windows

package locate

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var prunedDirs = map[string]bool{
	".git": true, ".svn": true, ".hg": true,
	"node_modules": true, "bower_components": true, "vendor": true,
	"target": true, ".gradle": true, ".idea": true,
	"dist": true, "out": true, "build": true, ".next": true, ".nuxt": true,
	"__pycache__": true, ".venv": true, "venv": true, ".cache": true,
	"bin": true, "obj": true, "pods": true, ".cursor": true, ".trash": true,
}

// searchResult records why a scan stopped.
type searchResult struct {
	Paths    []string
	TimedOut bool
	Scanned  int
}

// findFilesNamed walks root looking for a case-insensitive basename match.
func findFilesNamed(root, name string, deadline time.Time, maxResults int) searchResult {
	var result searchResult
	if root == "" || name == "" {
		return result
	}
	info, err := os.Stat(root)
	if err != nil {
		return result
	}
	if !info.IsDir() {
		if strings.EqualFold(filepath.Base(root), name) {
			result.Paths = append(result.Paths, root)
		}
		return result
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		result.Scanned++
		if result.Scanned%128 == 0 {
			if time.Now().After(deadline) {
				result.TimedOut = true
				return fs.SkipAll
			}
		}
		if walkErr != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != root && prunedDirs[strings.ToLower(d.Name())] {
				return fs.SkipDir
			}
			return nil
		}
		if strings.EqualFold(d.Name(), name) {
			result.Paths = append(result.Paths, path)
			if len(result.Paths) >= maxResults {
				return fs.SkipAll
			}
		}
		return nil
	})
	return result
}

// findDirsNamed walks root to a bounded depth looking for directories.
func findDirsNamed(root, name string, maxDepth int, deadline time.Time, maxResults int) []string {
	if root == "" || name == "" {
		return nil
	}
	root = filepath.Clean(root)
	rootDepth := strings.Count(root, string(os.PathSeparator))
	var found []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if time.Now().After(deadline) {
			return fs.SkipAll
		}
		if walkErr != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && prunedDirs[strings.ToLower(d.Name())] {
			return fs.SkipDir
		}
		depth := strings.Count(path, string(os.PathSeparator)) - rootDepth
		if depth > maxDepth {
			return fs.SkipDir
		}
		if strings.EqualFold(d.Name(), name) {
			found = append(found, path)
			if len(found) >= maxResults {
				return fs.SkipAll
			}
		}
		return nil
	})
	return found
}
