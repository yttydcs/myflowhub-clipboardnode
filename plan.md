# Plan - ClipboardNode MVP

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

- Plan is ready for user confirmation.

阻塞：是
禁止进入 3.2
禁止派发子Agent
