# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

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
