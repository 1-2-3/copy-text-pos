# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.7] - 2026-10-24

### Fixed

- Resolve JetBrains **Copy Reference** paths that are relative to a Java/Kotlin
  source root inside a nested Maven/Gradle module, such as
  `com/example/Foo.java:42`, instead of only trying the project root.

## [0.1.6] - 2026-10-24

### Fixed

- Obsidian window titles ending in a version suffix such as
  `Obsidian 1.14.5` no longer treat `Obsidian 1.14.5` as the vault name.
- Resolve Obsidian notes whose tab title is a frontmatter alias by reading the
  active vault's `.obsidian\workspace.json` and mapping open tabs to real files.

## [0.1.5] - 2026-10-24

### Changed

- Replaced the single “release in Explorer” switch with a configurable
  `releaseHotkeyProcesses` list. The default list contains `explorer.exe`.
- The settings page now lets users add/remove executable names; the global
  shortcut is released while any listed process is foreground and restored when
  the user switches away.
- Config schema v3 migrates the old `pauseHotkeyInExplorer` setting: `false`
  becomes an empty list, `true` or a missing value keeps Explorer.

## [0.1.4] - 2026-10-24

### Added

- File Explorer compatibility: while Explorer is foreground, Copy Text Pos
  temporarily unregisters its global shortcut so Explorer's native
  `Ctrl+Shift+C` “Copy as path” action keeps working. The shortcut returns when
  another application is focused. Controlled by `pauseHotkeyInExplorer`.

### Changed

- Shortcut suspension is integrated with settings changes, so changing the
  hotkey while Explorer is active is applied when the shortcut is released.

## [0.1.3] - 2026-10-24

### Changed

- Default clipboard output is now wrapped in double quotes:
  `"D:\projects\app\src\useAsync.ts:13-18"`.
- Added quoted presets and an explicit unquoted preset in Settings.
- Config schema v2 migrates the old unquoted default to the quoted default
  without touching custom templates.

## [0.1.2] - 2026-10-24

### Fixed

- Match Markdown table selections against the source table. Typora copies
  tables with padded columns and a reformatted separator row; separator rows are
  now ignored on both sides and column padding is normalized.
- Added regression coverage for copied tables with and without a separator row,
  including partial-row selections.

## [0.1.1] - 2026-10-24

### Fixed

- Decode Typora's hex-encoded `history.data` file (as written on Windows), so
  recent Typora documents resolve without adding a search root.
- Added regression tests for raw JSON, BOM-prefixed JSON and hex-encoded Typora
  history.

## [0.1.0] - 2026-10-24

### Added

- Native Windows tray application with a configurable global shortcut.
- Path and inclusive line-range resolution for VS Code, Cursor, Windsurf,
  VSCodium, JetBrains IDEs, Obsidian, Typora and title-based editors.
- Optional VS Code/Cursor bridge extension for exact path and range, including
  unsaved buffers.
- JetBrains *Copy Reference* probe and recent-project resolution.
- Obsidian vault and Typora recent-file indexes.
- Bounded, cached source search with exact, whitespace and Markdown-aware line
  matching.
- Bilingual (English/Chinese) loopback settings page with hotkey recording,
  output templates, search roots, notifications and autostart.
- Clipboard snapshot and restore on failed captures.
- Single-instance handling, taskbar restart recovery and per-user autostart.
- Windows icon/manifest resources, release build script and GitHub Actions.
- English and Chinese READMEs, editor-support notes and screenshots.
