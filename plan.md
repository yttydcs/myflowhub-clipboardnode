# Conditional Tag Release Builds Plan

## Goal

Allow `vX.Y.Z` tag releases to build and publish only platforms whose production prerequisites are configured. Platforms missing required signing secrets must be skipped instead of failing the whole release or publishing misleading unsigned production assets.

## Current State

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-ClipboardNode`
- Worktree: `D:\project\MyFlowHub3\worktrees\chore-conditional-release-builds\MyFlowHub-ClipboardNode`
- Branch: `chore/conditional-release-builds`
- Base: `master`
- Current stage: `4 - Change Archive`
- GitHub Actions secrets currently configured for Android only:
  - `ANDROID_KEYSTORE_BASE64`
  - `ANDROID_KEYSTORE_PASSWORD`
  - `ANDROID_KEY_ALIAS`
  - `ANDROID_KEY_PASSWORD`

## Requirements And Specs Impact

- Requirements impact: none.
- Specs impact: none.
- Related requirements: `docs/requirements/clipboard-sync.md` for platform support goals; no source-of-truth change required.
- Related specs: `docs/specs/clipboard-sync.md` for platform boundaries; no source-of-truth change required.
- Related lessons:
  - `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`
  - `docs/lessons/gomobile-mobile-bindings.md`

## Scope

### Must

- Keep manual `workflow_dispatch` release runs as dry-runs that do not publish a GitHub Release.
- On tag push, skip Windows/macOS/iOS/Android release build jobs when their required signing secret set is incomplete.
- Continue always building Linux and Web release assets.
- Publish a GitHub Release when at least one required artifact is available.
- Build release notes and checksum lists dynamically from available artifacts.
- Make skipped platforms visible in the release run summary and release notes.

### Optional

- Keep optional platform defaults such as Windows timestamp URL and iOS export method unchanged.

### Not Doing

- Generate or fake Windows/macOS/iOS production signing identities.
- Publish unsigned artifacts as signed production assets.
- Change debug-latest behavior.
- Change product runtime behavior.

## Architecture

- Add secret-availability outputs to `prepare-release`.
- Gate signed platform build jobs with job-level `if` conditions using those outputs.
- Keep Linux/Web jobs unconditional because they have no signing prerequisite in the current workflow.
- Make `publish-release` use `always()` plus explicit success/skipped checks so it can run after skipped platform jobs but still refuse failed jobs.
- Replace static `required_assets` with dynamic asset discovery and explicit minimum-asset validation.

## Task Checklist

### REL-1 - Compute Release Capability Flags

- Files: `.github/workflows/release.yml`
- Goal: expose per-platform build flags and status text from `prepare-release`.
- Acceptance:
  - Manual dry-run keeps all platform builds enabled where current behavior expects dry-run validation.
  - Tag release enables Android/Windows/macOS/iOS only when required secrets exist.
  - Linux/Web remain enabled.
- Tests:
  - YAML parse.
  - Static inspection of job outputs and conditions.
- Rollback:
  - Restore previous `prepare-release` outputs and validation behavior.

### REL-2 - Gate Platform Jobs

- Files: `.github/workflows/release.yml`
- Goal: skip unconfigured signed platform jobs cleanly.
- Acceptance:
  - Missing Windows/macOS/iOS secrets no longer fail tag releases.
  - Android still builds when Android secrets are configured.
  - Build jobs still fail on real build errors.
- Tests:
  - YAML parse.
  - Release workflow dry-run.
- Rollback:
  - Remove job-level `if` gates and restore hard secret validation.

### REL-3 - Publish Available Assets

- Files: `.github/workflows/release.yml`
- Goal: publish only artifacts actually produced by successful jobs.
- Acceptance:
  - Release does not require skipped platform assets.
  - Checksums cover the exact uploaded asset set.
  - Release notes list available assets and skipped platforms.
  - Publish job fails if no release assets exist or if any enabled build job fails.
- Tests:
  - YAML parse.
  - `gh workflow run release.yml` dry-run after push.
- Rollback:
  - Restore static `required_assets`.

### REL-4 - Update Release Documentation

- Files: `README.md`
- Goal: explain partial configured-platform tag release behavior.
- Acceptance:
  - README no longer states that every tag release requires all platform signing secrets.
  - README still warns that unconfigured platforms are skipped rather than published unsigned.
- Tests:
  - Markdown review.
- Rollback:
  - Restore previous release section wording.

## Parallelism Assessment

No sub-agent delegation. The write set is small, tightly coupled, and limited to one workflow file plus README wording. Parallel edits would increase merge risk without meaningful speed benefit.

## Risks

- GitHub Actions expression semantics around skipped `needs` jobs can accidentally skip `publish-release`; mitigate with `if: always()` and explicit result checks inside the publish job.
- Dynamic asset discovery could upload unintended files; mitigate by keeping artifact download scope limited to workflow artifacts and checking for known release asset extensions/names.
- Partial releases can be misread as full all-platform releases; mitigate with release notes and README skipped-platform language.

## Stage Gate

阻塞：否
进入 4

## Stage 3.2 Implementation Summary

- `REL-1`: added `prepare-release` outputs for per-platform build flags and human-readable platform status.
- `REL-2`: added job-level conditions for Windows, macOS, Android, and iOS release builds so tag releases skip platforms whose required signing secret set is incomplete.
- `REL-3`: changed `publish-release` to run after skipped jobs, validate enabled job results, discover available release assets, generate checksums for only those assets, and include platform status in release notes.
- `REL-4`: updated `README.md` to describe configured-platform releases and skipped unconfigured signed platforms.

## Stage 3.3 Code Review

- 需求覆盖: 通过. Tag releases can now publish configured platform assets while unconfigured signed platforms are skipped.
- 架构合理性: 通过. Release capability detection stays in `prepare-release`; platform jobs stay responsible for building; publishing stays responsible for asset selection.
- 性能风险: 通过. Added checks are small shell conditionals and static asset scanning over downloaded artifacts only.
- 可读性与一致性: 通过. Platform statuses are exposed once and reused in summaries and release notes.
- 可扩展性与配置化: 通过. Additional platform prerequisites can be added by extending the capability outputs and known asset list.
- 稳定性与安全: 通过. Unsigned production assets are not generated for unconfigured signed platforms; enabled jobs must still succeed before publishing.
- 测试覆盖情况: 通过. YAML structure, bash syntax, PowerShell syntax, capability simulations, publish asset simulation, and `git diff --check` passed. `actionlint` was not available locally.
- 子Agent治理与审计: 通过. Parallelism was assessed; no sub-agent was dispatched.

## Stage 4 Change Archive

Using `$m-docs`:

- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
- Change archive: `docs/change/2026-06-02_conditional-release-builds.md`
- Index updated: `docs/change/README.md`

Validation:

- Python/PyYAML workflow structural check: passed.
- Bash syntax check for release workflow scripts: passed, 16 scripts.
- PowerShell parser syntax check for release workflow scripts: passed, 2 scripts.
- Capability simulation for Android-only tag release secrets: passed.
- Capability simulation for manual dry-run without secrets: passed.
- Publish asset simulation with Linux/Web/Android assets only: passed.
- `git diff --check`: passed.
- Hosted `release.yml` dry-run on branch `chore/conditional-release-builds`: passed.
  - Run: `https://github.com/yttydcs/myflowhub-clipboardnode/actions/runs/26824592121`
  - Result: `success`
  - `Publish GitHub Release`: skipped as expected for manual dry-run.
  - `gh release view v0.0.0`: absent.
