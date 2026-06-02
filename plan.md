# Plan - Tag Release CI

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `chore/tag-release-ci`
- Base: `master` at `285ce22`
- Worktree: `D:/project/MyFlowHub3/worktrees/chore-tag-release-ci/MyFlowHub-ClipboardNode`
- Current Stage: `3.1 - Planning`

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
- `CI-REL-3`: validation passed for YAML parsing, workflow trigger assertions, release job dependency matching, release gate assertions, `git diff --check`, and Go tests.

## Stage 3.3 - Code Review

Stage 3.3 review result: passed.

- 需求覆盖: 通过. `debug-latest` remains on `master`; pushed `v*` tags now publish matching GitHub Releases.
- 架构合理性: 通过. The release job reuses existing all-platform build jobs instead of duplicating a workflow.
- 性能风险: 通过. Existing build cost is unchanged for PR/manual/master runs; only tag pushes add a publish job after the same build matrix.
- 可读性与一致性: 通过. Asset validation mirrors `publish-debug-latest`; release and debug publish jobs remain clearly separated.
- 可扩展性与配置化: 通过. Release tag pattern is explicit and can be tightened later if strict SemVer is required.
- 稳定性与安全: 通过. Global permissions stay read-only; publish jobs opt into `contents: write`; `debug-latest` is excluded from version release triggers.
- 测试覆盖情况: 通过. YAML parse, condition assertions, dependency assertions, `git diff --check`, and `GOWORK=off go test ./... -count=1` passed.
- 子Agent治理与审计: 通过. Parallelism was assessed in Stage 3.1; no sub-agent was dispatched.

阻塞：否
进入 4

## Stage 4 - Change Archive

使用 `$m-docs` 校验 change/lessons 路由、requirements/specs 影响和索引维护。

- Requirements impact: `none`; release automation does not change `docs/requirements/clipboard-sync.md`.
- Specs impact: `none`; release automation does not change `docs/specs/clipboard-sync.md`.
- Lessons impact: `none`; no new reusable failure mode emerged.
- Change archive: `docs/change/2026-06-02_tag-release-ci.md`.
- Related requirements:
  - `docs/requirements/clipboard-sync.md`
- Related specs:
  - `docs/specs/clipboard-sync.md`
- Related lessons:
  - `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`
- Indexes updated:
  - `docs/change/README.md`

Workflow is ready for branch commit, push, remote CI validation, and user decision on workflow end after hosted validation.
