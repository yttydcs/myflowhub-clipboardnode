# Startup Subscribe Timeout Half Connected

## Summary

ClipboardNode can appear authenticated locally while the current TCP session is not bound to the saved node id if a persisted auth snapshot is treated as a live login. In that state, TopicBus subscribe may time out and MyFlowHub-Win management views may not list the node.

## Lookup Hints

- `subscribe clipboard topic: context deadline exceeded`
- `auth_snapshot.json`
- `logged_in=true`
- `node_id` exists locally but Win console cannot find the node
- `ListNodesSimple`, `ListSubtreeSimple`, TopicBus subscribe timeout

## Symptoms

- ClipboardNode UI reports `subscribe clipboard topic: context deadline exceeded`.
- Local auth snapshot contains a non-zero `node_id` and `hub_id`.
- MyFlowHub-Win device tree does not show the ClipboardNode node under the Hub.
- TCP may still appear connected even though runtime did not start.

## Impact

Clipboard synchronization does not start. The user may misread the saved node id as proof that the current session is online, while the Hub management list only reflects live bound connections.

## Trigger Conditions

- A previous run saved `logged_in=true` in auth snapshot.
- A new process starts and reuses that snapshot without performing login on the new TCP session.
- Runtime then attempts TopicBus subscribe using the saved node id.
- Remote Hub does not route or respond to subscribe because the connection is not actually bound as that node.

## Root Cause

`logged_in` is a process/session property, not durable identity. Persisting and trusting it across process starts can skip login and leave the current TCP session unbound.

## Investigation Trail

1. Check `%APPDATA%/myflowhub/ClipboardNode/myflowhub/auth_snapshot.json`.
2. Compare `node_id` / `hub_id` against MyFlowHub-Win management tree output.
3. Trace `Engine.Start`: connect, ensure identity, runtime start, TopicBus subscribe.
4. Trace `Client.EnsureIdentity`: saved `LoggedIn=true` with matching device id can bypass login.
5. Confirm management tree lists live connection-manager children, not local snapshots.

## Resolution

- Treat loaded auth snapshot `LoggedIn` as false while preserving durable identity fields.
- Persist `logged_in=false` on `Client.Close`.
- Re-login on each fresh process connection before subscribing.
- Close transport on startup failure after connect so the UI does not show a half-connected state.

## Prevention / Guardrails

- Do not persist live session flags as reusable truth.
- Durable snapshot fields should be identity material only: device id, node id, hub id, role, and last action/message diagnostics.
- Startup tests should verify stale `logged_in=true` snapshots do not skip login.
- Startup failure tests should verify transport status returns to disconnected.

## Related Docs

- Related requirement: `docs/requirements/clipboard-sync.md`
- Related spec: `docs/specs/clipboard-sync.md`
- Related change: `docs/change/2026-06-02_clipboardnode-startup-lifecycle.md`
