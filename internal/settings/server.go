//go:build windows

// Package settings serves the local, token-protected configuration page.
package settings

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"copytextpos/internal/config"
	"copytextpos/internal/locate"
)

//go:embed assets/*
var embedded embed.FS

// LastCapture is shown on the settings page. It never contains selected text.
type LastCapture struct {
	Time       time.Time `json:"time"`
	Output     string    `json:"output"`
	Path       string    `json:"path,omitempty"`
	Mode       string    `json:"mode,omitempty"`
	Origin     string    `json:"origin,omitempty"`
	Note       string    `json:"note,omitempty"`
	DurationMs int64     `json:"durationMs"`
	Error      string    `json:"error,omitempty"`
}

// State is returned to the settings UI.
type State struct {
	Version      string              `json:"version"`
	Config       config.Config       `json:"config"`
	HotkeyActive bool                `json:"hotkeyActive"`
	HotkeyError  string              `json:"hotkeyError,omitempty"`
	ConfigPath   string              `json:"configPath"`
	LogPath      string              `json:"logPath"`
	BridgePath   string              `json:"bridgePath"`
	ExePath      string              `json:"exePath"`
	Bridge       locate.BridgeStatus `json:"bridge"`
	AutoStart    bool                `json:"autoStart"`
	LastCapture  *LastCapture        `json:"lastCapture,omitempty"`
}

// Deps connects the HTTP layer to the application.
type Deps struct {
	State  func() State
	Save   func(config.Config) (State, []string, error)
	Action func(string) (string, error)
}

// Server is the running loopback web server.
type Server struct {
	deps  Deps
	token string
	url   string
	http  *http.Server
	ln    net.Listener
}

// Start binds to 127.0.0.1 on an ephemeral port.
func Start(deps Deps) (*Server, error) {
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(tokenBytes)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{
		deps:  deps,
		token: token,
		ln:    ln,
		url:   fmt.Sprintf("http://%s/s/%s/", ln.Addr().String(), token),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	s.http = &http.Server{
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       45 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	go func() {
		_ = s.http.Serve(ln)
	}()
	return s, nil
}

// URL is the tokenized page URL. Treat it as a secret.
func (s *Server) URL() string { return s.url }

// Close stops the HTTP server.
func (s *Server) Close() error {
	return s.http.Close()
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	prefix := "/s/" + s.token + "/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	relative := strings.TrimPrefix(r.URL.Path, prefix)
	switch {
	case relative == "" || relative == "index.html":
		s.serveAsset(w, "assets/index.html", "text/html; charset=utf-8")
	case relative == "assets/app.css":
		s.serveAsset(w, "assets/app.css", "text/css; charset=utf-8")
	case relative == "assets/app.js":
		s.serveAsset(w, "assets/app.js", "application/javascript; charset=utf-8")
	case relative == "assets/icon.png":
		s.serveAsset(w, "assets/icon.png", "image/png")
	case relative == "api/state":
		s.handleState(w, r)
	case relative == "api/config":
		s.handleConfig(w, r)
	case relative == "api/action":
		s.handleAction(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) serveAsset(w http.ResponseWriter, name, contentType string) {
	data, err := fs.ReadFile(embedded, name)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, _ = w.Write(data)
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, s.deps.State())
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "cross-origin request rejected")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 256<<10))
	if err != nil {
		writeError(w, http.StatusBadRequest, "request body is too large")
		return
	}
	var cfg config.Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	state, warnings, err := s.deps.Save(cfg)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "state": state, "warnings": warnings,
	})
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "cross-origin request rejected")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<10))
	if err != nil {
		writeError(w, http.StatusBadRequest, "request body is too large")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(body, &req) != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "missing action name")
		return
	}
	message, err := s.deps.Action(req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": message})
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host) &&
		(u.Scheme == "http" || u.Scheme == "https")
}

func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, POST")
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": message})
}
