//go:build windows

package settings

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"copytextpos/internal/config"
	"copytextpos/internal/locate"
)

func TestServerAuthAndState(t *testing.T) {
	stateFn := func() State {
		return State{Version: "test", Config: config.Default(), HotkeyActive: true, Bridge: locate.BridgeStatus{}}
	}
	deps := Deps{
		State: stateFn,
		Save: func(in config.Config) (State, []string, error) {
			return stateFn(), nil, nil
		},
		Action: func(string) (string, error) { return "ok", nil },
	}
	server, err := Start(deps)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	resp, err := http.Get(server.URL() + "api/state")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("state status=%d", resp.StatusCode)
	}
	var state State
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		t.Fatal(err)
	}
	if state.Version != "test" || !state.HotkeyActive {
		t.Fatalf("unexpected state: %+v", state)
	}

	u, _ := url.Parse(server.URL())
	noToken := strings.TrimSuffix(server.URL(), u.Path) + "/"
	bad, err := http.Get(noToken)
	if err != nil {
		t.Fatal(err)
	}
	defer bad.Body.Close()
	if bad.StatusCode != http.StatusNotFound {
		t.Fatalf("untokenized status=%d", bad.StatusCode)
	}
}

func TestConfigPostRejectsBadOrigin(t *testing.T) {
	deps := Deps{
		State: func() State { return State{} },
		Save:  func(in config.Config) (State, []string, error) { return State{}, nil, nil },
		Action: func(string) (string, error) {
			return "", nil
		},
	}
	server, err := Start(deps)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	body := `{"hotkey":"Ctrl+Shift+C","format":"{path}:{range}"}`
	req, _ := http.NewRequest(http.MethodPost, server.URL()+"api/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d want 403", resp.StatusCode)
	}
}
