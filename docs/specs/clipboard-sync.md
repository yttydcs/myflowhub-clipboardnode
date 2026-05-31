# Clipboard Sync Specification

## Overall Solution

ClipboardNode is a standalone MyFlowHub node application. The first phase uses TopicBus as a small text event channel:

1. The node connects and logs in through the existing MyFlowHub client runtime.
2. When enabled, it subscribes to a configured clipboard topic.
3. A platform clipboard watcher reports local text changes to the shared runtime.
4. The shared runtime validates, hashes, deduplicates, and publishes a `clipboard.text.v1` event.
5. The runtime receives remote TopicBus events, validates them, deduplicates them, and asks the platform adapter to write the text clipboard.

TopicBus is selected for phase 1 because clipboard text sync is event-shaped, low-frequency, and does not need per-delivery ACK for the MVP. Stream or File remains the future path for large, binary, resumable, or guaranteed content.

## Alternatives Considered

- New subprotocol:
  - Rejected for phase 1 because no new wire semantics are required for small online text events.
- Stream text profile:
  - Better for bounded long text, ACK, backpressure, and future large payloads.
  - Deferred because phase 1 should stay small and reuse existing TopicBus.
- Embedding in MyFlowHub-Win:
  - Rejected because clipboard sync is a background platform node, not a Win console feature.
- Reusing MetricsNode:
  - Rejected because clipboard sync has different privacy and runtime semantics.

## Module Responsibilities

### `core/runtime`

- Own connection/auth orchestration.
- Own TopicBus subscribe, resubscribe, publish, and event receive.
- Own sync state, enablement, size limit, dedupe windows, and status reporting.
- Never call platform clipboard APIs directly.

### `core/clipboard`

- Define platform adapter interfaces:
  - `ReadText`
  - `WriteText`
  - `WatchText`
  - `Close`
- Define clipboard event and write-result types.

### `core/configstore`

- Persist non-sensitive runtime config.
- Must not persist clipboard text or event bodies.

### `windows`

- Wails host, tray or small control UI, Windows clipboard adapter, and Windows build scripts.
- Ensure clipboard operations obey Windows thread and message-loop requirements.

### `android`

- Android host, foreground service, permission and lifecycle handling, and clipboard adapter.
- Not required in the first implementation task unless explicitly planned.

### `nodemobile`

- gomobile bridge for Android shared runtime access.

## Data / Call Flow

### Startup

1. Load config.
2. Initialize runtime with clipboard adapter and TopicBus client.
3. Connect and login.
4. If `enabled=true`, subscribe to `topic`.
5. Start platform clipboard watcher.

### Local Clipboard To TopicBus

1. Adapter emits local text.
2. Runtime checks enabled state.
3. Runtime normalizes text, computes UTF-8 byte length, and rejects empty or oversize input.
4. Runtime computes SHA-256.
5. Runtime ignores unchanged text using last local/remote hash state.
6. Runtime builds `ClipboardTextEventV1`.
7. Runtime publishes TopicBus event:
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
}
```

Default `MaxInlineBytes` should be `65536`.

## Error Handling And Safety

- Empty topic: syncing cannot start; report invalid config.
- Not connected or not logged in: do not publish or subscribe; report waiting state.
- TopicBus subscribe failure: keep disabled subscription state and retry on reconnect or explicit enable.
- Invalid remote JSON: drop and record validation error without retry.
- Oversize local text: reject and report oversize status without publishing body.
- Hash mismatch: reject remote event and record validation error.
- Clipboard adapter failure: report error and do not mark the event as applied.
- Runtime shutdown: cancel watcher, unsubscribe best-effort, and close platform adapter.

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

## Extensibility Design Points

- Keep `version=1` payload so future payloads can add large-content references.
- Add `transfer` metadata later for Stream/File handoff without changing the phase 1 event name.
- Keep Android adapter isolated from Windows adapter.
- Keep TopicBus client behind an interface so tests do not need a live hub.
- Keep UI optional; runtime must be testable headlessly.

## Related Requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related External Specs

- `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/topicbus.md`
- `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/stream.md`

