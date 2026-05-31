# Plan - ClipboardNode MVP

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `feat/clipboard-node`
- Base: `master` at `0992111 chore: 初始化剪贴板节点仓库`
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-node/MyFlowHub-ClipboardNode`
- Current Stage: `3.2 - Implementation`

## Stage Records

### Initialization

- `guide.md`: read from `D:/project/MyFlowHub3/guide.md`; all implementation work must stay under `D:/project/MyFlowHub3/worktrees/`.
- Participating repo: `MyFlowHub-ClipboardNode`.
- Module boundary: independent node application; must not modify Proto/Core/SDK/SubProto/Server/Win unless a later plan explicitly adds a cross-repo task.
- Worktree confirmation: active worktree is `D:/project/MyFlowHub3/worktrees/feat-clipboard-node/MyFlowHub-ClipboardNode`.

### Stage 1 - Requirements Analysis

#### Goal

Build a standalone ClipboardNode repository for online text clipboard synchronization between trusted MyFlowHub devices.

#### Scope

- Must: independent node app, Windows first host, plain text sync, TopicBus small-event transport, safe default disabled, bounded payload, loop prevention, no plaintext logs.
- Optional: Android host, manual send action, pairing helpers, future large-content handoff.
- Not doing: images, files, rich text, offline replay, guaranteed delivery, TopicBus protocol changes, clipboard history persistence.

#### Use Cases

- Two enabled nodes on the same topic sync short text clipboard changes.
- Remote clipboard write does not create a feedback loop.
- Oversize local text is rejected and surfaced to the user.
- Reconnect triggers topic resubscription for future events only.

#### Functional Requirements

See [docs/requirements/clipboard-sync.md](docs/requirements/clipboard-sync.md).

#### Non-functional Requirements

Safety, privacy, explicit errors, bounded memory, no protocol changes, platform adapter isolation.

#### Inputs / Outputs

- Inputs: local clipboard text, TopicBus events, runtime config.
- Outputs: TopicBus publish events, local clipboard writes, status/errors without clipboard body.

#### Edge Cases

- Disabled sync, empty topic, invalid JSON, hash mismatch, local-origin event, duplicate event ID, oversize text, clipboard write failure, reconnect.

#### Acceptance Criteria

See [docs/requirements/clipboard-sync.md](docs/requirements/clipboard-sync.md).

#### Risks

TopicBus is best-effort and has no publish ACK or permission control; first phase is limited to trusted online small text sync.

#### Issue List

- None for Stage 1. Conservative MVP defaults are recorded in requirements.

### Stage 2 - Architecture Design

#### Overall Solution

Use TopicBus for `clipboard.text.v1` online small-text events. Keep shared runtime in `core/`, platform adapters in host directories, and future Android/gomobile support as a repository-level extension point.

#### Alternatives Considered

- New subprotocol: rejected for phase 1.
- Stream text profile: deferred for large or ACK-sensitive content.
- Embedding in Win: rejected because this is a standalone platform node.

#### Module Responsibilities

See [docs/specs/clipboard-sync.md](docs/specs/clipboard-sync.md).

#### Data / Call Flow

Startup, local clipboard publish, remote TopicBus apply, and shutdown flows are specified in [docs/specs/clipboard-sync.md](docs/specs/clipboard-sync.md).

#### Interface Drafts

Clipboard adapter, event payload, and runtime config drafts are specified in [docs/specs/clipboard-sync.md](docs/specs/clipboard-sync.md).

#### Error Handling and Safety

Reject invalid state explicitly, keep sync disabled by default, never log or persist clipboard text, and record validation failures.

#### Performance and Testing Strategy

Bound dedupe windows, hash once, cap inline payloads, and cover validation/dedupe/loop suppression with unit tests.

#### Extensibility Design Points

Versioned payload, TopicBus client interface, platform adapter boundary, future Stream/File handoff.

#### Issue List

- None for Stage 2.

### Stage 3.1 - Planning

#### Project Goal and Current State

The repository exists with documentation baselines only. No runtime or host code exists yet. The MVP should implement a testable headless core first, then a minimal Windows host, without touching other repositories.

#### Docs Governance Routing Decision

使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和索引维护。

- Requirements impact: add
- Specs impact: add
- Lessons impact: none
- Stable product truth: `docs/requirements/clipboard-sync.md`
- Stable technical truth: `docs/specs/clipboard-sync.md`
- Active workflow control: root `plan.md`
- Completed workflow archive: future `docs/change/YYYY-MM-DD_clipboard-node-mvp.md`

#### Related Requirements / Specs / Lessons

- Requirements: [docs/requirements/clipboard-sync.md](docs/requirements/clipboard-sync.md)
- Specs: [docs/specs/clipboard-sync.md](docs/specs/clipboard-sync.md)
- Lessons: none currently
- External specs:
  - `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/topicbus.md`
  - `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/stream.md`

#### Executable Task List

- `CLIP-1`: Define core event model and validation.
- `CLIP-2`: Implement runtime orchestration with fake TopicBus and fake clipboard tests.
- `CLIP-3`: Add Windows clipboard adapter and minimal Windows host skeleton.
- `CLIP-4`: Add configuration persistence and safe defaults.
- `CLIP-5`: Add validation, tests, and docs closeout.

#### Task Details

##### CLIP-1 - Core Event Model

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-node/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: implement versioned clipboard text event types, config types, validation, size limit, and hash helpers.
- Files / Modules: `core/clipboard`, `core/runtime`
- Write Set: new Go files under `core/`
- Acceptance: invalid payloads, oversize text, hash mismatch, and unsupported content types are rejected explicitly.
- Test Points: unit tests for validation and hash behavior.
- Rollback: remove new `core/clipboard` and `core/runtime` files for this task.

##### CLIP-2 - Runtime Orchestration

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-node/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: implement enable/disable, local clipboard event publish, remote event apply, dedupe, and loop suppression using interfaces.
- Files / Modules: `core/runtime`
- Write Set: runtime implementation and tests.
- Acceptance: fake clipboard and fake TopicBus tests prove local publish, remote apply, disabled no-op, duplicate drop, local-origin drop, and loop suppression.
- Test Points: `go test ./...` after Go module exists.
- Rollback: revert `core/runtime` implementation files.

##### CLIP-3 - Windows Host Skeleton

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-node/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: add minimal Windows host and clipboard adapter scaffold without full UI polish.
- Files / Modules: `windows/`, `scripts/`
- Write Set: Windows entrypoint, adapter, basic build script if needed.
- Acceptance: Windows package compiles or the blocker is documented with exact missing dependency.
- Test Points: package-level Go tests; build command when dependencies are available.
- Rollback: remove `windows/` and related scripts.

##### CLIP-4 - Config Persistence

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-node/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: persist non-sensitive settings with safe defaults.
- Files / Modules: `core/configstore`
- Write Set: config store implementation and tests.
- Acceptance: defaults are disabled, topic is explicit, max inline bytes defaults to 65536, clipboard text is never persisted.
- Test Points: config load/save tests.
- Rollback: remove `core/configstore`.

##### CLIP-5 - Validation And Closeout Prep

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-node/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: run available tests, update README/docs indexes, and prepare change archive input.
- Files / Modules: docs and test-related files only.
- Write Set: README/docs updates as needed.
- Acceptance: validation results are recorded; no unrelated repository files changed.
- Test Points: `git status`, `go test ./...`, build command if available.
- Rollback: revert docs updates from this task.

#### Dependencies

- Existing MyFlowHub TopicBus protocol.
- Go toolchain.
- Windows clipboard adapter dependency choice must stay minimal and be justified before implementation.

#### Risks and Notes

- TopicBus lacks delivery ACK; UI/status must avoid implying remote apply.
- Platform clipboard APIs may impose threading constraints.
- Android support may require a later dedicated task.
- Do not introduce hard-coded environment-specific values beyond repository-relative defaults and documented config.

#### Parallelism Assessment

- Parallel work is theoretically possible between core runtime and Windows adapter after `CLIP-1`.
- This workflow will stay single-agent for now because the repository is small and file ownership overlaps.

#### Issue List

- User confirmed continuation; Stage 3.2 may proceed.

阻塞：否
进入 3.2

### Stage 3.2 - Implementation

#### Parallelism Assessment

- `CLIP-1` and `CLIP-2` are tightly coupled because runtime orchestration depends on the event model.
- `CLIP-3` depends on the clipboard adapter interface from `CLIP-1`.
- `CLIP-4` can be implemented independently after the runtime config type exists.
- This implementation remains single-agent because the repository is small, the write sets overlap through shared config/runtime types, and host policy has not exposed a dedicated sub-agent dispatch tool in this turn.

#### File-level Change Plan

- `go.mod`: define the standalone Go module with the local MyFlowHub naming convention.
- `core/clipboard`: define narrow clipboard adapter contracts and text event metadata.
- `core/runtime`: implement config defaults, payload validation, event construction, TopicBus client interface, dedupe windows, local publish, remote apply, and status reporting.
- `core/configstore`: implement JSON persistence for non-sensitive config only.
- `windows`: implement a Windows clipboard adapter and polling watcher behind the core adapter interface.
- `cmd/clipboardnode`: add a minimal Windows-first headless host skeleton with explicit disabled-by-default behavior.
- `scripts`: add a narrow validation helper.
- `docs/change`: prepare Stage 4 archive after tests pass.
- `.gitignore`: ignore `.ace-tool/` produced by local indexing attempts (`CLIP-5` repository hygiene).

#### Implementation Result

- `CLIP-1`: completed. Added versioned `clipboard.text.v1` payload, validation, hash helpers, JSON parsing, and event tests.
- `CLIP-2`: completed. Added runtime orchestration with TopicBus and clipboard interfaces, local publish, remote apply, duplicate drop, local-origin drop, loop suppression, config switching, reconnect resubscribe, and fake integration tests.
- `CLIP-3`: completed as skeleton. Added Win32 text clipboard adapter, polling watcher, and headless Windows-first host skeleton. Live SDK TopicBus transport is explicitly not wired yet and fails clearly when `enabled=true`.
- `CLIP-4`: completed. Added JSON config store with disabled defaults and non-sensitive persistence tests.
- `CLIP-5`: completed. Added validation script, README notes, change archive, and `.ace-tool/` ignore rule.

#### Validation

- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `$env:GOWORK='off'; go build -o 'build/clipboardnode.exe' ./cmd/clipboardnode`: passed.
- `git diff --check`: passed.

#### Stage 3.3 - Code Review

- 需求覆盖：通过。MVP text-only、TopicBus best-effort、默认禁用、大小限制、无正文日志/配置、回环抑制均已覆盖。
- 架构合理性：通过。`core` 与平台 `windows` 分离，TopicBus 和剪贴板均通过接口隔离。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过。文本长度不做额外 byte slice 复制，hash 在本地路径复用一次，去重窗口有界。
- 可读性与一致性：通过。命名沿用需求/规格中的 event/config/runtime 术语。
- 可扩展性与配置化：通过。事件版本化，TopicBus/clipboard 接口化，配置默认值集中。
- 稳定性与安全：通过。启用需要 topic，禁用停止 watcher，错误显式返回，状态不含正文。
- 测试覆盖情况：通过。覆盖 payload 校验、hash mismatch、oversize、disabled no-op、local publish、remote apply、duplicate、local origin、loop suppression、配置存储、启用/禁用/重订阅。
- 子Agent治理与审计：通过。未派发子Agent，原因和执行轨迹已记录。

### Stage 4 - Change Archive

使用 `$m-docs` 校验变更归档、requirements/specs 影响和 lessons 需要性。

- Requirements impact: none
- Specs impact: none
- Lessons impact: none
- Related requirements: [docs/requirements/clipboard-sync.md](docs/requirements/clipboard-sync.md)
- Related specs: [docs/specs/clipboard-sync.md](docs/specs/clipboard-sync.md)
- Related lessons: none
- Change archive: [docs/change/2026-05-31_clipboard-node-mvp.md](docs/change/2026-05-31_clipboard-node-mvp.md)
