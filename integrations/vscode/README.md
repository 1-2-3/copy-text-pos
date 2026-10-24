# Copy Text Pos Bridge

A tiny companion extension for Copy Text Pos. It publishes the active
file and selection range to a local snapshot so the native Windows app can copy
an exact reference such as:

```text
D:\projects\android\src\composables\useAsync.ts:13-18
```

- No network access, telemetry or clipboard writes.
- Selected text is **not** written to disk; only the file path, line range and
  app/workspace names are stored.
- Works in VS Code, Cursor, Windsurf, VSCodium and other VS Code-compatible
  desktop editors on Windows.

## Install

From a checkout of the main project:

```powershell
cd integrations/vscode
npm run package
code --install-extension copy-text-pos-bridge-0.1.0.vsix
# or
cursor --install-extension copy-text-pos-bridge-0.1.0.vsix
```

Reload the editor window. Focus a text editor with a non-empty selection; the
snapshot appears in Copy Text Pos → Settings → Editor integrations.

Snapshot path:

```text
%LOCALAPPDATA%\CopyTextPos\vscode-bridge.json
```

## Notes

- Only the focused editor window publishes a snapshot. Focus changes are
  tracked so another window's stale selection cannot win.
- If no selection is active, the app falls back to its normal clipboard-based
  capture for that hotkey press.
- This extension is optional. Without it, the native app still uses window
  titles, workspace metadata and source searching, but the range is inferred
  rather than exact.
