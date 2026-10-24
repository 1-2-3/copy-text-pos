# Editor support in detail

This page explains how Copy Text Pos obtains a path and line range for each
editor family, and where the limits are. The short table lives in the main
[README](../README.md).

There is no universal Windows API for “the source file and source lines behind
the current selection”. Copy Text Pos always tries sources in this order:

1. **Exact editor bridge** — a fresh snapshot written by the VS Code companion
   extension (path and range, no clipboard copy).
2. **Absolute path in the window title** — many editors can be configured to
   show one.
3. **Editor-owned metadata** — recent workspaces, vault list, Typora history,
   JetBrains *Copy Reference*.
4. **Configured search roots** — a time-bounded file lookup.
5. **Selected-text search** — the clipboard selection is located in the source
   file on disk to derive inclusive line numbers.

If all path sources fail, the action fails and the previous clipboard is
restored.

## Support levels

| Level | Path | Range |
|---|---|---|
| **Exact** | From the editor bridge or a complete path in the title | From the editor bridge, or unambiguous title/line metadata |
| **Good** | From editor-owned metadata | Inferred from selected-text search |
| **Inferred** | Title filename plus search roots | Inferred; may be ambiguous |
| **Unsupported** | — | — |

“Inferred” is not a bug by itself: it is the only portable technique available
without a plugin in each editor.

---

## VS Code, Cursor, Windsurf, VSCodium

### With the bridge (recommended, exact)

The bridge extension is in [`integrations/vscode`](../integrations/vscode/README.md).
It publishes the active file and selected line range whenever the editor window
is focused.

- **Path:** exact.
- **Range:** exact, including unsaved (“dirty”) files.
- **Clipboard:** untouched; the native app does not send `Ctrl+C`.
- **Multi-window:** only the focused window writes the snapshot, so another
  window's stale selection does not win.
- **Install:** build or download the `.vsix`, then
  `code --install-extension copy-text-pos-bridge-0.1.0.vsix` (use `cursor` for
  Cursor). Reload the window.

The snapshot contains only paths, line numbers, app/workspace names and a
timestamp. Selected text is never written.

**Terminal caveat:** VS Code's extension API does not expose “the integrated
terminal currently has keyboard focus” reliably. If you invoke the hotkey while
the terminal panel is focused, the bridge may still point at the last editor
selection. Native terminal windows are skipped by the app, but the editor's
integrated terminal is not distinguishable at that level. Use the shortcut in
the editor text area.

### Without the bridge

- **Path:** from the title plus `workspaceStorage` indexes. The title format is
  usually `file.ts - workspace - Cursor`. The app matches the workspace name
  and searches that root for the file.
- **Range:** selected text is copied and searched in the file. Duplicate
  snippets can resolve to the first occurrence; unsaved edits can make the
  range stale.
- **Improve it:** set a title with the absolute path:

  ```json
  "window.title": "${activeEditorLong}${separator}${rootName}${separator}${appName}"
  ```

  Or add the project folder to **Search roots**.

`${activeEditorLong}` should render the full file path. Product support for
this variable can vary between VS Code versions and forks, but it is the
cheapest way to make title parsing exact.

### Remote SSH / dev containers

When the editor is attached to a remote filesystem, the local Windows path may
not exist. The bridge skips non-`file:` URIs. The app cannot access remote
source files, so remote-only documents are unsupported.

---

## JetBrains IDEs

This covers IntelliJ IDEA, Android Studio, PyCharm, WebStorm, GoLand, CLion,
Rider, RustRover, PhpStorm, RubyMine and related products.

- **Path:** the app sends the stock **Copy Reference** action
  (`Ctrl+Alt+Shift+C`) to the focused IDE, then reads the reference from the
  clipboard. Absolute references are used directly; relative references such as
  `src/main/java/Foo.java:42` are resolved against recent project roots and
  search roots.
- **Range:** the reference gives the start line. The app then searches the
  selected text in the file and chooses the occurrence nearest that anchor.
  If the file changed on disk, the end line is estimated from the number of
  selected lines.
- **No IDE plugin required.**
- **Keymap:** *Copy Reference* must be bound to `Ctrl+Alt+Shift+C` (the default).
  If you use a custom keymap, restore that binding or disable **JetBrains Copy
  Reference** in settings and rely on search roots.
- **Dirty files:** Copy Reference still returns a path and line, but the on-disk
  search may not match. Save the file for exact ranges.
- **Relative paths:** project roots are read from `recentProjects.xml` in
  `%APPDATA%\JetBrains\...` and `%APPDATA%\Google\...`. New Android Studio
  versions may not keep that file; adding the project to **Search roots** helps.
- **Clipboard:** the reference action temporarily replaces the clipboard, but
  the app restores the previous clipboard on failure and replaces it with the
  final reference on success.

Android Studio titles often look like `ProjectName – What's New`; the first
segment is used as the project name, and the reference probe supplies the real
file path.

---

## Obsidian

- **Path:** reads `%APPDATA%\obsidian\obsidian.json` to find registered vaults.
  It also reads the vault's `.obsidian\workspace.json` to map open tabs to real
  paths, which handles tabs whose title is a frontmatter alias instead of the
  filename. It then falls back to searching for the note name.
- **Range:** selected text is searched in the Markdown source. This is accurate
  for plain text and source-mode selections.
- **Rendered Markdown caveat:** if you select rendered text in reading/live
  preview, copied text may omit `**`, links, list markers or HTML. The matcher
  has a Markdown-aware fallback, but tables, embeds and complex formatting can
  still be ambiguous.
- **Multiple vaults:** the vault name from the title is used to disambiguate.
- **Fallback:** add the vault directory to **Search roots** if the registry is
  missing or the note is outside a registered vault.

---

## Typora

- **Path:** reads Typora's `%APPDATA%\Typora\history.data` recent-document list
  and matches the document name in the title. On Windows this file is usually a
  hex-encoded JSON blob; both hex and plain JSON are decoded.
- **Range:** selected text is searched in the Markdown source, with the same
  rendered-Markdown caveat as Obsidian. Typora copies tables as padded Markdown;
  table separator rows and column padding are normalized, so whole-table and
  row-range selections map back to their source lines.
- **Fallback:** add the document folder to **Search roots**.
- **Unsaved documents:** save first; an unsaved Typora buffer has no stable
  source path for the fallback search.

---

## Sublime Text, Notepad++, EmEditor, Neovim GUI, Windows Notepad

These editors do not have a known metadata adapter in v0.1.x. Use one or both
of:

1. **A full path in the window title.** Copy Text Pos will use it directly.
2. **Search roots.** Add the project/document folder. The app finds the file by
   the name in the title, then locates the selection in the source.

- Path is **inferred** unless the title includes an absolute path.
- Range is **inferred** from selected-text search.
- Windows 11 Notepad is a WinUI app and generally works when a search root is
  configured. Classic Notepad works the same way.
- Neovim GUIs often put `file.ts (13) - NVIM` in the title; the parenthesized
  line number is used as an anchor, improving duplicate handling.

---

## Unsupported targets

| Target | Why |
|---|---|
| Windows Terminal, cmd, PowerShell console | `Ctrl+C` is SIGINT, not a copy operation. The app skips native console windows. |
| Browser page text | There is no source file path for a DOM selection. |
| Word / Excel / PowerPoint | Not plain text files with source line ranges. |
| PDF viewers | No editable source selection with file path semantics. |
| Remote-only / virtual documents | The local Windows process cannot read the source file. |

## Verifying support on your machine

1. Select a few lines of code in the editor.
2. Press the configured hotkey.
3. Paste into a text field.
4. Check the tray notification and, if needed, the log at
   `%LOCALAPPDATA%\CopyTextPos\copy-text-pos.log`.

For VS Code family, check **Settings → Editor integrations**; it shows whether a
fresh bridge snapshot is present, which app/file wrote it, and how old it is.

## Adding an adapter

New exact integrations are welcome. A bridge adapter only needs to write JSON to
`%LOCALAPPDATA%\CopyTextPos\vscode-bridge.json` with this shape:

```json
{
  "version": 1,
  "timestamp": 1780000000000,
  "appName": "My Editor",
  "workspace": "project",
  "file": "D:\\projects\\project\\src\\main.ts",
  "startLine": 13,
  "endLine": 18,
  "hasSelection": true
}
```

`startLine`/`endLine` are 1-based inclusive. Only write while your editor window
is focused, and never include selected text. The native app validates freshness,
the file extension, the foreground process family and that the file name appears
in the focused window title.

For path-only adapters, add recent-file/workspace discovery under
`internal/locate`. The rest of the pipeline (selection matching, output
templates, clipboard safety) is shared.
