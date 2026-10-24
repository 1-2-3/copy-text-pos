'use strict';

// Minimal local test for the snapshot writer. It stubs the `vscode` module so
// the extension can run under plain Node. On non-Windows it exits successfully
// because the production extension is a no-op there.

const assert = require('assert');
const fs = require('fs');
const os = require('os');
const path = require('path');
const Module = require('module');

if (process.platform !== 'win32') {
  console.log('bridge test skipped: not Windows');
  process.exit(0);
}

const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'ctp-bridge-test-'));
process.env.LOCALAPPDATA = tempRoot;

const listeners = {};
const commandDisposables = [];
const fakeContext = {
  subscriptions: [
    { dispose() {} },
    { dispose() {} }
  ]
};
const fakeVSCode = {
  env: { appName: 'Cursor Test' },
  workspace: {
    name: 'android',
    onDidChangeWorkspaceFolders(handler) {
      listeners.workspace = handler;
      return { dispose() {} };
    }
  },
  window: {
    state: { focused: true },
    activeTextEditor: null,
    onDidChangeTextEditorSelection(handler) {
      listeners.selection = handler;
      return { dispose() {} };
    },
    onDidChangeActiveTextEditor(handler) {
      listeners.activeEditor = handler;
      return { dispose() {} };
    },
    onDidChangeWindowState(handler) {
      listeners.windowState = handler;
      return { dispose() {} };
    }
  },
  commands: {
    registerCommand(_name, handler) {
      listeners.command = handler;
      const disposable = { dispose() {} };
      commandDisposables.push(disposable);
      return disposable;
    }
  }
};

const originalLoad = Module._load;
Module._load = function (request, parent, isMain) {
  if (request === 'vscode') {
    return fakeVSCode;
  }
  return originalLoad.call(this, request, parent, isMain);
};

const extension = require('../extension.js');
const editorPath = path.join(tempRoot, 'src', 'useAsync.ts');
fakeVSCode.window.activeTextEditor = {
  document: {
    uri: { scheme: 'file', fsPath: editorPath },
    languageId: 'typescript',
    isDirty: true
  },
  selection: {
    isEmpty: false,
    start: { line: 12, character: 0 },
    end: { line: 18, character: 0 },
    active: { line: 18, character: 0 }
  }
};

extension.activate(fakeContext);

const bridgeFile = path.join(tempRoot, 'CopyTextPos', 'vscode-bridge.json');
assert.ok(fs.existsSync(bridgeFile), 'snapshot should be written immediately');
const snapshot = JSON.parse(fs.readFileSync(bridgeFile, 'utf8'));
assert.strictEqual(snapshot.version, 1);
assert.strictEqual(snapshot.appName, 'Cursor Test');
assert.strictEqual(snapshot.workspace, 'android');
assert.strictEqual(snapshot.file, editorPath);
assert.strictEqual(snapshot.startLine, 13);
assert.strictEqual(snapshot.endLine, 18, 'end at column 0 must not include line 19');
assert.strictEqual(snapshot.hasSelection, true);
assert.ok(!JSON.stringify(snapshot).includes('selectedText'));

extension.deactivate();
Module._load = originalLoad;
fs.rmSync(tempRoot, { recursive: true, force: true });
console.log('bridge test passed');
