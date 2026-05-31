# Plan - ClipboardNode debug-latest build automation

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `fix/debug-latest-ci`
- Base: `master` at `10dae6a chore: 添加debug-latest自动构建发布`
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-debug-latest-ci/MyFlowHub-ClipboardNode`
- Current Stage: `3.1 - Corrective planning after first remote run`

## Stage Records

### Initialization

- `guide.md`: not present in `MyFlowHub-ClipboardNode`.
- Participating repo: `MyFlowHub-ClipboardNode`.
- Participating modules:
  - GitHub Actions workflow under `.github/workflows/`.
  - Go validation and CLI build path.
  - Flutter app validation and Windows debug build path.
  - README/docs change archive.
- Base branch: `master`.
- Dedicated branch: `chore/debug-latest-build`.
- Dedicated worktree: `D:/project/MyFlowHub3/worktrees/chore-debug-latest-build/MyFlowHub-ClipboardNode`.
- Main repo path is control-plane only; implementation edits stay in this worktree.

### Stage 1 - Requirements Analysis

#### Goal

Add automated `debug-latest` build publishing for ClipboardNode so each accepted `master` push can produce a readily downloadable debug build without local packaging steps.

#### Scope

- Must:
  - run on GitHub Actions;
  - validate Go packages as an independent module with `GOWORK=off`;
  - build the Go CLI debug/support binary;
  - validate Flutter with `flutter pub get`, `flutter analyze`, and `flutter test`;
  - build the Flutter Windows debug application;
  - package the full Windows runner output, not only the `.exe`, because Flutter desktop needs bundled runtime/data files;
  - publish/update a prerelease and movable tag named `debug-latest` on `master` pushes;
  - keep manual `workflow_dispatch` available for validation without changing the release tag unless run from `master` push;
  - document how to find the latest debug build.
- Optional:
  - upload an Actions artifact for every workflow run.
  - include Go CLI binary in the release package as a separate asset.
- Not doing:
  - no product runtime behavior changes;
  - no protocol/subprotocol changes;
  - no mobile release packaging in this workflow;
  - no signing, notarization, installer, or production release channel.

#### Use Cases

- A developer pushes to `master` and the latest Windows debug package is refreshed on GitHub Releases.
- A developer manually runs the workflow to validate CI setup or inspect artifacts without moving the release channel from non-`master` branches.
- A tester downloads `debug-latest` to preview the current Windows app without building locally.

#### Functional Requirements

- Provide one workflow with build and publish jobs.
- The build job must fail explicitly if expected Windows or CLI artifacts are missing.
- The publish job must depend on successful build output.
- The publish job must force-update tag `debug-latest` to the current `master` commit.
- The release must be marked prerelease and overwritten safely with current artifacts.

#### Non-functional Requirements

- Keep automation deterministic enough for handoff: pin major action versions and use the repo SDK constraints.
- Keep workflow logs useful: output build paths and release URLs in the step summary.
- Avoid clipboard body leakage: CI does not run live sync or produce runtime data.
- Minimize repo churn and avoid unrelated formatting or generated file changes.

#### Inputs / Outputs

- Inputs:
  - repository source at the workflow commit;
  - GitHub Actions Windows runner;
  - Go and Flutter toolchains installed by the workflow;
  - GitHub token with `contents: write` for release publishing.
- Outputs:
  - Actions artifact containing Windows debug package and CLI binary;
  - GitHub prerelease `debug-latest`;
  - release assets for the Windows debug package and CLI binary;
  - README and change archive documentation.

#### Edge Cases

- Flutter SDK channel may not have the exact local development version; workflow must use the stable channel unless a future release pin is selected.
- Flutter Windows debug output layout can drift; packaging must locate and validate the expected `runner/Debug` directory.
- Release or tag may not exist on first run; publish logic must create it.
- Release assets may already exist; uploads must clobber current debug assets.
- Manual runs on non-`master` branches should build artifacts but not publish `debug-latest`.

#### Acceptance Criteria

- `.github/workflows/debug-latest.yml` exists and is syntactically coherent.
- Workflow triggers on `push` to `master`, pull requests, and manual dispatch.
- Build job produces a zipped Flutter Windows debug directory plus Go CLI executable.
- Publish job updates `debug-latest` only for `push` to `refs/heads/master`.
- README points users to the `debug-latest` prerelease.
- Local validations pass or any environment blocker is recorded.

#### Risks

- GitHub runner Flutter version availability can differ from local SDK `3.41.9`.
- Windows debug builds are not signed or optimized; they are preview artifacts only.
- Actual GitHub Actions execution can only be fully verified after the commit is pushed.

#### Issue List

- None blocking.

### Stage 2 - Architecture Design

#### Overall Solution

Add one GitHub Actions workflow with two jobs:

1. `build-windows-debug` runs validation and packaging on `windows-latest`.
2. `publish-debug-latest` downloads the artifact after a successful `master` push, moves tag `debug-latest`, creates or updates a prerelease, and uploads the latest assets.

This mirrors the existing MyFlowHub Android `debug-latest` release pattern while adapting it to this repository's `master` branch and Flutter Windows packaging shape.

#### Alternatives Considered

- Reuse the local `scripts/validate.ps1` only:
  - insufficient because it does not run Flutter validation or package release assets.
- Separate `ci.yml` and `debug-latest.yml`:
  - unnecessary for the first automation; one named workflow keeps the release dependency simple.
- Release only the Flutter `.exe`:
  - rejected because Flutter desktop requires bundled DLLs and `data/`.
- Use `stable` Flutter channel instead of pinning the local SDK revision:
  - selected for the first CI setup because the local SDK version is workspace-specific and may not be available through GitHub action version resolution.

#### Module Responsibilities

- `.github/workflows/debug-latest.yml`:
  - toolchain setup;
  - Go validation/build;
  - Flutter validation/build;
  - artifact packaging;
  - release publishing.
- `README.md`:
  - user-facing link and local validation context.
- `docs/change/`:
  - completed workflow archive after implementation.

#### Data / Call Flow

1. Checkout repository.
2. Install Go and Flutter.
3. Run Go tests and build CLI under `GOWORK=off`.
4. Run Flutter dependency restore, analysis, tests, and Windows debug build under `app/`.
5. Copy `app/build/windows/x64/runner/Debug` into a package directory.
6. Zip the package and upload it with the Go CLI executable as a workflow artifact.
7. On `master` push, download the artifact, update tag `debug-latest`, update/create prerelease, and upload assets with clobber semantics.

#### Interface Drafts

- Workflow artifact name: `myflowhub-clipboardnode-windows-debug`.
- Release tag: `debug-latest`.
- Release title: `Debug (latest)`.
- Release assets:
  - `myflowhub-clipboardnode-windows-debug.zip`
  - `clipboardnode-windows-amd64.exe`

#### Error Handling and Safety

- Fail fast on missing package directory, executable, zip, or CLI binary.
- Use `if-no-files-found: error` for build artifacts.
- Restrict release permissions to the publish job with `contents: write`.
- Keep pull request and manual builds read-only.

#### Performance and Testing Strategy

- Use Go and Flutter action caches where supported.
- Run Flutter and Go validation before packaging.
- Validate locally with Go tests, Flutter tests/analyze/build when the local SDK works, and `git diff --check`.
- Full publish validation happens after push through GitHub Actions.

#### Extensibility Design Points

- Future Android/iOS/macOS/Linux packages can add platform-specific jobs while keeping `publish-debug-latest` as a multi-asset publisher.
- Future release signing can be added as a separate production workflow without changing the debug prerelease contract.
- Future Flutter version pinning can be added once the project selects a published stable SDK version.

#### Issue List

- None blocking.

### Stage 3.1 - Planning

#### Project Goal and Current State

Current repo has Go validation and Flutter local validation instructions, but no GitHub Actions workflow. The next change adds CI build automation and a `debug-latest` release channel.

#### Docs Governance Routing Decision

使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。

- Requirements impact: none
- Specs impact: none
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons: `docs/lessons/flutter-windows-sdk-shared-bat-git.md`
- Stable product truth is unchanged because this workflow affects distribution automation only.
- Stable protocol/application architecture is unchanged because no runtime or protocol contract changes are planned.
- Active workflow control: root `plan.md`.
- Completed workflow archive: `docs/change/2026-05-31_debug-latest-build.md`.
- Lessons update: not planned unless validation exposes a reusable CI/toolchain failure mode.

#### Related Requirements / Specs / Lessons

- Requirements: [docs/requirements/clipboard-sync.md](docs/requirements/clipboard-sync.md)
- Specs: [docs/specs/clipboard-sync.md](docs/specs/clipboard-sync.md)
- Lessons: [docs/lessons/flutter-windows-sdk-shared-bat-git.md](docs/lessons/flutter-windows-sdk-shared-bat-git.md)

#### Executable Task List

- `CI-1`: Add GitHub Actions workflow for Windows debug build and `debug-latest` prerelease publishing.
- `CI-2`: Update README and change archive for the new debug release channel.
- `CI-3`: Run local validation and review the workflow against staged requirements.

#### Task Details

##### CI-1 - Debug-latest GitHub Actions workflow

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-debug-latest-build/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: add automated Windows debug build and prerelease publishing.
- Files / Modules: `.github/workflows/debug-latest.yml`
- Write Set: `.github/workflows/debug-latest.yml`
- Acceptance:
  - builds Go tests/CLI and Flutter Windows debug app;
  - uploads Actions artifact;
  - publishes `debug-latest` only on `master` push;
  - release assets are validated before upload.
- Test Points:
  - `git diff --check`;
  - local inspection of workflow commands;
  - full GitHub Actions run after push.
- Rollback: remove `.github/workflows/debug-latest.yml`.

##### CI-2 - README and change archive

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: document where to preview the latest debug build and record workflow output.
- Files / Modules: `README.md`, `docs/change/`, `docs/change/README.md`
- Write Set: `README.md`, `docs/change/2026-05-31_debug-latest-build.md`, `docs/change/README.md`
- Acceptance:
  - README names the `debug-latest` release channel and its scope;
  - archive records requirements/specs impact and rollback.
- Test Points:
  - `git diff --check`;
  - markdown path/link sanity by inspection.
- Rollback: revert README and archive additions.

##### CI-3 - Validation and review

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: verify changed files and perform required code review checklist.
- Files / Modules: changed files only.
- Write Set: none unless validation exposes a planned issue.
- Acceptance:
  - local validation results are recorded;
  - Stage 3.3 checklist passes;
  - residual risk of first remote Actions run is documented.
- Test Points:
  - `GOWORK=off go test ./... -count=1`;
  - `flutter analyze`;
  - `flutter test`;
  - `flutter build windows --debug`;
  - `git diff --check`;
  - `git status --short`.
- Rollback: no direct write set; fix or revert failing task outputs.

#### Dependencies

- GitHub Actions hosted Windows runner with Visual Studio desktop build tools.
- GitHub token permissions for releases on `master` pushes.
- Go module with `go 1.25.0`.
- Flutter stable channel satisfying `app/pubspec.yaml` SDK constraint.

#### Risks and Notes

- Local Flutter SDK is workspace-specific (`D:/project/MyFlowHub3/.tmp/tools/flutter-sdk-3.41.9/flutter`) and should not be encoded in CI.
- First release publish is fully verified only after GitHub executes the pushed workflow.
- Debug artifacts are intended for preview and should not be treated as signed production releases.

#### Parallelism Assessment

- Work can be split conceptually between workflow and docs, but both touch the same small release contract and need integrated review.
- No sub-agent dispatch is used because the write set is small and tightly coupled.

#### Issue List

- None blocking.

### Stage 3.2 - Implementation

#### Parallelism Assessment

- Potential split: workflow implementation and docs/archive updates.
- Sub-agent usage: none.
- Reason: the write set is small and coupled by the same release contract, so delegation would add coordination overhead without reducing risk.

#### File-level Change Summary

- `.github/workflows/debug-latest.yml`
  - Adds Windows debug build job.
  - Adds `debug-latest` prerelease publish job for `master` push only.
  - Packages full Flutter Windows debug runner directory plus Go CLI binary.
- `README.md`
  - Adds the debug preview release URL and artifact descriptions.
- `docs/change/2026-05-31_debug-latest-build.md`
  - Archives the workflow, task mapping, validation, risks, and rollback plan.
- `docs/change/README.md`
  - Indexes the new change archive.
- `plan.md`
  - Records requirements, architecture, implementation, review, and closeout state.

#### Task Results

- `CI-1`: completed.
- `CI-2`: completed.
- `CI-3`: completed.

### Stage 3.3 - Code Review

- 需求覆盖: 通过。Workflow covers master push release, manual/PR build validation, Windows debug package, Go CLI, and README visibility.
- 架构合理性: 通过。Build and publish jobs are separated; publish depends on build artifacts.
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）: 通过。CI performs expected build/package I/O only; no runtime code changes.
- 可读性与一致性: 通过。Workflow names, artifact names, and release names match the documented `debug-latest` contract.
- 可扩展性与配置化: 通过。Future platform assets can be added as additional build jobs/assets without changing the release tag contract.
- 稳定性与安全: 通过。Only the publish job has `contents: write`; build and PR/manual validation remain read-only unless the event is `master` push.
- 测试覆盖情况: 通过。Local Go/Flutter validations and packaging simulation passed; full publish path requires first GitHub Actions run after merge/push.
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）: 通过。No sub-agent dispatch; all changes map to `CI-1` through `CI-3`.

### Stage 4 - Change Archive

使用 `$m-docs` 校验变更归档、requirements/specs 影响和 lessons 入口。

- Change archive: [docs/change/2026-05-31_debug-latest-build.md](docs/change/2026-05-31_debug-latest-build.md)
- Requirements impact: none
- Specs impact: none
- Lessons impact: none
- Related requirements: [docs/requirements/clipboard-sync.md](docs/requirements/clipboard-sync.md)
- Related specs: [docs/specs/clipboard-sync.md](docs/specs/clipboard-sync.md)
- Related lessons: [docs/lessons/flutter-windows-sdk-shared-bat-git.md](docs/lessons/flutter-windows-sdk-shared-bat-git.md)
- Index update: [docs/change/README.md](docs/change/README.md)
- Workflow end: rolled back to Stage 3.1 after first remote run exposed two issues.

### Corrective Iteration - First Remote Run

#### Rollback Reason

GitHub Actions run `26717910962` uploaded artifacts and published `debug-latest`, but log inspection found one failed Flutter widget test. The PowerShell step continued because `$ErrorActionPreference = "Stop"` does not reliably stop on non-zero native command exits. Flutter `3.44.0` also introduced a `ListTile` assertion for a `DecoratedBox` background between the tile and its nearest `Material` ancestor.

#### Docs Governance Routing Decision

使用 `$m-docs` 校验修复计划、change 更新和 lessons 入口。

- Requirements impact: none
- Specs impact: none
- Lessons impact: add
- Related lessons:
  - `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`

#### Executable Task List

- `CI-4`: Harden GitHub Actions native command failure handling, pin Flutter `3.41.9`, and upgrade official GitHub Actions to Node 24-compatible major versions.
- `CI-5`: Replace the panel `DecoratedBox` background wrapper with a `Material` wrapper so descendant `ListTile` widgets render correctly under newer Flutter assertions.
- `CI-6`: Update change archive and add a reusable lesson for false-success CI publishing and Flutter `ListTile` material ancestry checks.
- `CI-7`: Re-run local validation, push, inspect the second remote run, and confirm `debug-latest` assets were refreshed only after all tests passed.

#### Task Details

##### CI-4 - Strict CI Native Command Handling

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-debug-latest-ci/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: ensure any failed Go or Flutter command fails the build job before packaging or publishing.
- Files / Modules: `.github/workflows/debug-latest.yml`
- Write Set: `.github/workflows/debug-latest.yml`
- Acceptance:
  - each native validation/build command has an explicit exit-code assertion;
  - Flutter version is pinned to `3.41.9`;
  - official GitHub Actions use current Node 24-compatible majors.
- Test Points:
  - YAML structure parse;
  - inspect remote job behavior after push.
- Rollback: revert workflow hardening block and action version changes.

##### CI-5 - Flutter Material Ancestor Compatibility

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: keep panel styling while ensuring `ListTile` background/ink behavior is backed by a `Material` ancestor.
- Files / Modules: `app/lib/features/shell/clipboard_shell.dart`
- Write Set: `app/lib/features/shell/clipboard_shell.dart`
- Acceptance:
  - `_Panel` uses `Material` with border shape and clipping;
  - widget tests pass locally and on GitHub Actions.
- Test Points:
  - `flutter analyze`;
  - `flutter test`;
  - `flutter build windows --debug`.
- Rollback: restore `_Panel` container wrapper.

##### CI-6 - Archive And Lesson Update

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: record the remote-run failure mode and quick checks.
- Files / Modules: `docs/change/`, `docs/lessons/`
- Write Set:
  - `docs/change/2026-05-31_debug-latest-build.md`
  - `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`
  - `docs/lessons/README.md`
- Acceptance: future debugging starts from the lesson instead of re-reading remote logs.
- Test Points: markdown link/index inspection.
- Rollback: revert lesson and archive update.

##### CI-7 - Corrective Validation

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: verify the corrected automation end to end.
- Files / Modules: changed files only.
- Write Set: none unless validation exposes a planned issue.
- Acceptance:
  - local validation passes;
  - second GitHub Actions run passes without publishing after a failed test;
  - `debug-latest` release points at the corrective commit.
- Test Points:
  - `GOWORK=off go test ./... -count=1`;
  - `flutter analyze`;
  - `flutter test`;
  - `flutter build windows --debug`;
  - YAML structure parse;
  - `git diff --check`;
  - `gh run watch`;
  - `gh release view debug-latest`.
- Rollback: no direct write set; fix or revert failing task outputs.

#### Parallelism Assessment

- Workflow, UI compatibility, and docs files are separable, but this is a small corrective iteration driven by one remote run.
- No sub-agent dispatch is used because integrated validation is the dominant cost and the write sets are small.

### Corrective Stage 3.2 - Implementation Result

- `CI-4`: completed.
  - `.github/workflows/debug-latest.yml` now checks `$LASTEXITCODE` after each Go/Flutter native command.
  - Flutter is pinned to `3.41.9`.
  - Official GitHub Actions were updated to `checkout@v6`, `setup-go@v6`, `upload-artifact@v7`, and `download-artifact@v8`.
- `CI-5`: completed.
  - `_Panel` now uses a shaped `Material` surface instead of a colored `DecoratedBox`, keeping the existing visual surface while satisfying `ListTile` material ancestry checks.
- `CI-6`: completed.
  - Change archive updated.
  - Lesson added and indexed.
- `CI-7`: local validation completed; remote validation pending after push.

### Corrective Stage 3.3 - Code Review

- 需求覆盖: 通过。Covers both false-success CI publishing and Flutter test failure.
- 架构合理性: 通过。Release job still depends on build job; build job now fails before packaging when validation fails.
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）: 通过。Only CI checks and one UI surface wrapper changed.
- 可读性与一致性: 通过。Workflow failure checks are explicit and colocated with native commands.
- 可扩展性与配置化: 通过。Flutter version pin is centralized in the setup step and can be updated intentionally later.
- 稳定性与安全: 通过。Publish permissions remain isolated to the release job.
- 测试覆盖情况: 通过 locally:
  - `GOWORK=off go test ./... -count=1`
  - `flutter analyze`
  - `flutter test`
  - `flutter build windows --debug`
  - YAML structure parse
  - packaging simulation
  - `git diff --check`
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）: 通过。No sub-agent dispatch.

阻塞：否
进入 corrective merge/push and remote validation
