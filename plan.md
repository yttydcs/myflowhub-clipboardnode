# User-Friendly Download Package

## Project Goal And Current State

Goal: make downloadable ClipboardNode desktop packages usable after manual
extraction without requiring users to download separate helpers or guess which
executable to start. The Windows desktop package should expose a professional
GUI executable entry, not a batch-file launcher.

Current state:

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-ClipboardNode`
- Branch: `fix/user-friendly-download-package`
- Base: `master`
- Worktree: `D:/project/MyFlowHub3/worktrees/fix-user-friendly-download-package/MyFlowHub-ClipboardNode`
- Current Stage: `4 - Change Archive`
- `debug-latest` publishes multiple assets. The Windows desktop zip already
  contains the Flutter runner and `clipboardnode-bridge.exe`, but users can
  confuse it with the Web bundle or standalone Go helper executables.
- The current Windows GUI executable name `myflowhub_clipboard.exe` is less
  user-facing than `ClipboardNode.exe`.

## Related Requirements / Specs / Lessons

- Requirements: `docs/requirements/clipboard-sync.md`
  - ClipboardNode must be a full cross-platform UI app, not only a headless
    node.
- Specs: `docs/specs/clipboard-sync.md`
  - Flutter UI shell is the product UI.
  - Desktop Flutter UI uses a narrow local JSON bridge to the Go engine.
- Related lessons:
  - `docs/lessons/debug-latest-ci-native-exit-flutter-material.md`
  - `docs/lessons/flutter-windows-sdk-shared-bat-git.md`

## Scope

Must:

- Keep the Windows desktop zip self-contained: Flutter runner plus local
  `clipboardnode-bridge.exe`.
- Rename the Windows Flutter GUI executable to `ClipboardNode.exe`.
- Add short package-local instructions that tell users to start
  `ClipboardNode.exe` and explain which files are helpers.
- Update README download guidance so users pick the Windows desktop zip and
  understand that the Web bundle is not the desktop package.
- Archive the workflow change.

Not in scope:

- Runtime sync behavior changes.
- A new installer/MSI/NSIS package.
- A true single-file Flutter executable. Flutter Windows still needs bundled
  DLLs and `data/` next to the executable.
- MyFlowHub protocol, auth, TopicBus, Stream, File, SDK, Server, Proto, or
  SubProto changes.

## Acceptance Criteria

- Windows debug and release builds produce `ClipboardNode.exe`.
- Windows debug and release zips include `README-WINDOWS.txt`.
- Users can start the extracted desktop package by double-clicking
  `ClipboardNode.exe`, with no separate downloads.
- README clearly says the standalone Go executables are not the desktop UI.
- README clearly says the Web zip is for web hosting/preview, not the Windows
  desktop quick-start package.
- Workflow syntax and changed package commands are validated locally.

## Tasks

### PKG-1 - Add Windows package GUI exe and package readme

- Files / Modules:
  - `app/windows/CMakeLists.txt`
  - `app/windows/runner/Runner.rc`
  - `.github/workflows/debug-latest.yml`
  - `.github/workflows/release.yml`
- Goal: expose a clear GUI executable name and package-local user guidance in
  Windows debug and release zips.
- Acceptance:
  - Flutter Windows produces `ClipboardNode.exe`.
  - `README-WINDOWS.txt` exists in the zip root.
  - The package readme tells users to double-click `ClipboardNode.exe`.
- Tests:
  - Static assertions over workflow, CMake, and resource files.
  - YAML parse.
- Rollback:
  - Restore `myflowhub_clipboard` binary name and remove package readme script
    blocks from both workflow files.

### PKG-2 - Clarify release/download documentation

- Files / Modules:
  - `README.md`
- Goal: make the correct download obvious and prevent confusion with Web and
  helper assets.
- Acceptance:
  - Debug and release sections include a Windows quick-start note.
  - Helper binaries are explicitly documented as non-UI support/diagnostic
    assets.
  - Web bundle is documented as not the Windows desktop launcher.
- Tests:
  - README guidance assertions and `git diff --check`.
- Rollback:
  - Restore previous release channel text.

### PKG-3 - Archive workflow outcome

- Files / Modules:
  - `docs/change/2026-06-02_user-friendly-download-package.md`
  - `docs/change/README.md`
- Goal: record scope, impact, validation, and rollback.
- Acceptance:
  - Archive includes task mapping and requirements/specs impact.
  - Change index links the new archive.
- Tests:
  - `git diff --check`.
- Rollback:
  - Remove archive and index entry.

## Parallelism Assessment

Owner: main agent.

No sub-agent delegation. The write set is small and coupled: executable name,
workflow package contents, README guidance, and archive wording need to stay
consistent.

## Stage 3.2 Implementation Summary

- `PKG-1`: updated `app/windows/CMakeLists.txt` and
  `app/windows/runner/Runner.rc` so Windows desktop builds produce
  `ClipboardNode.exe`.
- `PKG-1`: updated `.github/workflows/debug-latest.yml` and
  `.github/workflows/release.yml` so Windows desktop packages include
  `README-WINDOWS.txt` that points users to `ClipboardNode.exe`.
- `PKG-2`: updated `README.md` to call out the Windows desktop quick-start
  package, separate helper binaries from the UI, and clarify that Web bundles
  are not Windows desktop quick-start packages.
- `PKG-3`: added `docs/change/2026-06-02_user-friendly-download-package.md`
  and indexed it in `docs/change/README.md`.

## Stage 3.3 Code Review

- 需求覆盖: 通过. Downloaded Windows desktop zips are now self-contained and
  expose `ClipboardNode.exe` as the direct GUI entry.
- 架构合理性: 通过. Runtime and Flutter/Go bridge boundaries are unchanged.
- 性能风险: 通过. Only one small package readme is added to Windows zips.
- 可读性与一致性: 通过. Debug and release workflows use the same package readme
  pattern and README wording distinguishes desktop, Web, and helper assets.
- 可扩展性与配置化: 通过. Installer work remains out of scope and can be added
  later without changing the desktop zip contract.
- 稳定性与安全: 通过. No script launcher is used; users start the GUI executable
  produced by Flutter's Windows runner.
- 测试覆盖情况: 通过. Static assertions, YAML parse,
  `flutter build windows --debug`, and `git diff --check` passed. Go tests were
  not run because Go runtime/bridge code did not change.
- 子Agent治理与审计: 通过. Parallelism was assessed; no sub-agent was dispatched.

## Stage 4 Change Archive

Using `$m-docs`:

- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
- Change archive: `docs/change/2026-06-02_user-friendly-download-package.md`
- Index updated: `docs/change/README.md`

Validation:

- Workflow/package static assertions: passed.
- Windows executable naming assertions: passed.
- README guidance assertions: passed.
- Python/PyYAML parse for `.github/workflows/debug-latest.yml`: passed.
- Python/PyYAML parse for `.github/workflows/release.yml`: passed.
- `flutter build windows --debug`: passed and produced
  `app/build/windows/x64/runner/Debug/ClipboardNode.exe`.
- `git diff --check`: passed.

阻塞：否
进入 4
