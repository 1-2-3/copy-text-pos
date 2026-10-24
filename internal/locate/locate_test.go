//go:build windows

package locate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"copytextpos/internal/winapi"
)

func TestObsidianVersionedTitle(t *testing.T) {
	title := "血透医保控费采集终端方案和硬件需求 - obsi-doc - Obsidian 1.14.5"
	if got := StripAppSuffix(title); got != "血透医保控费采集终端方案和硬件需求 - obsi-doc" {
		t.Fatalf("StripAppSuffix=%q", got)
	}
	note, vault := obsidianNoteAndVault(title)
	if note != "血透医保控费采集终端方案和硬件需求" || vault != "obsi-doc" {
		t.Fatalf("note=%q vault=%q", note, vault)
	}
}

func TestObsidianWorkspaceFiles(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{
		"main": {
			"type": "tabs",
			"currentTab": 1,
			"children": [
				{"type":"leaf","state":{"type":"markdown","state":{"file":"notes/old.md"}}},
				{"type":"leaf","state":{"type":"markdown","state":{"file":"notes/current.md"}}},
				{"type":"leaf","state":{"type":"outline","state":{"file":"notes/current.md"}}}
			]
		}
	}`)
	paths := parseObsidianWorkspace(data, root)
	if len(paths) != 2 {
		t.Fatalf("paths=%#v", paths)
	}
	if paths[0] != filepath.Join(root, "notes", "current.md") {
		t.Fatalf("current tab should come first: %#v", paths)
	}
	if paths[1] != filepath.Join(root, "notes", "old.md") {
		t.Fatalf("unexpected paths: %#v", paths)
	}
}

func TestObsidianWorkspaceRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"main":{"type":"leaf","state":{"type":"markdown","state":{"file":"../secret.md"}}}}`)
	if paths := parseObsidianWorkspace(data, root); len(paths) != 0 {
		t.Fatalf("traversal path accepted: %#v", paths)
	}
}

func TestParseVSCodeTitle(t *testing.T) {
	cases := []struct {
		title     string
		file      string
		workspace string
	}{
		{"useAsync.ts - android - Cursor", "useAsync.ts", "android"},
		{"● useAsync.ts - android - Cursor", "useAsync.ts", "android"},
		{"my - file.ts - android - Visual Studio Code", "my - file.ts", "android"},
		{"notes.md - Cursor", "notes.md", ""},
	}
	for _, tc := range cases {
		got := ParseVSCodeTitle(tc.title)
		if got.FilePart != tc.file || got.Workspace != tc.workspace {
			t.Errorf("%q => file=%q workspace=%q; want %q/%q", tc.title, got.FilePart, got.Workspace, tc.file, tc.workspace)
		}
	}
}

func TestAbsolutePathHints(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my - file.ts")
	if err := os.WriteFile(path, []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	hints := AbsolutePathHints(path + " - project - Cursor")
	if len(hints) == 0 || !strings.EqualFold(hints[0].Path, path) {
		t.Fatalf("hints=%+v want %q", hints, path)
	}
	if hints[0].AnchorLine != 0 {
		t.Fatalf("unexpected anchor %d", hints[0].AnchorLine)
	}
}

func TestTitleAnchor(t *testing.T) {
	if got := TitleAnchor("file.ts:42 - NVIM"); got != 42 {
		t.Fatalf("got %d want 42", got)
	}
	if got := TitleAnchor("file.ts (17) - Editor"); got != 17 {
		t.Fatalf("got %d want 17", got)
	}
}

func TestFileURI(t *testing.T) {
	cases := map[string]string{
		"file:///d%3A/projects/android": `D:\projects\android`,
		"file:///c%3A/Users/user/a%20b": `C:\Users\user\a b`,
	}
	for in, want := range cases {
		got, err := fileURIToPath(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if !strings.EqualFold(got, want) {
			t.Errorf("%s => %q want %q", in, got, want)
		}
	}
}

func TestParseReference(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "src", "main", "Foo.java")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("class Foo {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	hint, ok := ParseReference(`src/main/Foo.java:12`, []string{dir}, nil)
	if !ok || hint.AnchorLine != 12 || !strings.EqualFold(hint.Path, target) {
		t.Fatalf("hint=%+v ok=%v", hint, ok)
	}
	if _, ok := ParseReference(`com.example.Foo:12`, []string{dir}, nil); ok {
		t.Fatal("class reference should not resolve to a file")
	}
}

func TestBridgeContext(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "useAsync.ts")
	if err := os.WriteFile(file, []byte("a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "bridge.json")
	ctx := BridgeContext{
		Version: 1, Timestamp: time.Now().UnixMilli(), AppName: "Cursor",
		Workspace: "android", File: file, StartLine: 13, EndLine: 18, HasSelection: true,
	}
	data, _ := json.Marshal(ctx)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	got, ok := BridgeContextForWindow(path, Window{
		Title:   "useAsync.ts - android - Cursor",
		Process: processForTest("Cursor.exe"),
	})
	if !ok || got.StartLine != 13 || got.EndLine != 18 {
		t.Fatalf("bridge result=%+v ok=%v", got, ok)
	}
}

func TestResolverIntegrationCursor(t *testing.T) {
	if os.Getenv("CTP_INTEGRATION") == "" {
		t.Skip("set CTP_INTEGRATION=1 to use local Cursor workspace metadata")
	}
	path := `D:\projects\android\src\composables\useAsync.ts`
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) < 18 {
		t.Skip("sample file is too short")
	}
	selection := strings.Join(lines[12:18], "\n")
	r := NewResolver(Options{
		EnableRecents: true, EnableBridge: false, EnableJetBrainsReference: true,
		MaxSearch: 5 * time.Second,
	})
	result, err := r.Resolve(Request{
		Window: Window{
			Title:   "● useAsync.ts - android - Cursor",
			Process: processForTest("Cursor.exe"),
		},
		Selection: selection,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(result.Path, path) || result.StartLine != 13 || result.EndLine != 18 {
		t.Fatalf("result=%+v", result)
	}
}

func processForTest(name string) winapi.Process {
	return winapi.Process{Name: name}
}
