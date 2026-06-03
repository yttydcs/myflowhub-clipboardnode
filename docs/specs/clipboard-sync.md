# Clipboard Sync Specification

## Overall Solution

ClipboardNode is a standalone cross-platform MyFlowHub clipboard application. It uses a shared synchronization engine, a cross-platform UI shell, and platform-specific clipboard adapters. The protocol strategy reuses existing MyFlowHub subprotocols and treats ClipboardNode messages as application-level payloads.

Small text synchronization uses TopicBus as an application event channel:

1. The node connects to the configured parent Hub / endpoint, then performs registration/login in the background through the existing MyFlowHub auth flow.
2. When enabled, it subscribes to a configured clipboard topic/channel.
3. A platform clipboard watcher reports local text changes to the shared runtime.
4. The shared runtime validates, hashes, deduplicates, and publishes a compact `clipboard.text.v1` event.
5. The runtime receives remote compact TopicBus events, validates them, computes derived metadata, deduplicates them, and asks the platform adapter to write the text clipboard.

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
- Own sync state, enablement, channel, size limit, dedupe windows, status reporting, and transfer manifest handling.
- Never call platform clipboard APIs directly.
- Expose UI-safe status without clipboard body leakage.
- Emit successful inline text decisions with an in-memory-only text field that the UI bridge may use for explicit local body history.

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
- Provide channel selection and sync policy settings.
- Provide desktop/mobile-appropriate send and receive controls.
- Provide recent transfer status without forced body exposure.
- Provide validation, transport, and platform permission status.
- Provide settings for inline size, auto-watch, auto-apply, local retention mode, and body history length.
- Maintain a body history list separate from activity/log metadata; body history is local in-memory UI state and defaults to the newest 256 text entries.

### Platform adapters

- Desktop adapters own automatic clipboard watching, local apply, tray/menu integration, notifications, and optional autostart.
- Mobile adapters own share-sheet/manual send, foreground notification/service behavior where allowed, local apply controls, and platform permission explanations.

### `core/configstore`

- Persist non-sensitive runtime config.
- Must not persist clipboard text or event bodies.

### `windows`

- Windows desktop host, tray or menu integration, Windows clipboard adapter, and Windows build scripts.
- Ensure clipboard operations obey Windows thread and message-loop requirements.

### `android`

- Android host, share-sheet integration, foreground notification/service where appropriate, permission/lifecycle handling, and clipboard adapter.
- Must not rely on unrestricted background clipboard watching.

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
4. Register or rebind the local device identity if needed.
5. Login with the local device identity.
6. If `enabled=true`, subscribe to `topic`.
7. Start platform clipboard watcher.

### Disconnect / Shutdown

1. Stop platform clipboard watchers.
2. Unsubscribe best-effort from the active topic.
3. Clear in-memory login state and node identity.
4. Close the transport/session.
5. Preserve non-sensitive configuration such as parent endpoint, topic, device label, size limits, and local policy.

### Local Clipboard To TopicBus

1. Desktop adapter emits local text automatically, or mobile/manual UI sends current/shared text.
2. Runtime checks enabled state.
3. Runtime normalizes text, computes UTF-8 byte length, and rejects empty or oversize input.
4. Runtime computes SHA-256.
5. Runtime ignores unchanged text using last local/remote hash state.
6. Runtime builds compact `ClipboardTextEventV1`.
7. Runtime publishes TopicBus application event:
   - topic: configured topic
   - name: `clipboard.text.v1`
   - payload: event JSON
8. Runtime emits a successful local-publish decision with metadata and in-memory text for bridge-side body history when enabled.

### TopicBus To Local Clipboard

1. Runtime receives TopicBus publish.
2. Runtime checks topic and event name.
3. Runtime validates payload version, identity fields, and text.
4. Runtime computes UTF-8 byte size and SHA-256 from the text.
5. Runtime ignores local-origin or duplicate events.
6. Runtime either writes text through the platform adapter or records the event as pending when auto-apply is off.
7. Runtime records the write hash/event ID to suppress loops after successful local apply.
8. Runtime emits a pending/applied decision with metadata and in-memory text for bridge-side body history when enabled.

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

### Runtime Config

```go
type Config struct {
    Enabled        bool   `json:"enabled"`
    ParentEndpoint string `json:"parent_endpoint"`
    Topic          string `json:"topic"`
    MaxInlineBytes int    `json:"max_inline_bytes"`
    DeviceLabel    string `json:"device_label,omitempty"`
    AutoWatch      bool   `json:"auto_watch"`
    AutoApply      bool   `json:"auto_apply"`
    HistoryRetention string `json:"history_retention"`
    HistoryLimit   int    `json:"history_limit"`
}
```

Default `ParentEndpoint` should be `127.0.0.1:9000`.
Default `MaxInlineBytes` should be `65536`.
Default `Enabled`, `AutoWatch`, and `AutoApply` should be conservative and off.
Default `HistoryRetention` should be `body`.
Default `HistoryLimit` should be `256`, and implementations should reject non-positive or unbounded limits.

### UI-safe Status

```go
type Status struct {
    Connected bool
    LoggedIn bool
    ParentEndpoint string
    Enabled bool
    Topic string
    DeviceLabel string
    AutoWatch bool
    AutoApply bool
    HistoryRetention string
    HistoryLimit int
    LastAction string
    LastEventID string
    LastSize int
    LastHashPrefix string
    LastError string
}
```

Status must not include clipboard text.

### UI Activity And Body History

Bridge activity events are metadata records by default. They may include an optional `text` field only when normalized local config has `history_retention=body` and the runtime decision came from a successful inline text publish, pending receive, or apply path. UI code must store that text only in the bounded body history list and must keep activity/log views metadata-only.

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
- Existing transfer protocol unavailable for large content: reject the transfer and show a UI-safe error without logging the body.
- Private topology assumption does not hold: require deployment guidance or future application-layer encryption; do not silently claim E2EE.

## Performance And Testing Strategy

- Keep dedupe windows bounded by count and age.
- Do not store unbounded event history.
- Hash text once per local and remote event path.
- Avoid logging full text after publish/apply.
- Retain body history only in the explicit bounded UI list; trim it to `history_limit` and clear it when retention is changed away from `body`.
- Unit test:
  - payload validation
  - locally computed text digest
  - local-origin rejection
  - duplicate event rejection
  - oversize rejection
  - disabled-state no-op behavior
  - remote apply loop suppression
- Integration test with fake TopicBus and fake clipboard adapter before platform tests.
- UI state tests should verify no clipboard body appears in diagnostics.
- Platform tests should be split by adapter and not require live MyFlowHub protocol changes.

## Extensibility Design Points

- Keep compact `v=1` payload so future payloads can add application-level fields without requiring TopicBus wire changes.
- Add distinct `clipboard.transfer.v1` manifests for Stream/File handoff without changing TopicBus semantics.
- Keep Android adapter isolated from Windows adapter.
- Keep TopicBus client behind an interface so tests do not need a live hub.
- Keep UI cross-platform; runtime must remain testable headlessly.
- Keep application-layer encryption as an optional future module that wraps ClipboardNode payloads without changing MyFlowHub subprotocol wire contracts.

## Protocol Compatibility Rules

- No ClipboardNode-specific subprotocol may be introduced by this repository.
- No existing MyFlowHub subprotocol action, header, routing rule, permission rule, or wire format may be changed by this repository.
- Topic strings remain application configuration and are passed to TopicBus exactly as configured after ClipboardNode-local validation.
- ClipboardNode event names and payload schemas are application contracts, not TopicBus protocol changes.
- Existing Stream/File references may be used only through their public contracts.

## Related Requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related External Specs

- `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/topicbus.md`
- `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/stream.md`

