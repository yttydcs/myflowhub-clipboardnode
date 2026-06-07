# 2026-06-07_android-auto-apply-remote

## 变更背景 / 目标

用户使用 GitHub 构建的 Android APK 与 Windows 桌面端联调时，Windows 端可以自动应用远端文本，但 Android 端在开启同步和自动应用远端文本后没有更新系统剪贴板。此前本地排查曾误判为本地 APK 可能缺少最新 AAR；进一步确认 GitHub debug workflow 会在 APK 前生成 `myflowhub.aar`，因此需要补强代码路径并重新推送触发 `debug-latest`。

## 具体变更内容

- `nodemobile/nodemobile.go`
  - 保留原有 `manualClipboard.WriteText` 到 `lastApplied` 的路径。
  - 增加 mobile-only decision drain 兜底：当 Kotlin 轮询 `TakeLastAppliedText` 而 `manualClipboard` 单槽为空时，从 Go engine decision channel 中非阻塞提取最新 `remote_applied` 文本。
  - 增加 `SetAppliedText`，用于明确表达 remote-applied handoff。
- `nodemobile/nodemobile_test.go`
  - 覆盖 `SetAppliedText`。
  - 覆盖 decision channel drain 时只返回最新 `remote_applied` 文本，并忽略 local/pending decision body。
- `app/android/.../MobileEngineChannel.kt`
  - Kotlin reflection 调用展开 `InvocationTargetException.targetException`。
  - MethodChannel 错误和 auto-apply polling 日志报告真实 gomobile/native cause，而不是只显示 `InvocationTargetException` 外壳。

## Requirements impact

none

## Specs impact

none

## Lessons impact

updated

## Related requirements

- `docs/requirements/clipboard-sync.md`

## Related specs

- `docs/specs/clipboard-sync.md`

## Related lessons

- `docs/lessons/gomobile-mobile-bindings.md`

## 对应 plan.md 任务映射

- `AA-1`: Strengthen mobile applied-text handoff in `nodemobile`.
- `AA-2`: Improve Android native reflection error reporting.
- `AA-3`: Add focused tests and validation.
- `AA-4`: Archive workflow and update reusable lesson cues.
- `AA-5`: Merge, push, and trigger GitHub debug build after user-requested push.

## 经验 / 教训摘要

- GitHub debug workflow 已经在 Android APK 前执行 `scripts/build_aar.sh`，因此 GitHub APK 不生效不能只归因于本地缺 AAR。
- Android remote auto-apply 是 Go runtime 接收并接受远端事件后，通过 mobile manual clipboard handoff 给 Kotlin，再由 Kotlin 写系统剪贴板；这个 handoff 需要可恢复、可观察。
- Kotlin 反射层如果不展开 `InvocationTargetException`，会隐藏真正的 gomobile 异常，导致现场问题难以定位。

## 可复用排查线索

- 症状:
  - Windows 自动应用远端文本正常，Android 开启同步和自动应用后系统剪贴板不变。
  - Android UI 只显示 `InvocationTargetException`，没有具体 gomobile 错误。
- 触发条件:
  - 使用 GitHub `myflowhub-clipboardnode-android-debug.apk`。
  - Android connected/logged in，topic route `sync_to_local=true`，`auto_apply=true`。
- 关键词:
  - `remote_applied`
  - `takeLastAppliedText`
  - `InvocationTargetException`
  - `myflowhub.aar`
  - `gomobile-build.log`
- 快速检查:
  - GitHub debug run 的 Android job 是否成功完成 `Build gomobile AAR`。
  - AAR `javap` 是否包含 lowerCamel `takeLastAppliedText`。
  - Android 端日志是否有真实 gomobile cause，而不是反射外壳。
  - Android topic route 是否开启 `sync_to_local`，并且 `auto_apply=true`。

## 关键设计决策与权衡

- 保留 TopicBus 和 runtime 协议不变，只补 mobile local handoff。
- 不把 clipboard body 写入 status/config/log；decision drain 仍是本地进程内的 mobile-only 文本交接。
- 没有添加 Android foreground service；本次目标是 app 连接存活时的远端自动应用，不承诺后台监听。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./nodemobile -count=1`: passed.
- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `.\scripts\build_aar.ps1 -OutFile app/android/app/libs/myflowhub.aar`: passed.
- `javap -classpath classes.jar com.myflowhub.gomobile.nodemobile.Nodemobile`: confirmed `start`, `applyEvent`, `setClipboardText`, `takeLastAppliedText`.
- Flutter local validation:
  - `flutter analyze` / `flutter test` attempted in parallel first and timed out because both contended for Flutter tool lock.
  - After cleanup, local `flutter --version` still hung on this Windows SDK; local Flutter validation is blocked by tool startup, not by this code path.
  - GitHub CI remains the APK build target and will run Android AAR + APK after push.

## 潜在影响

- Android `TakeLastAppliedText` now drains engine decisions when the manual clipboard slot is empty, so mobile decision channel buildup is reduced.
- In rare cases with multiple remote applies before one poll, the latest applied text wins; this matches current single-slot clipboard handoff behavior.
- Errors shown to Flutter/Android logs should become more specific and may expose previously hidden native causes.

## 回滚方案

- Revert this change commit to remove decision drain and reflection unwrapping.
- If pushed APK regresses, publish a revert commit on `master` to trigger a new `debug-latest` build.

## 子Agent执行轨迹

未派发子Agent。原因：变更集中在 mobile Go binding 和 Android Kotlin bridge，文件所有权小且强耦合，主 Agent 单独完成更可控。
