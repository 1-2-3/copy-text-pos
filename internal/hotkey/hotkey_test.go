package hotkey

import "testing"

func TestRoundTripDefault(t *testing.T) {
	spec, err := Parse("ctrl + shift + c")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := spec.String(), "Ctrl+Shift+C"; got != want {
		t.Fatalf("String()=%q want %q", got, want)
	}
	if spec.VK != 'C' || spec.Mods != ModControl|ModShift {
		t.Fatalf("unexpected spec: %+v", spec)
	}
}

func TestParseVariants(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Alt+F12", "Alt+F12"},
		{"Win+Shift+S", "Shift+Win+S"},
		{"Control+Alt+Delete", "Ctrl+Alt+Delete"},
		{"ctrl+shift+space", "Ctrl+Shift+Space"},
		{"cmdorctrl+x", ""},
	}
	for _, tc := range cases {
		got, err := Parse(tc.in)
		if tc.want == "" {
			if err == nil {
				t.Fatalf("Parse(%q) unexpectedly succeeded", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.in, err)
		}
		if got.String() != tc.want {
			t.Fatalf("Parse(%q).String()=%q want %q", tc.in, got.String(), tc.want)
		}
	}
}

func TestParseRejects(t *testing.T) {
	for _, in := range []string{"", "C", "Ctrl", "Ctrl+Shift", "Ctrl+Shift+C+D", "Ctrl+Shift+NotAKey"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) should fail", in)
		}
	}
}
