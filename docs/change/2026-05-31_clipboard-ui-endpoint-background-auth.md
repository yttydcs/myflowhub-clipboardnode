# 2026-05-31 Clipboard UI Endpoint And Background Auth

## Background

The ClipboardNode preview exposed a fixed parent endpoint and still needed an explicit product decision for registration, login, and session cleanup. These operations should be background lifecycle work rather than separate primary UI actions.

## Changes

- Added editable `parent_endpoint` support across Go runtime config, configstore coverage, Go bridge contracts, Flutter DTOs, preview bridge, and settings UI.
- Added UI-safe preview auth stages:
  - connect parent Hub;
  - background registration;
  - background login;
  - authenticated.
- Clear preview login state and node identity during disconnect and app disposal.
- Removed the redundant overview connection and security panels.
- Kept the top-bar connect/disconnect action as the single primary connection workflow.
- Fixed the switch off-state thumb/track colors and added a CJK-capable font fallback order.

## Requirements Impact

updated

The stable requirements now clarify configurable parent endpoint support and background auth/session cleanup.

## Specs Impact

updated

The technical specification now records startup registration/login and disconnect/shutdown cleanup order.

## Lessons Impact

none

No recurring troubleshooting rule emerged from this slice.

## Related Requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related Specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Related Lessons

- None.

## Related Plan

- Root `plan.md`
- Task mapping:
  - `APP-8`: parent Hub / endpoint configuration.
  - `APP-9`: background auth lifecycle.

## Reusable Checks

- Symptom: the app cannot target another Hub.
- Trigger: parent endpoint is still fixed in local defaults.
- Keywords: `parent_endpoint`, `ParentEndpoint`, `父节点`.
- Quick check: open settings, save a trimmed endpoint, and confirm the overview Hub value after connect.
- Symptom: register/login appears as a separate user workflow.
- Trigger: auth actions leak into primary UI instead of running during connect.
- Keywords: `authStage`, `后台注册`, `后台登录`, `登录态已清理`.
- Quick check: connect once, verify final `已认证`, disconnect, and verify node identity is cleared.

## Code Review

- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计：通过；本轮未派发子Agent。

## Key Decisions

- Background auth is modeled as lifecycle state, not as new MyFlowHub subprotocol behavior.
- The preview bridge demonstrates UI behavior only. Real Hub registration/login cleanup remains part of the future live transport adapter.
- Parent endpoint remains non-sensitive persisted configuration.

## Validation

- `dart format lib test`
  - passed.
- `flutter analyze`
  - passed.
- `flutter test`
  - passed; 5 widget tests.
- `flutter build windows`
  - passed; output `app/build/windows/x64/runner/Release/myflowhub_clipboard.exe`.
- `GOWORK=off go test ./... -count=1`
  - passed.
- `git diff --check`
  - passed.
- Windows preview
  - rebuilt and restarted locally.

## Rollback

- Revert `APP-8` `parent_endpoint` additions in config, contracts, UI, docs, and tests.
- Revert `APP-9` preview auth lifecycle state, docs, shell labels, and widget tests.
- Restore the removed overview panels only if they are required by a later UX decision.

## Sub-Agent Trace

- No sub-agent dispatch was used.
