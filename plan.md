# Plan - clipboardnode-device-id-display-name

## Workflow Information
- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `fix/clipboardnode-stable-device-id`
- Base: `master` at `926dec18f0d58511b4be3f8fc721553bb62cf6aa`
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-clipboardnode-stable-device-id/MyFlowHub-ClipboardNode`
- Current Stage: 4 - Change Archive Complete

## Stage Records

### Initialization
- guide.md: read from `D:/project/MyFlowHub3/guide.md`.
- base/worktree confirmation: implementation happens only in the dedicated worktree above.
- Participating repo: `MyFlowHub-ClipboardNode` only.
- Main repo generated build output is ignored for this workflow write set.

### Stage 1 - Requirements Analysis
#### Goal
Fix the `authenticate myflowhub node: invalid signature` path caused by changing the configured device identity, and make the user-visible node display name configurable without coupling it to the auth identity.

#### Scope
- Must: expose a configurable `device_id` as the MyFlowHub auth identity.
- Must: clear local login/auth snapshot data when `device_id` changes.
- Must: expose a configurable `display_name` and send it in auth register/login payloads.
- Must: keep legacy `device_label` config compatible by migrating it into `device_id` and `display_name` when the new fields are absent.
- Must: keep the change local to ClipboardNode without changing MyFlowHub wire contracts.
- Not doing: changing Server / SubProto auth verification rules, deleting node private keys automatically, or adding cross-repo protocol fields.

#### Use Cases
- A user edits the device ID after an earlier registration; ClipboardNode clears the old `auth_snapshot` instead of logging in with the old node id and new signature identity.
- A user edits only the display name; ClipboardNode keeps the same auth identity and sends the new display name during register/login.
- An existing config with only `device_label` remains usable after upgrade.

#### Functional Requirements
- Auth register/login signs with normalized `device_id`.
- Auth register/login payloads carry normalized `display_name`, falling back to `device_id` when blank.
- On config update, if the normalized old `device_id` and new `device_id` differ, the engine stops the current session and clears the auth snapshot.
- On startup, if the saved auth snapshot `device_id` differs from the normalized config `device_id`, the auth client clears the stale snapshot before registering/logging in.
- Legacy `device_label` is retained as a compatibility field and maps to display-name/status metadata.

#### Non-functional Requirements
- No clipboard text in logs or persisted config.
- Smallest safe change surface.
- No MyFlowHub protocol or server behavior changes.
- Focused automated regression coverage for config normalization and auth mismatch clearing.

#### Inputs / Outputs
- Inputs: runtime config fields `device_id`, `display_name`, legacy `device_label`, saved `auth_snapshot.json`, connect/login flow.
- Outputs: normalized config, UI-safe status, auth register/login payloads, cleared auth snapshot on device-id mismatch.

#### Edge Cases
- Existing config has `device_label` but no `device_id`.
- Existing snapshot has `device_id=old` but config has `device_id=new`.
- Existing snapshot has `node_id` but no `device_id`.
- Blank `display_name` falls back to `device_id`.

#### Acceptance Criteria
- Changing `device_id` clears `auth_snapshot.json` and avoids login with the stale node id.
- Changing `display_name` does not clear auth identity.
- Register/login payloads use `device_id` for identity and `display_name` for metadata.
- Existing targeted Go tests pass, and new tests cover normalization and auth mismatch clearing.

#### Risks
- Re-registering after a device-id change may require Hub approval; that is expected and preferable to `invalid signature`.
- Deleting `node_keys.json` automatically would be more destructive than needed, so this workflow clears only auth snapshot state.

### Stage 2 - Architecture Design
#### Overall Solution
Add first-class `DeviceID` and `DisplayName` runtime fields while keeping legacy `DeviceLabel`. Normalize config centrally so old `device_label` files migrate safely. Pass `DeviceID` and `DisplayName` separately through the engine into the MyFlowHub client. Clear stale auth snapshots both during config updates and during startup auth resolution.

#### Alternatives Considered
- Treat `device_label` as display name only and keep saved snapshot device id forever: rejected because the user clarified that editing device ID should clear login data.
- Delete `node_keys.json` on device-id change: rejected as unnecessarily destructive for this bug; the login data causing stale-node reuse is the auth snapshot.
- Add server-side display-name logic: rejected because existing auth payloads already include `DisplayName`.

#### Module Responsibilities
- `core/runtime`: normalize `device_id`, `display_name`, and legacy `device_label`.
- `core/engine`: compare old/new device IDs, stop runtime, and clear auth on identity change.
- `core/myflowhub`: register/login with separate identity/display values and clear mismatched snapshots before auth.
- `bridge` and `cmd/clipboardnode-bridge`: carry the new JSON fields and preserve compatibility.
- `app`: show separate settings fields for device ID and display name.

#### Data / Call Flow
1. UI sends `device_id` and `display_name` in `set_config`.
2. Bridge converts settings into normalized runtime config and saves it.
3. Engine compares old and new normalized `device_id`.
4. If changed, engine stops the active runtime and calls `ClearAuth`.
5. On connect, engine passes `device_id` and `display_name` to `EnsureIdentity`.
6. MyFlowHub client clears any mismatched saved snapshot, then registers/logs in using the configured identity.

#### Interface Drafts
- Runtime config: `DeviceID json:"device_id,omitempty"`, `DisplayName json:"display_name,omitempty"`, legacy `DeviceLabel json:"device_label,omitempty"`.
- Bridge settings/status: same new fields plus legacy `device_label`.
- Auth client: `EnsureIdentity(ctx, deviceID, displayName)`.

#### Error Handling and Safety
- Invalid or blank device ID normalizes to a safe default before auth.
- Auth clearing errors are returned instead of swallowed.
- Runtime stop errors during device-id change are returned to the bridge.
- Existing auth error propagation to UI remains intact.

#### Performance and Testing Strategy
- No additional network round trips.
- Config comparison uses normalized in-memory values.
- Run focused Go tests for runtime/configstore/bridge/engine/myflowhub plus Flutter analysis after UI changes.

#### Extensibility Design Points
- Future UI or CLI tooling can manage device identity without relying on legacy `device_label`.
- Display metadata remains independent from auth identity.

### Stage 3.1 - Planning
#### Docs Governance Routing Decision
Using `$m-docs` routing:
- Requirements impact: none
- Specs impact: none
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons: check whether a new auth-identity lesson is needed during stage 4
- Change archive destination: `docs/change/2026-06-03_clipboardnode-device-id-display-name.md`

#### Executable Task List
- T1: Add normalized config fields and bridge contract support for `device_id` / `display_name`.
- T2: Update engine and MyFlowHub auth client to clear stale auth snapshots on device-id changes.
- T3: Update Flutter settings UI and bridge parsing to configure both fields.
- T4: Add regression tests and run validation.
- T5: Archive the workflow and add a reusable lesson if warranted.

#### Task Details
##### T1 - Config and Bridge Contract
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-clipboardnode-stable-device-id/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Files / Modules: `core/runtime`, `core/configstore`, `bridge`, `cmd/clipboardnode-bridge`
- Write Set: `core/runtime/config.go`, `core/configstore/store_test.go`, `bridge/contract.go`, `bridge/contract_test.go`, `cmd/clipboardnode-bridge/main.go`
- Acceptance: old `device_label` configs normalize into new identity/display fields and status emits both new fields.
- Rollback: revert config/contract additions and tests.

##### T2 - Auth Clearing and Payloads
- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Files / Modules: `core/engine`, `core/myflowhub`
- Write Set: `core/engine/engine.go`, `core/engine/engine_test.go`, `core/myflowhub/client.go`, `core/myflowhub/client_test.go`
- Acceptance: mismatched auth snapshots are cleared before login; display-name-only changes do not clear auth.
- Rollback: revert engine/auth helper changes and tests.

##### T3 - Flutter Settings UI
- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Files / Modules: `app/lib/core/bridge`, `app/lib/features/shell`
- Write Set: `app/lib/core/bridge/*.dart`, `app/lib/features/shell/clipboard_shell.dart`
- Acceptance: settings page has separate device ID and display name inputs, and live/web bridges parse emitted status fields.
- Rollback: revert UI contract changes.

##### T4 - Validation
- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Files / Modules: tests and build output only
- Write Set: none
- Acceptance: targeted Go tests pass, Flutter analysis succeeds or any limitation is reported, and a bridge smoke validates device-id mismatch clearing.
- Rollback: fix failures or revert T1-T3.

##### T5 - Docs Archive
- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Files / Modules: `docs/change`, maybe `docs/lessons`
- Write Set: `docs/change/2026-06-03_clipboardnode-device-id-display-name.md`, optional lesson files and indexes
- Acceptance: change archive records requirements/spec impact and searchable troubleshooting cues.
- Rollback: remove archive if workflow is abandoned.

#### Dependencies
- Validation may use a temporary config directory and does not require mutating the user's live `%APPDATA%` files.

#### Risks and Notes
- New device IDs may require re-approval on the Hub after the snapshot is cleared.
- Generated Flutter plugin files already dirty in this worktree are prior build output and are not part of the planned write set.

#### Parallelism Assessment
- T1-T3 share field names and migration semantics across languages.
- Sub-agents: not used due small but tightly coupled write set and need for single-agent integration.

#### Issue List
- None blocking.

阻塞：否
进入 3.2

### Stage 3.2 - Implementation Summary
#### T1 - Config and Bridge Contract
- Status: completed.
- Changed `core/runtime.Config` to carry `device_id` and `display_name` separately while retaining legacy `device_label`.
- Centralized normalization so legacy `device_label` becomes both identity and display fallback when new fields are absent.
- Updated bridge settings/status JSON contracts and stdio bridge config conversion.

#### T2 - Auth Clearing and Payloads
- Status: completed.
- Updated engine startup to call `EnsureIdentity(ctx, deviceID, displayName)`.
- Updated config changes to stop the runtime and clear auth snapshot when normalized `device_id` changes.
- Updated MyFlowHub register/login payload helpers to sign and identify with `device_id` while sending `display_name` as metadata.
- Added stale snapshot clearing before register/login when saved snapshot identity does not match configured `device_id`.

#### T3 - Flutter Settings UI
- Status: completed.
- Added separate settings controls for device ID and display name.
- Updated live, web, mobile, and preview bridges to parse/emit `device_id`, `display_name`, and legacy `device_label` compatibly.
- Activity labels now use display name while preserving identity as device ID.

#### T4 - Validation
- Status: completed.
- `gofmt` and Dart format completed.
- `GOWORK=off go test ./... -count=1`: passed.
- `GOWORK=off go test -race ./core/myflowhub ./core/engine ./bridge -count=1`: passed.
- `GOWORK=off go build ./cmd/clipboardnode-bridge`: passed.
- `flutter analyze`: passed with Flutter `3.41.9`.
- `flutter test`: passed, 5 tests.
- `scripts/validate.ps1 -FlutterBin D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat`: passed, including Go tests, Go builds, Flutter analysis/tests, and `git diff --check`.
- Bridge smoke with temporary config/auth data:
  - `device_id` changed from `old-device` to `new-device`: config saved new identity/display fields and `myflowhub/auth_snapshot.json` was cleared.
  - only `display_name` changed for `same-device`: config saved the new display name and retained node identity in auth snapshot.

#### Notes
- Validation outputs include recurring local shell noise from conda initialization after successful commands; command exit codes were zero.
- Flutter generated plugin files show as modified in `git status` after Flutter tooling, but `git diff --name-only` has no content changes for them.
- Temporary smoke artifacts live under `build/smoke-device-id` and are generated validation output.

### Stage 3.3 - Code Review
- 需求覆盖: 通过. Device ID and display name are separate, legacy device label compatibility is preserved, and device ID changes clear local login state.
- 架构合理性: 通过. Normalization remains in `core/runtime`, engine owns config transition behavior, and MyFlowHub client owns auth payload/snapshot semantics.
- 性能风险: 通过. The change adds no repeated network calls, no O(n^2) paths, and only uses existing config/auth file writes on explicit settings changes.
- 可读性与一致性: 通过. Field names match JSON/API intent and surrounding code style; no broad refactor was introduced.
- 可扩展性与配置化: 通过. Future UI/CLI callers can configure identity and display metadata independently without relying on legacy `device_label`.
- 稳定性与安全: 通过. Auth clearing errors are returned, stale snapshots are not silently reused, and clipboard content is still excluded from status/config.
- 测试覆盖情况: 通过. Added/updated Go tests for contract, config persistence, engine auth clearing, auth payloads, and stale snapshot behavior; Flutter tests and analysis pass.
- 子Agent治理与审计: 通过. No sub-agents were used because the write set was small and tightly coupled.

阻塞：否
进入 4

### Stage 4 - Change Archive
- 使用 `$m-docs` 完成归档路由和需求/规格影响复核。
- Requirements impact: updated
- Specs impact: updated
- Lessons impact: updated
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons:
  - `docs/lessons/device-id-auth-snapshot-mismatch.md`
  - `docs/lessons/startup-subscribe-timeout-half-connected.md`
- Change archive: `docs/change/2026-06-03_clipboardnode-device-id-display-name.md`
- Docs index updates:
  - `docs/change/README.md`
  - `docs/lessons/README.md`
- Stable docs clarification:
  - `docs/requirements/clipboard-sync.md` now records separate device identity and display name requirements.
  - `docs/specs/clipboard-sync.md` now records `device_id`, `display_name`, and legacy `device_label` config/status semantics.
- Validation:
  - `GOWORK=off go test ./... -count=1` passed.
  - `GOWORK=off go test -race ./core/myflowhub ./core/engine ./bridge -count=1` passed.
  - `GOWORK=off go build ./cmd/clipboardnode-bridge` passed.
  - `flutter analyze` passed.
  - `flutter test` passed.
  - `scripts/validate.ps1 -FlutterBin D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat` passed.
  - Bridge smoke with temporary config/auth data passed for both `device_id` change and display-name-only change.
  - Final `git diff --check` passed after docs updates.
- SubAgents: none.

阻塞：否
等待是否结束 workflow

### Follow-up - Header Alignment UI Fix
- Date: 2026-06-04
- User symptom: desktop shell header text looked vertically off, and the overview header bottom border did not align with the left brand header border.
- Requirements impact: none
- Specs impact: none
- Lessons impact: none
- Task: make the desktop side navigation brand area and content top bar share one fixed header height and draw their bottom borders from matching containers.
- Files:
  - `app/lib/features/shell/clipboard_shell.dart`
  - `docs/change/2026-06-04_clipboardnode-header-alignment.md`
  - `docs/change/README.md`
- Implementation:
  - Added `_desktopHeaderHeight = 72`.
  - Replaced left brand `Padding + Divider` with a fixed-height decorated header.
  - Set the right top bar to the same fixed height.
  - Vertically centered `_BrandMark` contents with `Center` and `mainAxisSize: MainAxisSize.min`.
- Validation:
  - `dart format app/lib/features/shell/clipboard_shell.dart` passed.
  - `flutter analyze` passed.
  - `flutter test` passed, 5 tests.
  - `git diff --check` passed.
