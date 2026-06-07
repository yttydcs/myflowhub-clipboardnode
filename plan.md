# Plan - Android Remote Auto Apply

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `fix/android-auto-apply-remote`
- Base: `master` at `cb0bb10 merge: android clipboard automation`
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-android-auto-apply-remote`
- Current Stage: `Stage 4 complete; ready for merge and push`
- Skill route: `$m-autoflow`; using `$m-docs` for plan, requirements/specs impact, change archive, and lessons routing.

## Stage Records

### Initialization

- `guide.md`: not present in this repo.
- Participating repo: `MyFlowHub-ClipboardNode` only.
- Main repo is control-plane only: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`.
- Dedicated worktree created under `D:/project/MyFlowHub3/worktrees/fix-android-auto-apply-remote`.
- Main repo had no tracked changes before worktree creation; ignored build outputs existed and are not part of this workflow.

### Stage 1 - Requirements Analysis

#### Goal

Fix the GitHub-built Android APK path where enabling sync and automatic remote text apply does not update the Android system clipboard, while preserving Android foreground clipboard limitations and existing TopicBus protocol behavior.

#### Scope

- Must:
  - Keep the change inside `MyFlowHub-ClipboardNode`.
  - Preserve existing MyFlowHub TopicBus/Auth/SDK wire semantics.
  - Preserve explicit user enablement before reading/writing clipboard text.
  - Keep Android automatic remote apply working with the generated gomobile AAR used by GitHub debug builds.
  - Surface mobile native/reflection errors clearly instead of returning `InvocationTargetException` shells.
  - Avoid logging or persisting clipboard body text outside the already-scoped body-history behavior.
- Optional:
  - Drain mobile engine decisions if needed to keep Android mobile runtime observable and avoid decision queue buildup.
- Not doing:
  - Android background service or unrestricted background clipboard watching.
  - iOS live behavior changes.
  - TopicBus, server, SDK, or protocol changes.

#### Use Cases

1. User installs `myflowhub-clipboardnode-android-debug.apk` from GitHub `debug-latest`.
2. User connects Android and Windows to the same Hub/topic route.
3. Android has sync enabled, route `sync_to_local=true`, and automatic remote apply enabled.
4. Windows publishes a short text event.
5. Android accepts the remote event and writes the text to the Android system clipboard while the app process is alive.

#### Functional Requirements

- Android auto-apply must not depend on a stale or missing generated binding method without surfacing an error.
- Android mobile runtime should preserve the remote-applied text handoff from Go to Kotlin through a narrow local-only bridge.
- Android native bridge must continue using existing runtime topic filtering, dedupe, and apply policy.
- Reflection failures must report the underlying gomobile exception where possible.

#### Non-functional Requirements

- Minimal change surface.
- No clipboard body text in logs/status/config.
- Explicit failure rather than silent no-op.
- Maintain clear boundary: Go runtime accepts/applies events; Kotlin owns Android system clipboard writes.

#### Inputs / Outputs

- Inputs:
  - Android settings: `enabled`, `auto_apply`, topic route `sync_to_local`.
  - Remote TopicBus `clipboard.text.v1` messages.
  - Generated gomobile AAR exports.
- Outputs:
  - Android system clipboard write for accepted remote inline text.
  - UI-safe MethodChannel errors for native failures.
  - Change archive and lesson updates.

#### Edge Cases

- AAR missing or method mismatch: bridge should report the real method/class error.
- Remote event accepted faster than Kotlin polling: text handoff must remain available until consumed.
- Multiple remote events before a poll: newest remote applied text may overwrite older unapplied text; this matches current single-slot local clipboard semantics.
- Android foreground/background constraints still apply; this patch does not promise background clipboard watching.

#### Acceptance Criteria

- Unit tests cover mobile applied-text handoff and decision drain behavior.
- Android Kotlin reflection unwraps `InvocationTargetException` causes.
- Android build path still generates AAR before APK in GitHub workflow.
- `go test ./nodemobile -count=1` passes.
- `go test ./... -count=1` passes.
- Flutter analyze/test/build checks pass where toolchain permits.

#### Risks

- GitHub-built APK may still require user to download the artifact from the new workflow run, not the previous `debug-latest` asset.
- Android OS may restrict background behavior; foreground/app-alive apply is the supported target.

#### Issue List

- None.

### Stage 2 - Architecture Design

#### Overall Solution

Keep the existing Go runtime and TopicBus contract. Strengthen the mobile-only applied-text handoff by draining Go engine decisions in `nodemobile` and preserving the newest `remote_applied` text in the manual clipboard bridge. Android Kotlin continues polling `takeLastAppliedText()` and writing to the system clipboard. This avoids changing protocol payloads or exposing clipboard body in status.

Improve Android native error reporting by unwrapping reflection `InvocationTargetException` before returning MethodChannel errors. This makes missing/mismatched AAR methods visible in the UI instead of hiding them behind reflection shell errors.

#### Alternatives Considered

- Add a new TopicBus acknowledgment or delivery contract: rejected; out of scope and violates no protocol changes.
- Send clipboard text through status: rejected; violates privacy/status contract.
- Add Android foreground service: deferred; this issue is remote auto-apply while app is connected, not background watching.

#### Module Responsibilities

- `nodemobile`: mobile-only manual clipboard handoff, engine decision drain, exported helper tests.
- `app/android`: Android MethodChannel reflection and system clipboard write.
- `docs/change` and `docs/lessons`: workflow archive and reusable troubleshooting notes.

#### Data / Call Flow

1. Android starts gomobile engine with existing config.
2. Runtime subscribes to configured `sync_to_local` topics.
3. Remote TopicBus event enters `Runtime.HandleTopicBusMessage`.
4. If `auto_apply=true`, runtime writes to `manualClipboard.WriteText`.
5. `nodemobile` also drains engine decisions and preserves remote-applied text as a fallback handoff.
6. Kotlin polling calls `takeLastAppliedText`.
7. Kotlin writes non-empty text to Android `ClipboardManager`.

#### Interface Drafts

- Keep exported `TakeLastAppliedText() string`.
- No new public protocol or MethodChannel command required.

#### Error Handling and Safety

- Unwrap `InvocationTargetException.targetException` in Kotlin.
- Do not include clipboard text in errors/logs.
- Preserve bounded single-slot handoff.

#### Performance and Testing Strategy

- Nonblocking decision drain with bounded channel behavior.
- Focused Go tests for decision drain and applied text handoff.
- Kotlin change validated through build/analyze where available.

#### Extensibility Design Points

- Future Android foreground service can reuse the same mobile handoff.
- Future mobile event UI can add a separate body-safe activity path without changing TopicBus payloads.

### Stage 3.1 - Planning

#### Docs Governance Routing Decision

- Requirements impact: none; existing requirements already require Android/mobile limitations and remote apply policy.
- Specs impact: none; existing specs already describe Android `auto_apply` via Kotlin after Go acceptance.
- Lessons impact: update existing gomobile mobile binding lesson with remote auto-apply handoff / stale AAR lookup cues.
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons: `docs/lessons/gomobile-mobile-bindings.md`

#### Executable Task List

- `AA-1`: Strengthen mobile applied-text handoff in `nodemobile`.
- `AA-2`: Improve Android native reflection error reporting.
- `AA-3`: Add focused tests and validation.
- `AA-4`: Archive workflow and update reusable lesson cues.
- `AA-5`: Merge, push, and trigger GitHub debug build after user-requested push.

#### Task Details

##### AA-1 - Mobile Applied Text Handoff
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-android-auto-apply-remote`
- Plan Path: `plan.md`
- Goal: Keep remote-applied text available to Kotlin after Go auto-apply.
- Files / Modules: `nodemobile/nodemobile.go`, `nodemobile/nodemobile_test.go`
- Write Set: mobile-only Go binding files
- Acceptance: `TakeLastAppliedText` returns text after remote-applied decision drain.
- Test Points: `go test ./nodemobile -count=1`
- Rollback: revert nodemobile changes.

##### AA-2 - Android Error Reporting
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-android-auto-apply-remote`
- Plan Path: `plan.md`
- Goal: Return useful native errors instead of `InvocationTargetException` shells.
- Files / Modules: `app/android/app/src/main/kotlin/com/yttydcs/myflowhub/clipboardnode/MobileEngineChannel.kt`
- Write Set: Android Kotlin bridge
- Acceptance: reflection invoke unwraps target exception message.
- Test Points: Flutter/Android build where available.
- Rollback: revert Kotlin bridge change.

##### AA-3 - Validation
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-android-auto-apply-remote`
- Plan Path: `plan.md`
- Goal: Verify Go and app code paths.
- Files / Modules: tests/build outputs only
- Write Set: none
- Acceptance: required tests pass or blockers are documented.
- Test Points: Go tests, Flutter analyze/test/build as toolchain permits, AAR symbol proof.
- Rollback: none.

##### AA-4 - Archive and Lessons
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-android-auto-apply-remote`
- Plan Path: `plan.md`
- Goal: Record workflow and lookup cues.
- Files / Modules: `docs/change`, `docs/lessons`
- Write Set: docs only
- Acceptance: archive exists and lessons/index updated.
- Test Points: `git diff --check`
- Rollback: revert docs files.

##### AA-5 - Merge and Push
- Owner: main agent
- Worktree: control-plane repo after Stage 4 completion
- Plan Path: `plan.md`
- Goal: Push master to trigger GitHub `debug-latest`.
- Files / Modules: Git history
- Write Set: merge commit only
- Acceptance: `git push origin master` succeeds or proxy fallback succeeds.
- Test Points: final `git status --short --branch`.
- Rollback: create revert commit if pushed change must be undone.

#### Dependencies

- Local Go toolchain.
- Flutter/Android validation may be constrained by local Flutter tool hangs; GitHub CI remains the authoritative APK build trigger.

#### Risks and Notes

- Parallelism assessment: no sub-agent dispatch. The write set is small, cross-file logic is tightly coupled, and no sub-agent tool is required.
- Stage 3.2 may begin because stages 1, 2, and 3.1 are unblocked.

阻塞：否
进入 3.2

### Stage 3.2 - Implementation Summary

- `AA-1`: Completed. `nodemobile.TakeLastAppliedText` now uses the manual clipboard slot first, then drains engine decisions for the latest `remote_applied` text as a mobile-only fallback.
- `AA-2`: Completed. Android Kotlin reflection now unwraps `InvocationTargetException` in MethodChannel errors and polling logs.
- `AA-3`: Completed with local constraints.

#### Validation Results

- `$env:GOWORK='off'; go test ./nodemobile -count=1`: passed.
- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `.\scripts\build_aar.ps1 -OutFile app/android/app/libs/myflowhub.aar`: passed.
- `javap -classpath classes.jar com.myflowhub.gomobile.nodemobile.Nodemobile`: confirmed `start`, `applyEvent`, `setClipboardText`, `takeLastAppliedText`.
- Flutter local validation: attempted, but local Flutter tool hung after lock contention cleanup. GitHub CI will run Android AAR + APK after push.

### Stage 3.3 - Code Review

- 需求覆盖: 通过. Existing Android remote auto-apply requirement is addressed without changing protocol.
- 架构合理性: 通过. Go runtime remains the acceptance point; Kotlin remains Android system clipboard writer.
- 性能风险: 通过. Decision drain is nonblocking and bounded by existing channel contents.
- 可读性与一致性: 通过. Helper names describe mobile handoff behavior.
- 可扩展性与配置化: 通过. No new hard-coded environment values.
- 稳定性与安全: 通过. Clipboard body stays local-only and out of status/config/logs.
- 测试覆盖情况: 通过 with local Flutter limitation documented. Go tests cover the new handoff helper.
- 子Agent治理与审计: 通过. No sub-agents dispatched.

### Stage 4 - Change Archive

- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: none.
- Specs impact: none.
- Lessons impact: updated.
- Change archive: `docs/change/2026-06-07_android-auto-apply-remote.md`.
- Related lessons: `docs/lessons/gomobile-mobile-bindings.md`.
- Workflow end: user already requested code push/build trigger, so proceed to merge/push after final local checks.
