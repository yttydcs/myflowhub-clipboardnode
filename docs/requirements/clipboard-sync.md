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
- Subscribe to configured clipboard topics after successful node login.
- Default new configurations to the TopicBus topic `clipboard.text`.
- Allow users to configure multiple topic routes, each with independent remote-to-local and local-to-topic sync flags.
- Allow the user to configure the parent Hub / connection endpoint used by this node before connecting.
- Publish a clipboard event when local clipboard text changes and passes validation, only to topic routes with local-to-topic sync enabled.
- Apply a valid remote clipboard event to the local system clipboard only when the matching topic route has remote-to-local sync enabled.
- Prevent local publish and remote apply loops.
- Enforce a configurable inline text size limit.
- Keep clipboard text out of diagnostic logs, status, transfer records, and persistent configuration.
- Provide a local persisted clipboard body history that defaults to retaining the newest 256 text entries and can be configured or disabled by the user.
- Persist non-sensitive configuration, including topic routes, direction flags, local retention policy, and history limit.
- Keep send, receive, pending, transfer, and error records in logs or queue/status surfaces rather than using them as clipboard body history entries.
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
- Persisting clipboard body history in runtime config, diagnostic logs, status, transfer records, pending metadata, or any server-side store.

## Use Cases

1. A user enables ClipboardNode on trusted desktop devices connected to the same private MyFlowHub topology, both using the default `clipboard.text` topic.
2. The user copies a short text fragment on device A; device B receives the TopicBus event and updates its local clipboard according to local apply policy.
3. Device B's write to the system clipboard must not trigger an event loop back to device A.
4. A user copies or shares content on a mobile device; ClipboardNode offers manual send/share-sheet behavior that works within mobile OS clipboard restrictions.
5. A user wants to move large text, image, or file content; ClipboardNode sends a TopicBus manifest and uses an existing MyFlowHub transfer capability instead of splitting the body into TopicBus events.
6. A user copies a long text body that exceeds the configured inline limit; ClipboardNode rejects inline publish and either offers a supported transfer path or reports that it was not synchronized.
7. A device reconnects and re-subscribes to its configured clipboard topic routes; only new online events are synchronized unless a future explicit transfer/history feature is enabled.
8. A user disables sync; ClipboardNode stops publishing local changes, stops applying remote events, and stops platform clipboard watchers where applicable.
9. A user subscribes to several topics, allows one topic to update the local clipboard, and keeps another topic publish-only.
10. A user opens clipboard history, clicks an older text body, and ClipboardNode restores that text to the local clipboard while promoting that entry to the top of history.

## Functional Requirements

1. ClipboardNode must maintain a runtime state with connection status, auth state, parent Hub endpoint, enabled flag, active topic routes, max inline bytes, last local hash, recent event IDs, and last error.
2. ClipboardNode must treat `enabled=false` as the safe default.
3. ClipboardNode must require at least one non-empty TopicBus topic route before syncing.
4. ClipboardNode must normalize outbound text as UTF-8 and reject invalid or unsupported clipboard formats.
5. ClipboardNode must reject empty text by default, unless a future setting explicitly enables empty clipboard propagation.
6. ClipboardNode must reject outbound text whose UTF-8 byte length exceeds `max_inline_bytes`.
7. ClipboardNode must publish compact TopicBus text events with event identity, source tracking, and text body fields sufficient for dedupe and loop suppression.
8. ClipboardNode must ignore events whose `from` or `instance` fields identify the current runtime instance.
9. ClipboardNode must ignore duplicate `id` values within a bounded recent-event window.
10. ClipboardNode must compute text hashes locally and ignore events whose text hash matches a recent local write caused by the same remote event.
11. ClipboardNode must avoid logging clipboard text; logs may include event ID, topic, byte length, hash prefix, and status.
12. ClipboardNode must perform background registration/login after a transport connection is established, and must expose only UI-safe progress/error status.
13. ClipboardNode must resubscribe to every configured topic route after reconnect or login when enabled.
14. ClipboardNode must unsubscribe or stop applying remote events when disabled.
15. ClipboardNode must clear in-memory login state, node identity, subscriptions, and watchers when the user disconnects or the app shuts down.
16. ClipboardNode must expose enough state for UI or CLI diagnostics without revealing clipboard text.
17. ClipboardNode must maintain a separate local persisted body history for text clipboard entries when `history_retention=body`.
18. ClipboardNode must default `history_retention` to `body` and `history_limit` to `256`.
19. ClipboardNode must allow `history_retention=metadata` for metadata-only activity and `history_retention=none` for no local history retention.
20. ClipboardNode must validate `history_limit` as a positive bounded value and trim persisted body history to that limit.
21. Clicking a body history entry must restore that text to the local system clipboard and promote the restored text to the newest history position.
22. The newest body history entry should match the current clipboard text after a successful history restore.
23. ClipboardNode must validate `topics` as a non-empty, bounded list of unique topic routes.
24. Each topic route must include:
    - `topic`: trimmed TopicBus topic name.
    - `sync_to_local`: whether remote events from that topic may update the local clipboard or pending queue.
    - `sync_from_local`: whether local clipboard/manual text may publish to that topic.
25. Legacy single `topic` config remains a compatibility alias for the primary/default route; new defaults use `clipboard.text`.
26. ClipboardNode must ignore remote events from unknown topics and from topic routes with `sync_to_local=false`.
27. ClipboardNode must skip local publish when no route has `sync_from_local=true` and report that state without silently succeeding.
28. ClipboardNode must distinguish automatic desktop sync from mobile manual/share flows.
29. ClipboardNode must support at least these UI surfaces:
    - connection and login status;
    - parent Hub / endpoint configuration;
    - device identity, display name, and multi-topic route selection;
    - sync enable/disable;
    - local clipboard watch/apply policies;
    - inline size limit;
    - manual send current clipboard;
    - recent transfer status without forced body exposure;
    - error and validation status.
30. ClipboardNode must keep the authenticated device identity configurable separately from the user-visible display name.
31. ClipboardNode must clear or invalidate local login/session data when the configured authenticated device identity changes.
32. ClipboardNode must not treat publish success as remote apply success.
33. ClipboardNode must persist non-sensitive runtime configuration to a local config file, including `topics`, `history_retention`, and `history_limit`.
34. ClipboardNode must persist clipboard body history only in a dedicated local history store, separate from config and status payloads.
35. ClipboardNode must clear persisted clipboard body history when `history_retention` changes away from `body` or when the user clears recent history.
36. ClipboardNode must not create new MyFlowHub protocol actions or rely on server-side ClipboardNode-specific behavior.

## Non-functional Requirements

- Safety: default disabled, explicit enablement, bounded payload size, no plaintext logs, no implicit mobile background clipboard access.
- Privacy: clipboard text may appear only in the explicit local persisted body history, bounded by `history_limit`; it must not be logged, written to status, written to config, or sent to server-side history.
- Security model: private MyFlowHub topology plus authenticated node identity; no pairing or room key required by default.
- Reliability: reject invalid state explicitly and report errors.
- Performance: avoid repeated full-text processing; compute hash once per event path.
- Compatibility: use existing TopicBus publish/subscribe semantics and existing transfer protocols without protocol changes; keep legacy `topic` config as the primary route alias.
- Portability: keep platform-specific clipboard API usage behind a narrow adapter boundary.
- Maintainability: shared sync logic belongs in `core/` or `engine/`; host details belong in platform-specific adapters; UI logic belongs in the cross-platform app layer.

## Inputs / Outputs

### Inputs

- Local clipboard text from a platform adapter.
- TopicBus publish events on configured clipboard topic routes.
- Runtime config:
  - `enabled`
  - `parent_endpoint`
  - `topic`
  - `topics`
  - `max_inline_bytes`
  - `device_id`
  - `display_name`
  - `device_label` as a legacy compatibility alias
  - `auto_watch`
  - `auto_apply`
  - `history_retention`
  - `history_limit`
  - connection/auth defaults

### Outputs

- TopicBus publish event for accepted local clipboard changes.
- Local clipboard write for accepted remote events.
- Existing transfer protocol request or reference for large supported content.
- Runtime status and error records without clipboard body text.
- Local persisted clipboard body history entries when body history retention is enabled.
- Logs without clipboard body content.

### Topic Routes

New configs should use `topics` as the canonical route list:

```json
[
  {
    "topic": "clipboard.text",
    "sync_to_local": true,
    "sync_from_local": true
  }
]
```

The default topic is `clipboard.text`. The legacy scalar `topic` field remains the primary-topic compatibility alias and should normalize to the first route when `topics` is present. Topic route names must be trimmed, non-empty, unique, and bounded.

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

`clipboard.text` is the default TopicBus topic. `clipboard.text.v1` is the ClipboardNode application event name sent on the selected TopicBus topic route.

ClipboardNode computes UTF-8 byte size and SHA-256 locally from `text`; those
values are status and validation metadata, not required text-payload fields.

Future large-content announcements should use a distinct ClipboardNode event name such as `clipboard.transfer.v1` and carry only a manifest or reference to an existing Stream/File transfer, not a new subprotocol payload.

## Boundary Exceptions

- TopicBus publish does not ACK delivery. ClipboardNode must not present event publish as confirmed remote apply.
- TopicBus has no offline replay. ClipboardNode must not claim synchronization for events published while a device is disconnected or unsubscribed.
- TopicBus has no permission control in the current spec. ClipboardNode must rely on explicit topic configuration, authenticated nodes, and the private MyFlowHub topology by default.
- Topic routes are local ClipboardNode policy. They do not change TopicBus permission semantics and must not be presented as server-side ACLs.
- Text larger than the inline limit must not be split into multiple TopicBus events.
- Mobile platforms may restrict background clipboard access; ClipboardNode must provide manual/share flows instead of promising desktop-equivalent automatic watching.
- No ClipboardNode-specific Server, Proto, SDK, or SubProto changes are allowed in the default architecture.

## Acceptance Criteria

1. Requirements clearly state that ClipboardNode is an independent node application.
2. Requirements clearly state that the product is a full cross-platform UI application, not only a headless node.
3. Requirements clearly state that TopicBus is used only for ClipboardNode application events and small inline text.
4. Requirements clearly state the safe default is disabled.
5. Requirements clearly state clipboard text is persisted only in the dedicated local body-history store and is not logged, written to config, or written to status.
6. Requirements define loop prevention using compact origin fields, event IDs, and locally computed text hashes.
7. Requirements define an inline size limit and reject oversize text.
8. Requirements state that existing MyFlowHub protocols must be reused without modification.
9. Requirements state that private topology and node identity are the default security model, without pairing or room keys.
10. Requirements state that mobile clipboard limitations require manual/share flows.
11. Requirements state that local body history defaults to 256 persisted text entries and is configurable through `history_limit`.
12. Requirements state that the default TopicBus topic is `clipboard.text` and that multiple topic routes may independently control remote-to-local and local-to-topic sync.
13. Requirements state that restoring a history entry writes the selected text to the local clipboard and promotes it to the newest history position.

## Risks

- TopicBus currently has no publish ACK and no permission control, so ClipboardNode remains suitable for trusted private topology and online best-effort sync.
- Clipboard APIs are platform-specific and may require UI-thread, foreground-service, or permission handling.
- Automatic remote clipboard writes are sensitive; defaults and UI text must make enablement explicit.
- Large clipboard contents can stress JSON parsing and fanout; the inline limit is mandatory for phase 1.
- Flutter/Dart tooling is required for the cross-platform UI plan; if unavailable, implementation cannot proceed past planning without installing or configuring that toolchain.

