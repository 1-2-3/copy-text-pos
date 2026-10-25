# 编辑器支持详解

本文说明 Copy Text Pos 如何为各类编辑器获取路径与行号范围，以及限制在哪里。
简表见主 [README](../README.zh-CN.md)。

Windows 没有统一的 API 可以返回“当前选区对应的源文件和源码行号”。Copy Text Pos
始终按以下顺序寻找：

1. **精确编辑器桥接**：VS Code 伴生扩展写入的新鲜快照（路径和行号，不经过剪贴板）。
2. **窗口标题中的绝对路径**：很多编辑器可配置标题显示完整路径。
3. **编辑器自有元数据**：最近工作区、Vault 列表、Typora 历史、JetBrains
   *Copy Reference*。
4. **配置的搜索根目录**：限定时间内的文件查找。
5. **选中文本搜索**：在磁盘源文件中定位剪贴板选区，换算闭区间行号。

如果所有路径来源都失败，本次操作失败，并恢复原来的剪贴板。

## 支持级别

| 级别 | 路径 | 行号范围 |
|---|---|---|
| **精确** | 来自编辑器桥接或标题中的完整路径 | 来自桥接，或无歧义的标题元数据 |
| **良好** | 来自编辑器自有元数据 | 通过选中文本搜索推断 |
| **推断** | 标题文件名 + 搜索根目录 | 推断，可能有歧义 |
| **不支持** | — | — |

“推断”本身不是 bug：在不给每个编辑器写插件的前提下，这是唯一可移植的办法。

---

## VS Code、Cursor、Windsurf、VSCodium

### 使用桥接扩展（推荐，精确）

扩展源码位于 [`integrations/vscode`](../integrations/vscode/README.md)。编辑器
窗口获得焦点时，它上报当前文件与选区行号。

- **路径：** 精确。
- **范围：** 精确，包括未保存（dirty）的文件。
- **剪贴板：** 不被触碰，原生应用不发送 `Ctrl+C`。
- **多窗口：** 只有聚焦窗口写快照，因此另一个窗口的旧选区不会误胜出。
- **安装：** 构建或下载 `.vsix`，运行
  `code --install-extension copy-text-pos-bridge-0.1.0.vsix`（Cursor 用
  `cursor` 命令），然后重载窗口。

快照只包含路径、行号、应用/工作区名和时间戳，永远不写选中文字。

**终端注意事项：** VS Code 扩展 API 无法可靠判断“当前焦点在内置终端”。如果在
终端面板聚焦时按快捷键，桥接可能仍指向上一次编辑器选区。原生终端窗口会被应用
跳过，但编辑器内置终端在窗口级别无法区分。请在编辑器文本区使用快捷键。

### 不使用桥接

- **路径：** 从标题和 `workspaceStorage` 索引获取。常见标题是
  `file.ts - workspace - Cursor`。应用按工作区名匹配并搜索文件。
- **范围：** 复制选中文本并在文件中搜索。重复片段可能命中第一处；未保存修改
  可能让范围过期。
- **改善方式：** 在 `settings.json` 中设置绝对路径标题：

  ```json
  "window.title": "${activeEditorLong}${separator}${rootName}${separator}${appName}"
  ```

  或将项目目录加入 **搜索根目录**。

`${activeEditorLong}` 通常会渲染完整文件路径。不同 VS Code 版本/分支对该变量的
支持可能略有差异，但它是让标题解析变精确的最低成本方式。

### 远程 SSH / Dev Containers

编辑器连接远程文件系统时，本地 Windows 不一定存在对应路径。桥接会跳过非
`file:` URI。应用无法读取远程源码，因此纯远程文档不支持。

---

## JetBrains 系列

涵盖 IntelliJ IDEA、Android Studio、PyCharm、WebStorm、GoLand、CLion、Rider、
RustRover、PhpStorm、RubyMine 等。

- **路径：** 向前台 IDE 发送内置的 **Copy Reference** 动作
  （`Ctrl+Alt+Shift+C`），再从剪贴板读取引用。绝对引用直接使用；包相对引用
  （如 `com/example/Foo.java:42`）会结合项目根目录、标准源码根目录
  （`src/main/java`、`src/test/java` 及 Kotlin 对应目录）和多级 Maven/Gradle
  模块解析。
- **范围：** 引用给出起始行；应用在文件中搜索选中文本，并选择离该锚点最近的
  匹配。若磁盘内容已变化，则按选中文本行数估算结束行。
- **无需 IDE 插件。**
- **快捷键：** *Copy Reference* 必须绑定在 `Ctrl+Alt+Shift+C`（默认）。如果
  改过键位，请恢复，或在设置中关闭 **JetBrains Copy Reference** 改用搜索根目录。
- **未保存文件：** Copy Reference 仍会给出路径和行号，但磁盘搜索可能不匹配。
  需要精确范围时请先保存。
- **相对路径：** 项目根目录从 `%APPDATA%\JetBrains\...` 和
  `%APPDATA%\Google\...` 下的 `recentProjects.xml` 读取。新版 Android Studio
  可能不再保留该文件，此时请把项目加入 **搜索根目录**。
- **剪贴板：** 引用动作会暂时替换剪贴板；失败时应用恢复原剪贴板，成功时写入
  最终引用。

Android Studio 的标题通常类似 `ProjectName – What's New`；第一段用作项目名，
真正的文件路径由 Copy Reference 探测提供。

---

## Obsidian

- **路径：** 读取 `%APPDATA%\obsidian\obsidian.json` 中注册的 Vault，并读取
  Vault 的 `.obsidian\workspace.json`，把当前打开的标签页映射到真实文件路径；
  即使标签标题显示的是 frontmatter 别名而不是文件名也能定位。最后才按笔记名
  搜索。
- **范围：** 在 Markdown 源码中搜索选中文本。纯文本和源码模式选区通常准确。
- **渲染 Markdown 限制：** 在阅读视图/实时预览中选中的渲染文字，复制后可能不含
  `**`、链接、列表标记或 HTML。匹配器带有 Markdown 感知兜底，但表格、嵌入内容
  和复杂格式仍可能歧义。
- **多个 Vault：** 使用标题中的 Vault 名区分。
- **兜底：** 注册表丢失或笔记不在已注册 Vault 时，把 Vault 目录加入
  **搜索根目录**。

---

## Typora

- **路径：** 读取 Typora 的 `%APPDATA%\Typora\history.data` 最近文件列表，按
  标题中的文档名匹配。Windows 上该文件通常为十六进制编码的 JSON，应用会同时
  兼容十六进制与明文 JSON。
- **范围：** 在 Markdown 源码中搜索选中文本，同样有渲染 Markdown 限制。Typora
  复制表格时会输出带列对齐空格的 Markdown；应用会忽略表格分隔行并归一化列宽，
  因此整表或部分行选区都能映射回源码行号。Typora 还会在相邻有序列表项之间插入
  空行，匹配时也会忽略这些空白行。
- **兜底：** 在 Markdown 内容差异过大时，把文档目录加入 **搜索根目录**。
- **未保存文档：** 请先保存；未保存的 Typora 缓冲区没有稳定的源码路径可供搜索。

---

## Sublime Text、Notepad++、EmEditor、Neovim GUI、Windows 记事本

v0.1.x 没有这些编辑器的专用元数据适配器。可使用：

1. **窗口标题中的完整路径。** Copy Text Pos 会直接使用。
2. **搜索根目录。** 添加项目/文档目录；应用按标题中的文件名找到文件，再在源码
   中定位选区。

- 路径通常为**推断**，除非标题包含绝对路径。
- 范围由选中文本搜索**推断**。
- Windows 11 记事本是 WinUI 应用，配置搜索根目录后通常可用；经典记事本同理。
- Neovim GUI 常在标题中显示 `file.ts (13) - NVIM`；应用会把括号中的行号作为
  锚点，改善重复文本的匹配。

---

## 不支持的目标

| 目标 | 原因 |
|---|---|
| Windows Terminal、cmd、PowerShell 控制台 | `Ctrl+C` 是 SIGINT，不是复制。应用会跳过原生控制台窗口。 |
| 浏览器页面文字 | DOM 选区没有源文件路径。 |
| Word / Excel / PowerPoint | 不是带源码行号的纯文本文件。 |
| PDF 阅读器 | 没有可编辑的源文件选区语义。 |
| 纯远程/虚拟文档 | 本地 Windows 进程无法读取源文件。 |

## 在本机验证

1. 在编辑器中选中几行代码。
2. 按下配置的快捷键。
3. 粘贴到文本框。
4. 查看托盘通知；必要时查看
   `%LOCALAPPDATA%\CopyTextPos\copy-text-pos.log`。

VS Code 系列可以在 **设置 → 编辑器集成** 中查看是否存在新鲜桥接快照、由哪个
应用/文件写入、距今多久。

## 编写新的适配器

欢迎新增精确集成。桥接适配器只需向
`%LOCALAPPDATA%\CopyTextPos\vscode-bridge.json` 写入如下 JSON：

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

`startLine`/`endLine` 为 1-based 闭区间。只在自己窗口获得焦点时写入，永远不要
包含选中文字。原生应用会校验时间戳、文件扩展名、前台进程家族，以及文件名是否
出现在当前窗口标题中。

纯路径适配器可以放在 `internal/locate` 下；后续的选区匹配、输出模板、剪贴板
安全逻辑都是共用的。
