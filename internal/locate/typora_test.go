//go:build windows

package locate

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"copytextpos/internal/winapi"
)

const typoraJSON = `{"recentFolder":[],"recentDocument":[{"name":"editor-support.md","path":"D:\\projects\\copy-text-pos\\docs\\editor-support.md","data":1792841722474},{"name":"发布指南.md","path":"D:\\projects\\android\\docs\\发布指南.md","data":1792822125413}]}`

func TestParseTyporaHistoryHex(t *testing.T) {
	encoded := []byte(hex.EncodeToString([]byte(typoraJSON)))
	files, err := parseTyporaHistory(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files", len(files))
	}
	if files[0].Name != "editor-support.md" || files[0].Path != `D:\projects\copy-text-pos\docs\editor-support.md` {
		t.Fatalf("unexpected first file: %+v", files[0])
	}
}

func TestParseTyporaHistoryRawAndBOM(t *testing.T) {
	for _, data := range [][]byte{
		[]byte(typoraJSON),
		append([]byte{0xEF, 0xBB, 0xBF}, []byte(typoraJSON)...),
	} {
		files, err := parseTyporaHistory(data)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) == 0 || files[0].Name != "editor-support.md" {
			t.Fatalf("unexpected files: %+v", files)
		}
	}
}

func TestParseTyporaHistoryInvalid(t *testing.T) {
	if _, err := parseTyporaHistory([]byte("not json or hex")); err == nil {
		t.Fatal("expected error")
	}
}

func TestTyporaRecentIntegration(t *testing.T) {
	if os.Getenv("CTP_INTEGRATION") == "" {
		t.Skip("set CTP_INTEGRATION=1 to read the local Typora history")
	}
	var target string
	for _, file := range TyporaRecent() {
		if strings.EqualFold(file.Name, "editor-support.md") {
			target = file.Path
			break
		}
	}
	if target == "" {
		t.Skip("editor-support.md is not in Typora's recent list")
	}
	if _, err := os.Stat(target); err != nil {
		t.Skip(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) < 12 {
		t.Skip("document is too short")
	}
	selection := strings.Join(lines[9:11], "\n") // source lines 10-11
	r := NewResolver(Options{EnableRecents: true, MaxSearch: 3 * time.Second})
	result, err := r.Resolve(Request{
		Window: Window{
			Title:   filepath.Base(target) + " - Typora",
			Process: winapi.Process{Name: "Typora.exe"},
		},
		Selection: selection,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(result.Path, target) || result.StartLine != 10 || result.EndLine != 11 {
		t.Fatalf("result=%+v", result)
	}
}
