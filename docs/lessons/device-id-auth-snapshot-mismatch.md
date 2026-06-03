# Device ID Auth Snapshot Mismatch

## Summary

ClipboardNode authentication identity is the configured `device_id`, not the user-visible display name. If a user changes `device_id` after a previous registration, the saved auth snapshot can still contain the old node id and device id. Reusing that snapshot can trigger `authenticate myflowhub node: invalid signature` or stale-node login behavior.

## Lookup Hints

- `authenticate myflowhub node: invalid signature`
- `device_id`
- `display_name`
- `device_label`
- `auth_snapshot.json`
- changing device ID after registration
- node id remains from an old device identity

## Symptoms

- ClipboardNode fails during background auth after the device identifier is edited.
- The local config shows a new device identity, but `myflowhub/auth_snapshot.json` still contains the old identity or node id.
- Editing only the visible device label unexpectedly affects auth behavior if identity and display name are conflated.
- Re-registering may require Hub approval because the identity is now a different device.

## Impact

The app cannot connect and subscribe successfully until local login state is cleared or the configured identity matches the saved snapshot. Users may also confuse a display-name change with an identity change if the UI exposes only one field.

## Trigger Conditions

- A device registered or logged in with an earlier `device_id`.
- The user edits the configured device ID.
- ClipboardNode attempts to authenticate using the new identity while the auth snapshot still points at the old node identity.
- Legacy configs only contain `device_label`, so migration must preserve compatibility while separating identity from display metadata.

## Root Cause

Auth signatures and node reuse are keyed by the authenticated device identity. A display label is metadata, but a device ID change is an identity change. Keeping a stale auth snapshot after an identity change can reuse node metadata that no longer matches the configured signing identity.

## Investigation Trail

1. Inspect the runtime config for `device_id`, `display_name`, and legacy `device_label`.
2. Inspect `%APPDATA%/MyFlowHub/ClipboardNode/myflowhub/auth_snapshot.json` or the equivalent configured work directory.
3. Compare the saved snapshot `device_id` and `node_id` with the current config `device_id`.
4. Confirm the app clears the snapshot when `device_id` changes.
5. Confirm display-name-only edits keep the saved node identity and only update metadata on the next register/login.

## Resolution

- Add first-class `device_id` and `display_name` config fields.
- Normalize legacy `device_label` into both fields when new fields are absent.
- Use `device_id` for auth signatures and identity.
- Use `display_name` for register/login metadata and UI labels.
- Clear `auth_snapshot.json` when the configured `device_id` changes.
- Clear a stale snapshot during startup if the saved snapshot identity does not match configured `device_id`.

## Prevention / Guardrails

- Treat identity fields and display metadata as different concepts in UI, bridge contracts, config, and auth payloads.
- Do not automatically delete `node_keys.json` for this case; clearing the auth snapshot is enough to avoid stale node reuse.
- Add regression tests for identity changes, display-name-only changes, and legacy `device_label` migration.
- Smoke-test bridge `set_config` with a temporary config directory and `myflowhub/auth_snapshot.json` in the real workdir layout.

## Related Docs

- Related requirement: `docs/requirements/clipboard-sync.md`
- Related spec: `docs/specs/clipboard-sync.md`
- Related change: `docs/change/2026-06-03_clipboardnode-device-id-display-name.md`
