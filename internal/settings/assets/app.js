(() => {
  'use strict';

  const SAMPLE = 'D:\\projects\\android\\src\\composables\\useAsync.ts';
  const SAMPLE_NAME = 'useAsync.ts';
  const SAMPLE_DIR = 'D:\\projects\\android\\src\\composables';

  const I18N = {
    en: {
      settingsTitle: 'Copy Text Pos · Settings',
      tagline: 'Selected text → file path + line range',
      statusRunning: 'Running',
      statusUnavailable: 'Shortcut unavailable',
      shortcutEyebrow: 'GLOBAL SHORTCUT',
      heroTitle: 'Copy a reference, not a paragraph.',
      heroSubtitle: 'Select code or text in an editor, press the shortcut, and get an agent-ready file path with an inclusive line range.',
      recordPrompt: 'Press the new shortcut…',
      reset: 'Reset',
      hotkeyHint: 'Click the shortcut to record a different one. At least one modifier is required.',
      previewLabel: 'CLIPBOARD PREVIEW',
      previewLive: 'live',
      previewSingle: 'Single line',
      previewMeta: 'inclusive lines, zero extra formatting',
      pathTitle: 'Path discovery',
      pathDesc: "Copy Text Pos uses each editor's own metadata first, then falls back to a bounded file search.",
      bridgeToggleTitle: 'VS Code / Cursor bridge',
      bridgeToggleDesc: 'Exact path and range from the editor extension. Recommended.',
      jetbrainsToggleTitle: 'JetBrains Copy Reference',
      jetbrainsToggleDesc: 'Probe IntelliJ, Android Studio and friends with their built-in action.',
      recentsToggleTitle: 'Read editor recent files',
      recentsToggleDesc: 'Use local workspace, vault and recent-file lists to find paths.',
      searchRootsTitle: 'Search roots',
      searchRootsOptional: 'fallback only',
      searchRootsDesc: 'Folders to scan when an editor cannot tell us the full path. Keep them narrow.',
      add: 'Add',
      noRoots: 'No search roots yet. Editor metadata works without them for most supported apps.',
      removeRoot: 'Remove',
      rootInvalid: 'Use an absolute Windows folder path, for example D:\\projects.',
      outputTitle: 'Output format',
      outputDesc: 'Shape the clipboard string for your agent, editor or notes app.',
      presetLabel: 'Preset',
      presetDefault: 'Quoted path:start-end (recommended)',
      presetAlways: 'Quoted path:start-end (always a range)',
      presetUnquoted: 'Unquoted path:start-end',
      presetGithub: 'GitHub link fragment',
      presetMarkdown: 'Markdown link',
      presetCustom: 'Custom template',
      templateLabel: 'Template',
      templateHelp: 'The default wraps the complete reference in double quotes. {range} becomes 13 or 13-18; {gh_range} becomes L13 or L13-L18.',
      exampleLabel: 'Example output',
      integrationsTitle: 'Editor integrations',
      integrationsDesc: 'Optional helpers make ranges exact instead of inferred.',
      bridgeStatusTitle: 'Companion bridge',
      bridgeMissing: 'Snapshot not found. Install the extension for exact VS Code ranges.',
      notInstalled: 'not installed',
      installTitle: 'How to install the VS Code / Cursor bridge',
      installStep1: 'Download the .vsix from the GitHub release, or build it from integrations/vscode.',
      installStep2: "Install it in the editor's terminal:",
      installStep3: 'Reload the window. A snapshot appears here when an editor is focused.',
      editorVSCode: 'VS Code family',
      editorJetBrains: 'JetBrains IDEs',
      editorObsidian: 'Obsidian / Typora',
      levelExact: 'exact',
      levelGood: 'good',
      levelScanned: 'scanned',
      levelInferred: 'inferred',
      levelHeuristic: 'inferred',
      levelMaybe: 'depends',
      levelNo: '—',
      appTitle: 'Application',
      appDesc: 'Quiet by default. Everything stays on this computer.',
      notifyTitle: 'Notifications',
      notifyDesc: 'Show a tray balloon after each successful copy.',
      autostartTitle: 'Start with Windows',
      autostartDesc: 'Launch quietly at sign-in. No administrator rights required.',
      lastCaptureTitle: 'Last action',
      lastCaptureEmpty: 'Nothing copied yet. Select text in an editor and press the shortcut.',
      configFile: 'Config file',
      logFile: 'Log file',
      dataFolder: 'Data folder',
      open: 'Open',
      supportTitle: 'Editor support at a glance',
      supportDesc: 'There is no universal Windows API for a file path plus selection line range. Support is layered and honest.',
      supportBadge: 'local-first',
      tableEditor: 'Editor',
      tablePath: 'Path',
      tableRange: 'Line range',
      tableHow: 'How',
      howBridge: 'Bridge extension, window title, workspace index',
      howJetBrains: 'Copy Reference probe + selection search',
      howObsidian: 'Vault registry + Markdown-aware search',
      howTypora: 'Recent-file history + Markdown-aware search',
      howGeneric: 'Full path in title or configured search roots',
      editorUnsupported: 'Terminals · browsers · Office documents',
      howUnsupported: 'Not plain files with a selection. Terminal windows are skipped.',
      supportFootnote: 'Ranges are inclusive. Duplicate snippets without a bridge or title line hint resolve to the first occurrence. Inferred ranges can be wrong if the buffer is unsaved or the selection was rendered Markdown.',
      footerPrivacy: 'Local only · no telemetry · no network calls',
      footerSettings: 'Settings served from 127.0.0.1',
      allSaved: 'All changes saved',
      unsavedChanges: 'Unsaved changes',
      saving: 'Saving…',
      saveFailed: 'Could not save',
      saveChanges: 'Save changes',
      savedToast: 'Settings saved.',
      actionDone: 'Done.',
      actionFailed: 'Action failed.',
      hotkeyNeedsModifier: 'Use at least one modifier (Ctrl, Shift, Alt or Win).',
      hotkeyUnsupported: 'That key is not supported. Try a letter, number, F-key or navigation key.',
      recording: 'Recording…',
      bridgeDisabled: 'Bridge is disabled. Enable it above to use exact editor ranges.',
      bridgeWaiting: 'Bridge is running. Select some text in VS Code or Cursor.',
      bridgeStale: 'Snapshot is stale. Focus the editor again or move the caret.',
      bridgeActive: 'Active · exact selection range available.',
      bridgeSnapshot: 'Snapshot',
      bridgeDisabledBadge: 'disabled',
      bridgeInactiveBadge: 'waiting',
      bridgeActiveBadge: 'active',
      bridgeStaleBadge: 'stale',
      lastCaptureMeta: '{origin} · {mode} · {ms} ms',
      lastCaptureError: 'Error',
      noCapture: 'No captures yet',
      ageSeconds: '{n}s ago',
      ageMinutes: '{n}m ago',
      ageHours: '{n}h ago',
      languageName: '中文',
      rootAdded: 'Search root added.',
      rootRemoved: 'Search root removed.',
      configWarnings: 'Saved with warnings',
      restoreDefault: 'Default shortcut restored.',
      saveShortcut: 'Save',
    },
    zh: {
      settingsTitle: 'Copy Text Pos · 设置',
      tagline: '选中文字 → 文件路径 + 行号范围',
      statusRunning: '运行中',
      statusUnavailable: '快捷键不可用',
      shortcutEyebrow: '全局快捷键',
      heroTitle: '复制引用，不要整段文字。',
      heroSubtitle: '在编辑器中选中代码或文本，按下快捷键，即可得到可直接交给 Agent 的绝对路径与闭区间行号。',
      recordPrompt: '请按下新的快捷键…',
      reset: '恢复默认',
      hotkeyHint: '点击快捷键即可重新录制。至少需要一个修饰键。',
      previewLabel: '剪贴板预览',
      previewLive: '实时',
      previewSingle: '单行时',
      previewMeta: '闭区间行号，不附加额外格式',
      pathTitle: '路径发现',
      pathDesc: '优先读取编辑器自身提供的元数据，失败后在限定时间内扫描文件。',
      bridgeToggleTitle: 'VS Code / Cursor 桥接',
      bridgeToggleDesc: '由编辑器扩展提供精确的路径和行号范围，推荐开启。',
      jetbrainsToggleTitle: 'JetBrains Copy Reference',
      jetbrainsToggleDesc: '对 IntelliJ、Android Studio 等发送其内置的复制引用动作。',
      recentsToggleTitle: '读取编辑器最近文件',
      recentsToggleDesc: '利用本地工作区、仓库和最近文件索引定位路径。',
      searchRootsTitle: '搜索根目录',
      searchRootsOptional: '仅作兜底',
      searchRootsDesc: '当编辑器无法提供完整路径时扫描这些目录。范围越窄越快。',
      add: '添加',
      noRoots: '尚未添加搜索根目录。大多数受支持的编辑器无需它也能工作。',
      removeRoot: '移除',
      rootInvalid: '请输入 Windows 绝对目录，例如 D:\\projects。',
      outputTitle: '输出格式',
      outputDesc: '按 Agent、编辑器或笔记软件的需要定制剪贴板内容。',
      presetLabel: '预设',
      presetDefault: '带引号：路径:起始-结束（推荐）',
      presetAlways: '带引号：路径:起始-结束（始终范围）',
      presetUnquoted: '不带引号：路径:起始-结束',
      presetGithub: 'GitHub 行号片段',
      presetMarkdown: 'Markdown 链接',
      presetCustom: '自定义模板',
      templateLabel: '模板',
      templateHelp: '默认用双引号包裹整段引用。{range} 输出 13 或 13-18；{gh_range} 输出 L13 或 L13-L18。',
      exampleLabel: '输出示例',
      integrationsTitle: '编辑器集成',
      integrationsDesc: '可选扩展可把推断范围升级为精确范围。',
      bridgeStatusTitle: '桥接状态',
      bridgeMissing: '未发现快照。安装扩展后可获得 VS Code 的精确行号。',
      notInstalled: '未安装',
      installTitle: '如何安装 VS Code / Cursor 桥接扩展',
      installStep1: '从 GitHub Release 下载 .vsix，或从 integrations/vscode 目录自行打包。',
      installStep2: '在编辑器终端中安装：',
      installStep3: '重载窗口。聚焦编辑器并选中文本后，这里会显示快照。',
      editorVSCode: 'VS Code 系列',
      editorJetBrains: 'JetBrains 系列',
      editorObsidian: 'Obsidian / Typora',
      levelExact: '精确',
      levelGood: '良好',
      levelScanned: '扫描范围',
      levelInferred: '推断',
      levelHeuristic: '推断',
      levelMaybe: '视配置而定',
      levelNo: '—',
      appTitle: '应用设置',
      appDesc: '默认安静运行，所有数据都留在本机。',
      notifyTitle: '通知',
      notifyDesc: '每次成功复制后显示托盘气泡。',
      autostartTitle: '开机启动',
      autostartDesc: '登录后静默启动，无需管理员权限。',
      lastCaptureTitle: '最近一次操作',
      lastCaptureEmpty: '还没有复制记录。在编辑器中选中文字后按下快捷键即可。',
      configFile: '配置文件',
      logFile: '日志文件',
      dataFolder: '数据目录',
      open: '打开',
      supportTitle: '编辑器支持一览',
      supportDesc: 'Windows 没有统一 API 可以同时取得文件路径和选中行号，因此支持是分层且如实标注的。',
      supportBadge: '本地优先',
      tableEditor: '编辑器',
      tablePath: '路径',
      tableRange: '行号范围',
      tableHow: '实现方式',
      howBridge: '桥接扩展、窗口标题、工作区索引',
      howJetBrains: 'Copy Reference 探测 + 选中文本搜索',
      howObsidian: 'Vault 注册表 + Markdown 感知搜索',
      howTypora: '最近文件历史 + Markdown 感知搜索',
      howGeneric: '标题中的完整路径或搜索根目录',
      editorUnsupported: '终端 · 浏览器 · Office 文档',
      howUnsupported: '不是带选区概念的纯文本文件；终端窗口会被跳过。',
      supportFootnote: '行号范围为闭区间。没有桥接或标题行号提示时，重复文本取第一次出现的位置。文件未保存，或选中内容来自渲染后的 Markdown 时，推断范围可能不准确。',
      footerPrivacy: '纯本地 · 无遥测 · 无网络请求',
      footerSettings: '设置页由 127.0.0.1 提供',
      allSaved: '所有更改均已保存',
      unsavedChanges: '有未保存的更改',
      saving: '正在保存…',
      saveFailed: '保存失败',
      saveChanges: '保存更改',
      savedToast: '设置已保存。',
      actionDone: '操作完成。',
      actionFailed: '操作失败。',
      hotkeyNeedsModifier: '请至少使用一个修饰键（Ctrl、Shift、Alt 或 Win）。',
      hotkeyUnsupported: '不支持该按键。请使用字母、数字、F 键或导航键。',
      recording: '录制中…',
      bridgeDisabled: '桥接已关闭。开启后可获得编辑器的精确范围。',
      bridgeWaiting: '桥接正在运行。请在 VS Code 或 Cursor 中选中文本。',
      bridgeStale: '快照已过期。请重新聚焦编辑器或移动光标。',
      bridgeActive: '已连接 · 可获得精确选区范围。',
      bridgeSnapshot: '快照',
      bridgeDisabledBadge: '已关闭',
      bridgeInactiveBadge: '等待中',
      bridgeActiveBadge: '已连接',
      bridgeStaleBadge: '已过期',
      lastCaptureMeta: '{origin} · {mode} · {ms} 毫秒',
      lastCaptureError: '错误',
      noCapture: '暂无记录',
      ageSeconds: '{n} 秒前',
      ageMinutes: '{n} 分钟前',
      ageHours: '{n} 小时前',
      languageName: 'EN',
      rootAdded: '已添加搜索根目录。',
      rootRemoved: '已移除搜索根目录。',
      configWarnings: '已保存，但存在警告',
      restoreDefault: '已恢复默认快捷键。',
      saveShortcut: '保存',
    }
  };

  const PRESETS = new Set(['"{path}:{range}"', '"{path}:{start}-{end}"', '{path}:{range}', '{path}:{start}-{end}', '{path}#L{start}-L{end}', '[{name}]({path}#L{start}-L{end})']);

  let lang = localStorage.getItem('ctp-lang');
  if (lang !== 'en' && lang !== 'zh') {
    lang = (navigator.language || 'en').toLowerCase().startsWith('zh') ? 'zh' : 'en';
  }
  const queryLang = new URLSearchParams(location.search).get('lang');
  if (queryLang === 'en' || queryLang === 'zh') lang = queryLang;

  let state = null;
  let draft = null;
  let draftRoots = [];
  let draftHotkey = 'Ctrl+Shift+C';
  let recording = false;
  let dirty = false;
  let saving = false;
  let toastTimer = null;

  const $ = (id) => document.getElementById(id);
  const els = {};

  function t(key, vars) {
    let value = (I18N[lang] && I18N[lang][key]) || I18N.en[key] || key;
    if (vars) {
      for (const [name, replacement] of Object.entries(vars)) {
        value = value.replaceAll(`{${name}}`, String(replacement));
      }
    }
    return value;
  }

  function applyI18n() {
    document.documentElement.lang = lang === 'zh' ? 'zh-CN' : 'en';
    document.title = t('settingsTitle');
    document.querySelectorAll('[data-i18n]').forEach((node) => {
      const key = node.getAttribute('data-i18n');
      node.textContent = t(key);
    });
    document.querySelectorAll('[data-i18n-title]').forEach((node) => {
      node.title = t(node.getAttribute('data-i18n-title'));
    });
    els.langToggle.textContent = t('languageName');
    updateStaticStatus();
    renderBridge();
    renderLastCapture();
    updateSavebar();
  }

  async function api(path, options = {}) {
    const response = await fetch(path, {
      cache: 'no-store',
      headers: { 'Content-Type': 'application/json' },
      ...options,
    });
    let payload = null;
    try { payload = await response.json(); } catch (_) { /* empty */ }
    if (!response.ok || !payload || payload.ok === false) {
      throw new Error((payload && payload.error) || `HTTP ${response.status}`);
    }
    return payload;
  }

  async function loadState() {
    state = await api('api/state');
    applyState(state);
  }

  function applyState(next) {
    state = next;
    const cfg = next.config;
    draft = {
      version: cfg.version || 1,
      hotkey: cfg.hotkey || 'Ctrl+Shift+C',
      format: cfg.format || '{path}:{range}',
      searchRoots: Array.isArray(cfg.searchRoots) ? [...cfg.searchRoots] : [],
      enableVSCodeBridge: !!cfg.enableVSCodeBridge,
      enableJetBrainsReference: !!cfg.enableJetBrainsReference,
      enableEditorRecents: !!cfg.enableEditorRecents,
      notifications: !!cfg.notifications,
      startWithWindows: !!cfg.startWithWindows,
      maxSearchMs: cfg.maxSearchMs || 1800,
    };
    draftRoots = [...draft.searchRoots];
    draftHotkey = draft.hotkey;
    els.enableBridge.checked = draft.enableVSCodeBridge;
    els.enableJetBrains.checked = draft.enableJetBrainsReference;
    els.enableRecents.checked = draft.enableEditorRecents;
    els.notifications.checked = draft.notifications;
    els.startWithWindows.checked = draft.startWithWindows;
    els.formatInput.value = draft.format;
    syncPreset(draft.format);
    renderHotkey();
    renderRoots();
    setPreview();
    els.configPath.textContent = next.configPath || '—';
    els.logPath.textContent = next.logPath || '—';
    els.dataFolder.textContent = dirname(next.configPath || '');
    els.versionLabel.textContent = `Copy Text Pos ${next.version || ''}`.trim();
    renderBridge();
    renderLastCapture();
    dirty = false;
    updateStatus();
    updateSavebar();
  }

  function dirname(path) {
    if (!path) return '—';
    const idx = Math.max(path.lastIndexOf('\\'), path.lastIndexOf('/'));
    return idx > 0 ? path.slice(0, idx) : path;
  }

  function updateStaticStatus() {
    if (!state) return;
    const unavailable = !!state.hotkeyError;
    els.statusText.textContent = unavailable ? t('statusUnavailable') : t('statusRunning');
    els.statusPill.classList.toggle('is-warning', unavailable);
    els.statusPill.title = state.hotkeyError || '';
  }

  function updateStatus() {
    updateStaticStatus();
  }

  function setStatusClass() {
    els.savebar.classList.toggle('is-dirty', dirty && !saving);
    els.savebar.classList.toggle('is-saving', saving);
    els.savebar.classList.remove('is-error');
  }

  function updateSavebar(message, isError) {
    setStatusClass();
    els.savebar.classList.toggle('is-clean', !dirty && !saving && !isError);
    els.saveBtn.disabled = saving || (!dirty && !isError);
    if (saving) {
      els.saveStatus.textContent = t('saving');
      return;
    }
    if (isError) {
      els.savebar.classList.add('is-error');
      els.saveStatus.textContent = message || t('saveFailed');
      els.saveBtn.disabled = false;
      return;
    }
    els.saveStatus.textContent = dirty ? t('unsavedChanges') : t('allSaved');
  }

  function markDirty() {
    if (saving) return;
    dirty = true;
    updateSavebar();
  }

  function currentConfig() {
    return {
      version: draft ? draft.version : 1,
      hotkey: draftHotkey,
      format: els.formatInput.value.trim(),
      searchRoots: [...draftRoots],
      enableVSCodeBridge: els.enableBridge.checked,
      enableJetBrainsReference: els.enableJetBrains.checked,
      enableEditorRecents: els.enableRecents.checked,
      notifications: els.notifications.checked,
      startWithWindows: els.startWithWindows.checked,
      maxSearchMs: draft ? draft.maxSearchMs : 1800,
    };
  }

  async function save() {
    if (saving) return;
    const cfg = currentConfig();
    if (!cfg.format) {
      showToast(t('saveFailed'), true);
      return;
    }
    recording = false;
    els.hotkeyRecorder.classList.remove('is-recording');
    saving = true;
    dirty = true;
    updateSavebar();
    try {
      const payload = await api('api/config', { method: 'POST', body: JSON.stringify(cfg) });
      applyState(payload.state);
      saving = false;
      if (Array.isArray(payload.warnings) && payload.warnings.length) {
        showToast(`${t('configWarnings')}: ${payload.warnings.join(' · ')}`, true);
      } else {
        showToast(t('savedToast'));
      }
      updateSavebar();
    } catch (error) {
      saving = false;
      updateSavebar(error.message, true);
      showToast(error.message, true);
    }
  }

  function renderHotkey() {
    els.hotkeyKeys.replaceChildren();
    if (!draftHotkey) {
      els.hotkeyPrompt.textContent = t('recordPrompt');
      els.hotkeyKeys.appendChild(document.createTextNode('—'));
      return;
    }
    const parts = draftHotkey.split('+');
    parts.forEach((part, index) => {
      if (index > 0) {
        const plus = document.createElement('span');
        plus.className = 'keycap plus';
        plus.textContent = '+';
        els.hotkeyKeys.appendChild(plus);
      }
      const cap = document.createElement('span');
      cap.className = 'keycap';
      cap.textContent = part;
      els.hotkeyKeys.appendChild(cap);
    });
    els.hotkeyPrompt.textContent = t('recordPrompt');
  }

  function renderRoots() {
    els.rootList.replaceChildren();
    if (!draftRoots.length) {
      const empty = document.createElement('div');
      empty.className = 'empty-roots';
      empty.textContent = t('noRoots');
      els.rootList.appendChild(empty);
      return;
    }
    draftRoots.forEach((root) => {
      const row = document.createElement('div');
      row.className = 'root-item';
      const icon = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
      icon.setAttribute('viewBox', '0 0 24 24');
      const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
      path.setAttribute('d', 'M3 7.5A2.5 2.5 0 0 1 5.5 5h4l2 2h7A2.5 2.5 0 0 1 21 9.5v8A2.5 2.5 0 0 1 18.5 20h-13A2.5 2.5 0 0 1 3 17.5v-10Z');
      icon.appendChild(path);
      const label = document.createElement('span');
      label.textContent = root;
      label.title = root;
      const remove = document.createElement('button');
      remove.type = 'button';
      remove.className = 'root-remove';
      remove.textContent = '×';
      remove.title = t('removeRoot');
      remove.setAttribute('aria-label', t('removeRoot'));
      remove.addEventListener('click', () => {
        draftRoots = draftRoots.filter((item) => item !== root);
        renderRoots();
        markDirty();
      });
      row.append(icon, label, remove);
      els.rootList.appendChild(row);
    });
  }

  function addRoot() {
    let value = els.rootInput.value.trim().replace(/^"|"$/g, '');
    if (!value) return;
    if (!/^(?:[A-Za-z]:[\\/]|\\\\)/.test(value)) {
      showToast(t('rootInvalid'), true);
      return;
    }
    value = value.replace(/\//g, '\\').replace(/\\+$/, (match) => value.length <= 3 ? match : '');
    if (draftRoots.some((root) => root.toLowerCase() === value.toLowerCase())) {
      els.rootInput.value = '';
      return;
    }
    draftRoots.push(value);
    els.rootInput.value = '';
    renderRoots();
    markDirty();
  }

  function syncPreset(format) {
    const matching = PRESETS.has(format) ? format : 'custom';
    els.formatPreset.value = matching;
    els.formatInput.value = format;
    setPreview();
  }

  function previewFor(format, start, end) {
    const range = start === end ? String(start) : `${start}-${end}`;
    const gh = start === end ? `L${start}` : `L${start}-L${end}`;
    const map = {
      path: SAMPLE,
      name: SAMPLE_NAME,
      dir: SAMPLE_DIR,
      ext: 'ts',
      start: String(start),
      end: String(end),
      line: String(start),
      range,
      gh_range: gh,
      start_line: String(start),
      end_line: String(end),
    };
    let out = format;
    for (const [key, value] of Object.entries(map)) {
      out = out.split(`{${key}}`).join(value);
    }
    return out;
  }

  function setPreview() {
    const format = els.formatInput.value || '{path}:{range}';
    const main = previewFor(format, 13, 18);
    els.previewOutput.textContent = main;
    els.previewSingle.textContent = previewFor(format, 13, 13);
    if (els.formatExample) els.formatExample.textContent = main;
  }

  function keyNameFromEvent(event) {
    if (/^Key[A-Z]$/.test(event.code)) return event.code.slice(3);
    if (/^Digit[0-9]$/.test(event.code)) return event.code.slice(5);
    if (/^F(?:[1-9]|1[0-9]|2[0-4])$/.test(event.code)) return event.code;
    const map = {
      Space: 'Space', Enter: 'Enter', NumpadEnter: 'Enter', Tab: 'Tab',
      Backspace: 'Backspace', Delete: 'Delete', Insert: 'Insert',
      Home: 'Home', End: 'End', PageUp: 'PageUp', PageDown: 'PageDown',
      ArrowUp: 'Up', ArrowDown: 'Down', ArrowLeft: 'Left', ArrowRight: 'Right',
      Minus: 'Minus', Equal: 'Equal', Comma: 'Comma', Period: 'Period',
      Slash: 'Slash', Backslash: 'Backslash', Semicolon: 'Semicolon',
      Quote: 'Quote', BracketLeft: 'LeftBracket', BracketRight: 'RightBracket',
      Backquote: 'Backtick',
    };
    return map[event.code] || null;
  }

  function startRecording() {
    recording = true;
    els.hotkeyRecorder.classList.add('is-recording');
    els.hotkeyPrompt.textContent = t('recordPrompt');
    els.hotkeyRecorder.focus();
  }

  function stopRecording() {
    recording = false;
    els.hotkeyRecorder.classList.remove('is-recording');
    renderHotkey();
  }

  function handleRecordingKey(event) {
    if (!recording) return;
    event.preventDefault();
    event.stopPropagation();
    if (event.key === 'Escape') {
      stopRecording();
      return;
    }
    if (['Control', 'Shift', 'Alt', 'Meta'].includes(event.key)) return;
    if (!event.ctrlKey && !event.shiftKey && !event.altKey && !event.metaKey) {
      showToast(t('hotkeyNeedsModifier'), true);
      return;
    }
    const key = keyNameFromEvent(event);
    if (!key) {
      showToast(t('hotkeyUnsupported'), true);
      return;
    }
    const parts = [];
    if (event.ctrlKey) parts.push('Ctrl');
    if (event.shiftKey) parts.push('Shift');
    if (event.altKey) parts.push('Alt');
    if (event.metaKey) parts.push('Win');
    parts.push(key);
    draftHotkey = parts.join('+');
    stopRecording();
    markDirty();
  }

  function renderBridge() {
    if (!state || !draft) return;
    const bridge = state.bridge || {};
    const enabled = els.enableBridge.checked;
    let badgeClass = 'badge-muted';
    let badgeText = t('notInstalled');
    let detail = t('bridgeMissing');
    if (!enabled) {
      badgeText = t('bridgeDisabledBadge');
      detail = t('bridgeDisabled');
    } else if (bridge.present && bridge.fresh && bridge.hasSelection) {
      badgeClass = 'badge-exact';
      badgeText = t('bridgeActiveBadge');
      detail = `${t('bridgeActive')} ${formatAge(bridge.ageSeconds)}`;
    } else if (bridge.present && bridge.fresh) {
      badgeClass = 'badge-heuristic';
      badgeText = t('bridgeInactiveBadge');
      detail = t('bridgeWaiting');
    } else if (bridge.present) {
      badgeClass = 'badge-heuristic';
      badgeText = t('bridgeStaleBadge');
      detail = t('bridgeStale');
    }
    els.bridgeBadge.className = `badge ${badgeClass}`;
    els.bridgeBadge.textContent = badgeText;
    els.bridgeDetail.textContent = detail;
    els.bridgeFile.textContent = bridge.path || '';
  }

  function formatAge(seconds) {
    const value = Math.max(0, Number(seconds) || 0);
    if (value < 60) return t('ageSeconds', { n: Math.round(value) });
    if (value < 3600) return t('ageMinutes', { n: Math.round(value / 60) });
    return t('ageHours', { n: Math.round(value / 3600) });
  }

  function renderLastCapture() {
    if (!state) return;
    const capture = state.lastCapture;
    if (!capture) {
      els.lastCaptureBody.textContent = t('lastCaptureEmpty');
      els.lastCaptureTime.textContent = '';
      return;
    }
    els.lastCaptureBody.replaceChildren();
    if (capture.error) {
      const error = document.createElement('div');
      error.className = 'capture-error';
      error.textContent = `${t('lastCaptureError')}: ${capture.error}`;
      els.lastCaptureBody.appendChild(error);
    } else {
      const output = document.createElement('div');
      output.textContent = capture.output || '';
      const meta = document.createElement('span');
      meta.className = 'capture-note';
      const origin = capture.origin || 'editor';
      const mode = capture.mode || 'exact';
      meta.textContent = t('lastCaptureMeta', { origin, mode, ms: capture.durationMs || 0 });
      els.lastCaptureBody.append(output, meta);
      if (capture.note) {
        const note = document.createElement('span');
        note.className = 'capture-note';
        note.textContent = capture.note;
        els.lastCaptureBody.appendChild(note);
      }
    }
    const when = new Date(capture.time);
    els.lastCaptureTime.textContent = Number.isNaN(when.getTime())
      ? ''
      : when.toLocaleString(lang === 'zh' ? 'zh-CN' : 'en-US', { hour: '2-digit', minute: '2-digit', month: 'short', day: 'numeric' });
  }

  function showToast(message, isError = false) {
    els.toast.textContent = message;
    els.toast.classList.toggle('error', !!isError);
    els.toast.classList.add('show');
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(() => els.toast.classList.remove('show'), 3600);
  }

  async function runAction(name) {
    try {
      const result = await api('api/action', { method: 'POST', body: JSON.stringify({ name }) });
      showToast(result.message || t('actionDone'));
    } catch (error) {
      showToast(error.message || t('actionFailed'), true);
    }
  }

  function bind() {
    els.hotkeyRecorder.addEventListener('click', () => {
      if (recording) stopRecording();
      else startRecording();
    });
    els.hotkeyRecorder.addEventListener('blur', () => {
      if (recording) stopRecording();
    });
    document.addEventListener('keydown', handleRecordingKey, true);
    document.addEventListener('keydown', (event) => {
      if (!recording && (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
        event.preventDefault();
        save();
      }
    });
    els.resetHotkey.addEventListener('click', () => {
      draftHotkey = 'Ctrl+Shift+C';
      stopRecording();
      markDirty();
      showToast(t('restoreDefault'));
    });

    els.langToggle.addEventListener('click', () => {
      lang = lang === 'zh' ? 'en' : 'zh';
      localStorage.setItem('ctp-lang', lang);
      applyI18n();
      renderHotkey();
      renderRoots();
    });

    [els.enableBridge, els.enableJetBrains, els.enableRecents, els.notifications, els.startWithWindows]
      .forEach((input) => input.addEventListener('change', () => {
        renderBridge();
        markDirty();
      }));

    els.rootInput.addEventListener('keydown', (event) => {
      if (event.key === 'Enter') { event.preventDefault(); addRoot(); }
    });
    els.addRoot.addEventListener('click', addRoot);

    els.formatPreset.addEventListener('change', () => {
      if (els.formatPreset.value !== 'custom') {
        els.formatInput.value = els.formatPreset.value;
      }
      setPreview();
      markDirty();
    });
    els.formatInput.addEventListener('input', () => {
      syncPreset(els.formatInput.value);
      markDirty();
    });
    document.querySelectorAll('#placeholderChips button').forEach((button) => {
      button.addEventListener('click', () => {
        const token = button.dataset.token || '';
        const input = els.formatInput;
        const start = input.selectionStart ?? input.value.length;
        const end = input.selectionEnd ?? start;
        input.value = input.value.slice(0, start) + token + input.value.slice(end);
        input.focus();
        input.setSelectionRange(start + token.length, start + token.length);
        syncPreset(input.value);
        markDirty();
      });
    });

    document.querySelectorAll('[data-action]').forEach((button) => {
      button.addEventListener('click', () => runAction(button.dataset.action));
    });
    els.saveBtn.addEventListener('click', save);

    window.addEventListener('beforeunload', (event) => {
      if (dirty && !saving) {
        event.preventDefault();
        event.returnValue = '';
      }
    });
  }

  async function init() {
    for (const key of ['langToggle', 'statusPill', 'statusText', 'hotkeyRecorder', 'hotkeyKeys', 'hotkeyPrompt',
      'resetHotkey', 'previewOutput', 'previewSingle', 'enableBridge', 'enableJetBrains', 'enableRecents',
      'rootInput', 'rootList', 'addRoot', 'formatPreset', 'formatInput', 'formatExample', 'bridgeBadge', 'bridgeDetail',
      'bridgeFile', 'notifications', 'startWithWindows', 'lastCaptureTime', 'lastCaptureBody',
      'configPath', 'logPath', 'dataFolder', 'versionLabel', 'savebar', 'saveBtn', 'saveStatus', 'toast']) {
      els[key] = $(key);
    }
    bind();
    applyI18n();
    try {
      await loadState();
    } catch (error) {
      showToast(`Cannot load settings: ${error.message}`, true);
    }
  }

  document.addEventListener('DOMContentLoaded', init);
})();
