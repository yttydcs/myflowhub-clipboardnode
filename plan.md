# Plan - ClipboardNode Full-Platform Clipboard Sync

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Active worktree: `D:/project/MyFlowHub3/worktrees/feat-full-platform-clipboard-sync/MyFlowHub-ClipboardNode`
- Branch: `feat/full-platform-clipboard-sync`
- Base branch: `master`
- Base commit: `6a8c582551287a65283e337fe173eee9c1d6749f`
- Current stage: `4 - Change archive complete; awaiting workflow end confirmation`
- Control document: root `plan.md`
- Stage 3.2 entry: confirmed by user request on 2026-06-02: `请按照plan.md 实现`.

## Stage Records

### Initialization

- `$m-autoflow` is active because this workflow requires staged execution, task mapping, review, and change archival.
- `$m-docs` is active for plan routing and requirements/specs/lessons impact checks.
- `guide.md` was read from `D:/project/MyFlowHub3/guide.md`.
  - Worktrees must live under `D:/project/MyFlowHub3/worktrees`.
  - Commit messages should use Chinese after any conventional prefix.
  - PowerShell conda noise is known and unrelated to most command outcomes.
  - MyFlowHub MCP notification is attempted only after code modification, validation, commit, or workflow closeout when connected, logged in, and write-enabled.
- Main repo is not edited. All work must stay in the active worktree.
- `MyFlowHub-SubProto` is a server-side behavior reference for this workflow, not a client SDK dependency.

### Stage 1 - Requirements Analysis

#### Goal

Implement ClipboardNode as a complete engineered MyFlowHub clipboard application across native desktop, mobile, and a browser-policy-aware web mode.

#### Scope

Must:

- Keep ClipboardNode as an independent node application.
- Reuse existing MyFlowHub Auth, TopicBus, Stream/File-facing contracts without changing wire behavior.
- Use TopicBus application events for small inline UTF-8 text.
- Keep sync disabled by default.
- Keep clipboard bodies out of logs, config, status, and default activity history.
- Support real native desktop behavior on Windows, Linux, and macOS.
- Support mobile manual/share/apply behavior on Android and iOS without claiming unrestricted background clipboard watch.
- Support large-content transfer as metadata manifest plus existing transfer references, not TopicBus chunking.
- Provide platform-aware Flutter UI, validation, CI/build scripts, and handoff-ready docs archive.

Optional / platform-constrained:

- Web can be complete only as a browser-policy-aware mode: user-gesture clipboard read/write plus a local bridge or explicit diagnostic fallback. A hosted browser page cannot directly use the native Go MyFlowHub TCP engine or background clipboard watch.
- Tray/autostart/store signing are release follow-ups unless explicitly added.

Not doing:

- No new Clipboard subprotocol.
- No Server, Proto, SDK, SubProto, TopicBus, Stream, or File wire change.
- No application-layer E2EE in this workflow.
- No offline replay, remote apply ACK claim, or persistent clipboard body history by default.

#### Use Cases

- Desktop devices on a trusted private MyFlowHub topology sync small text over the same topic.
- Remote events can be pending until the user applies them when `auto_apply=false`.
- Android/iOS users can send shared/current text manually and apply received text through explicit user action.
- Oversize content produces a clear transfer or rejection state without logging the body.
- Reconnect restores login/subscription for future online events only.

#### Functional Requirements

- Connect validates endpoint, registers or reuses local identity, logs in, then subscribes when enabled.
- Enablement validates non-empty topic and starts only platform-allowed watch/apply paths.
- Local outbound text validates UTF-8, non-empty body, max inline size, hash, and duplicate/loop state.
- Remote inbound events validate topic, event name, payload version, content type, encoding, size, hash, source, duplicate ID, and loop hashes.
- Publish success means local publish only, not remote delivery/apply.
- Mobile must not expose unrestricted background clipboard watch.
- UI must expose endpoint, topic, device label, max inline bytes, auto-watch, auto-apply, manual send/read/apply, pending metadata, transfer state, and safe errors.

#### Non-functional Requirements

- Explicit errors; no silent swallowing on critical state changes.
- Bounded dedupe/pending/history state.
- No unnecessary repeated full-text processing.
- Platform adapters remain behind narrow boundaries.
- Tests cover critical runtime and bridge behavior.

#### Inputs / Outputs

- Inputs: local clipboard text, share/manual text, TopicBus events, runtime config.
- Outputs: TopicBus `clipboard.text.v1`, optional `clipboard.transfer.v1` manifests, local clipboard writes, metadata-only UI activity/status.

#### Edge Cases

- TopicBus subscriptions are in-memory and connection-scoped.
- TopicBus publish has no ACK and no replay.
- `TargetID=0` is broadcast-to-children, not "send to parent".
- Linux clipboard depends on X11/Wayland tooling availability.
- Android/iOS clipboard access is foreground/manual/share constrained.
- Web clipboard APIs require user gestures and cannot host the native engine by themselves.

#### Acceptance Criteria

- Windows/Linux/macOS native desktop targets have live engine integration and platform clipboard adapters.
- Android APK can include a generated gomobile AAR and use the live engine path.
- iOS build can include a generated gomobile XCFramework and use the live engine path; absent binding must be an explicit fallback, not reported as complete.
- Web has either a browser-safe local bridge mode or an explicitly scoped diagnostic mode recorded as a platform limitation.
- Go tests, Flutter analyze/test, native builds available locally, and CI platform builds are recorded.
- Stage 3.3 review passes before archive.

#### Risks

- Full-platform means platform-appropriate behavior, not identical background clipboard automation everywhere.
- iOS XCFramework generation requires macOS/Xcode.
- Android AAR generation requires Android SDK/NDK and gomobile.
- Web first-class sync needs a browser-safe bridge shape.

#### Issue List

- No requirements blocker for plan generation.
- Hosted CI run `26789125424` failed after the first archive pass: Android/iOS `.sh` scripts lacked executable permission when invoked directly, macOS/iOS jobs lacked Go setup before bridge/gomobile builds, and Android gomobile needed explicit SDK package preparation for API 26/NDK.
- Workflow was rolled back from Stage 4 to Stage 3.2 for `QA-1` CI-environment remediation only; no product behavior expansion was added in that pass.
- Hosted CI run `26789687407` passed for Go CLI, Windows, Linux, macOS, Android AAR/APK, iOS simulator XCFramework/app, and Web debug builds.
- User confirmation is required only before ending the workflow, merging, or cleaning up the worktree.

### Stage 2 - Architecture Design

#### Overall Solution

Use a shared Go engine plus platform adapters and a Flutter UI:

1. `core/myflowhub` wraps client-side MyFlowHub SDK/Auth/TopicBus behavior.
2. `core/runtime` owns validation, dedupe, loop suppression, pending events, transfer decisions, and UI-safe state.
3. `core/engine` wires transport, auth, config, clipboard adapter, and runtime lifecycle.
4. Desktop Flutter uses `cmd/clipboardnode-bridge` as a local JSON stdio bridge.
5. Android/iOS use `nodemobile` generated bindings through native platform channels.
6. Web uses preview/diagnostic today; full web requires a local browser-safe bridge task.

#### Alternatives Considered

- New Clipboard subprotocol: rejected.
- Import `MyFlowHub-SubProto` into ClipboardNode: rejected; it is server/handler code.
- Direct Dart MyFlowHub transport: deferred; Go SDK already exists and is reusable.
- Wails-only UI: rejected because mobile must share the app layer.

#### Module Responsibilities

- `core/auth`: local node keys and signing material; no clipboard body persistence.
- `core/myflowhub`: connect/register/login/subscribe/unsubscribe/publish and TopicBus receive.
- `core/runtime`: event validation, sync decisions, pending apply, transfer manifest, status.
- `core/clipboard`: adapter interfaces.
- `platform`: Windows/Linux/macOS adapter selection.
- `bridge`: JSON command/event contract.
- `cmd/clipboardnode`: foreground CLI/manual send host.
- `cmd/clipboardnode-bridge`: desktop Flutter engine bridge.
- `nodemobile`: exported mobile Go engine API.
- `app/android`: Kotlin platform channel, AAR loading, share intent/manual clipboard.
- `app/ios`: Swift platform channel, XCFramework loading, manual/share/apply.
- `app/lib`: Flutter state, controls, settings, and platform-aware UX.

#### Data / Call Flow

- Startup: UI loads settings -> engine starts -> connect -> register/login -> subscribe -> start allowed watcher/manual controls.
- Local publish: adapter/UI text -> runtime validation/hash/dedupe -> TopicBus publish -> metadata-only activity.
- Remote apply: TopicBus message -> runtime validation/dedupe/source checks -> pending or clipboard write -> loop suppression hash.
- Reconnect: transport recovery -> login -> resubscribe -> no replay claim.
- Shutdown: stop watchers -> unsubscribe best-effort -> clear in-memory auth/session -> close transport/adapters.

#### Interface Drafts

- Topic event: `clipboard.text.v1`.
- Transfer event: `clipboard.transfer.v1`.
- Bridge commands: `start`, `stop`, `updateConfig`, `sendText`, `readClipboard`, `applyEvent`, `status`, `clearRecent`.
- Mobile exports: `Start`, `Stop`, `UpdateConfig`, `SendText`, `SetClipboardText`, `ReadClipboard`, `ApplyEvent`, `Status`.

#### Error Handling and Safety

- Invalid config blocks connect/subscribe with explicit error.
- Invalid remote payload is dropped with metadata-only error.
- Clipboard adapter errors do not mark events applied.
- Missing mobile binding reports an explicit stub state.
- Clipboard body must not appear in status, config, logs, tests, or default history.

#### Performance and Testing Strategy

- Hash once per event path.
- Keep bounded dedupe and pending queues.
- Unit test runtime validation and bridge schema.
- Build native artifacts per platform.
- Use CI runners for Linux/macOS/iOS where Windows local validation cannot run them.

#### Extensibility Design Points

- Payload versioning allows future encryption or richer content.
- Transfer manifest remains application payload, not protocol extension.
- Platform adapter boundaries allow native improvements without touching runtime validation.
- Web bridge can be added without changing desktop/mobile bridge contracts.

#### Issue List

- No architecture blocker for plan confirmation.

## Stage 3.1 - Planning

### Docs Governance Routing Decision

使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。

- Canonical stable requirements: `docs/requirements/clipboard-sync.md`
- Canonical stable specs: `docs/specs/clipboard-sync.md`
- Workflow control: root `plan.md`
- Completed workflow archive destination: `docs/change/YYYY-MM-DD_clipboard-full-platform-sync.md`
- Reusable troubleshooting destination if needed: `docs/lessons/*.md`
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact at plan time: `none`; likely candidates after validation are gomobile version pinning and iOS XCFramework integration.

### Related Requirements / Specs / Lessons

- Related requirements:
  - `docs/requirements/clipboard-sync.md`
- Related specs:
  - `docs/specs/clipboard-sync.md`
  - `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/topicbus.md`
  - `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/stream.md`
- Related lessons:
  - `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`
  - `docs/lessons/flutter-windows-sdk-shared-bat-git.md`

### Cross-repo Reuse Decisions

- Reuse `MyFlowHub-SDK/await.Client` and transport patterns for client send/send-and-await/unmatched-frame handling.
- Reuse `MyFlowHub-Proto` TopicBus/Auth DTOs and constants.
- Reuse `MyFlowHub-MetricsNode` patterns for runtime lifecycle, auth snapshot/key persistence, reconnect/resubscribe, and gomobile singleton bridge.
- Use `MyFlowHub-Server/docs/specs/topicbus.md` as the TopicBus behavior truth.
- Use `MyFlowHub-SubProto` only to understand server-side handler behavior: exact topic matching, in-memory subscriptions, no ACK, no replay, no echo to publisher.
- Do not edit reference repos unless Stage 3.1 is reopened with additional worktrees.

### Worktree State Snapshot Before Stage 3.2 Completion

This subsection is retained as the planning-time snapshot. The authoritative current state is recorded in the Stage 3.2 implementation summary, Stage 3.3 review, and Stage 4 archive below.

Current uncommitted implementation already includes:

- `CORE-1`: mostly implemented.
- `CORE-2`: mostly implemented.
- `DESK-1`: implemented for local engine/CLI path; live two-node smoke still pending.
- `BRIDGE-1`: implemented for desktop JSON bridge.
- `DESK-2` / `DESK-3`: command-based Linux/macOS adapters implemented; hosted validation still required.
- `UI-1`: implemented enough for live controls, pending metadata, transfer state, and mobile/manual operations; final UX validation pending.
- `MOB-1`: in progress. `nodemobile`, Android channel, share/manual path, pinned AAR scripts, and CI-required AAR generation exist; Android live validation still pending.
- `MOB-2`: in progress. iOS now has XCFramework build scripts and an optional Swift binding path; macOS CI validation still pending.
- `WEB-1`: in progress. `clipboardnode-bridge` now has an opt-in localhost HTTP/SSE bridge and Flutter Web can use it with explicit dart-defines.
- `TRANSFER-1`: partial. Manifest and oversize decisions exist; no body chunking; transfer remains opaque reference skeleton.
- `QA-1`: pending final validation, Stage 3.3 review, and archive.

Validation already observed in this worktree before this plan refresh:

- `GOWORK=off go test ./... -count=1`: passed.
- `go build ./cmd/clipboardnode`: passed.
- `go build ./cmd/clipboardnode-bridge`: passed.
- `flutter analyze`: passed.
- `flutter test`: passed.
- `flutter build windows --debug`: passed.
- `flutter build apk --debug`: passed before Android AAR became CI-required; rerun required.
- `git diff --check`: passed.

These validations were rerun during Stage 3.2 and Stage 3.3; see the validation evidence below for final local results.

### Executable Task List

- `CORE-1`: Live MyFlowHub SDK/Auth/TopicBus adapter.
- `CORE-2`: Runtime lifecycle, config, reconnect/resubscribe, pending apply, diagnostics.
- `DESK-1`: Windows live engine and clipboard sync path.
- `BRIDGE-1`: Desktop JSON process bridge.
- `UI-1`: Flutter live UI, settings, privacy, pending/apply/transfer states.
- `DESK-2`: Linux clipboard adapter and validation.
- `DESK-3`: macOS clipboard adapter and validation.
- `MOB-1`: Android gomobile AAR integration and manual/share/apply validation.
- `MOB-2`: iOS gomobile XCFramework integration and manual/share/apply validation.
- `WEB-1`: Browser-policy-aware web mode.
- `TRANSFER-1`: Large-content transfer manifest and Stream/File reference skeleton.
- `QA-1`: Cross-platform validation, Stage 3.3 review, docs archive.

### Task Details

Task detail status values below are retained from the executable plan and may describe the state before final Stage 3.2 completion. The authoritative completion mapping is the Stage 3.2 implementation summary below.

#### CORE-1 - Live MyFlowHub SDK/Auth/TopicBus Adapter

- Status: mostly implemented; final review pending.
- Owner: main agent.
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-full-platform-clipboard-sync/MyFlowHub-ClipboardNode`
- Plan path: `plan.md`
- Goal: connect ClipboardNode to MyFlowHub as a real client node.
- Files / modules:
  - `core/auth/**`
  - `core/myflowhub/**`
  - `go.mod`
  - `go.sum`
- Acceptance:
  - Connect/register/login/subscribe/unsubscribe/publish/receive/close are wired through existing SDK/Proto behavior.
  - TopicBus publish targets the authenticated parent Hub, not `TargetID=0`.
  - Clipboard body is not logged.
- Test points:
  - `go test ./... -count=1`
  - auth key validation tests
  - fake transport/TopicBus tests where available
- Rollback:
  - remove `core/auth`, `core/myflowhub`, and restore runtime fake transport wiring.

#### CORE-2 - Runtime Lifecycle, Config, Pending Apply, Diagnostics

- Status: mostly implemented; final review pending.
- Owner: main agent.
- Goal: make shared runtime own safe sync lifecycle and UI-safe state.
- Files / modules:
  - `core/runtime/**`
  - `core/engine/**`
  - `core/configstore/**`
  - `bridge/contract.go`
- Acceptance:
  - `enabled=false` default is preserved.
  - `UpdateConfig` toggles enable/topic/watch/apply safely.
  - Pending remote events are bounded and can be explicitly applied.
  - Status/activity never includes clipboard body.
  - Disconnect clears in-memory login/session/subscription/watch state.
- Test points:
  - disabled no-op
  - invalid endpoint/topic
  - hash mismatch
  - duplicate/local-origin/loop ignore
  - oversize rejection
  - pending apply
  - no body leakage
- Rollback:
  - revert runtime lifecycle extensions while preserving core payload validation.

#### DESK-1 - Windows Live Desktop Path

- Status: implemented; two-node live smoke pending.
- Owner: main agent.
- Goal: make Windows the first fully runnable native desktop target.
- Files / modules:
  - `cmd/clipboardnode/**`
  - `platform/clipboard_windows.go`
  - `windows/**` if adapter changes are required
  - `README.md`
- Acceptance:
  - CLI/manual send and auto-watch can use the live engine.
  - Two local/private Hub-connected Windows instances can publish/receive/apply text and suppress loops.
  - Errors are UI/log safe.
- Test points:
  - Go tests
  - `go build ./cmd/clipboardnode`
  - manual two-node smoke when Hub is available
- Rollback:
  - return Windows CLI to non-live guard/fallback behavior.

#### BRIDGE-1 - Desktop JSON Process Bridge

- Status: implemented; final integration review pending.
- Owner: main agent.
- Goal: let Flutter desktop operate the live Go engine through a narrow process bridge.
- Files / modules:
  - `bridge/**`
  - `cmd/clipboardnode-bridge/**`
  - `app/lib/core/bridge/live_engine_bridge.dart`
  - desktop packaging in `.github/workflows/debug-latest.yml`
- Acceptance:
  - Desktop app launches or finds the bridge helper.
  - Commands have deterministic success/error responses.
  - Engine emits status/activity/decision events without body leakage.
  - Shutdown is deterministic.
- Test points:
  - Go bridge contract tests
  - Flutter tests
  - `go build ./cmd/clipboardnode-bridge`
  - `flutter build windows --debug`
- Rollback:
  - select `PreviewEngineBridge` for desktop and remove bridge helper packaging.

#### UI-1 - Flutter Live UI and Privacy Controls

- Status: mostly implemented; final UX/validation pass pending.
- Owner: main agent.
- Goal: provide platform-aware live controls instead of preview-only UI.
- Files / modules:
  - `app/lib/**`
  - `app/test/**`
  - `bridge/contract.go`
- Acceptance:
  - UI covers endpoint, topic, device label, max inline bytes, auto-watch, auto-apply, manual send/read/apply, pending activity, transfer state, errors, and clear recent.
  - Mobile controls do not imply background watch.
  - Activity is metadata-only.
- Test points:
  - `flutter analyze`
  - `flutter test`
  - manual desktop/mobile smoke
- Rollback:
  - restore preview controller selection while keeping stable DTOs.

#### DESK-2 - Linux Clipboard Adapter and Validation

- Status: implemented in command-adapter form; hosted/manual validation pending.
- Owner: main agent or delegated agent after confirmation.
- Goal: support Linux text read/write/watch with explicit unsupported-state reporting.
- Files / modules:
  - `platform/clipboard_unix.go`
  - `app/linux/**`
  - `.github/workflows/debug-latest.yml`
  - `README.md`
- Acceptance:
  - Detects Wayland/X11 command availability.
  - Uses `wl-copy`/`wl-paste`, `xclip`, or `xsel` without machine-specific paths.
  - Missing tooling returns explicit unsupported errors.
  - Linux Flutter debug build packages the bridge helper.
- Test points:
  - Go tests
  - Linux CI build
  - manual Linux smoke where display server is available
- Rollback:
  - mark Linux live adapter unsupported while keeping Flutter shell buildable.

#### DESK-3 - macOS Clipboard Adapter and Validation

- Status: implemented in command-adapter form; hosted/manual validation pending.
- Owner: main agent or delegated agent after confirmation.
- Goal: support macOS text read/write/watch with explicit errors.
- Files / modules:
  - `platform/clipboard_unix.go`
  - `app/macos/**`
  - `.github/workflows/debug-latest.yml`
  - `README.md`
- Acceptance:
  - Uses `pbpaste`/`pbcopy` safely.
  - Watch loop does not produce avoidable repeated processing.
  - macOS Flutter debug build packages the bridge helper.
- Test points:
  - Go tests
  - macOS CI build
  - manual macOS smoke if available
- Rollback:
  - mark macOS live adapter unsupported while keeping Flutter shell buildable.

#### MOB-1 - Android Gomobile AAR and Manual/Share/Apply Flow

- Status: in progress; must finish before Stage 3.3.
- Owner: main agent or delegated agent after confirmation.
- Goal: make Android a true live mobile target when the generated AAR is packaged.
- Files / modules:
  - `nodemobile/**`
  - `scripts/build_aar.ps1`
  - `scripts/build_aar.sh`
  - `app/android/**`
  - `app/lib/core/bridge/mobile_engine_bridge.dart`
  - `.github/workflows/debug-latest.yml`
- Required remaining work:
  - Pin `golang.org/x/mobile/cmd/gomobile` to the module version used by `go.mod`. Done in `scripts/build_aar.ps1` and `scripts/build_aar.sh`.
  - Verify generated AAR package/class names against `GoNodeBridge.resolveClass`.
  - Make CI artifact/log clearly distinguish "APK built with live AAR" from "APK built with stub". Done by making CI AAR generation required and uploading the AAR artifact.
  - Run or record Android AAR validation.
  - Decide whether `ACTION_SEND` should only preload manual clipboard state or trigger an explicit send action; current safe default is preload/manual send.
- Acceptance:
  - `Start`, `UpdateConfig`, `Stop`, `SendText`, `ReadClipboard`, `ApplyEvent`, `Status`, and share text preloading work through the AAR path.
  - APK can build with generated AAR and use live engine.
  - APK can still build with explicit stub only as fallback, never as proof of full Android completion.
  - No unrestricted background clipboard watch is claimed.
- Test points:
  - `go test ./... -count=1`
  - `scripts/build_aar.*`
  - `flutter build apk --debug`
  - emulator/device smoke if available
- Rollback:
  - remove AAR integration and keep Android as explicit preview/stub.

#### MOB-2 - iOS Gomobile XCFramework and Manual/Share/Apply Flow

- Status: in progress; `AppDelegate.swift` now delegates to a Swift channel that uses `Nodemobile.xcframework` when present and explicit stub fallback when absent.
- Owner: main agent or delegated agent after confirmation.
- Goal: make iOS a true live mobile target when the generated XCFramework is packaged.
- Files / modules:
  - `nodemobile/**`
  - new `scripts/build_ios_xcframework.sh`
  - optional `scripts/build_ios_xcframework.ps1` that fails clearly on non-macOS
  - `app/ios/**`
  - `app/lib/core/bridge/mobile_engine_bridge.dart`
  - `.github/workflows/debug-latest.yml`
  - `.gitignore`
- Required remaining work:
  - Add iOS XCFramework build script using `gomobile bind -target ios`. Done via `scripts/build_ios_xcframework.sh`.
  - Prefer output path `app/ios/Frameworks/Nodemobile.xcframework`. Done.
  - Ignore generated XCFramework artifacts. Done.
  - Add Swift bridge that calls generated symbols when `canImport(Nodemobile)` is true. Done in `app/ios/Runner/MobileEngineChannel.swift`.
  - Preserve explicit stub fallback when the framework is absent. Done.
  - Add Xcode search paths without making absent generated framework break normal stub builds. Initial xcconfig path added; build validation pending.
  - Validate module name and Swift-visible symbol names on macOS.
- Acceptance:
  - iOS simulator/device build can use live gomobile binding when the XCFramework exists.
  - Manual send/share/apply path works through the same `nodemobile` engine API.
  - Without framework, app reports "binding required" honestly.
  - No desktop-equivalent background clipboard watch is claimed.
- Test points:
  - macOS hosted `gomobile bind` / `flutter build ios --simulator --debug --no-codesign`
  - manual simulator/device smoke if available
  - `flutter test`
- Rollback:
  - remove XCFramework integration and keep iOS explicit stub state.

#### WEB-1 - Browser-policy-aware Web Mode

- Status: in progress.
- Owner: main agent.
- Goal: define and implement the strongest safe web behavior without pretending a browser has native clipboard/background TCP access.
- Files / modules:
  - `cmd/clipboardnode-bridge/**` if adding local WebSocket/HTTP bridge mode
  - `bridge/**`
  - `app/lib/core/bridge/**`
  - `app/web/**` if host changes are needed
  - `README.md`
- Preferred approach:
  - Add optional local bridge mode, for example `clipboardnode-bridge --web-listen 127.0.0.1:<port>`. Done.
  - Flutter Web connects only to localhost by explicit user configuration. Done with dart-defines.
  - Browser clipboard read/write remains user-gesture manual. Done by routing through explicit commands only.
  - Hosted web without local bridge remains diagnostic/preview with explicit status. Done.
- Acceptance:
  - Web target no longer silently behaves like native preview when user expects live mode.
  - Browser limitations are surfaced in UI and docs.
  - No insecure remote bridge default is introduced.
- Test points:
  - bridge tests for web transport if added
  - `flutter build web`
  - manual browser smoke with local bridge if implemented
- Rollback:
  - keep web diagnostic-only and document it as outside native full-platform sync.

#### TRANSFER-1 - Transfer Manifest and Existing Transfer Reference Skeleton

- Status: partial; final tests/docs pending.
- Owner: main agent.
- Goal: handle oversize or unsupported content without TopicBus body chunking.
- Files / modules:
  - `core/runtime/**`
  - `bridge/contract.go`
  - `app/lib/**`
  - `README.md`
- Acceptance:
  - Oversize inline text rejects when no transfer provider/reference is configured.
  - `clipboard.transfer.v1` carries only metadata and opaque existing-protocol reference.
  - UI shows transfer status without body leakage.
- Test points:
  - oversize rejection
  - manifest decision
  - no body leakage
- Rollback:
  - disable transfer manifest and retain oversize rejection only.

#### QA-1 - Validation, Code Review, Archive, Release Readiness

- Status: pending.
- Owner: main agent.
- Goal: prove the feature, complete Stage 3.3, and archive through `$m-docs`.
- Files / modules:
  - `.github/workflows/**`
  - `README.md`
  - `docs/change/**`
  - `docs/lessons/**` if reusable lessons emerge
  - `plan.md`
- Acceptance:
  - Stage 3.3 checklist passes.
  - Local and CI validations are recorded.
  - iOS/Android native binding gaps are resolved or explicitly scoped as blockers before completion.
  - Change archive maps every changed file group to task IDs.
  - Lessons are created only for reusable issues worth future lookup.
- Required validation:
  - `$env:GOWORK='off'; go test ./... -count=1`
  - `go build -o build/clipboardnode.exe ./cmd/clipboardnode`
  - `go build -o build/clipboardnode-bridge.exe ./cmd/clipboardnode-bridge`
  - `flutter analyze`
  - `flutter test`
  - `flutter build windows --debug`
  - `flutter build apk --debug`
  - `flutter build web --debug`
  - Android AAR build when Android SDK/NDK/gomobile are available
  - CI Linux/macOS/iOS simulator builds
  - `git diff --check`
  - live two-node MyFlowHub smoke when a Hub is available
- Rollback:
  - disable live bridge by platform selection or revert the feature branch task changes.

### Dependencies

- Local/private MyFlowHub Hub for live smoke tests.
- Go `1.25.x`.
- Flutter SDK selected for this repo.
- Android SDK/NDK/gomobile for AAR.
- macOS/Xcode/gomobile for iOS XCFramework and iOS simulator validation.
- GitHub hosted runners for Linux/macOS/iOS build proof.

### Risks and Notes

- Android/iOS completion is binding-and-validation sensitive; a stub build is not sufficient.
- Web completion needs a local bridge or must stay explicitly diagnostic.
- Generated AAR/XCFramework artifacts should stay ignored unless a release policy decides otherwise.
- Do not change MyFlowHub protocol semantics to force ClipboardNode behavior.
- Do not log clipboard body while debugging mobile bindings.

### Parallelism Assessment

Stage 3.2 parallelism assessment:

- `MOB-1` and `MOB-2` can split after shared `nodemobile` API is stable.
- `WEB-1` can split if it only touches web bridge files and does not alter mobile bridge DTOs.
- `DESK-2` / `DESK-3` validation can run independently on hosted runners.

The main agent must retain integration, conflict resolution, final review, and archive ownership.

No sub-agent dispatch is used in this implementation pass because no reliable sub-agent dispatch tool is exposed in the current host environment, and the remaining tasks converge through shared CI, README, and plan updates.

### Issue List

- Android native live path is implemented with pinned gomobile AAR scripts, CI-required AAR generation, local AAR class verification, and local debug APK build with live AAR.
- iOS native live path is implemented with pinned gomobile XCFramework scripts, Swift optional binding channel, explicit stub fallback, and CI-required macOS simulator binding/build. Local Windows validation cannot prove Swift module symbols; hosted macOS CI remains the authoritative proof.
- Web first-class live sync is implemented as an explicit localhost HTTP/SSE bridge mode plus dart-define opt-in. Hosted Web without the bridge remains diagnostic by explicit scope.

阻塞：否
进入 3.3

## Stage 3.2 - Implementation Summary

Stage 3.2 implementation is complete in the active worktree. All code and documentation changes are mapped to the confirmed task IDs:

- `CORE-1`: added live MyFlowHub SDK/Auth/TopicBus integration in `core/auth/**`, `core/myflowhub/**`, `core/engine/**`, and `go.mod` / `go.sum`.
- `CORE-2`: expanded `core/runtime/**` with safe defaults, config validation, lifecycle toggles, pending apply, reconnect/resubscribe, bounded dedupe, transfer decisions, and metadata-only status.
- `DESK-1`, `DESK-2`, `DESK-3`: added native desktop clipboard adapter paths under `platform/**`, CLI/manual send host updates, and desktop bridge packaging in CI.
- `BRIDGE-1`: added JSON command/event bridge and localhost Web bridge host in `bridge/**` and `cmd/clipboardnode-bridge/**`; command responses now return synchronous success/error status and error events encode explicit `ok:false`.
- `UI-1`: replaced preview-only Flutter flow with platform-aware bridge factory, live/mobile/web bridges, settings, manual send/read/apply controls, pending metadata, transfer status, and safe errors under `app/lib/**`.
- `MOB-1`: added `nodemobile/**`, pinned Android AAR build scripts, Kotlin platform channel, share-intent preloading/manual send path, CI-required AAR build, and Android minSdk 26 alignment for gomobile AAR merge.
- `MOB-2`: added pinned iOS XCFramework scripts, Swift optional binding channel with `canImport(Nodemobile)`, explicit stub fallback when the generated framework is absent, Xcode project wiring, and CI-required macOS simulator binding/build.
- `WEB-1`: added browser-policy-aware localhost bridge mode using `--web-listen` and `--web-token`, loopback-only validation, CORS/token checks, SSE events, and Flutter Web dart-define opt-in.
- `TRANSFER-1`: implemented metadata-only `clipboard.transfer.v1` manifest decisions, oversize rejection when no transfer reference is configured, UI transfer status, and tests that assert manifest/status do not leak clipboard bodies.
- `QA-1`: completed local validation, Stage 3.3 review, docs/change archive, and reusable lessons.

### Stage 3.2 Validation Evidence

Local validation completed on Windows:

- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `go build -o build\clipboardnode.exe .\cmd\clipboardnode`: passed.
- `go build -o build\clipboardnode-bridge.exe .\cmd\clipboardnode-bridge`: passed.
- Android gomobile AAR build to `app/android/app/libs/myflowhub.aar`: passed locally.
- AAR class check: `com/myflowhub/gomobile/nodemobile/Nodemobile.class` exists and matches the Kotlin resolver.
- `flutter analyze`: passed.
- `flutter test`: passed, 5 tests.
- `flutter build web --debug --dart-define=CLIPBOARDNODE_WEB_BRIDGE=http://127.0.0.1:18291 --dart-define=CLIPBOARDNODE_WEB_TOKEN=testtoken`: passed.
- `flutter build windows --debug`: passed.
- `flutter build apk --debug`: passed after the gomobile minSdk 26 alignment.
- `.\scripts\validate.ps1 -FlutterRoot D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter`: passed with Flutter 3.41.9; the script now fails explicitly when Flutter is missing unless `-SkipFlutter` is passed.
- Android default all-ABI AAR build: passed, including `arm64-v8a`, `armeabi-v7a`, `x86_64`, and `x86` native libraries.
- Local two-node MyFlowHub smoke: passed via `.\scripts\smoke_localhub_two_nodes.ps1 -ServerRoot D:\project\MyFlowHub3\repo\MyFlowHub-Server`; node A and node B logged in as node IDs `2` and `3`, both subscribed to the same topic, node A `send_text` returned `local_published`, and node B status changed to `remote_pending` with matching event ID, size `44`, and hash prefix. Smoke used `auto_watch=false` and `auto_apply=false` to avoid implicit system clipboard reads/writes.
- Remote Hub smoke attempt against `47.111.165.7:9000`: reached the Hub and Web bridge health checks passed, but both temporary nodes stopped at `authenticate myflowhub node: pending approval`; current MCP identity also cannot approve because login returns `invalid signature`.
- `git diff --check`: passed with CRLF warnings only.
- `git status --short --ignored`: generated AAR/build/cache directories remain ignored.

Hosted validation record:

- GitHub Actions run `26789125424`: failed on Android AAR, macOS app, and iOS simulator jobs while Go, Windows, Linux, and Web jobs passed.
- Android failure: `./scripts/build_aar.sh: Permission denied`; workflow now invokes the script through `bash`, prepares Android `platforms;android-26`, fixes Android SDK/NDK environment variables, and uses all four Android ABI targets.
- macOS failure: `go: command not found`; workflow now installs Go before building `clipboardnode-bridge`.
- iOS failure: `./scripts/build_ios_xcframework.sh: Permission denied`; workflow now installs Go and invokes the script through `bash`.
- Gomobile scripts now install both pinned `gomobile` and `gobind`, and prepend `$(go env GOPATH)/bin` to `PATH` before invoking the generated tools.
- CI remediation commit `a60ec93` was pushed to `origin/feat/full-platform-clipboard-sync`.
- GitHub Actions run `26789687407` passed on commit `a60ec93`: Go CLI, Windows debug, Linux debug, macOS debug, Android debug with required all-ABI gomobile AAR, iOS simulator debug with required `Nodemobile.xcframework`, and Web debug all succeeded.
- Run URL: `https://github.com/yttydcs/myflowhub-clipboardnode/actions/runs/26789687407`.
- Existing `debug-latest` release artifacts are still from `master` commit `6a8c582551287a65283e337fe173eee9c1d6749f`; feature-branch CI artifacts from run `26789687407` are the hosted evidence for this workflow.

## Stage 3.3 - Code Review

Stage 3.3 review result: passed.

- 需求覆盖: 通过. The implementation keeps ClipboardNode independent, uses existing MyFlowHub Auth/TopicBus contracts, preserves `enabled=false`, avoids protocol wire changes, supports native desktop, manual/share/apply mobile, and Web localhost bridge mode.
- 架构合理性: 通过. Shared runtime/engine logic remains in Go core packages, platform clipboard access stays behind adapters or native channels, and Flutter selects desktop/mobile/web bridges without changing MyFlowHub protocols.
- 性能风险: 通过. Runtime keeps bounded recent/pending state, hashes once per event path, does not add TopicBus chunking, and uses loopback SSE/HTTP only for explicit Web bridge mode.
- 可读性与一致性: 通过. Naming follows existing `core/runtime`, `bridge`, `nodemobile`, and Flutter bridge patterns; comments are limited to non-obvious generated-binding and minSdk decisions.
- 可扩展性与配置化: 通过. Transfer references are configured as opaque provider/ref metadata, Web bridge endpoint/token are dart-defines, and mobile bindings remain optional generated artifacts with explicit fallback.
- 稳定性与安全: 通过. Config validation fails explicitly, Web bridge binds only loopback, token auth is required for browser commands/events, mobile does not claim unrestricted background watch, and status/activity/default history exclude clipboard bodies.
- 测试覆盖情况: 通过. Runtime validation, no-body leakage, transfer manifest, bridge event encoding, Web bridge loopback/command paths, Flutter analyze/tests, native builds, Android AAR/APK, and validation script passed locally. GitHub Actions run `26789687407` passed the hosted Go, Windows, Linux, macOS, Android, iOS, and Web debug build matrix. Remote public Hub smoke remains limited by pending node approval, not by ClipboardNode implementation.
- 子Agent治理与审计: 通过. Parallel work was assessed in Stage 3.2; no sub-agent dispatch was used because the current host did not expose a reliable dispatch tool. Main agent retained integration, review, validation, and archive ownership.

阻塞：否
进入 4

## Stage 4 - Change Archive

使用 `$m-docs` 校验 change/lessons 路由、requirements/specs 影响和索引维护。

- Requirements impact: `none`; implementation follows `docs/requirements/clipboard-sync.md`.
- Specs impact: `none`; implementation follows `docs/specs/clipboard-sync.md` and does not change MyFlowHub protocol wire behavior.
- Lessons impact: `updated`; created reusable lessons for gomobile platform bindings and Web localhost bridge error propagation.
- Change archive: `docs/change/2026-06-02_clipboard-full-platform-sync.md`.
- Related lessons:
  - `docs/lessons/gomobile-mobile-bindings.md`
  - `docs/lessons/web-localhost-bridge-errors.md`
- Indexes updated:
  - `docs/change/README.md`
  - `docs/lessons/README.md`
- Hosted CI evidence:
  - `https://github.com/yttydcs/myflowhub-clipboardnode/actions/runs/26789687407`

Stage 4 archive is complete. Do not merge or clean the worktree until the user explicitly confirms workflow end.
