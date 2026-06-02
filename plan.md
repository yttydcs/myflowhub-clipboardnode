# Plan - Tag Release CI

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `chore/tag-release-ci`
- Base: `master` at `285ce22`
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Current Stage: `4 - Change Archive / release-mode revision`

## Stage Records

### Initialization

- guide.md: not present in the repository root.
- base/worktree confirmation: main repo is the control-plane; implementation will stay in the dedicated worktree above.
- Existing CI: `.github/workflows/debug-latest.yml` builds all debug artifacts and refreshes the `debug-latest` prerelease on `master` pushes. Tags are currently ignored.

### Stage 1 - Requirements Analysis

#### Goal

Keep the existing `debug-latest` channel and add automatic release publishing when a version tag is pushed.

#### Scope

- Must:
  - Preserve `master` push behavior that refreshes `debug-latest`.
  - Run the same platform build validation for release tags.
  - Publish a non-prerelease GitHub Release whose tag matches the pushed version tag.
  - Prevent `debug-latest` tag movement from recursively triggering a version release.
  - Keep release asset validation explicit and fail if an expected artifact is missing.
- Optional:
  - Improve workflow naming, summaries, and README release guidance.
- Not doing:
  - Production signing, notarization, store publishing, or installer generation.
  - Changing runtime clipboard behavior or MyFlowHub protocols.
  - Deleting the existing `debug-latest` release/tag.

#### Use Cases

1. A maintainer pushes `master`; CI validates all platforms and refreshes `debug-latest`.
2. A maintainer pushes `v1.2.3`; CI validates all platforms and creates or updates the `v1.2.3` Release with all expected assets.
3. The CI moves `debug-latest`; the workflow must not treat that movable tag as a stable release tag.
4. A platform artifact is missing; publish jobs fail before creating a misleading release.

#### Functional Requirements

- Trigger on `push` to `master`, `push` tags matching `v*`, `pull_request`, and `workflow_dispatch`.
- Keep `publish-debug-latest` gated to `github.event_name == 'push' && github.ref == 'refs/heads/master'`.
- Add a release publish job gated to `github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')`.
- The release job must depend on every build job and upload the same required assets as `debug-latest`.
- The release job must derive the release tag from `github.ref_name`, not from user input.
- Release notes must include commit SHA, run URL, and asset names.

#### Non-functional Requirements

- Avoid duplicated asset lists where practical, but keep the change small and reviewable.
- Do not weaken the existing native exit code guardrails.
- Keep GitHub token permissions minimal and scoped to publish jobs.
- Keep the workflow readable enough for future CI troubleshooting.

#### Inputs / Outputs

- Inputs:
  - GitHub Actions events: `push`, `pull_request`, `workflow_dispatch`.
  - Version tag name such as `v1.2.3`.
  - Existing build artifacts downloaded from Actions artifact storage.
- Outputs:
  - Existing `debug-latest` prerelease on `master` pushes.
  - New stable Release for pushed `v*` tags.
  - Step summary with release URL and asset links.

#### Edge Cases

- Missing artifact: publish job fails and prints discovered artifact files.
- Re-running a tag workflow: release edit/upload uses clobber behavior.
- Non-version tag such as `debug-latest`: ignored by trigger or release gate.
- Pull request and manual dispatch: build artifacts only; no publish job.

#### Acceptance Criteria

- Workflow syntax parses.
- `debug-latest` publish condition remains unchanged for `master`.
- Tag release publish path exists and is limited to `refs/tags/v*`.
- Required asset list is validated before release upload.
- README explains both debug preview and version tag release behavior.

#### Risks

- A broad tag trigger could turn movable tags into release runs; mitigated by `v*` tag filters and job condition.
- Without production signing, generated releases are still debug/preview artifacts unless later CI adds signed release builds.
- GitHub Actions tag workflow cannot be fully proven locally; validation will include syntax and condition inspection, then remote run after merge/tag push.

#### Issue List

- None.

### Stage 2 - Architecture Design

#### Overall Solution

Extend the existing `debug-latest` workflow instead of creating a separate duplicate workflow. The current build jobs already produce the assets needed for a release, so the safest change is to add tag triggers and a second publish job that consumes the same artifacts.

#### Alternatives Considered

- Separate `release.yml` workflow:
  - Rejected for this iteration because it would duplicate the full multi-platform build matrix and increase drift risk.
- Publish all tags:
  - Rejected because the workflow itself moves `debug-latest`; release publishing must be limited to stable version tags.
- Use GitHub Release generated notes only:
  - Not enough because these assets are custom debug artifacts and should be listed explicitly.

#### Module Responsibilities

- `.github/workflows/debug-latest.yml`:
  - Build all current platform artifacts.
  - Publish `debug-latest` for `master` pushes.
  - Publish version releases for `v*` tag pushes.
- `README.md`:
  - Document the two release channels and tag convention.
- `docs/change`:
  - Archive the workflow change and release behavior.

#### Data / Call Flow

1. GitHub receives `push` to `master` or `refs/tags/v*`.
2. Build jobs produce artifacts.
3. `publish-debug-latest` runs only for `refs/heads/master`.
4. `publish-tag-release` runs only for `refs/tags/v*`.
5. Publish job downloads artifacts with `merge-multiple: true`.
6. Publish job validates every required asset path.
7. Publish job creates/edits the matching release and uploads assets with `--clobber`.

#### Interface Drafts

- Version tag pattern: `v*`.
- Release tag source: `${GITHUB_REF_NAME}`.
- Release title: `MyFlowHub ClipboardNode ${GITHUB_REF_NAME}`.
- Required assets:
  - `myflowhub-clipboardnode-windows-debug.zip`
  - `myflowhub-clipboardnode-linux-debug.tar.gz`
  - `myflowhub-clipboardnode-macos-debug.zip`
  - `myflowhub-clipboardnode-android-debug.apk`
  - `myflowhub-clipboardnode-ios-simulator-debug.zip`
  - `myflowhub-clipboardnode-web-debug.zip`
  - `clipboardnode-windows-amd64.exe`
  - `clipboardnode-bridge-windows-amd64.exe`

#### Error Handling and Safety

- Keep `permissions: contents: read` globally; publish jobs opt into `contents: write`.
- Fail explicitly on missing assets.
- Use tag-derived values only; no free-form dispatch input for release tags.
- Keep release and debug concurrency groups separate.

#### Performance and Testing Strategy

- Build job cost is unchanged for `master` and PRs.
- Tag push intentionally pays the same all-platform build cost before release.
- Validate with workflow YAML parse, targeted text assertions, `git diff --check`, and existing Go tests.
- Remote proof requires pushing a `v*` tag after merge.

#### Extensibility Design Points

- Future signed production release builds can be added as separate artifacts or a separate release workflow once signing requirements are known.
- The tag pattern can be tightened from `v*` to SemVer-specific matching if GitHub Actions filter limitations are acceptable.

#### Issue List

- None.

### Stage 3.1 - Planning

#### Project Goal and Current State

Goal: improve CI/CD by adding automatic GitHub Release publishing for version tags while retaining the existing `debug-latest` prerelease channel.

Current state: all-platform debug CI exists and works; `debug-latest` is published only from `master` pushes; tag pushes are ignored.

#### Docs Governance Routing Decision

Using `$m-docs`:

- Requirements impact: none
- Specs impact: none
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons: `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`
- Stable product requirements/specs do not change because this workflow changes release automation only.
- A `docs/change` archive is required. A new lesson is not expected unless validation reveals a reusable CI failure mode.

#### Executable Task List

- [x] `CI-REL-1` - Extend workflow triggers and publish job.
- [x] `CI-REL-2` - Update release documentation.
- [x] `CI-REL-3` - Validate workflow syntax and release gating.
- [ ] `CI-REL-4` - Archive the change.

#### Task Details

##### CI-REL-1 - Extend workflow triggers and publish job

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: add `v*` tag release publishing without disrupting `debug-latest`.
- Files / Modules: `.github/workflows/debug-latest.yml`
- Write Set: workflow trigger and publish job.
- Acceptance: `master` publishes `debug-latest`; `refs/tags/v*` publishes matching stable release; non-version tags are ignored.
- Test Points: YAML parse; assert conditions and asset list.
- Rollback: revert workflow changes.

##### CI-REL-2 - Update release documentation

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: explain debug and version release channels.
- Files / Modules: `README.md`
- Write Set: Debug Preview / Release section text.
- Acceptance: README documents `debug-latest`, `v*` tag releases, and unsigned/debug artifact caveat.
- Test Points: read final README section.
- Rollback: revert README changes.

##### CI-REL-3 - Validate workflow syntax and release gating

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: catch syntax errors and accidental broad tag release behavior.
- Files / Modules: `.github/workflows/debug-latest.yml`
- Write Set: none expected except fixes if validation fails.
- Acceptance: YAML parses and release gate checks pass.
- Test Points: Ruby YAML parse or available parser; string assertions; `git diff --check`; `GOWORK=off go test ./... -count=1`.
- Rollback: fix or revert failing changes.

##### CI-REL-4 - Archive the change

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: create handoff-ready archive for the CI/CD change.
- Files / Modules: `docs/change/2026-06-02_tag-release-ci.md`, `docs/change/README.md`
- Write Set: new change archive and index.
- Acceptance: archive records requirements/specs/lessons impact, task mapping, validation, risks, and rollback.
- Test Points: read docs and index.
- Rollback: delete archive/index entry.

#### Dependencies

- GitHub CLI is useful for remote verification after push/tag, but local implementation does not require it.
- GitHub token in Actions must allow `contents: write` in publish jobs, matching current `debug-latest` behavior.

#### Risks and Notes

- This adds release-on-tag behavior for debug artifacts, not signed production binaries.
- If strict SemVer matching is required later, refine the tag filter and document it.
- PowerShell CI guardrail from the existing lesson must remain intact.

#### Parallelism Assessment

- Parallelism available: low.
- Rationale: workflow, README, and archive changes are tightly coupled and small; sub-agent delegation would add coordination overhead.
- Sub-agent use: none.

#### Issue List

- None.

阻塞：否
进入 3.2

## Stage 3.2 - Implementation Summary

- `CI-REL-1`: updated `.github/workflows/debug-latest.yml` so `push` triggers include `master` and `v*` tags. Added `publish-tag-release`, which depends on all existing build jobs, validates the same required artifact list, creates or updates the pushed version tag release, uploads assets with `--clobber`, and marks existing releases as non-draft, non-prerelease, and latest through the GitHub REST API.
- `CI-REL-2`: updated `README.md` from a debug-only section to release channels, documenting the existing `debug-latest` prerelease and the new `v*` tag release behavior.
- `CI-REL-3`: validation passed for YAML parsing, workflow trigger assertions, release job dependency matching, release gate assertions, `git diff --check`, Go tests, and hosted `workflow_dispatch` run `26801062682`.

## Stage 3.3 - Code Review

Stage 3.3 review result: passed.

- 需求覆盖: 通过. `debug-latest` remains on `master`; pushed `v*` tags now publish matching GitHub Releases.
- 架构合理性: 通过. The release job reuses existing all-platform build jobs instead of duplicating a workflow.
- 性能风险: 通过. Existing build cost is unchanged for PR/manual/master runs; only tag pushes add a publish job after the same build matrix.
- 可读性与一致性: 通过. Asset validation mirrors `publish-debug-latest`; release and debug publish jobs remain clearly separated.
- 可扩展性与配置化: 通过. Release tag pattern is explicit and can be tightened later if strict SemVer is required.
- 稳定性与安全: 通过. Global permissions stay read-only; publish jobs opt into `contents: write`; `debug-latest` is excluded from version release triggers.
- 测试覆盖情况: 通过. YAML parse, condition assertions, dependency assertions, `git diff --check`, and `GOWORK=off go test ./... -count=1` passed.
- 远端验证: 通过. GitHub Actions run `26801062682` on `chore/tag-release-ci` passed Go CLI, Windows, Linux, macOS, Android, iOS simulator, and Web build jobs. `Publish tag release` and `Publish debug-latest` were skipped as expected for `workflow_dispatch`.
- 子Agent治理与审计: 通过. Parallelism was assessed in Stage 3.1; no sub-agent was dispatched.

阻塞：否
进入 4

## Stage 4 - Change Archive

使用 `$m-docs` 校验 change/lessons 路由、requirements/specs 影响和索引维护。

- Requirements impact: `none`; release automation does not change `docs/requirements/clipboard-sync.md`.
- Specs impact: `none`; release automation does not change `docs/specs/clipboard-sync.md`.
- Lessons impact: `none`; no new reusable failure mode emerged.
- Change archive: `docs/change/2026-06-02_tag-release-ci.md`.
- Hosted validation: `https://github.com/yttydcs/myflowhub-clipboardnode/actions/runs/26801062682`
- Related requirements:
  - `docs/requirements/clipboard-sync.md`
- Related specs:
  - `docs/specs/clipboard-sync.md`
- Related lessons:
  - `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`
- Indexes updated:
  - `docs/change/README.md`

Workflow is ready for branch commit, push, remote CI validation, and user decision on workflow end after hosted validation.

## Scope Revision - Release-mode Production Packages

Reason: after the first tag-release implementation, the user clarified that a
version tag must not publish the same unsigned debug preview assets. The
workflow is rolling back the intermediate "tag release reuses debug assets"
approach and replacing it with a dedicated release-mode pipeline.

Rollback trace:

- Supersede `CI-REL-1` publish-on-tag behavior inside `debug-latest.yml`.
- Keep `debug-latest` as the preview/debug prerelease channel only.
- Add a new `release.yml` workflow for `vX.Y.Z` version tags.
- Update docs/archive so the release contract is not described as production
  signed until the required platform signing secrets are present.

### Stage 1 - Requirements Analysis Revision 2

#### Goal

Keep `debug-latest` for debug previews and implement a separate tag-driven
release workflow that builds release-mode packages for desktop, mobile, web,
and Go helper assets.

#### Scope

- Must:
  - Keep `debug-latest` behavior on `master` pushes.
  - Prevent `debug-latest` or any non-version tag from publishing a stable release.
  - Build release-mode Flutter assets for Windows, Linux, macOS, Android, iOS, and Web.
  - Build Go helper executables used by the desktop packaging path.
  - Publish a stable GitHub Release only for pushed version tags matching `vX.Y.Z`.
  - Validate and fail explicitly when a required release asset is missing.
  - Provide an Android release signing configuration driven by CI secrets.
  - Define the required CI secrets for Android, Windows signing, macOS signing/notarization, and iOS IPA export.
  - Keep `workflow_dispatch` usable for dry-run validation without creating a GitHub Release.
- Optional:
  - Sign Windows executables and macOS apps when the corresponding secrets are configured.
  - Produce Android APK and AAB assets.
- Not doing:
  - App Store / Play Store upload.
  - MSI/DMG installer generation.
  - Runtime clipboard, MyFlowHub protocol, or UI behavior changes.
  - Creating or pushing a real `v*` tag from this workflow without explicit user approval.

#### Use Cases

1. Maintainer pushes `master`; debug CI refreshes `debug-latest` only.
2. Maintainer pushes `v1.2.3`; release CI builds release-mode platform assets and publishes Release `v1.2.3`.
3. Maintainer runs `release.yml` manually on a branch; CI validates release build paths but publish is skipped.
4. Android release secrets are missing during a tag release; Android release job fails before a misleading APK/AAB is published.
5. iOS signing/profile secrets are missing during a tag release; iOS release job fails before a fake production IPA is published.

#### Functional Requirements

- `.github/workflows/debug-latest.yml` must ignore tag publishing and contain no stable release publish job.
- `.github/workflows/release.yml` must trigger on `push.tags: v*` and `workflow_dispatch`.
- The release workflow must derive `build-name` from the tag after removing the leading `v`.
- The release workflow must fail non-`vX.Y.Z` release tags early.
- The release workflow must publish only on tag push, not on manual branch dispatch.
- Android Gradle release signing must use release keystore values when provided and retain a clear local fallback for non-publishing dry runs.
- Release notes must include tag, commit SHA, run URL, release-mode assets, and signing/notarization status.

#### Non-functional Requirements

- Keep global GitHub Actions permissions read-only; publish job opts into `contents: write`.
- Avoid silent unsigned production claims.
- Keep the existing PowerShell native exit checks.
- Keep platform build failures local to their jobs and preserve build logs where useful.
- Keep changes narrow: workflow files, Android signing config, README, plan/change docs.

#### Inputs / Outputs

- Inputs:
  - `push` tag `vX.Y.Z`.
  - Manual `workflow_dispatch` dry-run `release_tag`.
  - GitHub Secrets / Variables for signing.
  - Generated gomobile AAR/XCFramework artifacts.
- Outputs:
  - GitHub Actions release artifacts.
  - Stable GitHub Release for pushed `vX.Y.Z` tags.
  - Release-mode archives: Windows zip, Linux tar.gz, macOS zip, Android APK/AAB, iOS IPA, Web zip, Windows Go helper executables.

#### Edge Cases

- Non-SemVer tag: release workflow fails before builds.
- Missing signing secret on tag release: affected signed platform job fails explicitly.
- Manual dispatch without signing secrets: dry-run release-mode builds can proceed where platform tooling permits, but publish is skipped.
- Re-run for an existing tag: release publish updates notes and uploads with `--clobber`.

#### Acceptance Criteria

- YAML syntax parses for both workflows.
- `debug-latest.yml` no longer has `publish-tag-release`.
- `release.yml` exists with release-mode build jobs and a tag-only publish gate.
- Android release signing is configurable by CI secrets and no longer hardcodes debug signing as the only release path.
- README documents debug and release channels, tag format, and signing secrets.
- Local Go tests and workflow assertions pass.

#### Risks

- Production signing is secret-dependent and cannot be fully proven locally.
- macOS notarization and iOS IPA export depend on Apple account/certificate state.
- Android release APK/AAB can build locally with debug fallback for dry-run, but a tag release should require real signing secrets.
- A real tag push would publish a stable release; avoid creating one during validation unless the user approves.

#### Issue List

- None.

### Stage 2 - Architecture Design Revision 2

#### Overall Solution

Split preview and release channels. `debug-latest.yml` returns to the existing
debug-only channel. A new `release.yml` owns version tag publishing and builds
release-mode artifacts with explicit signing gates for platforms that need
production credentials.

#### Alternatives Considered

- Keep one workflow and add release-mode branches inside it:
  - Rejected because the debug workflow is already large and mixing debug/release asset paths increases drift and publish-risk.
- Publish unsigned release-mode assets if secrets are missing:
  - Rejected for tag releases because the user explicitly asked to move beyond unsigned debug preview assets.
- Require all signing secrets for manual dry-runs:
  - Rejected because branch validation should prove CI structure without forcing release credentials into every test run.

#### Module Responsibilities

- `.github/workflows/debug-latest.yml`: debug preview build and `debug-latest` prerelease only.
- `.github/workflows/release.yml`: release tag validation, release-mode builds, signing/notarization gates, asset validation, and stable release publishing.
- `app/android/app/build.gradle.kts`: Android release signing config driven by Gradle properties or environment variables.
- `README.md`: release channel behavior, tag format, and signing secret contract.
- `docs/change`: workflow archive and validation record.

#### Data / Call Flow

1. Tag push `refs/tags/vX.Y.Z` starts `release.yml`.
2. `prepare-release` validates tag format and exposes `release_tag`, `build_name`, `build_number`, and publish mode.
3. Platform jobs build release-mode artifacts and package `build-info.txt`.
4. Signed platforms check secrets on tag releases and fail explicitly if required credentials are absent.
5. `publish-release` downloads artifacts, validates exact asset names, creates/updates the GitHub Release, and uploads with `--clobber`.
6. Manual `workflow_dispatch` follows steps 2-4 but skips `publish-release`.

#### Interface Drafts

- Release tag pattern: `vX.Y.Z`, for example `v1.2.3`.
- Manual dry-run input: `release_tag`, default `v0.0.0`.
- Android secrets:
  - `ANDROID_KEYSTORE_BASE64`
  - `ANDROID_KEYSTORE_PASSWORD`
  - `ANDROID_KEY_ALIAS`
  - `ANDROID_KEY_PASSWORD`
- Windows signing secrets:
  - `WINDOWS_CODESIGN_PFX_BASE64`
  - `WINDOWS_CODESIGN_PFX_PASSWORD`
  - optional `WINDOWS_CODESIGN_TIMESTAMP_URL`
- macOS signing/notarization secrets:
  - `MACOS_DEVELOPER_ID_CERT_BASE64`
  - `MACOS_DEVELOPER_ID_CERT_PASSWORD`
  - `MACOS_DEVELOPER_IDENTITY`
  - `APPLE_NOTARY_KEY_ID`
  - `APPLE_NOTARY_ISSUER_ID`
  - `APPLE_NOTARY_KEY_BASE64`
- iOS signing/export secrets:
  - `IOS_DISTRIBUTION_CERT_BASE64`
  - `IOS_DISTRIBUTION_CERT_PASSWORD`
  - `IOS_PROVISIONING_PROFILE_BASE64`
  - `IOS_DEVELOPMENT_TEAM`
  - optional `IOS_EXPORT_METHOD`

#### Error Handling and Safety

- Fail early on non-`vX.Y.Z` tag names.
- Fail tag releases when required signing/export secrets are missing.
- Keep dry-run `workflow_dispatch` publish-disabled.
- Validate all required release assets before creating/updating release notes.
- Never use user-provided release tag for automatic tag push; only `github.ref_name`.

#### Performance and Testing Strategy

- Debug CI cost is unchanged because tag release work moves to `release.yml`.
- Tag release cost is full-platform release build plus signing/notarization.
- Validate locally with YAML parse, job/asset assertions, Gradle signing config assertions, `git diff --check`, and Go tests.
- Remote validation uses `workflow_dispatch` on the branch, with publish skipped.

#### Extensibility Design Points

- Installer generation can be added as separate jobs without changing the release publish contract.
- Store uploads can depend on the signed APK/AAB/IPA jobs later.
- Strict SemVer variants can be added by changing only `prepare-release`.

#### Issue List

- None.

### Stage 3.1 - Planning Revision 2

#### Project Goal and Current State

Goal: replace the intermediate debug-asset tag release with a dedicated
release-mode package workflow while keeping `debug-latest` unchanged for debug
preview users.

Current state: branch `chore/tag-release-ci` contains the first tag-release
implementation and remote proof for debug workflow dispatch. This revision
will supersede that implementation before workflow closeout.

#### Docs Governance Routing Decision

Using `$m-docs`:

- Requirements impact: none
- Specs impact: none
- Related requirements: `docs/requirements/clipboard-sync.md`
- Related specs: `docs/specs/clipboard-sync.md`
- Related lessons:
  - `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`
  - `docs/lessons/gomobile-mobile-bindings.md`
- Stable product requirements/specs do not change because the release workflow changes packaging and CI/CD only.
- The existing `docs/change/2026-06-02_tag-release-ci.md` archive should be updated for the expanded scope; no new lessons are expected unless validation exposes a reusable failure mode.

#### Executable Task List Revision 2

- [x] `CI-REL-5` - Separate debug preview and release workflows.
- [x] `CI-REL-6` - Build desktop, web, and Go release-mode assets.
- [x] `CI-REL-7` - Add Android release signing config and APK/AAB packaging.
- [x] `CI-REL-8` - Add Apple release signing/notarization and iOS IPA packaging gates.
- [x] `CI-REL-9` - Update README and change archive.
- [x] `CI-REL-10` - Validate locally and record hosted dry-run limitation.

#### Task Details Revision 2

##### CI-REL-5 - Separate debug preview and release workflows

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: keep `debug-latest.yml` debug-only and create a dedicated `release.yml`.
- Files / Modules: `.github/workflows/debug-latest.yml`, `.github/workflows/release.yml`
- Write Set: workflow trigger/publish structure.
- Acceptance: debug workflow has no tag release publish job; release workflow triggers on `v*` tags and manual dry-runs.
- Test Points: YAML parse and job/trigger assertions.
- Rollback: delete `release.yml` and restore previous debug workflow.

##### CI-REL-6 - Build desktop, web, and Go release-mode assets

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: produce release-mode Windows, Linux, macOS, Web, and Windows Go helper assets.
- Files / Modules: `.github/workflows/release.yml`
- Write Set: release build jobs and packaging scripts inside workflow.
- Acceptance: release workflow packages exact required desktop/web/Go asset names and build metadata.
- Test Points: workflow assertions for `--release`, package paths, and required assets.
- Rollback: remove affected release jobs.

##### CI-REL-7 - Add Android release signing config and APK/AAB packaging

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: allow signed Android release APK/AAB from CI secrets while preserving dry-run fallback.
- Files / Modules: `app/android/app/build.gradle.kts`, `.github/workflows/release.yml`
- Write Set: Gradle release signing config and Android release job.
- Acceptance: tag releases require Android signing secrets; APK and AAB assets are packaged.
- Test Points: Gradle config assertions and workflow secret-gate assertions.
- Rollback: restore debug-signing release block and remove Android release job.

##### CI-REL-8 - Add Apple release signing/notarization and iOS IPA packaging gates

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: build macOS release app with optional notarization path and build signed iOS IPA when secrets are configured.
- Files / Modules: `.github/workflows/release.yml`
- Write Set: macOS release job and iOS release job.
- Acceptance: tag release requires Apple signing/export secrets; dry-run can validate unsigned release build paths where possible.
- Test Points: workflow assertions for secret gates, `flutter build macos --release`, iOS XCFramework generation, and IPA asset path.
- Rollback: remove Apple release jobs.

##### CI-REL-9 - Update README and change archive

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: document final debug/release behavior and CI secret contract.
- Files / Modules: `README.md`, `docs/change/2026-06-02_tag-release-ci.md`, `docs/change/README.md`
- Write Set: release documentation and archive.
- Acceptance: docs no longer claim tag releases publish debug assets; signing requirements are discoverable.
- Test Points: final read-through.
- Rollback: revert documentation edits.

##### CI-REL-10 - Validate locally and with hosted dry-run

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Plan Path: `plan.md`
- Goal: prove workflow syntax and non-publishing dry-run behavior before workflow closeout.
- Files / Modules: validation only except fixes.
- Write Set: none expected.
- Acceptance: local assertions and Go tests pass; hosted `workflow_dispatch` run passes or any credential-dependent limitation is explicitly recorded.
- Test Points: YAML parse, assertions, `git diff --check`, `GOWORK=off go test ./... -count=1`, optional `gh workflow run release.yml`.
- Rollback: fix or revert failing tasks.

#### Dependencies

- GitHub Actions hosted runners for Windows, Linux, macOS.
- GitHub Secrets for production signing on real tag releases.
- GitHub CLI for hosted dry-run validation.

#### Risks and Notes

- Codesigning/notarization cannot be proven locally without private certificates.
- A manual dry-run proves build structure, not production credential validity.
- iOS App Store / TestFlight upload remains outside this workflow.

#### Parallelism Assessment Revision 2

- Parallelism available: low.
- Rationale: release workflow asset names, docs, and Gradle signing config are tightly coupled; splitting implementation would risk divergent signing gates and asset validation.
- Sub-agent use: none.

#### Issue List

- None.

阻塞：否
进入 3.2

### Stage 3.2 - Implementation Summary Revision 2

- `CI-REL-5`: restored `.github/workflows/debug-latest.yml` to debug-only behavior with `tags-ignore: "**"` and removed the intermediate `publish-tag-release` job. Added `.github/workflows/release.yml` as the dedicated version release workflow.
- `CI-REL-6`: added release-mode Windows, Linux, macOS, Web, and Windows Go helper packaging jobs. Release assets use `*-release` names and include build metadata.
- `CI-REL-7`: updated `app/android/app/build.gradle.kts` so release signing reads CI/Gradle properties when provided and uses a local/dry-run debug fallback only when release signing is absent. Added Android release APK/AAB workflow packaging and tag-release secret gates.
- `CI-REL-8`: added macOS Developer ID signing/notarization gates and iOS distribution certificate/profile export gates. Manual dispatch can dry-run unsigned Apple build paths; tag release requires production credentials.
- `CI-REL-9`: updated `README.md` and `docs/change/2026-06-02_tag-release-ci.md` to document final debug/release channel behavior and signing secret contracts.
- `CI-REL-10`: local validation passed for workflow syntax/assertions, Android signing assertions, `git diff --check`, Go tests, and Android release APK dry-run build. Hosted dry-run dispatch was attempted after push and is blocked until `release.yml` exists on the default branch.

File-level design notes before editing:

- Debug preview and stable release are separate workflows to prevent debug artifacts from being described as production release packages.
- Manual `workflow_dispatch` for `release.yml` intentionally does not publish GitHub Releases.
- Real tag release fails fast when required signing secrets are absent, preventing misleading unsigned production assets.
- Android Gradle fallback remains only to support local/manual dry-runs; the workflow guards tag release publishing.

Parallelism assessment:

- Parallelism available: low.
- Sub-agent use: none.
- Rationale: release workflow asset names, publish validation, signing gates, README, plan, and archive are coupled; splitting implementation would increase drift risk.

### Stage 3.3 - Code Review Revision 2

Stage 3.3 review result: passed.

- 需求覆盖: 通过. `debug-latest` is preserved as debug preview; pushed `vX.Y.Z` tags use a dedicated release-mode workflow.
- 架构合理性: 通过. Debug and release publish paths are separated; release publish depends on all release build jobs.
- 性能风险: 通过. Debug CI cost is unchanged; release workflow runs only on version tag push or manual dry-run.
- 可读性与一致性: 通过. Release assets use consistent `*-release` names and publish validation uses one explicit required asset list.
- 可扩展性与配置化: 通过. Signing secrets are explicit per platform; installer/store uploads can be added as later jobs.
- 稳定性与安全: 通过. Global permissions remain read-only; publish job opts into `contents: write`; tag release fails when signing secrets are missing.
- 测试覆盖情况: 通过 for local scope. YAML parse, workflow assertions, Android signing assertions, `git diff --check`, `GOWORK=off go test ./... -count=1`, and Android release APK dry-run build passed.
- 子Agent治理与审计: 通过. Parallelism was assessed; no sub-agent was dispatched.

阻塞：否
进入 4

### Stage 4 - Change Archive Revision 2

使用 `$m-docs` 校验 change/lessons 路由、requirements/specs 影响和索引维护。

- Requirements impact: `none`; release automation and signing gates do not change `docs/requirements/clipboard-sync.md`.
- Specs impact: `none`; release automation and packaging do not change `docs/specs/clipboard-sync.md`.
- Lessons impact: `none`; no new reusable failure mode emerged.
- Change archive: `docs/change/2026-06-02_tag-release-ci.md`.
- Related requirements:
  - `docs/requirements/clipboard-sync.md`
- Related specs:
  - `docs/specs/clipboard-sync.md`
- Related lessons:
  - `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`
  - `docs/lessons/gomobile-mobile-bindings.md`
- Indexes updated:
  - No new change index entry was needed because the existing archive path stayed the same.

Local validation recorded:

- Python/PyYAML parsed `.github/workflows/debug-latest.yml` and `.github/workflows/release.yml`.
- Workflow assertions passed for debug-only tag ignore, release job set, publish gate, required dependencies, release-mode build commands, signing gates, and asset names.
- Android Gradle signing assertions passed.
- `git diff --check` passed.
- `$env:GOWORK='off'; go test ./... -count=1` passed.
- `flutter build apk --release --build-name 0.0.0 --build-number 1` passed locally, producing `app-release.apk` through the dry-run debug-signing fallback.

External validation record:

- Push branch commit.
- Hosted `workflow_dispatch` dry-run for `release.yml` was attempted with:
  - `gh workflow run release.yml --repo yttydcs/myflowhub-clipboardnode --ref chore/tag-release-ci -f release_tag=v0.0.0`
  - Result: blocked by GitHub API with `HTTP 404: workflow release.yml not found on the default branch`.
  - Reason: GitHub only exposes workflow dispatch for workflows already present on the default branch; current `release.yml` is new on `chore/tag-release-ci`.
  - Follow-up: after merge to `master`, run the same dry-run command before pushing a real `vX.Y.Z` tag.
- Do not create a real `vX.Y.Z` tag without explicit user approval because that would publish a stable release.
