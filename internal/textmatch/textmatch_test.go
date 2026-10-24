package textmatch

import (
	"strings"
	"testing"
)

func TestExactMultiLineTrailingNewline(t *testing.T) {
	content := "one\nline 13\nline 14\nline 15\nline 16\nline 17\nline 18\nline 19\n"
	selection := "line 13\nline 14\nline 15\nline 16\nline 17\nline 18\n"
	m, idx, count, err := Find(content, selection, Options{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.StartLine != 2 || m.EndLine != 7 {
		t.Fatalf("range=%d-%d want 2-7", m.StartLine, m.EndLine)
	}
	if idx != 0 || count != 1 || m.Mode != ModeExact {
		t.Fatalf("idx=%d count=%d mode=%s", idx, count, m.Mode)
	}
}

func TestCRLFAndPartialLine(t *testing.T) {
	content := "alpha\r\n  return useAsync(task)\r\nomega\r\n"
	m, err := FindSingle(content, "return useAsync(task)", Options{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.StartLine != 2 || m.EndLine != 2 {
		t.Fatalf("range=%d-%d want 2-2", m.StartLine, m.EndLine)
	}
}

func TestDuplicateUsesAnchor(t *testing.T) {
	content := "x\nreturn 1\ny\nreturn 2\nz\nreturn 3\n"
	m, _, count, err := Find(content, "return", Options{}, 4)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("count=%d want 3", count)
	}
	if m.StartLine != 4 || m.EndLine != 4 {
		t.Fatalf("range=%d-%d want 4-4", m.StartLine, m.EndLine)
	}
}

func TestWhitespaceFallback(t *testing.T) {
	content := "func main() {\n\tif x {\n\t\tfmt.Println(x)\n\t}\n}\n"
	selection := "    if x {\n        fmt.Println(x)\n    }"
	m, err := FindSingle(content, selection, Options{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.StartLine != 2 || m.EndLine != 4 || m.Mode != ModeWhitespace {
		t.Fatalf("unexpected match: %+v", m)
	}
}

func TestMarkdownFallback(t *testing.T) {
	content := "# Title\n\nThis is **bold** and a [link](https://example.com).\n\n- item one\n"
	selection := "This is bold and a link."
	m, err := FindSingle(content, selection, Options{Markdown: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.StartLine != 3 || m.EndLine != 3 || m.Mode != ModeMarkdown {
		t.Fatalf("unexpected match: %+v", m)
	}
}

func TestMarkdownTableCopy(t *testing.T) {
	content := "# 支持\n\n" +
		"| 级别 | 路径 | 行号范围 |\n" +
		"|---|---|---|\n" +
		"| **精确** | 来自编辑器桥接或标题中的完整路径 | 来自桥接，或无歧义的标题元数据 |\n" +
		"| **良好** | 来自编辑器自有元数据 | 通过选中文本搜索推断 |\n\n" +
		"后续段落。\n"

	// This is the shape Typora puts on the clipboard for a rendered table:
	// padded columns, preserved bold markers, and a padded separator row.
	selection := "| 级别       | 路径                             | 行号范围                       |\n" +
		"| ---------- | -------------------------------- | ------------------------------ |\n" +
		"| **精确**   | 来自编辑器桥接或标题中的完整路径 | 来自桥接，或无歧义的标题元数据 |\n" +
		"| **良好**   | 来自编辑器自有元数据             | 通过选中文本搜索推断           |"

	m, err := FindSingle(content, selection, Options{Markdown: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.StartLine != 3 || m.EndLine != 6 || m.Mode != ModeMarkdown {
		t.Fatalf("unexpected match: %+v", m)
	}

	// Some editors omit the separator row from copied text. The range should
	// still cover the entire source table including the separator.
	withoutSeparator := "| 级别 | 路径 | 行号范围 |\n" +
		"| **精确** | 来自编辑器桥接或标题中的完整路径 | 来自桥接，或无歧义的标题元数据 |\n" +
		"| **良好** | 来自编辑器自有元数据 | 通过选中文本搜索推断 |"
	m, err = FindSingle(content, withoutSeparator, Options{Markdown: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.StartLine != 3 || m.EndLine != 6 {
		t.Fatalf("unexpected match without separator: %+v", m)
	}

	// Selecting only data rows starts at the first row and ends at the last.
	rowsOnly := "| **精确**   | 来自编辑器桥接或标题中的完整路径 | 来自桥接，或无歧义的标题元数据 |\n" +
		"| **良好**   | 来自编辑器自有元数据             | 通过选中文本搜索推断           |"
	m, err = FindSingle(content, rowsOnly, Options{Markdown: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.StartLine != 5 || m.EndLine != 6 {
		t.Fatalf("unexpected rows-only match: %+v", m)
	}
}

func TestNotFound(t *testing.T) {
	if _, err := FindAll("abc\ndef\n", "not present", Options{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestEmpty(t *testing.T) {
	if _, err := FindAll("abc", "  \r\n ", Options{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestEstimatedEndLine(t *testing.T) {
	sel := "a\nb\nc\r\n"
	if got := EstimatedEndLine(10, sel); got != 12 {
		t.Fatalf("got %d want 12", got)
	}
}

func TestLargeFileLineNumbers(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= 10000; i++ {
		b.WriteString("line ")
		b.WriteString(strings.Repeat("x", i%7))
		b.WriteByte('\n')
	}
	b.WriteString("needle here\n")
	m, err := FindSingle(b.String(), "needle here", Options{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.StartLine != 10001 {
		t.Fatalf("start=%d want 10001", m.StartLine)
	}
}

func FindSingle(content, selection string, opts Options, anchor int) (Match, error) {
	m, _, _, err := Find(content, selection, opts, anchor)
	return m, err
}
