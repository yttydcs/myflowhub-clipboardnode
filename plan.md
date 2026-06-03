# Plan - ClipboardNode compact text payload

## Workflow State

- Stage: 4 archived
- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-clipboard-compact-payload/MyFlowHub-ClipboardNode`
- Branch: `feat/clipboard-compact-payload`
- Base: `master` at `926dec1`
- Owner: main agent

## Stage 1 - Requirements Analysis

### Goal

Replace the verbose `clipboard.text.v1` application payload with a compact payload. Compatibility with the previous full metadata payload is explicitly not required.

### Scope

Must:
- Keep ClipboardNode as an application-level TopicBus payload user.
- Keep MyFlowHub TopicBus/Auth/Server/Proto/SDK/SubProto wire contracts unchanged.
- Preserve dedupe, local-origin filtering, local publish loop suppression, inline size validation, and UI-safe status.
- Update stable requirements/specs because payload shape is a stable ClipboardNode contract.
- Update Go unit tests and local smoke expectations where they depend on text payload behavior.

Optional:
- Keep transfer manifest payload unchanged unless text payload changes require shared helper updates.

Not doing:
- No backward compatibility parser for the old verbose text payload.
- No new MyFlowHub subprotocol, TopicBus action, or server-side ClipboardNode behavior.
- No UI layout changes.

### Use Cases

1. Manual TopicBus test can publish a small compact `clipboard.text.v1` payload without computing size/hash in advance.
2. Local ClipboardNode sends compact text payloads to peers.
3. Remote compact text payloads are validated, deduped, optionally applied, and loop-suppressed using locally computed hashes.

### Functional Requirements

- Text payload must include only stable application identity and body fields needed by runtime:
  - version
  - event id
  - origin node
  - origin instance id
  - optional origin device
  - text
- Runtime must compute UTF-8 byte size and SHA-256 from `text`.
- Runtime must reject empty, invalid UTF-8, NUL-containing, oversize, empty-event-id, missing-origin, or missing-instance payloads.
- Runtime must keep current `Decision` size/hash metadata by computing it locally.
- Local-origin, duplicate-event-id, remote-pending, auto-apply, and suppress-hash behavior must remain intact.

### Non-functional Requirements

- Do not persist or log clipboard text.
- Avoid repeated hashing beyond current local/remote path needs.
- Keep change surface narrow to runtime payload, tests, and docs.

### Inputs / Outputs

Inputs:
- Local clipboard text.
- TopicBus `clipboard.text.v1` compact JSON payload.

Outputs:
- TopicBus publish with compact JSON payload.
- Local clipboard write or pending metadata.
- Status/activity metadata with computed size/hash prefix.

### Boundary Exceptions

- Existing deployed nodes using the old full payload will no longer interoperate in this branch by user instruction.
- TopicBus publish remains best-effort and is not a remote apply ACK.
- Oversize text still falls back to transfer manifest behavior when configured.

### Acceptance Criteria

- `ClipboardTextEventV1` marshals compact payloads.
- Parser rejects malformed compact payloads and no longer requires supplied size/hash/content metadata.
- Runtime tests cover remote compact payload apply/pending/dedupe/loop suppression.
- Docs show compact payload as the stable shape.
- Focused Go tests pass.

### Risks

- Manual tests that still publish the old full payload will fail.
- If a remote platform changes text bytes during clipboard write/read, suppress-hash loop detection can still miss the transformed echo.

## Stage 2 - Architecture Design

### Overall Solution

Keep event name `clipboard.text.v1` but redefine its ClipboardNode application payload to compact JSON. The payload carries identity and text; runtime computes derived metadata. This is an application contract change only and does not alter TopicBus wire format.

### Module Responsibilities

- `core/runtime/event.go`: define compact `ClipboardTextEventV1`, build, validate, marshal, parse, and hash helper behavior.
- `core/runtime/runtime.go`: continue using computed digest for publish/apply decisions and suppress hashes.
- `core/runtime/*_test.go`: update text payload tests and fixture builders.
- `docs/requirements` and `docs/specs`: update stable payload contract and validation language.
- `docs/change`: archive completed workflow after validation.

### Data / Call Flow

Local send:
1. Runtime validates local text and computes digest.
2. Runtime creates compact `ClipboardTextEventV1`.
3. Runtime marshals compact JSON and publishes it through TopicBus.
4. Decision metadata uses locally computed digest.

Remote receive:
1. Runtime parses compact JSON.
2. Runtime validates identity and text.
3. Runtime computes digest from text.
4. Runtime filters local-origin and duplicate event IDs.
5. Runtime either stores pending event or writes the clipboard.
6. Runtime records digest hash for loop suppression after write.

### Interface Draft

```go
type ClipboardTextEventV1 struct {
    Version          int    `json:"v"`
    EventID          string `json:"id"`
    OriginNode       uint32 `json:"from"`
    OriginInstanceID string `json:"instance"`
    OriginDevice     string `json:"device,omitempty"`
    Text             string `json:"text"`
}
```

### Error And Safety

- Missing `id`, `from`, `instance`, or `text` fails explicitly.
- Computed text digest enforces configured inline size.
- Clipboard body still must not be added to status/activity/log output.

### Performance And Testing

- Hash once per incoming payload after parse; reuse returned digest in runtime where useful.
- Run `go test ./core/runtime ./core/myflowhub ./bridge -count=1`.
- Run `go test ./... -count=1` if focused tests pass.

### Extensibility

- Keep version field as compact `v`.
- Leave transfer manifest unchanged for large content references.

## Stage 3.1 - Planning

Using `$m-docs`: this workflow changes stable payload truth, so:
- Requirements impact: updated
- Specs impact: updated
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons: none identified from `docs/lessons/README.md`

阻塞：否
进入 3.2

## Tasks

### T1 - Runtime compact payload

- Goal: replace full text payload fields with compact fields and local digest computation.
- Files:
  - `core/runtime/event.go`
  - `core/runtime/runtime.go`
- Acceptance:
  - Compact payload marshals with `v/id/from/instance/device/text`.
  - Parser computes digest for validation and runtime metadata.
  - Loop suppression remains hash-based.
- Tests:
  - `go test ./core/runtime -count=1`
- Rollback:
  - Revert `core/runtime/event.go` and `core/runtime/runtime.go`.

### T2 - Tests and fixtures

- Goal: align unit tests and fixtures with compact payload.
- Files:
  - `core/runtime/event_test.go`
  - `core/runtime/runtime_test.go`
  - any impacted Go tests found during validation
- Acceptance:
  - Tests assert compact JSON fields and no required wire `sha256/size/content_type/encoding/ts`.
- Tests:
  - `go test ./core/runtime ./core/myflowhub ./bridge -count=1`
- Rollback:
  - Revert changed test files.

### T3 - Documentation and archive

- Goal: update stable docs and archive workflow.
- Files:
  - `docs/requirements/clipboard-sync.md`
  - `docs/specs/clipboard-sync.md`
  - `docs/change/2026-06-03_compact-clipboard-text-payload.md`
- Acceptance:
  - Stable docs describe compact text payload and locally computed metadata.
  - Change archive captures tests and rollback.
- Tests:
  - Documentation review plus `git diff --check`.
- Rollback:
  - Revert docs and archive file.

## Parallelism Assessment

No sub-agent split. Runtime model, tests, and stable docs share one small contract and need tight sequencing to avoid incompatible intermediate states.

## Stage 3.2 - Implementation

### File-level Change Summary

- `core/runtime/event.go`: changed `ClipboardTextEventV1` to compact JSON fields, kept `Size` and `SHA256` as non-serialized derived runtime metadata, and normalized parsed/marshalled text events by computing digest from `Text`.
- `core/runtime/event_test.go`: updated text event tests to assert compact JSON output and digest derivation from compact payloads.
- `core/runtime/runtime_test.go`: updated local publish and local-origin fixtures to parse compact payloads and reject serialized derived fields.
- `docs/requirements/clipboard-sync.md`: updated stable text payload requirements and loop-prevention language.
- `docs/specs/clipboard-sync.md`: updated interface draft, call flow, validation, performance, and extensibility notes for compact payload.

### Task Mapping

- `T1`: completed runtime compact payload model and derived digest handling.
- `T2`: completed runtime fixture and validation tests.
- `T3`: completed stable requirement/spec updates and change archive.

### Validation

- `$env:GOWORK='off'; go test ./core/runtime -count=1`: passed.
- `$env:GOWORK='off'; go test ./core/runtime ./core/myflowhub ./bridge -count=1`: passed.
- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `git diff --check`: passed with CRLF warnings only.

## Stage 3.3 - Code Review

- 需求覆盖：通过。Compact text payload、无旧兼容、derived size/hash、local-origin/duplicate/loop suppression 均覆盖。
- 架构合理性：通过。变化限定在 ClipboardNode application payload，不改变 TopicBus/MyFlowHub wire contracts。
- 性能风险：通过。Digest 仍按每条 local/remote 事件路径计算；未引入额外 I/O、N+1 或锁竞争。
- 可读性与一致性：通过。Field names match compact contract; validation remains explicit.
- 可扩展性与配置化：通过。保留 `v=1` 和 `clipboard.transfer.v1` manifest 独立扩展点。
- 稳定性与安全：通过。远端 size/hash 不再受信，text 仍校验 empty/UTF-8/NUL/oversize，状态不暴露正文。
- 测试覆盖情况：通过。Focused runtime tests and full Go suite passed.
- 子Agent治理与审计：通过。No sub-agent dispatched; ownership and task mapping recorded.

## Stage 4 - Change Archive

Using `$m-docs`: archived the completed workflow in [docs/change/2026-06-03_compact-clipboard-text-payload.md](docs/change/2026-06-03_compact-clipboard-text-payload.md) and indexed it in [docs/change/README.md](docs/change/README.md).

- Requirements impact: updated
- Specs impact: updated
- Lessons impact: none
- Workflow end: awaiting user confirmation before merge/worktree cleanup.
