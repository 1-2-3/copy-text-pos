package render

import "testing"

func TestRenderDefault(t *testing.T) {
	got, err := Render(DefaultTemplate, Location{
		Path:  `D:\projects\android\src\composables\useAsync.ts`,
		Start: 13,
		End:   18,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `D:\projects\android\src\composables\useAsync.ts:13-18`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRenderSingleLineCollapsesRange(t *testing.T) {
	got, err := Render("{path}:{range}", Location{Path: `D:\a b\x.ts`, Start: 7, End: 7})
	if err != nil {
		t.Fatal(err)
	}
	if want := `D:\a b\x.ts:7`; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRenderPlaceholders(t *testing.T) {
	got, err := Render("{name}|{ext}|{dir}|{line}|{gh_range}", Location{
		Path:  `D:\src\main.go`,
		Start: 3,
		End:   9,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `main.go|go|D:\src|3|L3-L9`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidate(t *testing.T) {
	for _, bad := range []string{"", "no placeholders", "{path}:{wat}"} {
		if err := Validate(bad); err == nil {
			t.Errorf("Validate(%q) should fail", bad)
		}
	}
}
