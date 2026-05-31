# Plan - ClipboardNode Cross-platform App

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `feat/clipboard-node`
- Base: `master` at `0992111 chore: 初始化剪贴板节点仓库`
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-node/MyFlowHub-ClipboardNode`
- Current Stage: `3.1 - Planning`

## Stage Records

### Initialization

- `guide.md`: read from `D:/project/MyFlowHub3/guide.md`; all implementation work must stay under `D:/project/MyFlowHub3/worktrees/`.
- Participating repo: `MyFlowHub-ClipboardNode`.
- Existing branch state:
  - `d7906ba docs: 明确剪贴板同步需求与方案`
  - `bf9f5fb feat: 实现剪贴板节点MVP骨架`
- Current clean state before this planning update: worktree clean on `feat/clipboard-node`.
- New user constraints:
  - Need UI.
  - Need cross-platform app.
  - Do not treat this as MVP; build a complete engineered application.
  - No device pairing / room key; MyFlowHub topology is private.
  - Avoid modifying existing subprotocols as much as possible; default architecture must not modify them.

### Stage 1 - Requirements Analysis

#### Goal

Build ClipboardNode into a complete cross-platform MyFlowHub clipboard application for quickly copying content between trusted devices.

#### Scope

- Must:
  - cross-platform UI;
  - shared synchronization engine;
  - desktop automatic clipboard watching where supported;
  - mobile manual/share flows where background clipboard access is restricted;
  - TopicBus application events for small inline text;
  - existing Stream/File capabilities for future large-content transfer manifests;
  - private-network and authenticated-node security model;
  - no new MyFlowHub subprotocol and no existing wire-contract changes by default.
- Optional:
  - tray/menu integration beyond Windows;
  - Android foreground service;
  - iOS share extension;
  - bounded local transfer list;
  - future optional application-layer encryption for untrusted topology.
- Not doing by default:
  - device pairing;
  - room keys;
  - mandatory E2EE;
  - TopicBus/Stream/File/Proto/SDK/Server/SubProto wire changes;
  - offline replay or guaranteed delivery.

#### Use Cases

- Desktop to desktop: automatic text clipboard sync across trusted devices in the same private MyFlowHub topology.
- Mobile to desktop: user uses share/manual send to send text because mobile OS background clipboard access may be limited.
- Desktop to mobile: mobile app receives a TopicBus event and applies or presents content according to local policy.
- Large content: app sends a manifest and uses existing transfer protocol when available.
- Disable sync: app stops reading/writing clipboard and unsubscribes best-effort.

#### Functional Requirements

Stable requirements are in [docs/requirements/clipboard-sync.md](docs/requirements/clipboard-sync.md).

#### Non-functional Requirements

- Safety: disabled by default, explicit enablement, no silent clipboard access.
- Privacy: no plaintext logs; body retention only if user explicitly enables bounded local retention.
- Compatibility: no subprotocol changes by default.
- Portability: platform clipboard behavior behind adapters.
- Maintainability: shared engine remains headless-testable.

#### Inputs / Outputs

- Inputs:
  - platform clipboard text or shared content;
  - TopicBus publish events;
  - runtime and UI config;
  - existing transfer references for future large content.
- Outputs:
  - TopicBus application events;
  - local clipboard writes/apply prompts;
  - UI-safe status/errors;
  - transfer manifests/references.

#### Edge Cases

- Flutter/Dart toolchain unavailable.
- Mobile OS disallows background clipboard monitoring.
- TopicBus publish has no delivery ACK.
- Oversize content lacks available transfer backend.
- User disables sync while unsubscribe fails.
- Remote event is duplicate, local origin, hash mismatch, or unsupported type.

#### Acceptance Criteria

- Requirements explicitly describe a complete cross-platform UI application.
- Requirements explicitly reject pairing/room keys for the private-network default.
- Requirements explicitly forbid MyFlowHub subprotocol changes by default.
- Requirements distinguish desktop automatic sync from mobile manual/share flows.
- Requirements keep existing small-text event behavior compatible with current implementation.

#### Risks

- Flutter/Dart tooling is not installed or not on PATH in the current environment.
- Flutter + Go engine bridge choice affects build complexity across desktop and mobile.
- Existing Stream spec is a draft; large-content transfer may need a later dedicated workflow if production-ready transfer APIs are not available.

#### Issue List

- None for requirements after applying the user's new constraints.

### Stage 2 - Architecture Design

#### Overall Solution

Use a Flutter cross-platform app shell backed by a shared Go engine. Keep protocol use in the Go engine and expose UI-safe state/commands to Flutter through a narrow bridge. The engine reuses existing MyFlowHub SDK/auth/TopicBus contracts; ClipboardNode events remain application-level JSON payloads.

#### Alternatives Considered

- Wails-only UI:
  - Good desktop fit, rejected for full cross-platform target because it does not solve mobile.
- Native per-platform UI:
  - Maximum platform control, rejected as first approach because it duplicates product UI and state.
- Flutter UI shell:
  - Preferred if toolchain is available; one UI across desktop and mobile, with native/platform adapters for clipboard and OS integration.
- Mandatory E2EE:
  - Rejected for first full product under private MyFlowHub topology; optional future module only.
- New Clipboard subprotocol:
  - Rejected. TopicBus application events plus existing transfer protocols cover the target without wire changes.

#### Module Responsibilities

- `engine/` or expanded `core/`:
  - MyFlowHub connect/login orchestration;
  - TopicBus subscribe/publish/live receive adapter;
  - clipboard event validation/dedupe/loop suppression;
  - transfer manifest orchestration;
  - UI-safe status stream.
- `app/`:
  - Flutter UI shell;
  - connection/login screen;
  - device/channel/settings screens;
  - transfer activity view;
  - manual send/apply controls.
- `bridge/`:
  - command/event bridge between Flutter and Go engine;
  - JSON DTOs first to minimize cross-language type churn.
- `platform/<target>/`:
  - clipboard adapters;
  - tray/menu/notifications/autostart for desktop;
  - share-sheet/manual send/lifecycle for mobile.
- Existing `core/runtime`:
  - retained as first engine slice and refactored only when the bridge shape requires it.

#### Data / Call Flow

- Startup:
  1. UI loads persisted app settings.
  2. Engine initializes MyFlowHub SDK client and platform adapter.
  3. User connects/logs in.
  4. If enabled, engine subscribes to the configured TopicBus channel.
  5. Desktop starts watcher if `auto_watch=true`; mobile exposes manual/share send.
- Small text send:
  1. Adapter/UI supplies text.
  2. Engine validates size/type and computes hash.
  3. Engine publishes `clipboard.text.v1` through existing TopicBus.
  4. UI shows local publish status only, not remote apply confirmation.
- Small text receive:
  1. Engine receives TopicBus publish.
  2. Engine validates payload and drops local/duplicate/loop events.
  3. Engine writes clipboard automatically or exposes apply action based on local policy.
- Large content:
  1. Engine rejects inline send when content exceeds limit or type is not small text.
  2. If an existing transfer backend is available, engine publishes `clipboard.transfer.v1` manifest.
  3. UI tracks transfer status and apply/download action.

#### Interface Drafts

```go
type EngineCommand struct {
    Action string          `json:"action"`
    Data   json.RawMessage `json:"data,omitempty"`
}

type EngineEvent struct {
    Name string          `json:"name"`
    Data json.RawMessage `json:"data,omitempty"`
}
```

Initial actions:

- `connect`
- `login`
- `set_config`
- `send_text`
- `apply_event`
- `clear_recent`
- `shutdown`

Initial events:

- `status.changed`
- `transfer.updated`
- `clipboard.received`
- `error`

#### Error Handling and Safety

- Missing Flutter/Dart toolchain blocks UI implementation.
- Missing live TopicBus adapter blocks real sync but not UI shell prototyping.
- Mobile background clipboard restriction is a capability state, not a fatal error.
- UI and status DTOs must never include clipboard text unless user explicitly opens a content preview surface.
- Disabling sync must stop platform watchers before or regardless of unsubscribe success.

#### Performance and Testing Strategy

- Keep engine unit tests headless.
- Add bridge contract tests with JSON golden fixtures.
- Add Flutter widget tests for settings/status/activity screens.
- Add platform adapter tests where possible; use manual smoke tests for OS clipboard behavior.
- Keep dedupe windows bounded and avoid repeated full-text copies.

#### Extensibility Design Points

- Keep app-level event versioning.
- Keep encryption as a future payload wrapper, not a protocol change.
- Keep platform capability map so Android/iOS limitations are represented in UI.
- Keep transfer manifest independent of a single backend.

#### Issue List

- Flutter and Dart are not available on PATH in the current environment:
  - `flutter --version`: command not found.
  - `dart --version`: command not found.
  - This blocks Stage 3.2 implementation of the actual Flutter UI until the toolchain is installed or a different cross-platform UI stack is explicitly chosen.

### Stage 3.1 - Planning

#### Project Goal and Current State

The repository currently contains:

- stable docs for the initial text-sync model;
- Go core runtime for `clipboard.text.v1`;
- configstore;
- Windows clipboard adapter;
- headless command skeleton.

This iteration upgrades the target to a complete cross-platform application and updates the source-of-truth docs. Implementation should not start until the UI toolchain decision is confirmed and the toolchain is available.

#### Docs Governance Routing Decision

使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和索引维护。

- Requirements impact: clarify
- Specs impact: clarify
- Lessons impact: none
- Stable product truth: `docs/requirements/clipboard-sync.md`
- Stable technical truth: `docs/specs/clipboard-sync.md`
- Active workflow control: root `plan.md`
- Completed workflow archive: future `docs/change/YYYY-MM-DD_clipboard-cross-platform-app-plan.md`

#### Related Requirements / Specs / Lessons

- Requirements: [docs/requirements/clipboard-sync.md](docs/requirements/clipboard-sync.md)
- Specs: [docs/specs/clipboard-sync.md](docs/specs/clipboard-sync.md)
- Lessons: none currently
- External specs:
  - `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/topicbus.md`
  - `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/stream.md`

#### Executable Task List

- `APP-1`: Confirm cross-platform UI stack and install/verify toolchain.
- `APP-2`: Create Flutter app shell under `app/` after toolchain availability.
- `APP-3`: Define bridge DTOs and engine command/event API.
- `APP-4`: Wire existing Go runtime to a live MyFlowHub TopicBus adapter.
- `APP-5`: Implement initial UI screens: connection/login, sync status, channel/settings, manual send, recent activity.
- `APP-6`: Implement platform capability map and desktop/mobile clipboard policy split.
- `APP-7`: Add validation, widget tests, bridge tests, docs/change archive, and manual preview steps.

#### Task Details

##### APP-1 - UI Stack And Toolchain Gate

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-node/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: confirm Flutter as the UI stack or choose another cross-platform option; verify toolchain commands.
- Files / Modules: docs and scripts only unless toolchain install is explicitly requested.
- Write Set: `plan.md`, docs, optional toolchain check scripts.
- Acceptance: `flutter --version` and `dart --version` pass, or user explicitly chooses a different stack.
- Test Points: `flutter doctor -v` when Flutter is available.
- Rollback: revert docs/plan updates from this task.

##### APP-2 - Flutter App Shell

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: scaffold production-shaped Flutter app shell.
- Files / Modules: `app/`
- Write Set: `app/`
- Acceptance: app builds/runs for at least Windows target locally; initial navigation and responsive layout exist.
- Test Points: `flutter test`, `flutter run -d windows` or `flutter build windows`.
- Rollback: remove `app/`.

##### APP-3 - Bridge API

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: define stable JSON command/event bridge between Flutter UI and Go engine.
- Files / Modules: `bridge/`, `core/` or `engine/`
- Write Set: bridge and engine DTO files.
- Acceptance: bridge can exchange status/config/send-text commands in tests without live MyFlowHub.
- Test Points: Go unit tests and Flutter bridge tests.
- Rollback: revert bridge files.

##### APP-4 - Live TopicBus Adapter

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: wire existing MyFlowHub SDK/TopicBus publish/subscribe to `core/runtime.TopicBusClient`.
- Files / Modules: `engine/transport` or `core/runtime` adapter package.
- Write Set: adapter implementation and tests.
- Acceptance: fake integration still passes; live adapter compiles without modifying external protocol repos.
- Test Points: Go tests; optional local hub smoke if credentials/environment are available.
- Rollback: remove live adapter and keep fake runtime tests.

##### APP-5 - Initial UI Screens

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: implement usable UI for core workflows.
- Files / Modules: `app/`
- Write Set: Flutter UI files.
- Acceptance: user can see connection/login state, enable sync, configure topic/channel, send current text manually, and inspect recent status.
- Test Points: Flutter widget tests; desktop screenshot/manual preview.
- Rollback: revert UI screen files.

##### APP-6 - Platform Capability Policy

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: represent desktop/mobile clipboard capability differences in engine and UI.
- Files / Modules: `core/`, `app/`, `platform/`
- Write Set: capability DTOs, settings, platform notes.
- Acceptance: mobile limitations are visible and manual/share flow is first-class; desktop auto-watch remains configurable.
- Test Points: unit tests for policy defaults; UI tests for capability-dependent controls.
- Rollback: revert capability files.

##### APP-7 - Validation And Closeout

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: run tests, update docs/change, and prepare workflow closeout.
- Files / Modules: docs and scripts.
- Write Set: `docs/change/`, README, validation scripts.
- Acceptance: validation commands and residual risks are recorded; no unrelated repos changed.
- Test Points: Go tests, Flutter tests/build if available, `git diff --check`, `git status`.
- Rollback: revert closeout docs/scripts.

#### Dependencies

- Flutter/Dart SDK availability for implementation of `app/`.
- Existing MyFlowHub SDK client runtime.
- Existing TopicBus protocol and Go types.
- Future large-content path depends on existing Stream/File readiness; no new protocol work is allowed in this repo by default.

#### Risks and Notes

- Flutter is not currently available on PATH, so implementation is blocked unless the toolchain is installed/configured.
- Cross-platform clipboard behavior cannot be identical; product must expose capability differences honestly.
- Avoid editing `repo/MyFlowHub-*` external repositories unless a later explicit cross-repo plan is created.
- Do not claim E2EE or remote apply confirmation in UI.

#### Parallelism Assessment

- Parallel work is possible after `APP-1`:
  - Flutter UI shell (`APP-2`/`APP-5`) and Go live adapter (`APP-4`) can be split if sub-agent dispatch is available.
  - Bridge API (`APP-3`) gates both sides and should be owned by the main agent initially.
- No sub-agent dispatch is used in this planning turn.

#### Issue List

- Flutter/Dart toolchain missing:
  - `flutter --version`: command not found.
  - `dart --version`: command not found.

阻塞：是
禁止进入 3.2
禁止派发子Agent
