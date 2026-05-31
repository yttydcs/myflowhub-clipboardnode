# Clipboard Sync Specification

## Overall Solution

ClipboardNode is a standalone cross-platform MyFlowHub clipboard application. It uses a shared synchronization engine, a cross-platform UI shell, and platform-specific clipboard adapters. The protocol strategy reuses existing MyFlowHub subprotocols and treats ClipboardNode messages as application-level payloads.

Small text synchronization uses TopicBus as an application event channel:

1. The node connects and logs in through the existing MyFlowHub client runtime.
2. When enabled, it subscribes to a configured clipboard topic/channel.
3. A platform clipboard watcher reports local text changes to the shared runtime.
4. The shared runtime validates, hashes, deduplicates, and publishes a `clipboard.text.v1` event.
5. The runtime receives remote TopicBus events, validates them, deduplicates them, and asks the platform adapter to write the text clipboard.

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
 - Expose UI-safe state without clipboard body leakage.

### `core/clipboard`

- Define platform adapter interfaces:
  - `ReadText`
  - `WriteText`
  - `WatchText`
  - `Close`
- Define clipboard event and write-result types.

### Cross-platform app UI

- Provide connection/login controls.
- Provide channel selection and sync policy settings.
- Provide desktop/mobile-appropriate send and receive controls.
- Provide recent transfer status without forced body exposure.
- Provide validation, transport, and platform permission status.
- Provide settings for inline size, auto-watch, auto-apply, and local retention.

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
3. Connect and login.
4. If `enabled=true`, subscribe to `topic`.
5. Start platform clipboard watcher.

### Local Clipboard To TopicBus

1. Desktop adapter emits local text automatically, or mobile/manual UI sends current/shared text.
2. Runtime checks enabled state.
3. Runtime normalizes text, computes UTF-8 byte length, and rejects empty or oversize input.
4. Runtime computes SHA-256.
5. Runtime ignores unchanged text using last local/remote hash state.
6. Runtime builds `ClipboardTextEventV1`.
7. Runtime publishes TopicBus application event:
   - topic: configured topic
   - name: `clipboard.text.v1`
   - payload: event JSON

### TopicBus To Local Clipboard

1. Runtime receives TopicBus publish.
2. Runtime checks topic and event name.
3. Runtime validates payload version, content type, encoding, size, hash, and text.
4. Runtime ignores local-origin or duplicate events.
5. Runtime writes text through the platform adapter.
6. Runtime records the write hash/event ID to suppress loops.

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
    Version          int    `json:"version"`
    EventID          string `json:"event_id"`
    OriginNode       uint32 `json:"origin_node"`
    OriginInstanceID string `json:"origin_instance_id"`
    OriginDevice     string `json:"origin_device,omitempty"`
    ContentType      string `json:"content_type"`
    Encoding         string `json:"encoding"`
    Size             int    `json:"size"`
    SHA256           string `json:"sha256"`
    Text             string `json:"text"`
    TS               int64  `json:"ts"`
}
```

### Runtime Config

```go
type Config struct {
    Enabled        bool   `json:"enabled"`
    Topic          string `json:"topic"`
    MaxInlineBytes int    `json:"max_inline_bytes"`
    DeviceLabel    string `json:"device_label,omitempty"`
    AutoWatch      bool   `json:"auto_watch"`
    AutoApply      bool   `json:"auto_apply"`
    HistoryRetention string `json:"history_retention"`
}
```

Default `MaxInlineBytes` should be `65536`.
Default `Enabled`, `AutoWatch`, `AutoApply`, and persistent body retention should be conservative and off unless the user enables them explicitly.

### UI-safe Status

```go
type Status struct {
    Connected bool
    LoggedIn bool
    Enabled bool
    Topic string
    DeviceLabel string
    AutoWatch bool
    AutoApply bool
    LastAction string
    LastEventID string
    LastSize int
    LastHashPrefix string
    LastError string
}
```

Status must not include clipboard text.

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
- Not connected or not logged in: do not publish or subscribe; report waiting state.
- TopicBus subscribe failure: keep disabled subscription state and retry on reconnect or explicit enable.
- Invalid remote JSON: drop and record validation error without retry.
- Oversize local text: reject and report oversize status without publishing body.
- Hash mismatch: reject remote event and record validation error.
- Clipboard adapter failure: report error and do not mark the event as applied.
- Runtime shutdown: cancel watcher, unsubscribe best-effort, and close platform adapter.
- Mobile background clipboard access unavailable: present manual/share actions rather than reporting a fatal app error.
- Existing transfer protocol unavailable for large content: reject the transfer and show a UI-safe error without logging the body.
- Private topology assumption does not hold: require deployment guidance or future application-layer encryption; do not silently claim E2EE.

## Performance And Testing Strategy

- Keep dedupe windows bounded by count and age.
- Do not store unbounded event history.
- Hash text once per local and remote event path.
- Avoid logging or retaining full text after publish/apply.
- Unit test:
  - payload validation
  - hash mismatch rejection
  - local-origin rejection
  - duplicate event rejection
  - oversize rejection
  - disabled-state no-op behavior
  - remote apply loop suppression
- Integration test with fake TopicBus and fake clipboard adapter before platform tests.
- UI state tests should verify no clipboard body appears in diagnostics.
- Platform tests should be split by adapter and not require live MyFlowHub protocol changes.

## Extensibility Design Points

- Keep `version=1` payload so future payloads can add large-content references.
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

