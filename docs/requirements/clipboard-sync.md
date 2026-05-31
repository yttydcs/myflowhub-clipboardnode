# Clipboard Sync Requirements

## Goal

ClipboardNode provides a standalone MyFlowHub node application for online clipboard synchronization between trusted devices.

The first phase focuses on low-latency text clipboard events over the existing TopicBus protocol. It must not introduce a new subprotocol, modify TopicBus wire semantics, or embed the sync runtime into MyFlowHub-Win, MyFlowHub-Android, Server, or SubProto.

## Scope

### Must

- Run as an independent node application repository.
- Support Windows as the first host target.
- Keep Android host support in the repository design so it can be added without moving shared logic.
- Synchronize plain UTF-8 text clipboard content through TopicBus.
- Require explicit user enablement before reading from or writing to the system clipboard.
- Subscribe to a configured clipboard topic after successful node login.
- Publish a clipboard event when local clipboard text changes and passes validation.
- Apply a valid remote clipboard event to the local system clipboard.
- Prevent local publish and remote apply loops.
- Enforce a configurable inline text size limit.
- Keep clipboard text out of logs, status history, and persistent configuration.
- Surface sync status, validation failures, and transport errors without silently swallowing them.

### Optional

- Android host and gomobile bridge.
- Manual "send current clipboard" action.
- A small local UI for connection status, enablement, topic, and limits.
- Topic pairing helpers for generating or copying a shared group topic.
- Future large-content handoff through Stream or File.

### Not In Scope

- Image clipboard synchronization.
- File clipboard synchronization.
- Rich text, HTML, RTF, or application-specific clipboard formats.
- Offline queueing, replay, or history recovery.
- End-to-end delivery acknowledgement beyond TopicBus subscribe control responses.
- Modifying TopicBus permission semantics.
- Persisting clipboard history.

## Use Cases

1. A user enables ClipboardNode on two trusted devices connected to the same MyFlowHub topology, both using the same clipboard topic.
2. The user copies a short text fragment on device A; device B receives the TopicBus event and updates its local clipboard.
3. Device B's write to the system clipboard must not trigger an event loop back to device A.
4. A user copies a long text body that exceeds the configured inline limit; ClipboardNode rejects the event and reports that it was not synchronized.
5. A device reconnects and re-subscribes to its clipboard topic; only new online events are synchronized.
6. A user disables sync; ClipboardNode stops publishing local changes and stops applying remote events.

## Functional Requirements

1. ClipboardNode must maintain a runtime state with connection status, auth state, enabled flag, active topic, max inline bytes, last local hash, recent event IDs, and last error.
2. ClipboardNode must treat `enabled=false` as the safe default.
3. ClipboardNode must require a non-empty TopicBus topic before syncing.
4. ClipboardNode must normalize outbound text as UTF-8 and reject invalid or unsupported clipboard formats.
5. ClipboardNode must reject empty text by default, unless a future setting explicitly enables empty clipboard propagation.
6. ClipboardNode must reject outbound text whose UTF-8 byte length exceeds `max_inline_bytes`.
7. ClipboardNode must publish TopicBus events with event metadata sufficient for dedupe and source tracking.
8. ClipboardNode must ignore events whose `origin_node` or `origin_instance_id` identify the current runtime instance.
9. ClipboardNode must ignore duplicate `event_id` values within a bounded recent-event window.
10. ClipboardNode must ignore events whose text hash matches a recent local write caused by the same remote event.
11. ClipboardNode must avoid logging clipboard text; logs may include event ID, topic, byte length, hash prefix, and status.
12. ClipboardNode must resubscribe after reconnect or login when enabled.
13. ClipboardNode must unsubscribe or stop applying remote events when disabled.
14. ClipboardNode must expose enough state for UI or CLI diagnostics without revealing clipboard text.

## Non-functional Requirements

- Safety: default disabled, explicit enablement, bounded payload size, no plaintext logs.
- Privacy: clipboard text must be transient in memory and not persisted as history.
- Reliability: reject invalid state explicitly and report errors.
- Performance: avoid repeated full-text processing; compute hash once per event path.
- Compatibility: use existing TopicBus publish/subscribe semantics without protocol changes.
- Portability: keep platform-specific clipboard API usage behind a narrow adapter boundary.
- Maintainability: shared sync logic belongs in `core/`; host details belong in `windows/` and `android/`.

## Inputs / Outputs

### Inputs

- Local clipboard text from a platform adapter.
- TopicBus publish events on the configured clipboard topic.
- Runtime config:
  - `enabled`
  - `topic`
  - `max_inline_bytes`
  - `device_label`
  - connection/auth defaults

### Outputs

- TopicBus publish event for accepted local clipboard changes.
- Local clipboard write for accepted remote events.
- Runtime status and error records.
- Logs without clipboard body content.

## Topic Event Payload

Initial payload shape:

```json
{
  "version": 1,
  "event_id": "uuid",
  "origin_node": 12,
  "origin_instance_id": "runtime-uuid",
  "origin_device": "win-laptop",
  "content_type": "text/plain",
  "encoding": "utf-8",
  "size": 42,
  "sha256": "hex",
  "text": "hello",
  "ts": 1760000000000
}
```

The event name should be `clipboard.text.v1`.

## Boundary Exceptions

- TopicBus publish does not ACK delivery. ClipboardNode must not present event publish as confirmed remote apply.
- TopicBus has no offline replay. ClipboardNode must not claim synchronization for events published while a device is disconnected or unsubscribed.
- TopicBus has no permission control in the current spec. ClipboardNode must rely on explicit topic configuration and trusted topology for phase 1.
- Text larger than the inline limit must not be split into multiple TopicBus events in phase 1.

## Acceptance Criteria

1. Requirements clearly state that ClipboardNode is an independent node application.
2. Requirements clearly state that phase 1 synchronizes only plain text.
3. Requirements clearly state that TopicBus is used only for online small text events.
4. Requirements clearly state the safe default is disabled.
5. Requirements clearly state no clipboard text is persisted or logged.
6. Requirements define loop prevention using origin metadata, event IDs, and text hashes.
7. Requirements define an inline size limit and reject oversize text.
8. Requirements state that images, files, rich text, offline replay, and guaranteed delivery are out of scope.

## Risks

- TopicBus currently has no publish ACK and no permission control, so this feature is suitable only for trusted topics and online best-effort sync.
- Clipboard APIs are platform-specific and may require UI-thread, foreground-service, or permission handling.
- Automatic remote clipboard writes are sensitive; defaults and UI text must make enablement explicit.
- Large clipboard contents can stress JSON parsing and fanout; the inline limit is mandatory for phase 1.

