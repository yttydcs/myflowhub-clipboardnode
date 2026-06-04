# Clipboard Sync Specification

## Overall Solution

ClipboardNode is a standalone cross-platform MyFlowHub clipboard application. It uses a shared synchronization engine, a cross-platform UI shell, and platform-specific clipboard adapters. The protocol strategy reuses existing MyFlowHub subprotocols and treats ClipboardNode messages as application-level payloads.

Small text synchronization uses TopicBus as an application event channel:

1. The node connects to the configured parent Hub / endpoint, then performs registration/login in the background through the existing MyFlowHub auth flow.
2. When enabled, it subscribes to every configured clipboard topic route.
3. A platform clipboard watcher reports local text changes to the shared runtime.
4. The shared runtime validates, hashes, deduplicates, and publishes a compact `clipboard.text.v1` event to every route with local-to-topic sync enabled.
5. The runtime receives remote compact TopicBus events, validates the route policy, validates the payload, computes derived metadata, deduplicates it, and asks the platform adapter to write the text clipboard only when remote-to-local sync is enabled.

TopicBus is selected for small inline text because clipboard text sync is event-shaped, low-frequency, and does not need per-delivery ACK for the default private-network use case. Existing Stream or File capabilities remain the transfer path for large, binary, resumable, or file content. ClipboardNode must not add or modify MyFlowHub subprotocol wire contracts.

The default security model is the private MyFlowHub topology plus authenticated node identity. Device pairing, room keys, and application-layer end-to-end encryption are not required for the first full product.

## Alternatives Considered

- New subprotocol:
  - Rejected because ClipboardNode can express small content as TopicBus application events and larger content as manifests for existing transfer protocols.
- Stream text profile:
  - Better for bounded long text, ACK, backpressure, and future large payloads.
  - Reused only when already available as an existing MyFlowHub capability; ClipboardNode must not modify Stream wire semantics in this repository.
- Embedding in MyFlowHub-Win:
  - Rejected because clipboard sync is a complete cross-platform product, not a Win console feature.
- Reusing MetricsNode:
  - Rejected because clipboard sync has different privacy and runtime semantics.
- Pairing / room key security:
  - Rejected for the default product because the expected deployment is a private MyFlowHub topology with authenticated trusted nodes.
- Mandatory E2EE:
  - Deferred. It can be introduced later as an application-layer option for untrusted hub or multi-tenant deployments, without changing the MyFlowHub subprotocols.
- Wails-only UI:
  - Rejected for the complete product because it targets desktop well but does not cover mobile as the shared UI direction.
- Flutter UI shell:
  - Preferred for the complete product because it gives one cross-platform UI layer while platform clipboard behavior stays in native adapters. Implementation requires Flutter/Dart tooling.

## Module Responsibilities

### `engine` / `core/runtime`

- Own connection/auth orchestration through existing MyFlowHub SDK/client runtime.
- Own TopicBus subscribe, resubscribe, publish, and event receive.
- Own sync state, enablement, topic routes, size limit, dedupe windows, status reporting, and transfer manifest handling.
- Never call platform clipboard APIs directly.
- Expose UI-safe status without clipboard body leakage.
- Emit successful inline text decisions with a local-only text field that the UI bridge may use for explicit local body history persistence.

### `core/clipboard`

- Define platform adapter interfaces:
  - `ReadText`
  - `WriteText`
  - `WatchText`
  - `Close`
- Define clipboard event and write-result types.

### Cross-platform app UI

- Provide a single connect/disconnect control; registration, login, and login-state cleanup run in the background.
- Provide parent Hub / endpoint settings before connection.
- Provide topic route selection and per-route sync policy settings.
- Provide desktop/mobile-appropriate send and receive controls.
- Provide recent transfer status without forced body exposure.
- Provide validation, transport, and platform permission status.
- Provide settings for inline size, auto-watch, auto-apply, local retention mode, and body history length.
- Maintain a body history list separate from activity/log metadata; body history is local persisted UI state and defaults to the newest 256 text entries.
- Restore clicked body history entries to the local clipboard and promote the restored text to the top of the body history list.

### Platform adapters

- Desktop adapters own automatic clipboard watching, local apply, tray/menu integration, notifications, and optional autostart.
- Mobile adapters own share-sheet/manual send, foreground notification/service behavior where allowed, local apply controls, and platform permission explanations.

### `core/configstore`

- Persist non-sensitive runtime config.
- Must not persist clipboard text or event bodies.

### `cmd/clipboardnode-bridge`

- Own the local body-history store used by the desktop/live bridge.
- Persist body history in `history.json` next to `config.json`.
- Emit `history.updated` events with the bounded body history list.
- Clear the persisted body history when retention is not `body` or the user clears recent history.
- Must not include body history in status payloads or runtime config.

### `windows`

- Windows desktop host, tray or menu integration, Windows clipboard adapter, and Windows build scripts.
- Ensure clipboard operations obey Windows thread and message-loop requirements.

### `android`

- Android host, share-sheet integration, foreground notification/service where appropriate, permission/lifecycle handling, and clipboard adapter.
- Must not rely on unrestricted background clipboard watching.
- Android `auto_watch` is a foreground app policy. When enabled through the live mobile bridge, Kotlin registers an Android `ClipboardManager` primary-clip listener while ClipboardNode is started, forwards text into the gomobile manual clipboard adapter, and lets the existing runtime publish through configured `sync_from_local` topic routes.
- Android `auto_apply` writes accepted remote text to the Android system clipboard through the Kotlin platform channel after the Go runtime has accepted the event. This does not imply background clipboard access or server-side delivery acknowledgement.

### `ios`

- iOS host, share extension/manual send flow, clipboard apply controls, and platform permission/lifecycle handling.

### `linux` / `macos`

- Desktop clipboard adapters, tray/menu integration, notifications, and autostart support where practical.

### `bridge`

- Bridge the shared Go engine to the selected cross-platform UI shell.
- For Flutter, prefer a narrow JSON command/event bridge at first to reduce cross-language type churn.
- Mobile Go integration may use gomobile where practical; desktop may use a local process/FFI bridge depending on toolchain maturity.

## Data / Call Flow

### Startup

1. Load config.
2. Initialize engine with platform clipboard adapter and TopicBus client.
3. Connect to the configured `parent_endpoint`.
4. Register or rebind the local device identity from `device_id` if needed.
5. Login with the local device identity and send `display_name` as auth metadata.
6. If `enabled=true`, subscribe to every configured `topics[].topic`.
7. Start platform clipboard watcher.

### Disconnect / Shutdown

1. Stop platform clipboard watchers.
2. Unsubscribe best-effort from every active subscribed topic.
3. Clear in-memory login state and node identity.
4. Close the transport/session.
5. Preserve non-sensitive configuration such as parent endpoint, topic, device label, size limits, and local policy.

### Local Clipboard To TopicBus

1. Desktop adapter emits local text automatically, Android foreground listener or mobile/manual UI sends current/shared text.
2. Runtime checks enabled state.
3. Runtime normalizes text, computes UTF-8 byte length, and rejects empty or oversize input.
4. Runtime computes SHA-256.
5. Runtime ignores unchanged text using last local/remote hash state.
6. Runtime builds compact `ClipboardTextEventV1`.
7. Runtime publishes TopicBus application event to every topic route where `sync_from_local=true`:
   - topic: route topic
   - name: `clipboard.text.v1`
   - payload: event JSON
8. Runtime emits a successful local-publish decision with topic metadata and in-memory text for bridge-side body history when enabled.
9. If no route has `sync_from_local=true`, runtime emits an ignored local-policy decision and does not publish.

### TopicBus To Local Clipboard

1. Runtime receives TopicBus publish.
2. Runtime checks whether the message topic matches a configured route.
3. Runtime ignores unknown topics or routes where `sync_to_local=false`.
4. Runtime checks event name.
5. Runtime validates payload version, identity fields, and text.
6. Runtime computes UTF-8 byte size and SHA-256 from the text.
7. Runtime ignores local-origin or duplicate events.
8. Runtime either writes text through the platform adapter or records the event as pending when auto-apply is off. Android live mobile uses a gomobile manual adapter internally, then the Kotlin channel writes accepted applied text to the Android system clipboard.
9. Runtime records the write hash/event ID to suppress loops after successful local apply.
10. Runtime emits a pending/applied decision with topic metadata. Applied decisions may include in-memory text for bridge-side body history when enabled; pending decisions remain metadata-only at the bridge boundary.

### Body History Restore

1. The local bridge body-history store saves only text entries from successful inline text decisions when `history_retention=body`.
2. Clicking a history item writes the selected body text to the local system clipboard through the active platform bridge.
3. After a successful write, UI sends `restore_history` to the bridge, and the bridge promotes that body to the newest persisted history position while removing older duplicate entries with the same text.
4. Restore failures must surface as UI errors; they must not silently reorder history.
5. Restore may trigger normal local clipboard watching on desktop. If sync is enabled and route policy permits local-to-topic publish, that follow-up publish is consistent with the restored clipboard becoming the current local clipboard.

### Large Content Transfer

1. UI or adapter detects content that is unsupported for inline TopicBus or exceeds the configured inline limit.
2. Engine builds a `clipboard.transfer.v1` manifest with content type, size, hash, source metadata, and transfer reference.
3. Actual bytes move through an existing MyFlowHub Stream or File capability when available.
4. Receiver UI shows the transfer status and asks for apply/download when platform policy requires user action.
5. TopicBus is not used to split or carry large content bodies.

## Interface Drafts

### Clipboard Adapter

```go
type TextEvent struct {
    Text string
    Source string
    ObservedAt time.Time
}

type Adapter interface {
    ReadText(ctx context.Context) (string, error)
    WriteText(ctx context.Context, text string) error
    WatchText(ctx context.Context) (<-chan TextEvent, error)
    Close() error
}
```

### Topic Payload

```go
type ClipboardTextEventV1 struct {
    Version          int    `json:"v"`
    EventID          string `json:"id"`
    OriginNode       uint32 `json:"from"`
    OriginInstanceID string `json:"instance"`
    OriginDevice     string `json:"device,omitempty"`
    Text             string `json:"text"`
    Size             int    `json:"-"`
    SHA256           string `json:"-"`
}
```

`Size` and `SHA256` are runtime-derived fields computed from `Text` after parse
or before publish; they are not serialized in the `clipboard.text.v1` payload.

`clipboard.text` is the default TopicBus topic. `clipboard.text.v1` is the ClipboardNode application event name sent on the selected TopicBus topic route.

### Runtime Config

```go
type Config struct {
    Enabled        bool   `json:"enabled"`
    ParentEndpoint string `json:"parent_endpoint"`
    Topic          string `json:"topic"`
    Topics         []TopicRoute `json:"topics,omitempty"`
    MaxInlineBytes int    `json:"max_inline_bytes"`
    DeviceID       string `json:"device_id,omitempty"`
    DisplayName    string `json:"display_name,omitempty"`
    DeviceLabel    string `json:"device_label,omitempty"`
    AutoWatch      bool   `json:"auto_watch"`
    AutoApply      bool   `json:"auto_apply"`
    HistoryRetention string `json:"history_retention"`
    HistoryLimit   int    `json:"history_limit"`
}

type TopicRoute struct {
    Topic         string `json:"topic"`
    SyncToLocal   bool   `json:"sync_to_local"`
    SyncFromLocal bool   `json:"sync_from_local"`
}
```

Default `ParentEndpoint` should be `127.0.0.1:9000`.
Default `MaxInlineBytes` should be `65536`.
Default `DeviceID` should be `local-device`.
`DisplayName` falls back to `DeviceID` when blank.
`DeviceLabel` is retained as a legacy compatibility alias and should normalize to the display name.
Default `Enabled`, `AutoWatch`, and `AutoApply` should be conservative and off.
Default `HistoryRetention` should be `body`.
Default `HistoryLimit` should be `256`, and implementations should reject non-positive or unbounded limits.
Default `Topic` should be `clipboard.text`.
Default `Topics` should contain one route for `clipboard.text` with both `SyncToLocal` and `SyncFromLocal` enabled.
When `Topics` is present, it is the canonical route list and `Topic` is the primary compatibility alias. A legacy config with only `Topic` normalizes to one route with both directions enabled. An explicitly empty route list is invalid.

### UI-safe Status

```go
type Status struct {
    Connected bool
    LoggedIn bool
    ParentEndpoint string
    Enabled bool
    Topic string
    Topics []TopicRoute
    DeviceID string
    DisplayName string
    DeviceLabel string
    AutoWatch bool
    AutoApply bool
    HistoryRetention string
    HistoryLimit int
    PendingEventID string
    PendingTopic string
    PendingSize int
    PendingHashPrefix string
    LastAction string
    LastEventID string
    LastSize int
    LastHashPrefix string
    LastError string
}
```

Status must not include clipboard text.
Pending status should include only the pending event ID, topic, size, and hash prefix.

### UI Activity And Body History

Bridge activity events are metadata records by default. They must include topic metadata when available. They may include an optional `text` field only when normalized local config has `history_retention=body` and the runtime decision came from a successful inline text publish or apply path. Pending receive activity remains metadata-only until the user or auto-apply path writes the text to the local clipboard. UI code must store body text only in the bounded body history list and must keep activity/log views metadata-only. Mobile MethodChannel decision responses may carry the same scoped text only for local-published and remote-applied decisions; status, config, transfer, and pending metadata remain body-free.

### Bridge History Contract

```go
const ActionRestoreHistory = "restore_history"
const EventHistoryUpdated = "history.updated"

type HistoryEntry struct {
    ID          string `json:"id"`
    Kind        string `json:"kind"`
    Text        string `json:"text"`
    Topic       string `json:"topic,omitempty"`
    DeviceLabel string `json:"device_label,omitempty"`
    ByteSize    int    `json:"byte_size"`
    HashPrefix  string `json:"hash_prefix,omitempty"`
    TimestampMS int64  `json:"timestamp_ms"`
}
```

`history.updated` carries a JSON array of `HistoryEntry`. This event is the only live bridge event that intentionally carries persisted clipboard body text. Status, config, activity logs, transfer records, and pending metadata must remain body-free except for the already-scoped activity `text` field used to feed the history store.

### Transfer Manifest Draft

```json
{
  "version": 1,
  "event_id": "uuid",
  "origin_node": 12,
  "origin_instance_id": "runtime-uuid",
  "origin_device": "win-laptop",
  "content_type": "text/plain",
  "size": 1048577,
  "sha256": "hex",
  "transfer": {
    "protocol": "file",
    "ref": "opaque-existing-protocol-reference"
  },
  "ts": 1760000000000
}
```

The manifest is a ClipboardNode application payload. It must not require a TopicBus, Stream, File, Server, Proto, SDK, or SubProto wire change.

## Error Handling And Safety

- Empty topic: syncing cannot start; report invalid config.
- Empty `topics`: syncing cannot start; report invalid config.
- Duplicate route topics: reject config explicitly instead of merging silently.
- Remote event on unknown topic or route with `sync_to_local=false`: ignore and record metadata-only ignored decision.
- Local publish with no `sync_from_local=true` routes: ignore and record metadata-only ignored decision.
- Empty parent endpoint: connect/login cannot start; report invalid config.
- Register/login failure: keep transport cleanup best-effort, report UI-safe auth status, and do not subscribe or publish.
- Not connected or not logged in: do not publish or subscribe; report waiting state.
- TopicBus subscribe failure: keep disabled subscription state and retry on reconnect or explicit enable.
- Invalid remote JSON: drop and record validation error without retry.
- Oversize local text: reject and report oversize status without publishing body.
- Text digest: compute UTF-8 byte length and SHA-256 locally; reject invalid or oversize text and record validation error.
- Clipboard adapter failure: report error and do not mark the event as applied.
- Runtime shutdown: cancel watcher, unsubscribe best-effort, and close platform adapter.
- Mobile background clipboard access unavailable: present manual/share actions rather than reporting a fatal app error.
- Android foreground listener unavailable or gomobile binding missing: keep the explicit stub/native binding error visible and do not claim live automatic sync.
- Existing transfer protocol unavailable for large content: reject the transfer and show a UI-safe error without logging the body.
- Private topology assumption does not hold: require deployment guidance or future application-layer encryption; do not silently claim E2EE.

## Performance And Testing Strategy

- Keep dedupe windows bounded by count and age.
- Do not store unbounded event history.
- Hash text once per local and remote event path.
- Avoid logging full text after publish/apply.
- Retain body history only in the explicit bounded local history store; trim it to `history_limit` and clear it when retention is changed away from `body`.
- Retain at most 32 topic routes by default, reject empty or duplicate topics, and subscribe/unsubscribe by set difference when settings change.
- Unit test:
  - payload validation
  - locally computed text digest
  - local-origin rejection
  - duplicate event rejection
  - oversize rejection
  - disabled-state no-op behavior
  - remote apply loop suppression
  - topic route normalization and duplicate rejection
  - multi-topic subscribe/resubscribe/unsubscribe
  - local publish fan-out by `sync_from_local`
  - remote apply gating by `sync_to_local`
- Integration test with fake TopicBus and fake clipboard adapter before platform tests.
- UI state tests should verify no clipboard body appears in diagnostics and that restoring history promotes the selected body to the top.
- Platform tests should be split by adapter and not require live MyFlowHub protocol changes.

## Extensibility Design Points

- Keep compact `v=1` payload so future payloads can add application-level fields without requiring TopicBus wire changes.
- Add distinct `clipboard.transfer.v1` manifests for Stream/File handoff without changing TopicBus semantics.
- Keep Android adapter isolated from Windows adapter.
- Keep TopicBus client behind an interface so tests do not need a live hub.
- Keep UI cross-platform; runtime must remain testable headlessly.
- Keep route policy in runtime config so future per-topic labels, paused state, or filtering can extend `TopicRoute` without changing TopicBus.
- Keep application-layer encryption as an optional future module that wraps ClipboardNode payloads without changing MyFlowHub subprotocol wire contracts.

## Protocol Compatibility Rules

- No ClipboardNode-specific subprotocol may be introduced by this repository.
- No existing MyFlowHub subprotocol action, header, routing rule, permission rule, or wire format may be changed by this repository.
- Topic strings remain application configuration and are passed to TopicBus exactly as configured after ClipboardNode-local validation.
- `clipboard.text` is the default TopicBus topic, while `clipboard.text.v1` remains the ClipboardNode text event name.
- Topic route direction flags are local ClipboardNode policy only and do not create TopicBus ACLs.
- ClipboardNode event names and payload schemas are application contracts, not TopicBus protocol changes.
- Existing Stream/File references may be used only through their public contracts.

## Related Requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related External Specs

- `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/topicbus.md`
- `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/stream.md`

