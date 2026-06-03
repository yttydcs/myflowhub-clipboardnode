# Plan - clipboardnode-startup-lifecycle

## Workflow Information
- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `fix/clipboardnode-startup-lifecycle`
- Base: `master` at `3c0e539`
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-clipboardnode-startup-lifecycle/MyFlowHub-ClipboardNode`
- Current Stage: 4 - Change Archive

## Stage Records

### Initialization
- guide.md: read from `D:/project/MyFlowHub3/guide.md`; worktrees must stay under `D:/project/MyFlowHub3/worktrees`.
- base/worktree confirmation: main repo `master` was clean; dedicated worktree created for this workflow.
- Participating repo: `MyFlowHub-ClipboardNode` only.

### Stage 1 - Requirements Analysis
#### Goal
Make ClipboardNode fail cleanly when TopicBus subscription times out and improve node identification in management UIs without changing MyFlowHub wire protocols.

#### Scope
- Must: clean up the transport/session when startup fails after connection or authentication.
- Must: treat persisted `logged_in` as stale session state on process start so reconnect always performs login and binds the new TCP session.
- Must: surface the startup error without leaving a misleading connected state.
- Must: include a stable display name in auth register/login payloads when the configured device label is available.
- Optional: add focused unit tests for failure cleanup and auth payload display names.
- Not doing: changing TopicBus, Auth, Management, SDK, Server, SubProto, or remote Hub behavior.

#### Use Cases
- A user connects ClipboardNode to a Hub where TopicBus subscribe does not return; the app reports failure and does not appear half-started.
- A user opens MyFlowHub-Win device tree after ClipboardNode connects; if the Hub lists the node, the display name should be meaningful.

#### Functional Requirements
- Startup must connect, authenticate, and subscribe before reporting a started runtime.
- A failed runtime subscription must clear the local transport connection state.
- Auth payloads should carry the user-visible device label as display metadata when protocol fields support it.
- Existing auth snapshot behavior must remain compatible for `device_id`, `node_id`, and `hub_id`, while `logged_in` remains an in-memory session property.

#### Non-functional Requirements
- No clipboard text may be logged or persisted.
- Keep the change local to ClipboardNode.
- Preserve existing public behavior for successful startup.
- Avoid live Hub dependency in tests.

#### Inputs / Outputs
- Inputs: runtime config, `device_label`, parent endpoint, TopicBus subscription result.
- Outputs: UI-safe status, auth payload, transport cleanup on startup failure.

#### Edge Cases
- Missing device label falls back to `clipboardnode`.
- Connect failure should not try to close an unconnected session beyond best effort.
- Runtime construction failure after auth should also close the transport.
- Subscribe timeout remains reported as the original error.

#### Acceptance Criteria
- `Engine.Start` closes `myflowhub.Client` on any failure after `Connect` succeeds.
- Loading an auth snapshot with `logged_in=true` does not skip login on the next process start.
- Closing the client persists `logged_in=false` without dropping the saved node identity.
- `Status().Connected` becomes false after a startup failure caused by runtime subscribe failure.
- Register/login payloads include display name when supported by proto structs.
- Existing targeted Go tests pass.

#### Risks
- The remote Hub may still not respond to TopicBus subscribe; this change makes the local state correct but does not fix remote routing/module deployment.
- If the currently deployed auth server ignores display names, node visibility still depends on management listing online sessions.

### Stage 2 - Architecture Design
#### Overall Solution
Keep the fix inside ClipboardNode's engine and MyFlowHub client wrappers. `Engine.Start` will use a local cleanup guard once `Connect` succeeds and release the guard only after `Runtime.Start` succeeds. Auth snapshots will preserve reusable identity fields but normalize session login state to false when loaded from disk. Auth register/login will populate existing display-name fields, if present in the imported protocol types.

#### Alternatives Considered
- Retry subscribe automatically: deferred because it changes runtime behavior and could mask remote protocol issues.
- Increase subscribe timeout: rejected because the observed problem is no response, not slow response.
- Change Win console tree: rejected because it correctly lists online management children and cannot infer ClipboardNode auth snapshots.

#### Module Responsibilities
- `core/engine`: connection lifecycle orchestration and startup cleanup.
- `core/myflowhub`: auth payload construction and request tests.
- `core/runtime`: unchanged; it already records `transport_failed` and returns explicit subscribe errors.

#### Data / Call Flow
1. `Engine.Start` calls transport `Connect`.
2. A cleanup guard is armed.
3. `EnsureIdentity` reuses saved node identity but logs in again unless the current in-memory session is already logged in.
4. `EnsureIdentity` registers/logs in using device label metadata.
5. Runtime is created and subscribes to TopicBus.
6. On any error before successful runtime start, cleanup guard closes transport and original error is returned.
7. On success, cleanup guard is released and normal runtime consumption begins.

#### Interface Drafts
- No new public interfaces.
- Existing auth payloads use `DisplayName` when present.

#### Error Handling and Safety
- Cleanup errors are best-effort and should not replace the root startup error.
- Existing `recordError` continues to store the root error for UI status.
- No clipboard payload data enters logs or status.

#### Performance and Testing Strategy
- No new polling or repeated network I/O.
- Unit tests use fakes or `net.Pipe`/local SDK hooks rather than live Hub.
- Run `go test ./core/engine ./core/myflowhub ./core/runtime -count=1`.

#### Extensibility Design Points
- Startup cleanup is local and does not constrain future reconnect/backoff behavior.
- Display-name helper can be reused if future auth metadata expands.

### Stage 3.1 - Planning
#### Project Goal and Current State
Current runtime can leave an established transport and logged-in auth snapshot after TopicBus subscription times out, causing misleading UI state. The remote Hub timeout remains a separate external concern.

#### Docs Governance Routing Decision
Using `$m-docs` routing:
- Requirements impact: none
- Specs impact: none
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons: create `docs/lessons/startup-subscribe-timeout-half-connected.md`
- Change archive destination: `docs/change/2026-06-02_clipboardnode-startup-lifecycle.md`

#### Executable Task List
- T1: Clean up transport when `Engine.Start` fails after connection.
- T2: Treat persisted auth login state as stale and include ClipboardNode display name in auth register/login payloads.
- T3: Add focused tests and run targeted validation.
- T4: Archive the workflow and reusable lesson.

#### Task Details
##### T1 - Startup Failure Cleanup
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-clipboardnode-startup-lifecycle/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: ensure failed subscribe/runtime startup does not leave transport connected.
- Files / Modules: `core/engine`
- Write Set: `core/engine/engine.go`, `core/engine/*_test.go`
- Acceptance: subscribe failure returns original error and transport status is disconnected.
- Test Points: fake transport/runtime path or engine test using fake transport.
- Rollback: revert engine cleanup changes and tests.

##### T2 - Auth Session Snapshot and Display Name
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-clipboardnode-startup-lifecycle/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: send configured device label as display name during register/login and never trust persisted `logged_in=true` as proof of a live session.
- Files / Modules: `core/myflowhub`
- Write Set: `core/myflowhub/client.go`, `core/myflowhub/client_test.go`
- Acceptance: payload includes display name, loaded snapshots force `LoggedIn=false`, and `Close` persists `logged_in=false`.
- Test Points: request payload and snapshot persistence unit tests.
- Rollback: revert display-name payload fields and tests.

##### T3 - Validation
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-clipboardnode-startup-lifecycle/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: verify changed behavior and no regressions in targeted packages.
- Files / Modules: tests only
- Write Set: none outside test files.
- Acceptance: targeted Go tests pass.
- Test Points: `go test ./core/engine ./core/myflowhub ./core/runtime -count=1`.
- Rollback: fix failed tests or revert task changes.

##### T4 - Docs Archive
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-clipboardnode-startup-lifecycle/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: create change archive and lesson.
- Files / Modules: `docs/change`, `docs/lessons`
- Write Set: `docs/change/2026-06-02_clipboardnode-startup-lifecycle.md`, `docs/lessons/startup-subscribe-timeout-half-connected.md`, `docs/lessons/README.md`
- Acceptance: change and lesson are discoverable and record requirements/spec impact.
- Test Points: docs content review.
- Rollback: remove archive/lesson if workflow is abandoned.

#### Dependencies
- Remote Hub investigation remains external and is not required for this local lifecycle fix.

#### Risks and Notes
- This fix will make failed startup appear disconnected; users may still need remote Hub logs to resolve subscribe timeout.
- No sub-agents are planned because T1/T2 share startup/auth context and the write set is small.

#### Parallelism Assessment
- Independent Task IDs exist, but T1/T2 are tightly coupled through startup identity flow and tests are small.
- Sub-agents: not used due tight coupling, limited write set, and need for main-agent integration under `$m-autoflow`.

#### Issue List
- None blocking.

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
- T1 completed: `Engine.Start` now closes transport on any failure after a successful connect and before runtime startup is fully established.
- T2 completed: loaded auth snapshots force `LoggedIn=false`, `Close` persists `logged_in=false`, and register/login payloads include `display_name`.
- T3 completed: added tests for startup failure cleanup, display-name payloads, stale loaded login state, and close persistence.
- Sub-agents: not used because write set and context were tightly coupled.

### Stage 3.3 - Code Review
- 需求覆盖: 通过
- 架构合理性: 通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）: 通过
- 可读性与一致性: 通过
- 可扩展性与配置化: 通过
- 稳定性与安全: 通过
- 测试覆盖情况: 通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）: 通过；未使用子Agent

### Stage 4 - Change Archive
- 使用 `$m-docs` 完成归档路由复核。
- Requirements impact: none
- Specs impact: none
- Lessons impact: updated
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons: `docs/lessons/startup-subscribe-timeout-half-connected.md`
- Change archive: `docs/change/2026-06-02_clipboardnode-startup-lifecycle.md`
- Validation:
  - `GOWORK=off go test ./core/engine ./core/myflowhub ./core/runtime -count=1` passed
  - `GOWORK=off go test ./... -count=1` passed
  - `git diff --check` passed
  - 临时运行目录复制 `%APPDATA%\MyFlowHub\ClipboardNode` 的 `config.json` 与 `node_keys.json`，使用修复后的 `build/clipboardnode-bridge.exe` 执行 `connect`，返回 `connected=true`、`logged_in=true`、`node_id=14`、`subscribed=true`
  - 使用当前 MCP `node_id=15` 查询 `myflowhub_management_list_subtree`，Hub 1 返回节点 `8 (NAS MyFlowHub MCP)`、`15 (AI MCP)`、`14 (local-device)`、`1`，确认 ClipboardNode 在线可见
