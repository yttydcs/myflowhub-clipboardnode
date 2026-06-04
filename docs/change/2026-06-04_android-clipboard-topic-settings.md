# 2026-06-04 Android Clipboard Topic Settings

## 变更背景 / 目标

用户反馈 Android 平台的“自动监听”和“自动应用”不可用，并要求 Topic 订阅配置不要继续和其他同步配置挤在一起，而是单独作为一组设置。

本轮目标是在不改 MyFlowHub 协议的前提下，让 Android live mobile bridge 支持前台剪贴板监听和远端文本应用到系统剪贴板，并将 Flutter 设置页的 Topic route editor 拆成独立面板。

## 具体变更内容

- Android native bridge:
  - `MobileEngineChannel.kt` 增加 Android `ClipboardManager.OnPrimaryClipChangedListener`。
  - `auto_watch=true` 且同步启用时，前台 listener 读取 Android 系统剪贴板文本，写入 gomobile manual clipboard，再走现有 `ReadClipboard` 发布路径。
  - `auto_apply=true` 时启动前台本地轮询，消费 Go runtime 已应用的远端正文并写入 Android 系统剪贴板。
  - 手动 `applyEvent` 成功后立即写 Android 系统剪贴板。
  - native 写剪贴板后抑制下一次由自身写入触发的 listener 回环。
  - Kotlin reflection method names updated to gomobile-generated lowerCamel names such as `start`, `applyEvent`, and `takeLastAppliedText`.
- gomobile:
  - `manualClipboard` 区分本地输入和 runtime 远端写入。
  - 新增 `TakeLastAppliedText`，只返回并清空 runtime 远端已应用正文。
  - mobile-only decision JSON 只在 `local_published` 和 `remote_applied` 时携带 `Text`，pending 仍保持正文隔离。
- Flutter:
  - `MobileEngineBridge` 改为平台化 capability：Android 开启前台 `automaticWatch` 和 `autoApply`，iOS 仍保持手动/share。
  - Settings 页面将 `Topic 订阅` 独立为单独 panel，保留 route editor、方向开关和独立保存按钮。
- Specs:
  - `docs/specs/clipboard-sync.md` 补充 Android foreground listener、native apply、mobile MethodChannel 文本边界。

## Requirements impact

none

现有需求已规定移动端需要尊重后台限制，并提供 manual/share/local apply 控制。本轮是在该范围内实现 Android 前台能力。

## Specs impact

updated

`docs/specs/clipboard-sync.md` 已澄清 Android `auto_watch` / `auto_apply` 的平台边界，以及 mobile MethodChannel 只在 local-published / remote-applied 决策里携带正文。

## Lessons impact

updated

Updated `docs/lessons/gomobile-mobile-bindings.md` because this workflow found a reusable live-binding pitfall: gomobile Java method names must be verified with `javap` and called by their lowerCamel generated names from Kotlin reflection.

## Related requirements

- `docs/requirements/clipboard-sync.md`

## Related specs

- `docs/specs/clipboard-sync.md`

## Related lessons

- `docs/lessons/gomobile-mobile-bindings.md`

## 对应 plan.md 任务映射

- T1: Android mobile clipboard policy bridge.
- T2: Flutter capability model and settings UI split.
- T3: Specs/archive documentation.
- T4: Validation and code review.

## 经验 / 教训摘要

- Android live clipboard policy belongs in the Kotlin platform channel; Go runtime remains shared and protocol-neutral.
- Go `Decision.Text` remains hidden from generic JSON, so mobile command responses need a narrow local-only DTO when UI history or native clipboard write needs the accepted body.
- Topic route UI can be separated without changing the existing `settings.topics` JSON contract.

## 可复用排查线索

- Symptoms: Android settings disables automatic watch/apply; Android remote apply reports success but system clipboard does not change; Topic routes appear inside general sync settings.
- Trigger conditions: Android uses `MobileEngineBridge`; `PreviewEngineBridge` mobile capability reports `autoApply=false`; gomobile manual clipboard has no Android `ClipboardManager` write path.
- Keywords: `MobileEngineBridge`, `MobileEngineChannel`, `TakeLastAppliedText`, `auto_watch`, `auto_apply`, `ClipboardManager`, `Topic 订阅`, `manualClipboard`.
- Quick checks:
  - Confirm `MobileEngineBridge.capabilityForPlatform(isAndroid: true, ...)` reports `automaticWatch=true` and `autoApply=true`.
  - Confirm `MobileEngineChannel.kt` registers listener only when config has `enabled=true` and `auto_watch=true`.
  - Confirm pending decision JSON does not include `Text`.

## 关键设计决策与权衡

- Did not add a background service. Android clipboard background behavior needs a foreground notification/lifecycle design and is out of scope.
- Kept Android system clipboard writes in Kotlin rather than Go because Android `ClipboardManager` is platform API.
- Used a foreground polling helper for auto-apply because gomobile does not currently push runtime decisions into Kotlin; this keeps the change narrow and bounded.

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./nodemobile -count=1`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat analyze` from `app`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat test` from `app`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat build apk --debug` from `app`: passed before AAR generation, validating Android/Kotlin/Flutter stub-channel compilation.
- `.\scripts\build_aar.ps1 -OutFile app/android/app/libs/myflowhub.aar`: passed. The generated AAR is ignored by git.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat build apk --debug` from `app`: passed with generated `myflowhub.aar` present.
- `D:\rj\androidstudio\jbr\bin\javap.exe -classpath classes.jar com.myflowhub.gomobile.nodemobile.Nodemobile`: confirmed generated methods include `applyEvent`, `setClipboardText`, and `takeLastAppliedText`.

## 潜在影响

- Android `auto_watch` is foreground app behavior only; users should not interpret it as background clipboard monitoring.
- If a stale AAR is packaged without the new gomobile `takeLastAppliedText` export, live Android apply will fail at reflection time until the AAR is regenerated.
- Android auto-apply UI history updates remain bounded by the current mobile bridge event surface; direct manual apply responses can carry body history text, while background applied text is applied to the system clipboard through native polling.

## 回滚方案

- Revert `nodemobile/nodemobile.go` and `nodemobile/nodemobile_test.go` changes for `TakeLastAppliedText` and mobile decision serialization.
- Revert `MobileEngineChannel.kt` listener/polling/system clipboard write changes.
- Revert `MobileEngineBridge` Android capability change.
- Revert `clipboard_shell.dart` Topic panel split and widget test changes.
- Revert `docs/specs/clipboard-sync.md`, this archive, and `docs/change/README.md` index entry.
- Revert `docs/lessons/gomobile-mobile-bindings.md` and `docs/lessons/README.md` lesson updates.

## 子Agent执行轨迹

No sub-agent was dispatched. The Android native bridge, gomobile helper, Dart capability model, and UI settings split share one coupled platform boundary.
