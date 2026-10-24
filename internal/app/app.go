//go:build windows

// Package app wires configuration, capture, path discovery, tray UI and the
// local settings server together.
package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"copytextpos/internal/assets"
	"copytextpos/internal/autostart"
	"copytextpos/internal/capture"
	"copytextpos/internal/config"
	"copytextpos/internal/hotkey"
	"copytextpos/internal/locate"
	"copytextpos/internal/logging"
	"copytextpos/internal/render"
	"copytextpos/internal/settings"
	"copytextpos/internal/tray"
	"copytextpos/internal/version"
	"copytextpos/internal/winapi"
)

const (
	commandCopy     = 1
	commandSettings = 2
	commandConfig   = 3
	commandLogs     = 4
	commandAbout    = 5
	commandExit     = 6
)

// App is the long-lived tray application.
type App struct {
	cfgPath    string
	logPath    string
	bridgePath string
	exePath    string
	chinese    bool
	firstRun   bool

	cfg       atomic.Pointer[config.Config]
	resolver  *locate.Resolver
	log       *logging.Logger
	tray      *tray.Tray
	server    *settings.Server
	serverErr error

	captureMu sync.Mutex
	saveMu    sync.Mutex
	lastMu    sync.Mutex
	last      *settings.LastCapture
	quit      atomic.Bool
	// suppressUntil prevents the JetBrains Copy Reference probe from
	// re-triggering our own global shortcut when it contains Ctrl+Shift+C.
	suppressUntil atomic.Int64
}

// New loads configuration and creates the tray application. The caller must
// call Run to start the message loop.
//
// New locks the calling goroutine to its OS thread because the hidden tray
// window and RegisterHotKey are thread-affine. Run must be called on the same
// goroutine.
func New(configPath string, debug bool) (*App, error) {
	runtime.LockOSThread()
	winapi.SetDPIAware()

	if configPath == "" {
		var err error
		configPath, err = config.DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	logPath, err := config.LogPath()
	if err != nil {
		return nil, err
	}
	bridgePath, err := config.BridgePath()
	if err != nil {
		return nil, err
	}
	exePath, err := os.Executable()
	if err != nil {
		exePath = "copy-text-pos.exe"
	}
	_ = os.MkdirAll(filepath.Dir(logPath), 0o700)
	logger := logging.New(logPath, debug)
	a := &App{
		cfgPath:    configPath,
		logPath:    logPath,
		bridgePath: bridgePath,
		exePath:    exePath,
		log:        logger,
		chinese:    (winapi.UserDefaultUILanguage() & 0x3ff) == 0x04,
	}

	cfg, firstRun, recovered, loadErr := loadOrRecoverConfig(configPath)
	if loadErr != nil {
		a.log.Error("load config: %v", loadErr)
	}
	a.firstRun = firstRun
	if firstRun || recovered || cfg.Migrated {
		if err := config.Save(configPath, cfg); err != nil {
			a.log.Error("write initial config: %v", err)
		} else if cfg.Migrated {
			a.log.Info("migrated configuration to schema v%d", config.CurrentVersion)
		}
	}
	if recovered {
		a.log.Info("invalid config was preserved as a .bad file and defaults were restored")
	}
	a.cfg.Store(&cfg)
	a.resolver = locate.NewResolver(resolverOptions(cfg, bridgePath))

	spec, err := hotkey.Parse(cfg.Hotkey)
	if err != nil {
		a.log.Error("invalid hotkey %q: %v", cfg.Hotkey, err)
		spec = hotkey.Default()
	}

	items := a.menuItems()
	tr, err := tray.New(tray.Config{
		Tooltip:   a.tooltipFor(cfg.Hotkey),
		IconData:  assets.Icon,
		Items:     items,
		OnCommand: a.handleMenuCommand,
		OnHotkey: func() {
			if a.hotkeySuppressed() {
				a.log.Info("ignored hotkey triggered by our own JetBrains probe")
				return
			}
			go a.CaptureCurrent()
		},
		OnSettings: a.OpenSettings,
	})
	if err != nil {
		logger.Error("create tray: %v", err)
		logger.Close()
		return nil, err
	}
	a.tray = tr

	if err := tr.ApplyHotkey(&spec); err != nil {
		a.log.Error("register hotkey %s: %v", spec.String(), err)
	} else {
		a.log.Info("registered global hotkey %s", spec.String())
	}

	server, serverErr := settings.Start(settings.Deps{
		State:  a.State,
		Save:   a.SaveConfig,
		Action: a.doAction,
	})
	if serverErr != nil {
		a.serverErr = serverErr
		a.log.Error("start settings server: %v", serverErr)
	} else {
		a.server = server
		if debug {
			a.log.Info("settings URL: %s", server.URL())
		} else {
			a.log.Info("settings server listening on loopback")
		}
	}

	if err := autostart.Set(cfg.StartWithWindows, exePath); err != nil {
		a.log.Error("apply autostart=%v: %v", cfg.StartWithWindows, err)
	}

	a.log.Info("started version=%s config=%s exe=%s", version.String(), configPath, exePath)
	return a, nil
}

func loadOrRecoverConfig(path string) (cfg config.Config, firstRun, recovered bool, err error) {
	cfg, err = config.Load(path)
	if err == nil {
		return cfg, false, false, nil
	}
	if errors.Is(err, config.ErrNotFound) {
		return config.Default(), true, false, nil
	}
	// Preserve the old file, then run with defaults instead of leaving the user
	// without a working tray app.
	badPath := fmt.Sprintf("%s.bad-%s", path, time.Now().Format("20060102-150405"))
	if renameErr := os.Rename(path, badPath); renameErr != nil {
		return config.Default(), false, false, fmt.Errorf("%w (could not preserve bad config: %v)", err, renameErr)
	}
	return config.Default(), false, true, err
}

func resolverOptions(cfg config.Config, bridgePath string) locate.Options {
	return locate.Options{
		SearchRoots:              append([]string(nil), cfg.SearchRoots...),
		EnableRecents:            cfg.EnableEditorRecents,
		EnableBridge:             cfg.EnableVSCodeBridge,
		EnableJetBrainsReference: cfg.EnableJetBrainsReference,
		MaxSearch:                time.Duration(cfg.MaxSearchMs) * time.Millisecond,
		BridgePath:               bridgePath,
	}
}

func (a *App) menuItems() []tray.Item {
	tr := func(en, zh string) string {
		if a.chinese {
			return zh
		}
		return en
	}
	return []tray.Item{
		{ID: commandCopy, Label: tr("Copy path at cursor", "复制当前位置路径"), Default: true},
		{ID: commandSettings, Label: tr("Settings…", "设置…")},
		{Label: "", Separator: true},
		{ID: commandConfig, Label: tr("Open config file", "打开配置文件")},
		{ID: commandLogs, Label: tr("Open log", "打开日志")},
		{ID: commandAbout, Label: tr("About", "关于")},
		{Label: "", Separator: true},
		{ID: commandExit, Label: tr("Exit", "退出")},
	}
}

func (a *App) tooltipFor(hotkeyText string) string {
	return "Copy Text Pos · " + hotkeyText
}

func (a *App) handleMenuCommand(id int) {
	switch id {
	case commandCopy:
		go a.CaptureCurrent()
	case commandSettings:
		a.OpenSettings()
	case commandConfig:
		if err := winapi.ShellExecute(a.cfgPath); err != nil {
			a.forceNotify("Copy Text Pos", err.Error())
		}
	case commandLogs:
		if err := winapi.ShellExecute(a.logPath); err != nil {
			a.forceNotify("Copy Text Pos", err.Error())
		}
	case commandAbout:
		a.forceNotify("Copy Text Pos", version.Full())
	case commandExit:
		a.Quit()
	}
}

// Run blocks on the Windows message loop.
func (a *App) Run() error {
	go func() {
		time.Sleep(700 * time.Millisecond)
		if a.firstRun {
			a.forceNotify(
				a.tr("Copy Text Pos", "Copy Text Pos"),
				a.tr("Running in the tray. Select text and press "+a.currentHotkey()+".",
					"已在托盘运行。选中文字后按 "+a.currentHotkey()+"。"),
			)
		}
		if err := a.tray.HotkeyError(); err != nil {
			a.forceNotify(a.tr("Shortcut unavailable", "快捷键不可用"), err.Error())
		}
		if a.serverErr != nil {
			a.forceNotify(a.tr("Settings unavailable", "设置页不可用"), a.serverErr.Error())
		}
	}()
	// RegisterHotKey is global and would otherwise steal Explorer's own
	// Ctrl+Shift+C "Copy as path" shortcut. Poll the foreground window cheaply
	// and release our hotkey only while Explorer is active.
	go func() {
		time.Sleep(300 * time.Millisecond)
		a.watchExplorerForeground()
	}()
	err := a.tray.Run()
	if a.server != nil {
		_ = a.server.Close()
	}
	a.log.Info("message loop stopped")
	a.log.Close()
	return err
}

func (a *App) watchExplorerForeground() {
	ticker := time.NewTicker(75 * time.Millisecond)
	defer ticker.Stop()
	var lastHWND winapi.HWND
	foregroundProcess := ""
	for {
		if a.quit.Load() {
			return
		}
		hwnd := winapi.ForegroundWindow()
		if hwnd != lastHWND {
			lastHWND = hwnd
			foregroundProcess = ""
			if hwnd != 0 {
				pid := winapi.WindowProcessID(winapi.RootAncestor(hwnd))
				if process, err := winapi.ProcessInfo(pid); err == nil {
					foregroundProcess = process.Name
				}
			}
		}
		cfg := a.cfg.Load()
		wantSuspended := cfg != nil && cfg.ReleasesHotkeyFor(foregroundProcess)
		if wantSuspended != a.tray.HotkeySuspended() {
			if err := a.tray.SetHotkeySuspended(wantSuspended); err != nil {
				a.log.Error("set hotkey suspension=%v: %v", wantSuspended, err)
				if !wantSuspended {
					a.forceNotify(
						a.tr("Shortcut unavailable", "快捷键不可用"),
						err.Error(),
					)
				} else {
					// Retry on the next tick; unregistration rarely fails.
					lastHWND = 0
				}
			} else if wantSuspended {
				a.log.Info("hotkey paused while %s is foreground", foregroundProcess)
			} else {
				a.log.Info("hotkey resumed")
			}
		}
		<-ticker.C
	}
}

// Quit asks the tray to close.
func (a *App) Quit() {
	if a.quit.CompareAndSwap(false, true) {
		a.log.Info("exit requested")
		a.tray.Quit()
	}
}

// OpenSettings opens the local settings page in the default browser.
func (a *App) OpenSettings() {
	if a.server == nil {
		a.forceNotify(a.tr("Settings unavailable", "设置页不可用"), a.settingsError())
		return
	}
	if err := winapi.ShellExecute(a.server.URL()); err != nil {
		a.forceNotify(a.tr("Could not open settings", "无法打开设置页"), err.Error())
	}
}

func (a *App) settingsError() string {
	if a.serverErr != nil {
		return a.serverErr.Error()
	}
	return a.tr("The local settings server did not start.", "本地设置服务未能启动。")
}

func (a *App) doAction(name string) (string, error) {
	switch name {
	case "open-config":
		return a.tr("Opened config file.", "已打开配置文件。"), winapi.ShellExecute(a.cfgPath)
	case "open-logs":
		return a.tr("Opened log file.", "已打开日志。"), winapi.ShellExecute(a.logPath)
	case "open-data-folder":
		dir, err := config.LocalDataDir()
		if err != nil {
			return "", err
		}
		return a.tr("Opened data folder.", "已打开数据目录。"), winapi.ShellExecute(dir)
	default:
		return "", fmt.Errorf("unknown action %q", name)
	}
}

func (a *App) currentHotkey() string {
	cfg := a.cfg.Load()
	if cfg == nil {
		return hotkey.Default().String()
	}
	return cfg.Hotkey
}

func (a *App) tr(en, zh string) string {
	if a.chinese {
		return zh
	}
	return en
}

// State returns the settings page state.
func (a *App) State() settings.State {
	cfg := a.cfg.Load()
	if cfg == nil {
		defaultCfg := config.Default()
		cfg = &defaultCfg
	}
	var hotkeyErr error
	if a.tray != nil {
		hotkeyErr = a.tray.HotkeyError()
	}
	autoStart := cfg.StartWithWindows
	if actual, err := autostart.IsEnabled(); err == nil {
		autoStart = actual
	}
	bridge := locate.BridgeStatusForWindow(a.bridgePath, locate.Window{})
	a.lastMu.Lock()
	last := a.last
	if last != nil {
		copyLast := *last
		last = &copyLast
	}
	a.lastMu.Unlock()
	state := settings.State{
		Version:      version.String(),
		Config:       *cfg,
		HotkeyActive: hotkeyErr == nil,
		ConfigPath:   a.cfgPath,
		LogPath:      a.logPath,
		BridgePath:   a.bridgePath,
		ExePath:      a.exePath,
		Bridge:       bridge,
		AutoStart:    autoStart,
		LastCapture:  last,
	}
	if hotkeyErr != nil {
		state.HotkeyError = hotkeyErr.Error()
	}
	return state
}

// SaveConfig validates, applies and persists settings.
func (a *App) SaveConfig(in config.Config) (settings.State, []string, error) {
	a.saveMu.Lock()
	defer a.saveMu.Unlock()

	in.Normalize()
	if err := in.Validate(); err != nil {
		return a.State(), nil, err
	}
	spec, err := hotkey.Parse(in.Hotkey)
	if err != nil {
		return a.State(), nil, err
	}
	old := a.cfg.Load()
	needHotkey := old == nil || spec.String() != old.Hotkey || a.tray.HotkeyError() != nil
	if needHotkey {
		if err := a.tray.ApplyHotkey(&spec); err != nil {
			return a.State(), nil, err
		}
	}
	if err := config.Save(a.cfgPath, in); err != nil {
		if needHotkey && old != nil {
			if oldSpec, parseErr := hotkey.Parse(old.Hotkey); parseErr == nil {
				_ = a.tray.ApplyHotkey(&oldSpec)
			}
		}
		return a.State(), nil, err
	}

	newCfg := in
	a.cfg.Store(&newCfg)
	a.resolver.SetOptions(resolverOptions(newCfg, a.bridgePath))
	a.tray.SetTooltip(a.tooltipFor(newCfg.Hotkey))

	warnings := newCfg.Warnings()
	if err := autostart.Set(newCfg.StartWithWindows, a.exePath); err != nil {
		warnings = append(warnings, "start with Windows: "+err.Error())
	}
	a.log.Info("configuration saved hotkey=%s roots=%d format=%q", newCfg.Hotkey, len(newCfg.SearchRoots), newCfg.Format)
	return a.State(), warnings, nil
}

// CaptureCurrent performs one full hotkey action.
func (a *App) CaptureCurrent() {
	if a.hotkeySuppressed() {
		a.log.Info("ignored hotkey during synthetic input")
		return
	}
	if !a.captureMu.TryLock() {
		a.forceNotify(a.tr("Copy Text Pos", "Copy Text Pos"), a.tr("Another capture is still running.", "上一次复制仍在进行中。"))
		return
	}
	defer a.captureMu.Unlock()

	started := time.Now()
	w, err := locate.Foreground()
	if err != nil {
		a.finishError(started, err)
		return
	}
	if w.PID == winapi.CurrentProcessID() {
		return
	}
	if winapi.IsTerminalClass(w.Class) {
		a.finishError(started, errors.New(a.tr(
			"Terminal windows are not supported; select text in an editor pane.",
			"不支持终端窗口；请在编辑器文本区中选择内容。",
		)))
		return
	}

	cfg := a.cfg.Load()
	if cfg == nil {
		a.finishError(started, errors.New("configuration is unavailable"))
		return
	}
	// The foreground watcher should already have released the hotkey, but there
	// is a short race after switching windows. Never steal the native shortcut
	// of a configured passthrough process or overwrite its clipboard.
	if cfg.ReleasesHotkeyFor(w.Process.Name) {
		a.log.Info("ignored capture because %s is configured for shortcut passthrough", w.Process.Name)
		return
	}
	a.log.Info("capture start window=%q process=%s pid=%d hwnd=0x%X", logTitle(w.Title), w.Process.Name, w.PID, w.Handle)

	if cfg.EnableVSCodeBridge {
		if result, ok := a.resolver.BridgeResult(w); ok {
			if winapi.ForegroundWindow() != w.Handle {
				a.finishError(started, errors.New("focus changed while reading the editor snapshot"))
				return
			}
			output, err := render.Render(cfg.Format, render.Location{Path: result.Path, Start: result.StartLine, End: result.EndLine})
			if err != nil {
				a.finishError(started, err)
				return
			}
			if err := winapi.WriteClipboardText(output); err != nil {
				a.finishError(started, fmt.Errorf("write clipboard: %w", err))
				return
			}
			a.finishSuccess(started, cfg, result, output)
			return
		}
	}

	spec, err := hotkey.Parse(cfg.Hotkey)
	if err != nil {
		spec = hotkey.Default()
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	session := capture.Begin(w.Handle, spec)
	defer session.Close()

	selection, err := session.CopySelection()
	if err != nil {
		a.finishError(started, err)
		return
	}
	a.log.Info("selection copied bytes=%d ole_restore=%v", len(selection), session.OLEAvailable())

	reference := ""
	if cfg.EnableJetBrainsReference && w.JetBrains() {
		// The stock Copy Reference shortcut is Ctrl+Alt+Shift+C. Depending on
		// how Windows matches extra modifiers, that can also fire our global
		// shortcut. Ignore our own synthetic event for a short window.
		a.suppressUntil.Store(time.Now().Add(2 * time.Second).UnixNano())
		reference, err = session.CopyJetBrainsReference()
		if err != nil {
			a.log.Error("JetBrains reference probe: %v", err)
		} else {
			a.log.Info("JetBrains reference received bytes=%d", len(reference))
		}
	}

	result, err := a.resolver.Resolve(locate.Request{
		Window:        w,
		Selection:     selection,
		ReferenceText: reference,
	})
	if err != nil {
		a.finishError(started, err)
		return
	}
	if winapi.ForegroundWindow() != w.Handle {
		a.finishError(started, errors.New("focus changed while locating the file; nothing was copied"))
		return
	}
	output, err := render.Render(cfg.Format, render.Location{
		Path:  result.Path,
		Start: result.StartLine,
		End:   result.EndLine,
	})
	if err != nil {
		a.finishError(started, err)
		return
	}
	if err := session.WriteResult(output); err != nil {
		a.finishError(started, fmt.Errorf("write clipboard: %w", err))
		return
	}
	a.finishSuccess(started, cfg, result, output)
}

func (a *App) finishSuccess(started time.Time, cfg *config.Config, result locate.Result, output string) {
	elapsed := time.Since(started)
	a.record(&settings.LastCapture{
		Time:       time.Now(),
		Output:     output,
		Path:       result.Path,
		Mode:       result.Mode,
		Origin:     result.Origin,
		Note:       result.Note,
		DurationMs: elapsed.Milliseconds(),
	})
	a.log.Info("capture success duration=%dms path=%q range=%d-%d origin=%s mode=%s matches=%d note=%q",
		elapsed.Milliseconds(), result.Path, result.StartLine, result.EndLine, result.Origin, result.Mode, result.Matches, result.Note)

	if !cfg.Notifications {
		return
	}
	title := a.tr("Path and lines copied", "已复制路径与行号")
	message := truncateRunes(output, 220)
	if result.Note != "" {
		title = a.tr("Copied with a note", "已复制（有提示）")
		message = truncateRunes(output, 150) + "\n" + truncateRunes(result.Note, 100)
	}
	a.tray.Notify(title, message, false)
}

func (a *App) finishError(started time.Time, err error) {
	elapsed := time.Since(started)
	a.record(&settings.LastCapture{
		Time:       time.Now(),
		DurationMs: elapsed.Milliseconds(),
		Error:      err.Error(),
	})
	a.log.Error("capture failed duration=%dms error=%v", elapsed.Milliseconds(), err)
	a.forceNotify(a.tr("Copy failed", "复制失败"), truncateRunes(err.Error(), 240))
}

func (a *App) record(capture *settings.LastCapture) {
	a.lastMu.Lock()
	a.last = capture
	a.lastMu.Unlock()
}

func (a *App) hotkeySuppressed() bool {
	until := a.suppressUntil.Load()
	return until != 0 && time.Now().UnixNano() < until
}

func (a *App) forceNotify(title, message string) {
	if a.tray == nil {
		return
	}
	a.tray.Notify(title, truncateRunes(message, 240), true)
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func logTitle(s string) string {
	return truncateRunes(s, 160)
}
