'use strict';

// Copy Text Pos Bridge
//
// This extension does not copy anything itself. While its VS Code window is
// focused it publishes a tiny JSON snapshot (active file + selected line range)
// to %LOCALAPPDATA%\CopyTextPos\vscode-bridge.json. The native tray app reads
// that snapshot, so VS Code, Cursor and compatible editors get an exact range
// without sending synthetic Ctrl+C. Selected text is never written to disk.

const fs = require('fs');
const os = require('os');
const path = require('path');
const vscode = require('vscode');

const VERSION = 1;
const HEARTBEAT_MS = 30000;

let focused = vscode.window.state.focused;
let emptyTimer = null;
let heartbeat = null;
let lastError = '';
let lastWrite = 0;

function bridgeDir() {
  const base = process.env.LOCALAPPDATA || path.join(os.homedir(), 'AppData', 'Local');
  return path.join(base, 'CopyTextPos');
}

function bridgeFile() {
  return path.join(bridgeDir(), 'vscode-bridge.json');
}

function currentContext() {
  const editor = vscode.window.activeTextEditor;
  const context = {
    version: VERSION,
    timestamp: Date.now(),
    appName: vscode.env.appName,
    workspace: vscode.workspace.name || '',
    file: '',
    startLine: 0,
    endLine: 0,
    hasSelection: false,
    pid: process.pid
  };
  if (!editor || editor.document.uri.scheme !== 'file') {
    return context;
  }
  const selection = editor.selection;
  context.file = editor.document.uri.fsPath;
  context.languageId = editor.document.languageId;
  context.dirty = editor.document.isDirty;
  if (!selection || selection.isEmpty) {
    context.startLine = selection ? selection.active.line + 1 : 0;
    context.endLine = context.startLine;
    return context;
  }
  context.startLine = selection.start.line + 1;
  let endLine = selection.end.line + 1;
  // A selection ending at column 0 does not include that next line.
  if (selection.end.character === 0 && selection.end.line > selection.start.line) {
    endLine -= 1;
  }
  context.endLine = Math.max(context.startLine, endLine);
  context.hasSelection = true;
  return context;
}

function writeSnapshot() {
  if (process.platform !== 'win32' || !focused) {
    return;
  }
  try {
    fs.mkdirSync(bridgeDir(), { recursive: true });
    const file = bridgeFile();
    const temporary = `${file}.${process.pid}.tmp`;
    fs.writeFileSync(temporary, JSON.stringify(currentContext()), 'utf8');
    fs.renameSync(temporary, file);
    lastWrite = Date.now();
    lastError = '';
  } catch (error) {
    lastError = error && error.message ? error.message : String(error);
  }
}

function scheduleWrite(immediate) {
  if (!focused) {
    return;
  }
  if (emptyTimer) {
    clearTimeout(emptyTimer);
    emptyTimer = null;
  }
  if (immediate) {
    writeSnapshot();
    return;
  }
  emptyTimer = setTimeout(() => {
    emptyTimer = null;
    writeSnapshot();
  }, 180);
}

function activate(context) {
  if (process.platform !== 'win32') {
    return;
  }

  context.subscriptions.push(
    vscode.window.onDidChangeTextEditorSelection((event) => {
      scheduleWrite(!event.selections || event.selections.every((selection) => !selection.isEmpty));
    }),
    vscode.window.onDidChangeActiveTextEditor(() => scheduleWrite(true)),
    vscode.window.onDidChangeWindowState((state) => {
      focused = state.focused;
      if (focused) {
        scheduleWrite(true);
      }
    }),
    vscode.workspace.onDidChangeWorkspaceFolders(() => scheduleWrite(true)),
    vscode.commands.registerCommand('copyTextPos.showBridgeStatus', () => {
      const context = currentContext();
      const detail = context.hasSelection
        ? `${context.file}:${context.startLine}-${context.endLine}`
        : context.file
          ? `${context.file} (no selection)`
          : 'No file editor is active.';
      vscode.window.showInformationMessage(
        `Copy Text Pos bridge: ${detail}. Snapshot: ${bridgeFile()}`
      );
    })
  );

  heartbeat = setInterval(() => {
    if (focused) {
      writeSnapshot();
    }
  }, HEARTBEAT_MS);

  context.subscriptions.push({
    dispose() {
      if (emptyTimer) {
        clearTimeout(emptyTimer);
      }
      if (heartbeat) {
        clearInterval(heartbeat);
      }
    }
  });

  scheduleWrite(true);
}

function deactivate() {
  if (emptyTimer) {
    clearTimeout(emptyTimer);
    emptyTimer = null;
  }
  if (heartbeat) {
    clearInterval(heartbeat);
    heartbeat = null;
  }
}

module.exports = { activate, deactivate };
