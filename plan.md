# Windows Unsigned Release Preview Plan

## Goal

Allow tag releases to include a clearly labeled unsigned Windows preview package when Windows production code-signing secrets are not configured, without presenting it as a signed production Windows asset.

## Current State

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-ClipboardNode`
- Worktree: `D:\project\MyFlowHub3\worktrees\chore-windows-unsigned-release-preview\MyFlowHub-ClipboardNode`
- Branch: `chore/windows-unsigned-release-preview`
- Base: `master`
- Current stage: `4 - Change Archive`
- Current release behavior:
  - Linux and Web always build.
  - Android builds when signing secrets are configured.
  - Windows is skipped when `WINDOWS_CODESIGN_PFX_*` secrets are missing.

## Requirements And Specs Impact

- Requirements impact: none.
- Specs impact: none.
- Related requirements: `docs/requirements/clipboard-sync.md`.
- Related specs: `docs/specs/clipboard-sync.md`.
- Related lessons:
  - `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`

## Scope

### Must

- Build Windows on tag releases even when Windows code-signing secrets are missing.
- Keep signing when `WINDOWS_CODESIGN_PFX_BASE64` and `WINDOWS_CODESIGN_PFX_PASSWORD` exist.
- Publish unsigned Windows assets only with explicit preview naming.
- Keep signed Windows asset names unchanged when signing secrets exist.
- Make Release Notes and README state that unsigned Windows preview assets are not trusted production signed packages.
- Keep macOS/iOS skip behavior unchanged until their signing materials exist.

### Not Doing

- Add Windows code signing certificates.
- Add Azure Artifact Signing or SignPath integration.
- Change runtime/product behavior.
- Reissue the already-published `v0.1.0` release unless requested separately.

## Architecture

- Change Windows capability detection to always build Windows, with status text distinguishing signed production from unsigned preview.
- In the Windows packaging step, select output names based on whether signing secrets are present.
- Signed outputs retain:
  - `myflowhub-clipboardnode-windows-release.zip`
  - `clipboardnode-windows-amd64.exe`
  - `clipboardnode-bridge-windows-amd64.exe`
- Unsigned preview outputs use:
  - `myflowhub-clipboardnode-windows-unsigned-preview.zip`
  - `clipboardnode-windows-amd64-unsigned-preview.exe`
  - `clipboardnode-bridge-windows-amd64-unsigned-preview.exe`
- Keep publish asset selection dynamic by adding the unsigned preview names to `known_assets`.

## Task Checklist

### WIN-1 - Enable Windows Build With Preview Status

- Files: `.github/workflows/release.yml`
- Goal: Windows job runs for tag releases without signing secrets.
- Acceptance:
  - `build_windows=true` for tag release even when Windows signing secrets are missing.
  - `windows_status` says unsigned preview when secrets are missing.
  - Signed status remains signed production when secrets exist.
- Tests:
  - YAML structural checks.
  - Capability simulation.
- Rollback:
  - Restore `secret_status "windows"` hard gate.

### WIN-2 - Rename Unsigned Windows Outputs

- Files: `.github/workflows/release.yml`
- Goal: unsigned Windows artifacts are visibly marked as preview.
- Acceptance:
  - Unsigned zip/exe names include `unsigned-preview`.
  - Signed zip/exe names remain unchanged.
  - Package README inside unsigned zip warns that the package is unsigned preview.
- Tests:
  - Static script checks.
  - Bash/Pwsh syntax checks.
- Rollback:
  - Restore static signed output names.

### WIN-3 - Publish And Document Preview Assets

- Files:
  - `.github/workflows/release.yml`
  - `README.md`
- Goal: Release upload and user docs describe unsigned Windows preview explicitly.
- Acceptance:
  - `known_assets` includes unsigned Windows preview names.
  - README release asset list includes unsigned Windows preview package.
  - README still documents required secrets for signed Windows production package.
- Tests:
  - Publish asset simulation.
  - Markdown review.
- Rollback:
  - Remove unsigned preview names and README wording.

### WIN-4 - Archive Workflow

- Files:
  - `docs/change/YYYY-MM-DD_windows-unsigned-release-preview.md`
  - `docs/change/README.md`
- Goal: record scope, validation, and rollback.
- Acceptance:
  - Archive includes requirements/specs/lessons impact and task mapping.
- Tests:
  - `git diff --check`.
- Rollback:
  - Remove archive and index entry.

## Parallelism Assessment

No sub-agent delegation. The change is a tightly coupled workflow/documentation edit in one repo.

## Risks

- Users may treat unsigned preview as production. Mitigate with asset names, release notes status, README wording, and package-local README warning.
- Windows publish names can diverge from upload names. Mitigate by deriving names once and adding both signed and unsigned names to `known_assets`.
- Signed Windows path must remain unchanged for future certificate configuration.

## Stage Gate

阻塞：否
进入 3.3

## Stage 3.2 Implementation Summary

- `WIN-1`: changed Windows platform capability detection so tag releases build Windows even without code-signing secrets, with status text marking unsigned preview mode.
- `WIN-2`: changed Windows packaging to derive output names from signing availability:
  - signed path keeps `myflowhub-clipboardnode-windows-release.zip`, `clipboardnode-windows-amd64.exe`, and `clipboardnode-bridge-windows-amd64.exe`;
  - unsigned path emits `myflowhub-clipboardnode-windows-unsigned-preview.zip`, `clipboardnode-windows-amd64-unsigned-preview.exe`, and `clipboardnode-bridge-windows-amd64-unsigned-preview.exe`.
- `WIN-2`: added package-local `README-WINDOWS.txt` warning for unsigned preview packages.
- `WIN-3`: added unsigned Windows preview names to publish asset discovery and documented the behavior in `README.md`.

## Stage 3.3 Code Review

- 需求覆盖: 通过. Tag releases can include Windows release-mode assets without code-signing secrets while clearly marking them unsigned preview.
- 架构合理性: 通过. Capability detection remains in `prepare-release`; Windows build remains one job; publish logic remains dynamic asset discovery.
- 性能风险: 通过. Changes only affect CI naming/packaging conditionals and do not add repeated expensive work beyond the already-enabled Windows build.
- 可读性与一致性: 通过. Signed and unsigned names are derived once in the Windows packaging script.
- 可扩展性与配置化: 通过. Future Windows signing secrets automatically restore the signed production asset names.
- 稳定性与安全: 通过. Unsigned Windows artifacts carry explicit preview names and package-local warnings; signed production naming is reserved for configured signing secrets.
- 测试覆盖情况: 通过. YAML structure, bash syntax, PowerShell syntax, Windows capability simulations, publish asset simulation, and `git diff --check` passed.
- 子Agent治理与审计: 通过. Parallelism was assessed; no sub-agent was dispatched.

## Stage 4 Change Archive

- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: none.
- Specs impact: none.
- Lessons impact: none. 本次没有暴露新的可复用故障模式；归档中记录 Windows unsigned preview 排查线索即可。
- Archive: `docs/change/2026-06-02_windows-unsigned-release-preview.md`.
- Index updated: `docs/change/README.md`.
- Hosted GitHub Actions dry-run:
  - Run: <https://github.com/yttydcs/myflowhub-clipboardnode/actions/runs/26831337705>
  - Result: success.
  - `Publish GitHub Release`: skipped as expected for manual dry-run.
  - `gh release view v0.0.0`: absent; no accidental Release was published.
- Windows artifact inspection:
  - Artifact outer name: `myflowhub-clipboardnode-windows-release`.
  - Internal release asset files include `myflowhub-clipboardnode-windows-unsigned-preview.zip`, `clipboardnode-windows-amd64-unsigned-preview.exe`, and `clipboardnode-bridge-windows-amd64-unsigned-preview.exe`.
  - Expanded package root contains `README-WINDOWS.txt` warning that the Windows package is unsigned preview, may trigger Unknown Publisher or SmartScreen warnings, and must not be treated as a signed production release.

## Workflow End Gate

阻塞：否
等待用户确认 `结束workflow` 后再合并、推送 master、删除远端分支并清理 worktree。
