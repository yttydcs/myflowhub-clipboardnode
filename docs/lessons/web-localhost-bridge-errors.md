# Web Localhost Bridge Errors

## Summary

Flutter Web cannot directly use the native Go MyFlowHub engine or unrestricted background clipboard APIs. Live Web mode must be explicit: a loopback-only local bridge, token authentication, user-gesture clipboard commands, and command responses that expose real success or error state.

## Lookup Hints

- `--web-listen`
- `CLIPBOARDNODE_WEB_BRIDGE`
- `CLIPBOARDNODE_WEB_TOKEN`
- `/health`
- `/events`
- `/command`
- `ok:false`
- `omitempty`
- `loopback`

## Symptoms

- Web UI appears to accept a command but no sync action occurs.
- Errors are visible in bridge logs but not in Dart state.
- SSE error event omits `ok:false`, so the browser side treats the event as accepted/default.
- Bridge starts on an unsafe non-loopback address or browser calls fail due missing token/CORS.

## Impact

- Users can mistake diagnostic Web mode for live native sync.
- Failed commands can look successful if HTTP only returns `accepted`.
- Security posture weakens if the local engine control surface is exposed beyond loopback or without token auth.

## Trigger Conditions

- Web bridge endpoint/token dart-defines are omitted.
- `/command` returns only asynchronous acceptance instead of synchronous `ok`/`error` status.
- Go JSON tags omit false booleans, especially `ok:false`.
- `--web-listen` host is not an explicit loopback address.
- Browser clipboard behavior is treated like desktop background watch.

## Root Cause

Web has browser sandbox constraints and cannot share the same trust or capability model as native desktop. The localhost bridge is a separate control surface, so both transport safety and error propagation need explicit contracts.

## Investigation Trail

1. Confirm Web build used `CLIPBOARDNODE_WEB_BRIDGE` and `CLIPBOARDNODE_WEB_TOKEN`.
2. Call `/health` on the bridge.
3. Verify `/command` requires token auth and returns JSON containing `accepted`, `ok`, `error`, and `status`.
4. Inspect `/events` SSE payloads and confirm failures encode `ok:false`.
5. Check `--web-listen` validation rejects non-loopback hosts.

## Resolution

- Add loopback-only `--web-listen` validation.
- Require token authentication for Web bridge status, events, and commands.
- Return synchronous command result JSON from `/command`.
- Remove `omitempty` from `EngineEvent.OK` so `ok:false` is encoded.
- Keep hosted Web without local bridge in explicit diagnostic/preview mode.

## Prevention / Guardrails

- Never imply hosted Web has native engine or background clipboard access.
- Keep Web bridge endpoint/token opt-in via dart-defines or explicit configuration.
- Test both success and failure command responses.
- Include `ok:false` assertions in bridge contract tests.
- Bind only to loopback addresses.

## Related Requirements / Specs / Changes

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)
- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)
- [../change/2026-06-02_clipboard-full-platform-sync.md](../change/2026-06-02_clipboard-full-platform-sync.md)
