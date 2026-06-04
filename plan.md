# Plan - Android Clipboard Policy and Topic Settings Split

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `fix/android-clipboard-topic-settings`
- Base: `master` at `f4f99ae merge: 合并剪贴板多 Topic 历史持久化`
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-android-clipboard-topic-settings`
- Current Stage: `Stage 3.1 complete; entering implementation`
- Skill route: `$m-autoflow`; using `$m-docs` for plan, requirements/specs impact, change archive, and lessons routing.

## Stage Records

### Initialization

- `guide.md`: read from `D:/project/MyFlowHub3/guide.md`.
- Participating repo: `MyFlowHub-ClipboardNode` only.
- Main repo path requested by the user under `D:/project/monkeys/repo` did not exist; actual repo is `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`.
- Base/worktree confirmation: created dedicated worktree under `D:/project/MyFlowHub3/worktrees/fix-android-clipboard-topic-settings` on branch `fix/android-clipboard-topic-settings`.
- Main repo status: `master...origin/master`, clean before worktree creation.

### Stage 1 - Requirements Analysis

#### Goal

Fix Android ClipboardNode behavior so Android users can enable foreground automatic clipboard listening and automatic remote apply where the live mobile binding is available, while keeping mobile OS limitations explicit. Also move TopicBus subscription route settings out of the general sync configuration group into a separate settings group.

#### Scope

- Must:
  - Keep the change inside `MyFlowHub-ClipboardNode`.
  - Preserve existing MyFlowHub TopicBus/Auth/SDK protocol semantics.
  - Keep mobile background clipboard restrictions explicit; Android support is foreground/manual/share based, not unrestricted background watching.
  - Enable Android UI capability for `auto_watch` and `auto_apply` when using the Android mobile bridge.
  - Make Android `auto_watch` start a platform clipboard listener that feeds the existing Go mobile runtime.
  - Make Android `auto_apply` and manual pending apply write the accepted remote text to the Android system clipboard.
  - Keep clipboard body text out of logs/status/config except existing scoped history behavior.
  - Put Topic route subscription rows in their own panel instead of mixing them with parent/device/inline/transfer sync fields.
- Optional:
  - Add a small native-to-Go JSON helper if needed to carry remote text across the Android MethodChannel boundary.
- Not doing:
  - Android background service, persistent notification, or unrestricted background clipboard access.
  - iOS live clipboard policy changes.
  - Protocol changes or server-side changes.

#### Use Cases

1. Android user enables sync, turns on automatic listening, copies text while the app is in the foreground, and ClipboardNode publishes it through configured `sync_from_local` topic routes.
2. Android user enables automatic apply and receives a remote text event; ClipboardNode writes the accepted text to the Android system clipboard.
3. Android user keeps automatic apply disabled, receives a pending event, taps apply, and ClipboardNode writes that pending text to the Android system clipboard.
4. User opens settings and sees Topic subscription routes as a distinct group from endpoint, device identity, inline limit, transfer, and local policy settings.

#### Functional Requirements

- Android foreground clipboard listening must be opt-in through existing `auto_watch`.
- Android automatic remote writes must be opt-in through existing `auto_apply`.
- Android manual/share send must continue to work.
- Live mobile native binding absence must still fail explicitly or fall back to preview/stub reporting rather than pretending live sync works.
- Topic rows must keep existing validation: non-empty, unique, bounded routes with `sync_to_local` and `sync_from_local` flags.

#### Non-functional Requirements

- Minimal change surface: use existing runtime/config/TopicBus logic wherever possible.
- Privacy: do not add clipboard body text to status/config/log contracts.
- Reliability: invalid native states should return explicit MethodChannel errors.
- Portability: do not enable Android-only capabilities for iOS.
- Maintainability: keep Android platform clipboard behavior in `app/android`, and shared sync logic in Go runtime/nodemobile.

#### Inputs / Outputs

- Inputs:
  - Android system clipboard text while app is active.
  - Android share intent text.
  - Remote TopicBus text events accepted by the Go runtime.
  - User settings for `auto_watch`, `auto_apply`, and topic routes.
- Outputs:
  - Existing Go runtime publish/apply decisions.
  - Android system clipboard write for remote applied text.
  - UI-safe capability/status/settings updates.
  - Separated Topic subscription settings panel.

#### Edge Cases

- `auto_watch=true` with missing gomobile binding: report stub/native binding error.
- Android clipboard is empty or non-text: report explicit read error for manual read, ignore automatic listener events that have no text.
- Auto listener sees duplicate text: rely on existing runtime hash loop/unchanged suppression.
- Remote apply decision has no text in the serialized decision JSON: native bridge must obtain the applied text through an explicit, bounded, local-only helper instead of logging it.
- iOS uses `MobileEngineBridge` too; Android-only capability must not accidentally mark iOS auto watch/apply as available.

#### Acceptance Criteria

- Android settings switches for automatic listening and automatic apply are enabled in the UI capability model.
- Android auto watch can feed local clipboard changes into `readClipboard`/runtime through a native listener path.
- Android remote apply and manual apply can write accepted remote text to Android system clipboard.
- Topic subscription route editor is rendered in a separate settings group.
- Tests cover Android capability selection, applied-text JSON handling, and settings UI split.

#### Risks

- Android clipboard access is foreground/lifecycle constrained and should not be presented as background sync.
- Go `Decision.Text` is intentionally not serialized; Android needs a narrow local bridge contract without broadening status/log privacy boundaries.
- Flutter widget tests may need stable finders after moving panels.

#### Issue List

- None.

### Stage 2 - Architecture Design

#### Overall Solution

Keep the existing Go runtime and TopicBus behavior. Add a small mobile binding helper that returns the last applied text only after `ApplyEvent`, and have Android Kotlin write that text to the system clipboard. Add an Android `ClipboardManager.OnPrimaryClipChangedListener` while live mobile bridge is started with `auto_watch=true`, feeding text into the existing manual mobile clipboard adapter and `ReadClipboard` path.

The Flutter mobile capability model becomes platform-specific: Android exposes foreground auto watch and auto apply; iOS keeps the existing manual/share capability until an iOS adapter is implemented.

The settings UI is restructured into separate panels:

- `同步配置`: enablement, parent endpoint, device identity, inline limit, transfer fields, save button.
- `Topic 订阅`: topic route editor and route save button.
- `本机策略`: auto watch, auto apply, history retention, and history limit.

#### Alternatives Considered

- Serialize `Decision.Text` for all mobile decisions: rejected because the existing privacy boundary intentionally omits text from generic decision/status JSON.
- Put Android system clipboard writes inside Go `manualClipboard`: rejected because Go code cannot call Android `ClipboardManager`; Kotlin owns the platform clipboard API.
- Add a foreground service now: rejected because the immediate bug is the unavailable foreground settings and platform bridge; background service requires a larger lifecycle/notification design.

#### Module Responsibilities

- `nodemobile`: keep in-memory clipboard adapter; expose a narrow helper to return and clear the last applied text for Android native code.
- `app/android/.../MobileEngineChannel.kt`: manage Android system clipboard read/write/listener and bridge calls.
- `app/lib/core/bridge/mobile_engine_bridge.dart`: expose Android capabilities without enabling iOS capabilities.
- `app/lib/features/shell/clipboard_shell.dart`: split Topic routes into a dedicated settings panel.
- `app/test`: cover capability and UI layout behavior.
- `docs/specs/clipboard-sync.md`: clarify Android foreground listener/apply platform boundary.

#### Data / Call Flow

- Android start:
  1. Flutter sends settings to Kotlin `start`.
  2. Kotlin starts gomobile engine.
  3. If `auto_watch=true`, Kotlin registers a foreground clipboard listener.
- Android local clipboard:
  1. Clipboard listener reads current text.
  2. Kotlin calls `SetClipboardText`.
  3. Kotlin calls `ReadClipboard`.
  4. Existing Go runtime publishes to `sync_from_local` routes.
- Android remote apply:
  1. Go runtime accepts remote text and writes it to the mobile manual adapter.
  2. Kotlin receives `ApplyEvent` or auto-apply decision.
  3. Kotlin calls a narrow helper for the last applied text.
  4. Kotlin writes that text to Android `ClipboardManager`.

#### Interface Drafts

- `nodemobile.TakeLastAppliedText() (string, error)`
- Kotlin `NodeBridge.takeLastAppliedText(): String`
- Kotlin private helpers:
  - `syncWatcher(configJson: String)`
  - `setSystemClipboard(text: String)`
  - `applyDecisionToSystemClipboard(decisionJson: String)`

#### Error Handling and Safety

- Empty Android clipboard reads return explicit errors for manual read.
- Clipboard listener ignores empty/non-text changes and avoids reentrant publish when Kotlin itself just wrote the clipboard.
- `TakeLastAppliedText` errors if no applied text is available, and Kotlin treats that as an apply failure only when the decision action is `remote_applied`.
- Native binding missing continues to report a stub error.

#### Performance and Testing Strategy

- Listener registration is one per started bridge and removed on stop or setting update.
- Native listener does no polling and relies on existing runtime dedupe.
- Tests:
  - Flutter model test for Android-only mobile capability.
  - Flutter widget test for separate Topic subscription panel.
  - Go nodemobile test for last-applied-text helper.
  - Targeted Go tests and Flutter analyze/test.

#### Extensibility Design Points

- Foreground Android listener can later be moved behind a foreground service without changing runtime policy.
- The narrow applied-text helper keeps generic status/activity contracts body-free.
- Topic settings panel can later add labels or per-topic paused state without crowding connection settings.

### Stage 3.1 - Planning

#### Project Goal and Current State

Current Android mobile UI inherits mobile preview capabilities with `automaticWatch=false` and `autoApply=false`, disabling both policy switches. The Android Kotlin channel can read the system clipboard manually and receive share intents, but it has no clipboard listener and no path to write remote applied text to Android `ClipboardManager`. Topic route editing is currently inside the general `同步配置` panel.

#### Docs Governance Routing Decision

Using `$m-docs`:

- Requirements impact: none. Existing requirements already require mobile manual/share flows, local apply controls, and platform limitation respect.
- Specs impact: clarify. Specs should explicitly describe Android foreground listener/apply boundaries.
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons: `docs/lessons/gomobile-mobile-bindings.md`
- New lesson: not planned unless implementation reveals a reusable failure mode beyond the existing gomobile lesson.

#### Executable Task List

- T1 - Android mobile clipboard policy bridge.
- T2 - Flutter capability model and settings UI split.
- T3 - Specs/archive documentation.
- T4 - Validation and code review.

#### Task Details

##### T1 - Android mobile clipboard policy bridge

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-android-clipboard-topic-settings`
- Plan Path: `D:/project/MyFlowHub3/worktrees/fix-android-clipboard-topic-settings/plan.md`
- Goal: Enable Android foreground auto watch and auto apply through the existing gomobile runtime.
- Files / Modules: `nodemobile/nodemobile.go`, `app/android/app/src/main/kotlin/com/yttydcs/myflowhub/clipboardnode/MobileEngineChannel.kt`
- Write Set: listed files and optional Go test.
- Acceptance: Android native channel registers/removes listener based on `auto_watch`, writes applied remote text to system clipboard, and keeps stub errors explicit.
- Test Points: `GOWORK=off go test ./nodemobile -count=1`; Android build if tooling permits.
- Rollback: revert T1 hunks.

##### T2 - Flutter capability model and settings UI split

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-android-clipboard-topic-settings`
- Plan Path: `D:/project/MyFlowHub3/worktrees/fix-android-clipboard-topic-settings/plan.md`
- Goal: Show Android auto watch/apply as available and separate Topic subscriptions into their own settings panel.
- Files / Modules: `app/lib/core/bridge/mobile_engine_bridge.dart`, `app/lib/features/shell/clipboard_shell.dart`, `app/test/widget_test.dart`
- Write Set: listed Dart files.
- Acceptance: Android capability is platform-specific; iOS remains manual/share; Topic route editor is no longer inside the general sync fields.
- Test Points: Flutter analyze/test.
- Rollback: revert T2 hunks.

##### T3 - Specs/archive documentation

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-android-clipboard-topic-settings`
- Plan Path: `D:/project/MyFlowHub3/worktrees/fix-android-clipboard-topic-settings/plan.md`
- Goal: Clarify Android foreground clipboard adapter boundary and archive the workflow.
- Files / Modules: `docs/specs/clipboard-sync.md`, `docs/change/2026-06-04_android-clipboard-topic-settings.md`, optional `docs/change/README.md`
- Write Set: docs only.
- Acceptance: archive records specs impact, tests, rollback, and task mapping.
- Test Points: docs diff and `git diff --check`.
- Rollback: revert T3 docs.

##### T4 - Validation and code review

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-android-clipboard-topic-settings`
- Plan Path: `D:/project/MyFlowHub3/worktrees/fix-android-clipboard-topic-settings/plan.md`
- Goal: Run targeted validation, perform Stage 3.3 review, fix mapped failures, then enter Stage 4 archive.
- Files / Modules: no planned writes except validation-driven fixes mapped to T1/T2/T3.
- Write Set: none unless failures are found.
- Acceptance: targeted Go tests, Flutter analyze/test, and whitespace checks pass or failures are recorded.
- Test Points: commands recorded in archive.
- Rollback: revert validation-driven fixes by task.

#### Dependencies

- Local Go toolchain.
- Local Flutter SDK at `D:/project/MyFlowHub3/.tmp/tools/flutter-sdk-3.41.9/flutter`.
- Android Gradle/Flutter build tooling for APK validation if available.
- No server/proto/sdk repo changes expected.

#### Risks and Notes

- Android clipboard listener is foreground app behavior; background service is intentionally out of scope.
- If gomobile AAR is absent, APK may still build with the explicit stub, but live Android behavior cannot be fully validated on this machine.
- Android Kotlin code must avoid publishing a clipboard change caused by its own remote apply.

#### Parallelism Assessment

No sub-agent is used. Android native bridge, gomobile helper, Dart capability state, and widget tests share one tightly coupled behavior boundary.

#### Issue List

- None.

阻塞：否
进入 3.2

### Stage 3.2 - Implementation

#### Task Mapping

- T1 Android mobile clipboard policy bridge:
  - Added Android foreground `ClipboardManager` listener controlled by `enabled && auto_watch`.
  - Added Android applied-text polling controlled by `enabled && auto_apply`.
  - Added Android system clipboard write after manual pending apply and after remote auto apply.
  - Added gomobile `TakeLastAppliedText` / `takeLastAppliedText` boundary and local-only decision serialization for body-bearing local/applied decisions.
  - Fixed Kotlin reflection to call gomobile-generated lowerCamel method names.
- T2 Flutter capability model and settings UI split:
  - Android capability now exposes foreground auto watch and auto apply.
  - iOS remains manual/share only.
  - Topic route editor moved to a separate `Topic 订阅` settings panel with its own save action.
- T3 Specs/archive documentation:
  - Clarified Android foreground listener/native apply boundaries in `docs/specs/clipboard-sync.md`.
  - Updated gomobile lesson for `javap` method-name verification.
  - Added `docs/change/2026-06-04_android-clipboard-topic-settings.md` and change index entry.
- T4 Validation and code review:
  - Ran Go, Flutter, Android APK, AAR generation, and `javap` validation listed below.

#### Implementation Notes

- Android automatic listening is foreground app behavior only; no background service was added.
- Kotlin owns Android system clipboard reads/writes. Go runtime remains protocol-neutral and continues to use the mobile manual clipboard adapter.
- Generic Go decision/status JSON stays body-free. The gomobile command response uses a mobile-only DTO that carries `Text` only for `local_published` and `remote_applied`.
- Pending receive decisions remain metadata-only.
- The generated AAR is ignored by git and must be regenerated before live Android packaging.

#### Validation During 3.2

- `$env:GOWORK='off'; go test ./nodemobile -count=1`: passed.
- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat analyze` from `app`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat test` from `app`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat build apk --debug` from `app`: passed before AAR generation.
- `.\scripts\build_aar.ps1 -OutFile app/android/app/libs/myflowhub.aar`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat build apk --debug` from `app`: passed with generated AAR present.
- `D:\rj\androidstudio\jbr\bin\javap.exe -classpath classes.jar com.myflowhub.gomobile.nodemobile.Nodemobile`: confirmed `applyEvent`, `setClipboardText`, and `takeLastAppliedText`.

阻塞：否
进入 3.3

### Stage 3.3 - Code Review

#### Review Checklist

- 需求覆盖: 通过. Android 自动监听/自动应用变为可用的前台策略，Topic 订阅已拆为独立配置组。
- 架构合理性: 通过. Android platform API 留在 Kotlin，Go runtime 仍负责共享同步策略。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）: 通过. Android listener 无轮询；auto-apply polling 仅在 `auto_apply` 启用时前台运行，周期 500ms，消费本地内存字段。
- 可读性与一致性: 通过. Kotlin `ClipboardPolicy`、`takeLastAppliedText` 和 Dart `capabilityForPlatform` 命名清晰。
- 可扩展性与配置化: 通过. 后续 foreground service 可复用现有 listener/apply boundary，不改 runtime config。
- 稳定性与安全: 通过. Missing binding 保留 stub error；pending/status/config 不携带正文；listener 抑制本机 remote apply 回环。
- 测试覆盖情况: 通过. Go 覆盖 applied text helper 和 mobile decision text boundary；Flutter 覆盖 Android/iOS capability 和 Topic panel；Android APK/AAR build 通过。
- 子Agent治理与审计: 通过. 未派发子Agent，原因是 Android native/gomobile/Dart capability 强耦合。

#### Review Fixes

- Fixed Kotlin reflection method names after `javap` showed gomobile Java exports are lowerCamel, not PascalCase.
- Added Android warning logs for native listener/shared-text handoff failures without logging clipboard body.
- Updated gomobile lesson with `javap` method verification.

阻塞：否
进入 4

### Stage 4 - Archive

#### Docs Impact

- Requirements impact: none.
- Specs impact: updated.
- Lessons impact: updated.
- Related requirements: `docs/requirements/clipboard-sync.md`.
- Related specs: `docs/specs/clipboard-sync.md`.
- Related lessons: `docs/lessons/gomobile-mobile-bindings.md`.

#### Archive

- Created `docs/change/2026-06-04_android-clipboard-topic-settings.md`.
- Updated `docs/change/README.md`.
- Updated `docs/lessons/gomobile-mobile-bindings.md`.
- Updated `docs/lessons/README.md`.

阻塞：否
等待用户确认是否结束 workflow
