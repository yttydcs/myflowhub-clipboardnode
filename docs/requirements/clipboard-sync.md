# Clipboard Sync Requirements

## Goal

ClipboardNode provides a standalone, cross-platform MyFlowHub clipboard application for quickly copying text and supported content between trusted devices in a private MyFlowHub topology.

The product must be designed as a complete engineered application, not a throwaway MVP. It must provide a usable UI, a shared synchronization engine, platform-specific clipboard integrations, and a protocol strategy that reuses existing MyFlowHub capabilities without adding or modifying subprotocol wire semantics.

Security is based on the private MyFlowHub network, authenticated node identity, explicit local enablement, and per-device settings. The first full product does not require device pairing, room keys, or application-layer end-to-end encryption. Future encryption can be added only if ClipboardNode must run across an untrusted hub or a multi-tenant topology.

## Scope

### Must

- Run as an independent node application repository.
- Provide a cross-platform UI for desktop and mobile targets.
- Support Windows as the first runnable desktop target, while keeping macOS, Linux, Android, and iOS in the repository architecture.
- Keep shared synchronization logic out of platform UI layers.
- Synchronize plain UTF-8 text clipboard content through TopicBus application events.
- Reuse existing MyFlowHub subprotocols only; do not add a Clipboard subprotocol and do not change TopicBus, Stream, File, Server, Proto, SDK, or SubProto wire contracts unless a later explicit cross-repo workflow approves it.
- Treat TopicBus payload as ClipboardNode application JSON, not as a protocol extension.
- Use existing Stream or File capabilities for future large text, image, file, or binary transfers; TopicBus may carry only small inline content or transfer manifests.
- Require explicit user enablement before reading from or writing to the system clipboard.
- Run node registration/login in the background as part of the connect lifecycle; the primary UI must not require separate manual register or login actions.
- Clear local login/session state in the background when the user disconnects or the app shuts down.
- Subscribe to a configured clipboard topic after successful node login.
- Allow the user to configure the parent Hub / connection endpoint used by this node before connecting.
- Publish a clipboard event when local clipboard text changes and passes validation.
- Apply a valid remote clipboard event to the local system clipboard.
- Prevent local publish and remote apply loops.
- Enforce a configurable inline text size limit.
- Keep clipboard text out of logs, status history, and persistent configuration.
- Surface sync status, validation failures, and transport errors without silently swallowing them.
- Provide device/channel status, recent transfer status, settings, manual send, receive/apply controls, and clear privacy controls in the UI.
- Respect platform limitations: desktop can support automatic clipboard watching; mobile must support manual send/share-sheet flows and cannot rely on unrestricted background clipboard watching.

### Optional

- macOS and Linux tray/menu bar integration.
- Android foreground service and share-sheet integration.
- iOS share extension.
- Local bounded recent-transfer list with body visibility controlled by user settings.
- Future application-layer encryption for untrusted topology deployments.

### Not In Scope

- Creating a new MyFlowHub subprotocol.
- Modifying existing TopicBus, Stream, File, Auth, Management, Proto, SDK, Server, or SubProto wire behavior.
- Device pairing or room-key setup for the private-network default model.
- Application-layer end-to-end encryption in the first full product.
- Offline queueing, replay, or history recovery.
- End-to-end delivery acknowledgement beyond TopicBus subscribe control responses.
- Modifying TopicBus permission semantics.
- Persisting full clipboard history without explicit local user enablement and local retention controls.

## Use Cases

1. A user enables ClipboardNode on trusted desktop devices connected to the same private MyFlowHub topology, both using the same clipboard channel.
2. The user copies a short text fragment on device A; device B receives the TopicBus event and updates its local clipboard according to local apply policy.
3. Device B's write to the system clipboard must not trigger an event loop back to device A.
4. A user copies or shares content on a mobile device; ClipboardNode offers manual send/share-sheet behavior that works within mobile OS clipboard restrictions.
5. A user wants to move large text, image, or file content; ClipboardNode sends a TopicBus manifest and uses an existing MyFlowHub transfer capability instead of splitting the body into TopicBus events.
6. A user copies a long text body that exceeds the configured inline limit; ClipboardNode rejects inline publish and either offers a supported transfer path or reports that it was not synchronized.
7. A device reconnects and re-subscribes to its clipboard channel; only new online events are synchronized unless a future explicit transfer/history feature is enabled.
8. A user disables sync; ClipboardNode stops publishing local changes, stops applying remote events, and stops platform clipboard watchers where applicable.

## Functional Requirements

1. ClipboardNode must maintain a runtime state with connection status, auth state, parent Hub endpoint, enabled flag, active topic, max inline bytes, last local hash, recent event IDs, and last error.
2. ClipboardNode must treat `enabled=false` as the safe default.
3. ClipboardNode must require a non-empty TopicBus topic or channel before syncing.
4. ClipboardNode must normalize outbound text as UTF-8 and reject invalid or unsupported clipboard formats.
5. ClipboardNode must reject empty text by default, unless a future setting explicitly enables empty clipboard propagation.
6. ClipboardNode must reject outbound text whose UTF-8 byte length exceeds `max_inline_bytes`.
7. ClipboardNode must publish compact TopicBus text events with event identity, source tracking, and text body fields sufficient for dedupe and loop suppression.
8. ClipboardNode must ignore events whose `from` or `instance` fields identify the current runtime instance.
9. ClipboardNode must ignore duplicate `id` values within a bounded recent-event window.
10. ClipboardNode must compute text hashes locally and ignore events whose text hash matches a recent local write caused by the same remote event.
11. ClipboardNode must avoid logging clipboard text; logs may include event ID, topic, byte length, hash prefix, and status.
12. ClipboardNode must perform background registration/login after a transport connection is established, and must expose only UI-safe progress/error status.
13. ClipboardNode must resubscribe after reconnect or login when enabled.
14. ClipboardNode must unsubscribe or stop applying remote events when disabled.
15. ClipboardNode must clear in-memory login state, node identity, subscriptions, and watchers when the user disconnects or the app shuts down.
16. ClipboardNode must expose enough state for UI or CLI diagnostics without revealing clipboard text.
17. ClipboardNode must distinguish automatic desktop sync from mobile manual/share flows.
18. ClipboardNode must support at least these UI surfaces:
    - connection and login status;
    - parent Hub / endpoint configuration;
    - device identity, display name, and channel selection;
    - sync enable/disable;
    - local clipboard watch/apply policies;
    - inline size limit;
    - manual send current clipboard;
    - recent transfer status without forced body exposure;
    - error and validation status.
19. ClipboardNode must keep the authenticated device identity configurable separately from the user-visible display name.
20. ClipboardNode must clear or invalidate local login/session data when the configured authenticated device identity changes.
21. ClipboardNode must not treat publish success as remote apply success.
22. ClipboardNode must not create new MyFlowHub protocol actions or rely on server-side ClipboardNode-specific behavior.

## Non-functional Requirements

- Safety: default disabled, explicit enablement, bounded payload size, no plaintext logs, no implicit mobile background clipboard access.
- Privacy: clipboard text must be transient in memory unless the user explicitly enables local bounded retention.
- Security model: private MyFlowHub topology plus authenticated node identity; no pairing or room key required by default.
- Reliability: reject invalid state explicitly and report errors.
- Performance: avoid repeated full-text processing; compute hash once per event path.
- Compatibility: use existing TopicBus publish/subscribe semantics and existing transfer protocols without protocol changes.
- Portability: keep platform-specific clipboard API usage behind a narrow adapter boundary.
- Maintainability: shared sync logic belongs in `core/` or `engine/`; host details belong in platform-specific adapters; UI logic belongs in the cross-platform app layer.

## Inputs / Outputs

### Inputs

- Local clipboard text from a platform adapter.
- TopicBus publish events on the configured clipboard topic.
- Runtime config:
  - `enabled`
  - `parent_endpoint`
  - `topic`
  - `max_inline_bytes`
  - `device_id`
  - `display_name`
  - `device_label` as a legacy compatibility alias
  - `auto_watch`
  - `auto_apply`
  - `history_retention`
  - connection/auth defaults

### Outputs

- TopicBus publish event for accepted local clipboard changes.
- Local clipboard write for accepted remote events.
- Existing transfer protocol request or reference for large supported content.
- Runtime status and error records.
- Logs without clipboard body content.

## Topic Event Payload

Initial small-text payload shape:

```json
{
  "v": 1,
  "id": "uuid",
  "from": 12,
  "instance": "runtime-uuid",
  "device": "win-laptop",
  "text": "hello"
}
```

The event name should be `clipboard.text.v1`.

ClipboardNode computes UTF-8 byte size and SHA-256 locally from `text`; those
values are status and validation metadata, not required text-payload fields.

Future large-content announcements should use a distinct ClipboardNode event name such as `clipboard.transfer.v1` and carry only a manifest or reference to an existing Stream/File transfer, not a new subprotocol payload.

## Boundary Exceptions

- TopicBus publish does not ACK delivery. ClipboardNode must not present event publish as confirmed remote apply.
- TopicBus has no offline replay. ClipboardNode must not claim synchronization for events published while a device is disconnected or unsubscribed.
- TopicBus has no permission control in the current spec. ClipboardNode must rely on explicit topic configuration, authenticated nodes, and the private MyFlowHub topology by default.
- Text larger than the inline limit must not be split into multiple TopicBus events.
- Mobile platforms may restrict background clipboard access; ClipboardNode must provide manual/share flows instead of promising desktop-equivalent automatic watching.
- No ClipboardNode-specific Server, Proto, SDK, or SubProto changes are allowed in the default architecture.

## Acceptance Criteria

1. Requirements clearly state that ClipboardNode is an independent node application.
2. Requirements clearly state that the product is a full cross-platform UI application, not only a headless node.
3. Requirements clearly state that TopicBus is used only for ClipboardNode application events and small inline text.
4. Requirements clearly state the safe default is disabled.
5. Requirements clearly state no clipboard text is persisted or logged.
6. Requirements define loop prevention using compact origin fields, event IDs, and locally computed text hashes.
7. Requirements define an inline size limit and reject oversize text.
8. Requirements state that existing MyFlowHub protocols must be reused without modification.
9. Requirements state that private topology and node identity are the default security model, without pairing or room keys.
10. Requirements state that mobile clipboard limitations require manual/share flows.

## Risks

- TopicBus currently has no publish ACK and no permission control, so ClipboardNode remains suitable for trusted private topology and online best-effort sync.
- Clipboard APIs are platform-specific and may require UI-thread, foreground-service, or permission handling.
- Automatic remote clipboard writes are sensitive; defaults and UI text must make enablement explicit.
- Large clipboard contents can stress JSON parsing and fanout; the inline limit is mandatory for phase 1.
- Flutter/Dart tooling is required for the cross-platform UI plan; if unavailable, implementation cannot proceed past planning without installing or configuring that toolchain.

