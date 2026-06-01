# Plan - ClipboardNode debug-latest all-platform builds

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `chore/debug-latest-all-platforms`
- Base: `master` at `a954c79 fix: 修复debug-latest首跑校验问题`
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-debug-latest-all-platforms/MyFlowHub-ClipboardNode`
- Current Stage: `3.1 - Planning confirmed`

## Stage Records

### Initialization

- `guide.md`: not present.
- Participating repo: `MyFlowHub-ClipboardNode`.
- Participating modules:
  - GitHub Actions workflow under `.github/workflows/`.
  - Flutter platform host directories under `app/`.
  - README and docs archive/index entries.
- Base branch: `master`.
- Dedicated branch: `chore/debug-latest-all-platforms`.
- Dedicated worktree: `D:/project/MyFlowHub3/worktrees/chore-debug-latest-all-platforms/MyFlowHub-ClipboardNode`.
- Main repo path is control-plane only; implementation edits stay in this worktree.

### Stage 1 - Requirements Analysis

#### Goal

Extend the existing `debug-latest` automation from a Windows-only preview into an all-platform debug build workflow so each accepted `master` push compiles every supported Flutter target and refreshes the prerelease assets.

#### Scope

Must:

- keep the `debug-latest` release channel and movable tag contract;
- compile Windows, Linux, macOS, Android debug APK, iOS simulator debug app, and Web;
- keep `workflow_dispatch` and pull request validation without publishing unless the event is a `master` push;
- preserve explicit native-command exit-code checks in PowerShell steps;
- package platform outputs with names that clearly identify platform and debug channel;
- keep Go module validation and the existing Windows CLI debug binary build independent of Flutter platform jobs;
- update README and change archive so users know the release is all-platform.

Optional:

- build additional Go CLI helper binaries after non-Windows host adapters exist.

Not doing:

- no ClipboardNode runtime behavior changes;
- no MyFlowHub protocol or subprotocol changes;
- no signing, notarization, TestFlight, Play Store, installer, or production release channel;
- no device pairing, room key, or encryption change.

#### Use Cases

- A tester downloads `debug-latest` assets for the platform they are using.
- A developer opens a pull request and sees whether the Flutter shell still compiles across supported targets.
- A `master` push refreshes the prerelease only after every required build job succeeds.

#### Functional Requirements

- The workflow must fail before publish if any platform build fails.
- Missing build outputs must be detected explicitly.
- Linux/macOS platform host directories must exist in the Flutter app if those targets are compiled.
- The publish job must download and validate all required assets before moving the `debug-latest` tag.
- Release notes and step summary must list every uploaded asset.

#### Non-functional Requirements

- Keep CI behavior deterministic and auditable.
- Minimize generated Flutter churn and inspect generated platform files before committing.
- Keep package naming consistent and easy to script against.
- Avoid logging clipboard bodies or running live sync in CI.

#### Inputs / Outputs

Inputs:

- repository source at the workflow commit;
- GitHub-hosted Windows, Ubuntu, and macOS runners;
- Go and Flutter toolchains from the workflow;
- Android and Apple platform tooling available on hosted runners.

Outputs:

- Actions artifacts for Go CLI binaries and each Flutter platform package;
- GitHub prerelease `debug-latest` with all current debug assets on `master` pushes;
- generated Linux/macOS Flutter host directories if required by the build;
- README and `docs/change` archive updates.

#### Edge Cases

- iOS device builds require signing, so CI must build simulator output with `--no-codesign`.
- macOS debug apps are unsigned and are preview-only.
- Linux desktop builds require GTK/CMake/Ninja dependencies.
- Web output is a directory and must be archived before upload.
- GitHub Actions manual runs on non-`master` branches must not move `debug-latest`.

#### Acceptance Criteria

- `.github/workflows/debug-latest.yml` has separate build coverage for Windows, Linux, macOS, Android, iOS simulator, Web, and Go CLI.
- Publish depends on all platform jobs.
- Release assets are validated by name before upload.
- README names all available debug assets.
- Local validations pass where the Windows host can run them; unsupported local platform builds are recorded as remote-only validation.

#### Risks

- Full validation of Linux/macOS/iOS/Android CI behavior requires GitHub-hosted runners after push.
- Flutter-generated Linux/macOS files may need future platform-specific customization before production distribution.
- Debug builds are unsigned and should not be treated as production packages.

#### Issue List

- None blocking.

### Stage 2 - Architecture Design

#### Overall Solution

Use a multi-job GitHub Actions workflow:

1. `build-go-cli` validates Go and builds the existing Windows helper CLI binary.
2. `build-windows-debug` builds and packages the Windows Flutter debug runner.
3. `build-linux-debug` installs Linux desktop dependencies, builds, and packages the Linux bundle.
4. `build-macos-debug` builds and packages the macOS `.app`.
5. `build-android-debug` builds the Android debug APK.
6. `build-ios-debug` builds the iOS simulator `.app` without code signing.
7. `build-web-debug` builds and packages the Web output.
8. `publish-debug-latest` runs only on `master` push, waits for all jobs, validates assets, moves the tag, updates the prerelease, and uploads all assets.

This keeps platform concerns isolated while making the release gate depend on the complete build matrix.

#### Alternatives Considered

- Single matrix job:
  - rejected because package paths, host runners, and validation commands differ significantly by platform.
- Keep Windows-only and add manual instructions:
  - rejected because the user explicitly wants every platform compiled by automation.
- Production mobile builds:
  - rejected because signing and store distribution are separate release concerns.

#### Module Responsibilities

- `.github/workflows/debug-latest.yml`: all CI build, package, artifact, and release logic.
- `app/linux`, `app/macos`: Flutter-generated desktop host projects needed for Linux/macOS build targets.
- `README.md`: user-facing preview asset documentation.
- `docs/change`: completed workflow archive and index.

#### Data / Call Flow

1. Checkout source.
2. Restore/setup toolchains per runner.
3. Run validation in the relevant job.
4. Build each platform debug artifact.
5. Copy or archive outputs into `dist/`.
6. Upload each job's artifacts.
7. On `master` push, download all artifacts, validate expected file names, update `debug-latest`, and upload assets with clobber semantics.

#### Interface Drafts

- Release tag: `debug-latest`.
- Release title: `Debug (latest)`.
- Asset names:
  - `myflowhub-clipboardnode-windows-debug.zip`
  - `myflowhub-clipboardnode-linux-debug.tar.gz`
  - `myflowhub-clipboardnode-macos-debug.zip`
  - `myflowhub-clipboardnode-android-debug.apk`
  - `myflowhub-clipboardnode-ios-simulator-debug.zip`
  - `myflowhub-clipboardnode-web-debug.zip`
  - `clipboardnode-windows-amd64.exe`

#### Error Handling and Safety

- Validate every expected output path before upload.
- Use `if-no-files-found: error` for artifacts.
- Keep publish `contents: write` scoped to the publish job.
- Keep debug release update disabled for pull requests and non-`master` manual runs.
- Preserve explicit `$LASTEXITCODE` checks for native PowerShell commands.

#### Performance and Testing Strategy

- Use platform-specific jobs so long builds run in parallel.
- Run Go tests once in the Go job.
- Run Flutter `pub get`, `analyze`, and `test` in one desktop job, then platform build jobs compile their targets.
- Validate locally on Windows with Go tests, Flutter analyze/test, Windows build, Web build, YAML parsing, and `git diff --check`.
- Validate Linux/macOS/iOS/Android through GitHub Actions after push.

#### Extensibility Design Points

- Signing/notarization can be added later as production release jobs without changing `debug-latest`.
- Additional architecture variants can be added as new jobs/assets.
- Future app-layer encryption or protocol behavior remains outside this CI-only workflow.

#### Issue List

- None blocking.

### Stage 3.1 - Planning

#### Project Goal and Current State

Current `debug-latest` publishes only a Windows Flutter debug package and a Windows Go CLI binary. The app already has Android, iOS, Web, and Windows Flutter host directories, but lacks Linux and macOS host directories, so those must be generated before the CI can compile those targets.

#### Docs Governance Routing Decision

使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。

- Requirements impact: none
- Specs impact: none
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons:
  - `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`
  - `docs/lessons/flutter-windows-sdk-shared-bat-git.md`
- Stable product truth is unchanged because this workflow affects distribution automation only.
- Stable protocol/application architecture is unchanged because no runtime or protocol contract changes are planned.
- Active workflow control: root `plan.md`.
- Completed workflow archive: `docs/change/2026-06-01_debug-latest-all-platforms.md`.
- Lessons update: not planned unless validation exposes a reusable CI/toolchain failure mode.

#### Related Requirements / Specs / Lessons

- Requirements: [docs/requirements/clipboard-sync.md](docs/requirements/clipboard-sync.md)
- Specs: [docs/specs/clipboard-sync.md](docs/specs/clipboard-sync.md)
- Lessons: [docs/lessons/debug-latest-ci-native-exit-flutter-material.md](docs/lessons/debug-latest-ci-native-exit-flutter-material.md), [docs/lessons/flutter-windows-sdk-shared-bat-git.md](docs/lessons/flutter-windows-sdk-shared-bat-git.md)

#### Executable Task List

- `CI-8`: Generate and inspect missing Flutter Linux/macOS host projects.
- `CI-9`: Expand `debug-latest` workflow to build and publish all platform debug artifacts.
- `CI-10`: Update README and change archive/index for all-platform debug previews.
- `CI-11`: Run local validation, perform mandatory code review, commit, push, and inspect remote Actions.

#### Task Details

##### CI-8 - Flutter Linux/macOS host projects

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-debug-latest-all-platforms/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: add the missing Flutter desktop platform shells required by CI builds.
- Files / Modules: `app/linux`, `app/macos`, `app/.metadata`
- Write Set: generated Flutter host files only.
- Acceptance:
  - `flutter create --platforms=linux,macos .` succeeds in `app/`;
  - generated diffs are limited to Linux/macOS host scaffolding and Flutter metadata;
  - no application UI/runtime code is changed by generation.
- Test Points:
  - inspect `git status` and `git diff --stat`;
  - later `flutter build linux`/`flutter build macos` in CI.
- Rollback: remove generated `app/linux`, `app/macos`, and the corresponding `.metadata` entries.

##### CI-9 - All-platform debug-latest workflow

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: compile every supported debug target and publish all assets only after every job succeeds.
- Files / Modules: `.github/workflows/debug-latest.yml`
- Write Set: `.github/workflows/debug-latest.yml`
- Acceptance:
  - jobs cover Go CLI, Windows, Linux, macOS, Android, iOS simulator, and Web;
  - publish job needs every build job;
  - missing required assets fail the publish job;
  - release notes list all assets.
- Test Points:
  - YAML parse;
  - local inspection;
  - remote GitHub Actions run after push.
- Rollback: restore previous Windows-only workflow.

##### CI-10 - README and change archive

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: document all-platform preview assets and archive the workflow.
- Files / Modules: `README.md`, `docs/change/`, `docs/change/README.md`
- Write Set:
  - `README.md`
  - `docs/change/2026-06-01_debug-latest-all-platforms.md`
  - `docs/change/README.md`
- Acceptance:
  - README describes all debug assets and the master-only publish rule;
  - archive records requirements/specs impact, task mapping, validation, and rollback.
- Test Points:
  - `git diff --check`;
  - markdown link/path inspection.
- Rollback: revert README and archive/index changes.

##### CI-11 - Validation, review, push, and remote verification

- Owner: main agent
- Worktree: same
- Plan Path: `plan.md`
- Goal: verify local buildable targets, perform Stage 3.3 review, and use GitHub Actions for remote-only platforms.
- Files / Modules: changed files only.
- Write Set: none unless validation exposes a planned issue.
- Acceptance:
  - local Go and Flutter validations pass where supported on Windows;
  - Stage 3.3 checklist passes;
  - commit is pushed to `origin`;
  - remote run is inspected and result is reported.
- Test Points:
  - `GOWORK=off go test ./... -count=1`;
  - `flutter analyze`;
  - `flutter test`;
  - `flutter build windows --debug`;
  - `flutter build web`;
  - YAML parse;
  - `git diff --check`;
  - `gh run watch` when available.
- Rollback: fix forward if validation exposes CI defects, or revert the branch before merging.

#### Dependencies

- GitHub hosted runners for Windows, Ubuntu, and macOS.
- Flutter `3.41.9` stable and Dart SDK satisfying `app/pubspec.yaml`.
- Go `1.25.x`.
- Android/iOS platform toolchains from hosted runners.

#### Risks and Notes

- Local Windows cannot compile Linux, macOS, iOS, or Android in the same way as hosted runners; remote CI is required for full proof.
- iOS simulator and macOS outputs are unsigned debug previews.
- The existing lesson on PowerShell native exits remains mandatory; do not remove `$LASTEXITCODE` checks.

#### Parallelism Assessment

- CI jobs are designed to run in parallel in GitHub Actions.
- No sub-agent dispatch is used locally because the write set is small and the workflow/release contract needs integrated review.

#### Issue List

- None blocking.

阻塞：否
进入 3.2

### Stage 3.2 - Implementation

#### Parallelism Assessment

- GitHub Actions build work is split into parallel jobs by platform.
- Local implementation stayed single-agent because the release contract, asset names, and docs need one integrated review.
- No sub-agent dispatch was used.

#### File-level Change Summary

- `.github/workflows/debug-latest.yml`
  - Adds `build-go-cli`, `build-linux-debug`, `build-macos-debug`, `build-android-debug`, `build-ios-debug`, and `build-web-debug`.
  - Keeps Windows validation with explicit `$LASTEXITCODE` checks.
  - Makes `publish-debug-latest` depend on every build job and validate every expected asset before upload.
- `app/.metadata`
  - Adds Linux and macOS to tracked Flutter platforms while preserving existing Android/iOS/Web/Windows entries.
- `app/linux/**`
  - Adds Flutter-generated Linux desktop host project.
- `app/macos/**`
  - Adds Flutter-generated macOS desktop host project.
- `README.md`
  - Documents all platform debug assets and clarifies unsigned preview status.
- `docs/change/2026-06-01_debug-latest-all-platforms.md`
  - Archives this workflow, requirements/specs impact, task mapping, validation, and rollback.
- `docs/change/README.md`
  - Indexes the new change archive.

#### Task Results

- `CI-8`: completed.
  - `flutter create --platforms=linux,macos .` generated host projects.
  - IDE metadata generated by Flutter was removed because it is ignored and unrelated.
  - `.metadata` was corrected to include all existing platforms plus Linux/macOS.
- `CI-9`: completed.
  - Workflow now builds Go CLI, Windows, Linux, macOS, Android, iOS simulator, and Web.
  - Publish validates seven required assets before release upload.
- `CI-10`: completed.
  - README and `docs/change` index/archive updated.
- `CI-11`: local validation completed; remote validation pending after push.

### Stage 3.3 - Code Review

- 需求覆盖: 通过。The workflow compiles all requested platforms and keeps publish limited to successful `master` pushes.
- 架构合理性: 通过。Platform-specific jobs avoid path/runner coupling; publish depends on all required jobs.
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）: 通过。Go tests run once; Flutter platform builds run in parallel; packaging copies only build outputs.
- 可读性与一致性: 通过。Flutter asset names share one `myflowhub-clipboardnode-<platform>-debug` pattern and release validation centralizes expected names.
- 可扩展性与配置化: 通过。Flutter version is centralized in workflow env; future signing/release jobs can be added without changing debug channel.
- 稳定性与安全: 通过。Publish permission remains scoped to publish job; unsigned preview status is documented; no clipboard runtime or protocol code changed.
- 测试覆盖情况: 通过 locally with remote-only caveat:
  - `GOWORK=off go test ./... -count=1`: passed.
  - Go Windows CLI cross-compile: passed. Non-Windows CLI assets are intentionally not published because the CLI still uses the Windows host adapter.
  - YAML structure parse: passed.
  - `actions/setup-java` latest tag check: `v5.2.0`.
  - `flutter analyze`: passed.
  - `flutter test`: passed, 5 tests.
  - `flutter build windows --debug`: passed.
  - `flutter build web --debug`: passed.
  - `git diff --check`: passed.
  - `actionlint`: not installed locally.
  - Linux/macOS/Android/iOS simulator Flutter builds require GitHub hosted runners and are pending remote validation after push.
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）: 通过。No sub-agent dispatch; all file changes map to `CI-8` through `CI-11`.

阻塞：否
进入 Stage 4 / push validation

### Stage 4 - Change Archive

使用 `$m-docs` 校验变更归档、requirements/specs 影响和 lessons 入口。

- Change archive: [docs/change/2026-06-01_debug-latest-all-platforms.md](docs/change/2026-06-01_debug-latest-all-platforms.md)
- Requirements impact: none
- Specs impact: none
- Lessons impact: none
- Related requirements: [docs/requirements/clipboard-sync.md](docs/requirements/clipboard-sync.md)
- Related specs: [docs/specs/clipboard-sync.md](docs/specs/clipboard-sync.md)
- Related lessons:
  - [docs/lessons/debug-latest-ci-native-exit-flutter-material.md](docs/lessons/debug-latest-ci-native-exit-flutter-material.md)
  - [docs/lessons/flutter-windows-sdk-shared-bat-git.md](docs/lessons/flutter-windows-sdk-shared-bat-git.md)
- Index update: [docs/change/README.md](docs/change/README.md)
- New reusable lesson: not needed; no new recurring failure mode was discovered.
- Workflow end: not requested yet; next step is commit, push, and remote Actions validation.
