# 2026-06-02_clipboard-full-platform-sync

## 变更背景 / 目标

ClipboardNode 需要从预览 UI 和局部 runtime 走到完整的全平台同步实现：桌面使用真实 Go engine 和剪贴板 adapter，Android/iOS 使用 gomobile 绑定路径，Web 在浏览器限制内通过显式 localhost bridge 接入，同步逻辑继续复用 MyFlowHub 现有 Auth、TopicBus、Stream/File-facing contract，不新增 Clipboard 子协议。

## 具体变更内容

- 新增 live MyFlowHub client/auth/engine 层，支持连接、注册/登录、订阅、发布、接收、重订阅和关闭。
- 扩展 runtime，覆盖安全默认值、配置校验、enable/topic/watch/apply 切换、pending apply、loop suppression、bounded dedupe、oversize transfer manifest、metadata-only status/activity。
- 新增 desktop JSON bridge 和 `clipboardnode-bridge`，并支持 `--web-listen` loopback HTTP/SSE bridge、token auth、CORS、`/health`、`/status`、`/events`、`/command`。
- Flutter UI 接入 platform bridge factory、desktop live bridge、mobile channel bridge、Web localhost bridge和显式 preview fallback，并提供设置、手动发送/读取、待应用、传输状态和隐私安全错误。
- Android 新增 pinned gomobile AAR 构建脚本、Kotlin platform channel、share intent 预载/manual send flow，并让 APK 构建在 live AAR 存在时真实打包绑定。
- iOS 新增 pinned XCFramework 构建脚本、Swift `MobileEngineChannel`、`canImport(Nodemobile)` live path、缺失 framework 时的显式 stub fallback 和 Xcode project wiring。
- CI 扩展为构建 bridge helper、Android AAR、iOS XCFramework，并上传 gomobile 构建日志/产物。
- README 更新全平台构建、Android AAR、iOS XCFramework 和 Web localhost bridge 使用说明。

## Requirements impact

none

本次实现既有 `docs/requirements/clipboard-sync.md`：独立节点应用、全平台 UI、TopicBus 小文本事件、默认禁用、无剪贴板正文日志/配置/默认历史、移动手动/share 流和不修改 MyFlowHub wire contract。

## Specs impact

none

本次实现既有 `docs/specs/clipboard-sync.md` 的模块边界和事件契约。`clipboard.text.v1` / `clipboard.transfer.v1` 仍是 ClipboardNode 应用 payload，不改变 TopicBus、Stream、File、Server、Proto、SDK 或 SubProto 协议。

## Lessons impact

updated

新增两个可复用 lessons：

- [../lessons/gomobile-mobile-bindings.md](../lessons/gomobile-mobile-bindings.md)
- [../lessons/web-localhost-bridge-errors.md](../lessons/web-localhost-bridge-errors.md)

## Related requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)
- `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/topicbus.md`
- `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/stream.md`

## Related lessons

- [../lessons/debug-latest-ci-native-exit-flutter-material.md](../lessons/debug-latest-ci-native-exit-flutter-material.md)
- [../lessons/flutter-windows-sdk-shared-bat-git.md](../lessons/flutter-windows-sdk-shared-bat-git.md)
- [../lessons/gomobile-mobile-bindings.md](../lessons/gomobile-mobile-bindings.md)
- [../lessons/web-localhost-bridge-errors.md](../lessons/web-localhost-bridge-errors.md)

## 对应 plan.md 任务映射

- `CORE-1`: `core/auth/**`, `core/myflowhub/**`, `core/engine/**`, `go.mod`, `go.sum`.
- `CORE-2`: `core/runtime/**`, `core/configstore/**`, `bridge/contract.go`.
- `DESK-1`, `DESK-2`, `DESK-3`: `platform/**`, `cmd/clipboardnode/**`, desktop CI packaging.
- `BRIDGE-1`: `bridge/**`, `cmd/clipboardnode-bridge/**`, `app/lib/core/bridge/live_engine_bridge.dart`.
- `UI-1`: `app/lib/**`, `app/test/**`.
- `MOB-1`: `nodemobile/**`, `scripts/build_aar.*`, `app/android/**`, mobile Flutter bridge.
- `MOB-2`: `scripts/build_ios_xcframework.*`, `app/ios/**`, `nodemobile/**`, mobile Flutter bridge.
- `WEB-1`: `cmd/clipboardnode-bridge/**`, `app/lib/core/bridge/web_engine_bridge.dart`, README Web bridge docs.
- `TRANSFER-1`: `core/runtime/**`, `bridge/contract.go`, `app/lib/**`, README transfer notes.
- `QA-1`: `plan.md`, `docs/change/**`, `docs/lessons/**`, CI workflow, validation evidence.

## 经验 / 教训摘要

- gomobile generated artifacts should stay ignored, but CI must build them before claiming live Android/iOS completion.
- Android gomobile AAR declares minSdk 26; the app minSdk must align when the live AAR is packaged.
- iOS XCFramework proof is macOS/Xcode-only; Windows can validate scripts and Flutter shared code, but cannot prove Swift module symbols.
- A Web build cannot silently imply native engine access. Live Web mode needs explicit localhost bridge configuration and browser-safe manual clipboard behavior.
- Bridge error contracts need synchronous command result fields and explicit `ok:false`; omitting false booleans can make Dart treat failures as accepted commands.
- Hosted CI needs Go setup on macOS/iOS jobs before building the bridge or gomobile bindings; `.sh` scripts should be invoked through `bash` unless executable bits are guaranteed in Git.
- Android gomobile `-androidapi 26` needs `platforms;android-26`, NDK, and `ANDROID_HOME`/`ANDROID_NDK_HOME` prepared on the runner.

## 可复用排查线索

- 症状: Android APK builds in stub mode, but live engine calls fail at runtime.
- 触发条件: `app/android/app/libs/myflowhub.aar` missing, gomobile version drift, wrong generated Java package, or app minSdk lower than AAR minSdk.
- 关键词: `Nodemobile.class`, `com.myflowhub.gomobile.nodemobile.Nodemobile`, `minSdk 26`, `scripts/build_aar`.
- 快速检查: build AAR first, inspect class names with `jar tf` or a ZIP reader, then run `flutter build apk --debug`.

- 症状: Web UI accepts a command but status shows no real action or no error.
- 触发条件: `/command` returns only accepted/202, SSE error omits `ok:false`, or bridge is not loopback/token-authenticated.
- 关键词: `CLIPBOARDNODE_WEB_BRIDGE`, `CLIPBOARDNODE_WEB_TOKEN`, `--web-listen`, `ok:false`, `/events`, `/command`.
- 快速检查: call `/health`, verify token header, inspect `/command` JSON for `ok` and `error`, and check SSE event payloads.

## 关键设计决策与权衡

- 保留 Go runtime/engine 为同步核心，Flutter 只负责 UI 和 platform-aware bridge selection，避免把 MyFlowHub protocol 和平台剪贴板逻辑散落到 UI。
- 桌面使用 process bridge 而不是直接 Dart SDK/FFI，减少跨语言类型 churn，并方便 Web localhost bridge 复用同一 command/event contract。
- Android/iOS 使用 generated gomobile binding；生成物不入库，但 CI 和脚本提供可重复构建证据。
- iOS 缺少 `Nodemobile.xcframework` 时明确报告 binding required，不把 stub build 当成完整 iOS live completion。
- Web 只允许显式 loopback bridge，不引入远程 bridge 默认值，避免浏览器页面直接暴露本机 engine 控制面。
- Oversize 内容不做 TopicBus chunking，只发布 metadata manifest 或明确 rejection。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `go build -o build\clipboardnode.exe .\cmd\clipboardnode`: passed.
- `go build -o build\clipboardnode-bridge.exe .\cmd\clipboardnode-bridge`: passed.
- Android gomobile AAR build: passed locally, generated `app/android/app/libs/myflowhub.aar`.
- AAR class verification: passed, `com/myflowhub/gomobile/nodemobile/Nodemobile.class` exists.
- `flutter analyze`: passed.
- `flutter test`: passed, 5 tests.
- `flutter build web --debug --dart-define=CLIPBOARDNODE_WEB_BRIDGE=http://127.0.0.1:18291 --dart-define=CLIPBOARDNODE_WEB_TOKEN=testtoken`: passed.
- `flutter build windows --debug`: passed.
- `flutter build apk --debug`: passed after minSdk 26 alignment.
- `.\scripts\validate.ps1 -FlutterRoot D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter`: passed with Flutter 3.41.9; the script now fails explicitly when Flutter is missing unless `-SkipFlutter` is passed.
- Android default all-ABI AAR build: passed, including `arm64-v8a`, `armeabi-v7a`, `x86_64`, and `x86` native libraries.
- Local two-node MyFlowHub smoke: passed via `.\scripts\smoke_localhub_two_nodes.ps1 -ServerRoot D:\project\MyFlowHub3\repo\MyFlowHub-Server`; node A and node B logged in as node IDs `2` and `3`, both subscribed to the same topic, node A `send_text` returned `local_published`, and node B status changed to `remote_pending` with matching event ID, size `44`, and hash prefix. Smoke used `auto_watch=false` and `auto_apply=false` to avoid implicit system clipboard reads/writes.
- Remote Hub smoke attempt against `47.111.165.7:9000`: reached the Hub and Web bridge health checks passed, but both temporary nodes stopped at `authenticate myflowhub node: pending approval`; current MCP identity also cannot approve because login returns `invalid signature`.
- `git diff --check`: passed with CRLF warnings only.
- GitHub Actions run `26789125424`: failed initially on Android AAR, macOS app, and iOS simulator jobs; Go CLI, Windows, Linux, and Web jobs passed. Failures were CI-environment issues: direct `.sh` execution without executable bits, missing Go setup on macOS/iOS, and incomplete Android SDK package preparation.
- CI remediation commit `a60ec93`: added Go setup for macOS/iOS/Android hosted jobs, invoked shell scripts through `bash`, prepared Android `platforms;android-26` and NDK, installed pinned `gomobile` plus `gobind`, and ensured Go bin is on `PATH`.
- GitHub Actions run `26789687407`: passed on commit `a60ec93`.
  - Run: `https://github.com/yttydcs/myflowhub-clipboardnode/actions/runs/26789687407`
  - Passed jobs: Go CLI, Windows debug, Linux debug, macOS debug, Android debug with required all-ABI gomobile AAR/APK, iOS simulator debug with required `Nodemobile.xcframework`, and Web debug.
  - `Publish debug-latest` was skipped because the run was on `feat/full-platform-clipboard-sync`, not `master`.

Not proven:

- Remote public Hub end-to-end smoke is not proven because temporary nodes against `47.111.165.7:9000` remain pending approval and the current MCP identity cannot approve them. Local two-node Hub smoke passed and proves the ClipboardNode publish/receive/pending path.

## 潜在影响

- Android app minimum SDK is now effectively 26 when live gomobile AAR is included.
- CI runtime increases because Android/iOS gomobile binding builds are required for debug validation.
- Web live mode now depends on a separate local bridge process and token configuration.
- iOS live completion depends on macOS CI proof for generated module names and Swift-visible symbols.

## 回滚方案

- Disable platform bridge selection back to preview/stub bridge for affected targets.
- Revert `cmd/clipboardnode-bridge` Web HTTP/SSE mode and keep Web diagnostic-only.
- Remove gomobile build steps from CI and keep Android/iOS explicit stub state.
- Revert `core/myflowhub`, `core/engine`, and runtime live lifecycle changes if protocol-facing integration needs to be paused.
- Generated artifacts are ignored and can be deleted without source rollback.

## 子Agent执行轨迹

Stage 3.2 and 3.3 parallelism was assessed. No sub-agent was dispatched because the current host environment did not expose a reliable sub-agent dispatch tool. The main agent retained implementation integration, conflict resolution, validation, review, and archive ownership.
