# 2026-06-02_user-friendly-download-package

## 变更背景 / 目标

用户可以接受手动解压 Windows 包，但不能接受下载后还需要额外下载 helper
或猜测启动哪个文件。当前 `debug-latest` 同时发布 Windows desktop zip、
Web bundle、Go CLI helper 和 Go bridge helper，容易误下载 Web bundle 或直接
启动 helper，导致看到命令行/JSON 而不是图形界面。

本次目标是让 Windows desktop zip 保持自包含，并在解压后提供清晰的 GUI
exe 启动入口。

## 具体变更内容

- `.github/workflows/debug-latest.yml`
  - Windows debug zip 根目录新增 `README-WINDOWS.txt`。
  - 说明 `clipboardnode-bridge.exe` 是 UI 内部使用的 local engine helper，
    不需要单独下载或直接启动。
- `.github/workflows/release.yml`
  - Windows release zip 增加同样的 package readme。
  - 保持 release-mode `flutter build windows --release` 和签名路径不变。
- `app/windows/CMakeLists.txt`, `app/windows/runner/Runner.rc`
  - Windows Flutter runner 输出从 `myflowhub_clipboard.exe` 改为
    `ClipboardNode.exe`。
  - Windows version resource 的 `OriginalFilename` 同步为 `ClipboardNode.exe`。
- `README.md`
  - Debug Preview 和 Version Releases 均增加 Windows quick-start 下载说明。
  - 明确 Web zip 是 hosting/browser bundle，不是 Windows 桌面快速启动包。
  - 明确独立 Go exe 是 CLI/bridge helper，不是桌面 UI。

## Requirements impact

none

本次实现已有要求：ClipboardNode 应作为完整 UI 应用提供，而不是只作为
headless node。没有新增或修改同步、隐私、协议、平台能力要求。

## Specs impact

none

本次不改变 Flutter UI 与 Go bridge 的技术边界，只改变发布包内的启动入口
和说明文件。

## Lessons impact

none

这是发布包体验修正，不新增可复用底层故障模式；排查线索记录在本 change
archive 中即可。

## Related requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Related lessons

- [../lessons/debug-latest-ci-native-exit-flutter-material.md](../lessons/debug-latest-ci-native-exit-flutter-material.md)
- [../lessons/flutter-windows-sdk-shared-bat-git.md](../lessons/flutter-windows-sdk-shared-bat-git.md)

## 对应 plan.md 任务映射

- `PKG-1`: `.github/workflows/debug-latest.yml`, `.github/workflows/release.yml`,
  `app/windows/CMakeLists.txt`, `app/windows/runner/Runner.rc`
- `PKG-2`: `README.md`
- `PKG-3`: `docs/change/2026-06-02_user-friendly-download-package.md`, `docs/change/README.md`

## 经验 / 教训摘要

- Flutter Windows 包是目录型产物，不应让用户从多个 helper exe 中猜启动入口；
  GUI exe 应使用面向用户的名字。
- Windows desktop zip 必须自带 bridge helper；用户不应再单独下载
  `clipboardnode-bridge-windows-amd64.exe` 才能启动 UI。
- Web bundle 和 desktop bundle 要在文档中明确区分，否则用户会下载到
  `index.html`/JS 文件而不是桌面启动包。

## 可复用排查线索

- 症状:
  - 下载后只有 `index.html`、`main.dart.js`、`assets/` 等 Web 文件。
  - 双击后看到 JSON `status.changed` 或命令行窗口，没有 GUI。
- 触发条件:
  - 下载了 `myflowhub-clipboardnode-web-*.zip`。
  - 直接启动了 `clipboardnode-bridge*.exe` 或 `clipboardnode-windows-amd64.exe`。
- 关键词:
  - `ClipboardNode.exe`
  - `clipboardnode-bridge.exe`
  - `myflowhub-clipboardnode-windows-debug.zip`
  - `myflowhub-clipboardnode-web-debug.zip`
- 快速检查:
  - Windows desktop zip 根目录应包含 `ClipboardNode.exe`、
    `flutter_windows.dll`、`data/` 和 `clipboardnode-bridge.exe`。
  - 如果根目录是 `index.html` 和 JS 文件，说明下载的是 Web bundle。

## 关键设计决策与权衡

- 选择重命名 GUI exe 和加入 package readme，而不是新增 bat launcher。这样可以
  满足“手动解压可接受”的要求，并保持启动入口更专业。
- 保留现有 asset names，避免破坏已有 release workflow 和下载脚本。
- 不做单文件 exe：Flutter Windows runner 仍需要 `flutter_windows.dll` 和 `data/`
  目录与 GUI exe 放在一起。真正的单文件体验应通过 installer 或自解压包另行实现。

## 测试与验证方式 / 结果

- Workflow/package static assertions: 通过。
  - `.github/workflows/debug-latest.yml` 和 `.github/workflows/release.yml`
    均包含 `README-WINDOWS.txt`、`ClipboardNode.exe` 启动说明和 bridge
    helper 说明。
- README guidance assertions: 通过。
  - 文档包含 Windows quick-start、无需单独下载 helper、Web bundle 不是桌面包、
    helper 不是桌面 UI 等说明。
- Python/PyYAML 解析 `.github/workflows/debug-latest.yml`: 通过。
- Python/PyYAML 解析 `.github/workflows/release.yml`: 通过。
- `flutter build windows --debug`: 通过。
  - 结果: 生成 `app/build/windows/x64/runner/Debug/ClipboardNode.exe`。
- `git diff --check`: 通过。

未运行 Go 测试：本次没有改 Go runtime、bridge 或协议代码。

## 潜在影响

- Windows zip 增加一个小的 package readme，体积影响可以忽略。
- 用户下载 Windows desktop zip 后可以直接双击 `ClipboardNode.exe`。
- 独立 helper exe 仍作为 CI asset 保留，用于诊断和集成，但文档不再把它们表现为
  普通用户启动入口。

## 回滚方案

- 从 `.github/workflows/debug-latest.yml` 和 `.github/workflows/release.yml`
  删除 package readme 注入逻辑。
- 将 `app/windows/CMakeLists.txt` 和 `app/windows/runner/Runner.rc` 恢复为
  `myflowhub_clipboard.exe`。
- 将 `README.md` release channel 文案恢复到原先的 asset 列表描述。
- 删除本 change archive 和 `docs/change/README.md` 索引项。

## 子Agent执行轨迹

未派发子Agent；主 agent 完成打包脚本、文档、归档与验证。
