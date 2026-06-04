# Plan - Clipboard multi-topic history controls

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `feat/clipboard-multitopic-history`
- Base: `master` at `38d206c fix(ui): polish clipboard history settings`
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history`
- Current Stage: `Stage 4 archive refreshed; awaiting workflow end confirmation`
- Skill route: `$m-autoflow`, with `$m-docs` for plan, requirements, specs, and archive routing.

## Stage Records

### Initialization

- `guide.md`: read from `D:/project/MyFlowHub3/guide.md`.
- Participating repo: `MyFlowHub-ClipboardNode` only.
- Base/worktree confirmation: created dedicated worktree under `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history` on branch `feat/clipboard-multitopic-history`.
- Main repo status: control-plane workspace has unrelated dirty files; implementation stays inside this worktree.

### Stage 1 - Requirements Analysis

#### Goal

Change ClipboardNode so history is a local clipboard text body list, transfer/send/receive activity is shown in logs, clicking a history item restores that text to the local clipboard and promotes it to the top, and users can subscribe to multiple TopicBus topics with per-topic direction controls. The default topic is `clipboard.text`.

#### Scope

- Must:
  - Keep ClipboardNode as a single-repo change with no MyFlowHub protocol changes.
  - Add multi-topic subscription settings with `sync_to_local` and `sync_from_local` direction flags per topic.
  - Preserve legacy `topic` compatibility while defaulting new config/UI to `clipboard.text`.
  - Subscribe to every configured topic when sync is enabled.
  - Publish local clipboard/manual text only to topics whose `sync_from_local` is enabled.
  - Apply or queue remote text only from topics whose `sync_to_local` is enabled.
  - Keep activity/log views metadata-only and move send/receive/pending status there.
  - Make body history display clipboard text entries without send/receive labeling.
  - Restore a clicked history entry to the local clipboard and promote that entry to the top.
- Optional:
  - Add richer per-topic status counts if it falls out naturally.
- Not doing:
  - New MyFlowHub subprotocols or server-side changes.
  - Persisting clipboard text bodies.
  - Delivery acknowledgements or offline replay.

#### Use Cases

1. User keeps the default `clipboard.text` topic with both directions enabled.
2. User subscribes to several trusted topics but disables remote-to-local apply for one topic.
3. User publishes local clipboard changes only to selected subscribed topics.
4. User opens history, clicks an old text body, and the app restores it to the local system clipboard with that body at the top of history.
5. User reviews send/receive/pending/error metadata in logs without clipboard body leakage.

#### Functional Requirements

- Config must normalize, validate, and expose a bounded non-empty topic list.
- Topic names must be trimmed and duplicate topics must fail explicitly.
- Enabling sync requires at least one valid topic.
- A topic route may independently enable remote-to-local and local-to-topic sync.
- Runtime must subscribe/unsubscribe/resubscribe all configured topics.
- Runtime must ignore remote messages from unknown topics or from routes with remote-to-local disabled.
- Runtime local publish and transfer manifest publish must fan out to local-to-topic enabled routes.
- UI must allow adding/removing topic rows and toggling both directions.
- UI history must be body-oriented and clickable for restore.

#### Non-functional Requirements

- Clipboard body text remains in memory only.
- Logs/status/config must not persist or expose full body text.
- Keep config and bridge contract backward compatible with existing `topic`.
- Avoid repeated text hashing and avoid unbounded history or topic growth.
- Keep module boundaries: runtime owns sync policy; Flutter owns UI history presentation/restore affordance.

#### Inputs / Outputs

- Inputs:
  - Existing settings plus `topics`.
  - Local clipboard/manual text.
  - TopicBus messages from any subscribed topic.
  - User clicks on body history entries.
- Outputs:
  - TopicBus publishes to selected local-to-topic routes.
  - Local clipboard write on accepted remote apply or history restore.
  - Metadata-only activity/log entries.
  - In-memory bounded body history.

#### Edge Cases

- Empty topic row while enabled: explicit validation error.
- Duplicate topic names: explicit validation error.
- No local-to-topic route: local text change is ignored without publishing.
- Remote message on subscribed but local-disabled route: ignored and logged as metadata.
- Restore fails because platform clipboard write is unavailable: show explicit UI error.
- Legacy config with only `topic`: normalize to one route with both directions enabled.

#### Acceptance Criteria

- Default settings show `clipboard.text`.
- Multiple topic rows can be configured and serialized through Flutter -> bridge -> runtime.
- Runtime tests prove multi-topic subscribe, publish fan-out, remote apply gating, and duplicate validation.
- Widget/state tests prove body history is text-focused and restore promotes clicked text.
- Logs contain send/receive/pending/error metadata while history contains text bodies.

#### Risks

- Cross-language JSON contract drift between Go bridge and Flutter settings/status parsing.
- UI complexity from dynamic topic row controllers.
- Restore through Flutter clipboard may behave differently per platform; failures must surface.

#### Issue List

- None.

### Stage 2 - Architecture Design

#### Overall Solution

Add a compatible topic route model to runtime config and bridge contracts:

```text
TopicRoute {
  topic: string
  sync_to_local: bool
  sync_from_local: bool
}
```

The existing `topic` field remains as the primary/default topic and legacy compatibility alias. `topics` is the canonical multi-topic list once present. The runtime subscribes to every route topic and uses direction flags at publish/apply decision points. Flutter renders and edits the route list, while activity/log records remain metadata-only.

#### Alternatives Considered

- Replace `topic` completely: rejected because existing config, bridge tests, and status payloads use `topic`.
- Separate subscribe topics from publish topics: rejected for now because one row with two direction flags is simpler for users and maps directly to the request.
- Push history restore through Go runtime: deferred because Flutter can write the local clipboard on a user gesture and update local in-memory history without adding engine commands.

#### Module Responsibilities

- `core/runtime`: config normalization, topic route validation, subscribe/unsubscribe diffs, publish fan-out, remote route gating, status.
- `core/configstore`: persist non-sensitive multi-topic settings without body text.
- `bridge`: expose `topics` in settings/status and include topic metadata in activity details.
- `cmd/clipboardnode-bridge`: map bridge settings to runtime config and runtime status to bridge status.
- `app/lib/core/bridge`: Dart model, validation, parsing, and history restore helpers.
- `app/lib/features/shell`: topic management UI, body-only history tile, restore click affordance, logs.
- `docs/requirements` and `docs/specs`: stable truth update for multi-topic and history/log split.

#### Data / Call Flow

- Settings save: UI topic rows -> `ClipboardSettings.topics` -> bridge JSON -> `runtime.Config.Topics` -> normalized route list.
- Startup/reconnect: runtime subscribes to all configured route topics.
- Local text: runtime validates and hashes once, builds one payload, publishes it to every `sync_from_local=true` route.
- Remote message: runtime looks up `msg.Topic`; unknown or `sync_to_local=false` routes are ignored; enabled routes proceed through existing dedupe, pending, and apply logic.
- History restore: UI writes selected body to local clipboard and promotes the entry to the top of in-memory history.

#### Interface Drafts

- Go:
  - `const DefaultTopic = "clipboard.text"`
  - `type TopicRoute struct { Topic string; SyncToLocal bool; SyncFromLocal bool }`
  - `Config.Topics []TopicRoute`
  - `Status.Topics []TopicRoute`
  - `Decision.Topic string`
- Dart:
  - `class TopicSyncConfig`
  - `ClipboardSettings.topics`
  - `normalizeTopicSyncConfigs(...)`
  - `promoteClipboardHistoryEntry(...)`
  - `ClipboardEngineBridge.restoreHistory(ClipboardHistoryEntry entry)`

#### Error Handling and Safety

- Trim and validate all topic inputs.
- Reject duplicate topics instead of merging silently.
- Keep body text out of status, logs, persisted config, and JSON decisions.
- Surface platform clipboard restore errors to `lastError`.

#### Performance and Testing Strategy

- Keep topic route count bounded by UI/validation guardrail.
- Hash local text once, reuse payload for fan-out.
- Unit tests:
  - config default and route normalization;
  - duplicate and empty topic validation;
  - multi-topic subscribe/resubscribe/unsubscribe;
  - publish fan-out to local-to-topic routes only;
  - remote apply ignore when local direction is disabled.
- Flutter tests:
  - default topic is `clipboard.text`;
  - history restore promotes clicked text;
  - settings render topic row controls.

#### Extensibility Design Points

- Direction flags can later grow per-topic labels, paused state, or read-only subscriptions without changing TopicBus.
- Legacy `topic` continues to make older config/status consumers work.
- Activity topic metadata provides future filtering without body exposure.

#### Issue List

- None.

### Stage 3.1 - Planning

#### Project Goal and Current State

Current implementation has a single `topic` setting (`clipboard/shared` in Flutter defaults), single-topic runtime subscription/publish/apply, and body history generated from activity events. The new goal is multi-topic directional sync plus a body-focused restoreable clipboard history.

#### Docs Governance Routing Decision

- Requirements impact: add
- Specs impact: add
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons: none currently; existing `docs/lessons/flutter-switch-hover-state-layer.md` was read as UI context, but no new recurring lesson is known yet.
- `docs/README.md`, `docs/requirements/README.md`, and `docs/specs/README.md` topology remains unchanged because existing leaf docs are updated in place.

#### Executable Task List

- T1 - Update stable docs for multi-topic directional sync and body-only history behavior.
- T2 - Extend Go runtime/config/bridge contract for multi-topic routes.
- T3 - Update Flutter bridge models, settings UI, and history restore behavior.
- T4 - Add or update Go and Flutter tests.
- T5 - Run validation and fix regressions.
- T6 - Archive the completed workflow in `docs/change`.

#### Task Details

##### T1 - Stable docs

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history/plan.md`
- Goal: Update requirements/specs for default topic `clipboard.text`, multi-topic route list, per-topic direction flags, and history/log split.
- Files / Modules: `docs/requirements/clipboard-sync.md`, `docs/specs/clipboard-sync.md`
- Write Set: those two docs only.
- Acceptance: stable docs describe the new behavior without duplicating change archive content.
- Test Points: docs reviewed by diff.
- Rollback: revert doc hunks for this task.

##### T2 - Runtime and bridge contract

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history/plan.md`
- Goal: Add topic routes to Go config/status/settings and enforce runtime multi-topic behavior.
- Files / Modules: `core/runtime`, `core/configstore`, `bridge`, `cmd/clipboardnode`, `cmd/clipboardnode-bridge`
- Write Set: Go source and tests under listed modules.
- Acceptance: legacy `topic` still works; new `topics` list subscribes/publishes/applies by direction; invalid topics fail explicitly.
- Test Points: `GOWORK=off go test ./core/runtime ./core/configstore ./bridge ./cmd/clipboardnode-bridge -count=1`
- Rollback: revert Go source and test hunks for this task.

##### T3 - Flutter UI and state

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history/plan.md`
- Goal: Add topic row settings, update defaults, parse `topics`, render history as text bodies, and restore clicked history entries.
- Files / Modules: `app/lib/core/bridge`, `app/lib/core/controller`, `app/lib/features/shell`, `app/test`
- Write Set: Flutter source and widget tests under listed modules.
- Acceptance: settings support multiple topics and direction toggles; history tiles restore and promote text; logs remain activity metadata.
- Test Points: Flutter analyze/test when tooling is available.
- Rollback: revert Dart source and test hunks for this task.

##### T4 - Validation coverage

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history/plan.md`
- Goal: Ensure changed behavior has regression coverage.
- Files / Modules: Go tests and Flutter widget/state tests.
- Write Set: `*_test.go`, `app/test/widget_test.dart`
- Acceptance: tests cover default topic, validation, fan-out, gating, and restore promotion.
- Test Points: targeted Go tests, Flutter analyze/test.
- Rollback: revert test hunks.

##### T5 - Validation

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history/plan.md`
- Goal: Run repository validation appropriate to the change surface.
- Files / Modules: no planned writes except fixing failures mapped back to T1-T4.
- Write Set: none unless validation exposes a mapped defect.
- Acceptance: Go targeted tests and `git diff --check` pass; Flutter analyze/test attempted with local Flutter SDK.
- Test Points: commands recorded in final archive.
- Rollback: revert any validation-driven fixes by task.

##### T6 - Change archive

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history/plan.md`
- Goal: Create `docs/change/2026-06-04_clipboard-multitopic-history.md` with task mapping, docs impact, validation, and rollback notes.
- Files / Modules: `docs/change`
- Write Set: one change archive; update `docs/change/README.md` only if index style requires it.
- Acceptance: archive contains required `$m-autoflow` Stage 4 sections.
- Test Points: docs diff review and `git diff --check`.
- Rollback: delete archive/index hunk.

#### Dependencies

- Local Go toolchain.
- Local Flutter SDK at `D:/project/MyFlowHub3/.tmp/tools/flutter-sdk-3.41.9/flutter` for UI validation.
- No server/proto/sdk repo changes expected.

#### Risks and Notes

- Cross-language contract updates require synchronized Go and Dart tests.
- Platform clipboard restore uses Flutter clipboard APIs; live engine watcher may publish restored text if auto-watch and local-to-topic routes are enabled, which is consistent with actual clipboard synchronization.
- Activity logs remain metadata-only even when history retention is body.

#### Parallelism Assessment

- Potential independent tasks exist, but Go runtime contract, bridge contract, Dart parser, and UI settings all share the same JSON field names and compatibility rules.
- Sub-agents are not used because the write sets are tightly coupled and integration risk is higher than the benefit for this single-repo change.

#### Issue List

- None.

阻塞：否
进入 3.2

### Stage 3.2 - Persistent History Implementation Follow-up

#### Task Mapping

- `T7` docs/plan: requirements/specs now define dedicated local persisted body history and persistent non-sensitive config; plan records the Stage 4 rollback and new task IDs.
- `T8` Go bridge persistence: added `history.json` store, `restore_history`, `history.updated`, history clear/trim/retention handling, and Go tests.
- `T9` Dart live/web bridge: added history entry serialization, persisted history list parsing, live/web `history.updated` handling, web `/history` fallback fetch, and restore command persistence.
- `T10` validation/archive: ran Go/Flutter validation and refreshed this plan plus the change archive.

#### Implementation Notes

- Config persistence remains in `core/configstore`: `set_config` saves normalized `topics`, direction flags, `history_retention`, and `history_limit` to `config.json`.
- Clipboard body history is intentionally separate in `history.json` next to `config.json`.
- `history.updated` is the dedicated body-history event. Status, config, transfer records, pending metadata, and activity log state remain body-free.
- Retention changes to `metadata` or `none` clear the persisted body history; `body` retention trims to `history_limit`.
- Restore writes the selected text to the local clipboard first, then sends `restore_history` so the Go store persists the promoted entry.

#### Validation During Follow-up

- `$env:GOWORK='off'; go test ./bridge ./cmd/clipboardnode-bridge -count=1`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat analyze` from `app`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat test` from `app`: passed.
- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `git diff --check`: passed.

阻塞：否
进入 3.3

### Stage 3.3 - Persistent History Code Review Follow-up

#### Review Checklist

- 需求覆盖: 通过. 历史正文现在可本地持久化，配置也继续持久化 multi-topic 和历史设置。
- 架构合理性: 通过. 正文历史 store 位于 bridge 本地边界，runtime/configstore 不保存正文。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）: 通过. history store 只在 append/promote/clear/settings-change 时写盘，列表 bounded。
- 可读性与一致性: 通过. Go 和 Dart contract 使用显式 `HistoryEntry`、`restore_history`、`history.updated` 命名。
- 可扩展性与配置化: 通过. `history_limit`、`history_retention` 继续驱动 trim/clear 策略。
- 稳定性与安全: 通过. 非 body retention 清空本地正文历史，status/config/log 不承载正文。
- 测试覆盖情况: 通过. Go 覆盖 store reload/dedupe/trim/clear/promote；Flutter 覆盖 history payload parse 和 restore JSON。
- 子Agent治理与审计: 通过. 未派发子Agent，原因是 Go/Dart JSON contract 强耦合。

阻塞：否
进入 4

### Stage 4 - Archive Refresh Follow-up

#### Docs Impact

- Requirements impact: updated.
- Specs impact: updated.
- Lessons impact: none;本轮是需求澄清和实现补齐，没有新增可复用排障模式。
- Related requirements: `docs/requirements/clipboard-sync.md`.
- Related specs: `docs/specs/clipboard-sync.md`.
- Related lessons: none.

#### Archive

- Refreshed `docs/change/2026-06-04_clipboard-multitopic-history.md` with persistent history behavior and validation.

阻塞：否
等待用户确认是否结束 workflow

### Stage 3.2 - Implementation

#### Task Mapping

- `T1` stable docs: updated `docs/requirements/clipboard-sync.md` and `docs/specs/clipboard-sync.md` for multi-topic route policy, default `clipboard.text`, history/log separation, and restore behavior.
- `T2` runtime and bridge contract: implemented Go `TopicRoute`, normalized `topics`, multi-topic subscribe/unsubscribe/resubscribe, fan-out publish, remote route gating, `pending_topic`, and activity topic metadata.
- `T3` Flutter UI and state: implemented `TopicSyncConfig`, multi-topic settings rows, direction toggles, default `clipboard.text`, clickable body-only history restore, and pending metadata-only body-history filtering.
- `T4` validation coverage: added Go tests for route validation, fan-out, remote gating, multi-topic lifecycle, partial reconnect cleanup, and bridge contract mapping; added Flutter tests for topic normalization, restore promotion, pending history filtering, and UI labels.
- `T5` validation: ran Go and Flutter validation commands listed below.
- `T6` change archive: created `docs/change/2026-06-04_clipboard-multitopic-history.md` and updated `docs/change/README.md`.

#### Implementation Notes

- Legacy scalar `topic` remains supported. When `topics` is present, it is canonical and `topic` normalizes to the first route.
- The default route is `clipboard.text` with both `sync_to_local` and `sync_from_local` enabled.
- Runtime subscribes to every route topic, but publish/apply is gated by each route's direction flags.
- Pending remote text remains in the queue/log as metadata until applied; it is not added to body history at the bridge/UI boundary.
- Restoring a body history entry writes the text to the platform clipboard where supported and promotes the entry to the top of local in-memory history.

#### Validation During 3.2

- `$env:GOWORK='off'; go test ./core/runtime ./core/configstore ./bridge ./cmd/clipboardnode-bridge -count=1`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat analyze` from `app`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat test` from `app`: passed.

阻塞：否
进入 3.3

### Stage 3.3 - Code Review

#### Review Checklist

- Requirements alignment: pass. History/log separation, restore promotion, multi-topic routes, and default topic match the user request.
- Architecture boundaries: pass. Runtime owns sync policy; Flutter owns body history presentation and restore affordance; bridge maps JSON contracts only.
- Cross-language contract: pass. `topic`, `topics`, `pending_topic`, and activity `topic` are mapped across Go bridge and Dart bridge parsers.
- Privacy boundary: pass. Status/transfer/pending activity remain body-free; only local-published and applied decisions can carry bridge activity text for body history.
- Error handling: pass. Empty/duplicate topics fail explicitly; no publish route and disabled remote-to-local route produce explicit ignored decisions.
- Tests: pass. Go and Flutter coverage was added for changed behavior.
- Parallelism/subagents: no sub-agent dispatched because the runtime/bridge/Dart/UI fields are tightly coupled and share one JSON contract.

#### Review Fixes

- Added `pending_topic` status mapping through Go bridge and Dart live/web/mobile parsers.
- Changed pending remote activity to metadata-only at the bridge boundary and filtered pending out of Dart body history.
- Added cleanup for partial multi-topic subscription failures during reconnect.
- Added topic metadata to known-topic failure decisions so logs retain the relevant topic.

阻塞：否
进入 4

### Stage 4 - Archive

#### Docs Impact

- Requirements impact: updated.
- Specs impact: updated.
- Lessons impact: none.
- Related requirements: `docs/requirements/clipboard-sync.md`.
- Related specs: `docs/specs/clipboard-sync.md`.
- Related lessons: none.

#### Archive

- Created `docs/change/2026-06-04_clipboard-multitopic-history.md`.
- Updated `docs/change/README.md`.

#### Final Validation

- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat analyze` from `app`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat test` from `app`: passed.
- `git diff --check`: passed.
- `rg -n "[ \t]+$" docs\change\2026-06-04_clipboard-multitopic-history.md`: no trailing whitespace matches.

阻塞：否
等待用户确认是否结束 workflow

### Stage 1 - Requirements Analysis Follow-up

#### Rollback Reason

用户确认剪贴板历史也需要持久化，且配置必须持久化。上一轮 Stage 1/2/3.1 明确将“剪贴板正文不落盘”作为不做项，因此 workflow 从 Stage 4 回滚到新的 Stage 1/2/3.1 后继续。

#### Goal

在已完成的多 topic 和历史/日志拆分基础上，补齐本地剪贴板正文历史持久化；配置持久化继续走现有 configstore，但必须确认 `topics`、方向开关、`history_retention` 和 `history_limit` 都会保存。

#### Scope

- Must:
  - 将正文历史持久化到本地独立 history store。
  - history store 与 runtime config/status/log/transfer/pending metadata 分离。
  - `history_retention=body` 时加载、追加、去重、裁剪、restore 后置顶并持久化。
  - `history_retention=metadata|none` 或用户清空最近记录时清除已持久化正文历史。
  - 保持配置持久化，覆盖 multi-topic route list 和方向开关。
- Not doing:
  - 将剪贴板正文写入 `config.json`。
  - server-side 历史、离线 replay、TopicBus 协议变更。

#### Acceptance Criteria

- 重启 live bridge 后，`history_retention=body` 的历史正文通过 `history.updated` 回到 UI。
- restore 历史条目后，该条正文写入本机剪贴板并成为持久化历史顶部。
- `history_retention` 改为 `metadata` 或 `none` 后，本地持久化正文历史被清空。
- `set_config` 保存的 config 仍包含 `topics`、方向开关和历史配置，不包含剪贴板正文。

### Stage 2 - Architecture Design Follow-up

#### Overall Solution

新增 `cmd/clipboardnode-bridge` 本地 `history.json` store，与现有 `config.json` 同目录。Go bridge 在活动事件进入 body history、清空最近记录、配置 retention 改变、以及 Dart restore 后调用该 store，并通过 `history.updated` 事件把完整 bounded history 发给 live/web UI。Dart live/web 解析 `history.updated`；restore 成功写系统剪贴板后发送 `restore_history` 命令，由 Go store 完成持久化置顶。

#### Module Responsibilities

- `core/configstore`: 继续只保存非敏感配置。
- `bridge`: 增加 history command/event 和 `HistoryEntry` JSON contract。
- `cmd/clipboardnode-bridge`: 管理 `history.json`、持久化策略、清空策略和 event emission。
- `app/lib/core/bridge`: 解析 history list，restore 后通知 bridge 持久化。

#### Error Handling and Safety

- history JSON 读取/写入失败必须显式返回错误。
- retention 非 body 时不得继续返回或保留持久化正文历史。
- status/config/log 不承载历史正文，历史正文只通过专用 history contract 进入 UI。

### Stage 3.1 - Replanning Follow-up

#### Docs Governance Routing Decision

- Requirements impact: clarify
- Specs impact: clarify
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons: none;本轮是需求澄清，不是可复用故障排查经验。

#### Additional Executable Task List

- T7 - Update stable docs and plan for persistent local body history.
- T8 - Implement bridge history persistence and Go contract/tests.
- T9 - Update Dart live/web history event handling and restore persistence.
- T10 - Re-run validation, code review, and refresh change archive.

##### T7 - Persistent history docs

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history/plan.md`
- Goal: Replace in-memory-only history requirement with a local dedicated persisted history store while preserving config/status/log privacy boundaries.
- Files / Modules: `docs/requirements/clipboard-sync.md`, `docs/specs/clipboard-sync.md`, `plan.md`
- Write Set: listed docs only.
- Acceptance: docs no longer conflict with persisted history requirement and explicitly keep clipboard bodies out of config/status/logs.
- Test Points: docs diff and whitespace checks.
- Rollback: revert T7 hunks.

##### T8 - Go bridge persistent history

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history/plan.md`
- Goal: Add `history.json` store, `restore_history`, `history.updated`, clear/trim/retention handling, and Go tests.
- Files / Modules: `bridge`, `cmd/clipboardnode-bridge`
- Write Set: Go bridge source and tests.
- Acceptance: history persists across store reload, de-dupes/trims, clears on non-body retention, and restore promotion persists.
- Test Points: `GOWORK=off go test ./bridge ./cmd/clipboardnode-bridge -count=1`
- Rollback: revert T8 files/hunks.

##### T9 - Dart live/web persistent history handling

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history/plan.md`
- Goal: Parse `history.updated` arrays and send `restore_history` after successful live/web clipboard writes.
- Files / Modules: `app/lib/core/bridge`, `app/test`
- Write Set: Dart bridge source and tests.
- Acceptance: live/web can load persisted history lists and restore commands update persisted order.
- Test Points: `flutter analyze`, `flutter test`.
- Rollback: revert T9 files/hunks.

##### T10 - Validation and archive refresh

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-clipboard-multitopic-history/plan.md`
- Goal: Run Go/Flutter validation, perform Stage 3.3 review, and update `docs/change/2026-06-04_clipboard-multitopic-history.md`.
- Files / Modules: tests and change archive.
- Write Set: `docs/change/2026-06-04_clipboard-multitopic-history.md` plus validation-driven fixes mapped to T8/T9.
- Acceptance: validation results and rollback notes include persistent history.
- Test Points: full commands recorded in archive.
- Rollback: revert archive/fix hunks.

#### Parallelism Assessment

No sub-agent is used. The Go history contract, bridge command/event names, Dart parser, and restore behavior share one JSON boundary and must be integrated atomically.

阻塞：否
进入 3.2
