# 2026-05-31 Clipboard Cross-platform App Shell

## Background

ClipboardNode needed to move from a headless Windows-first skeleton toward a complete cross-platform application with UI, while preserving the existing protocol decision: small text uses TopicBus application events and no MyFlowHub subprotocol wire contract changes are introduced.

## Changes

- Created `app/` Flutter project with Windows, Android, iOS, and Web targets.
- Implemented a responsive ClipboardNode UI shell:
  - overview;
  - manual send;
  - settings;
  - activity metadata.
- Added Flutter DTOs and preview bridge under `app/lib/core/bridge`.
- Added Go `bridge/` package for JSON command/event contract:
  - `EngineCommand`;
  - `EngineEvent`;
  - `Settings`;
  - `Status`;
  - `Activity`;
  - `PlatformCapability`.
- Extended Go runtime config with:
  - `auto_watch`;
  - `auto_apply`;
  - `history_retention`.
- Updated `README.md` with Flutter validation and preview commands.
- Updated `plan.md` with APP-1 completion, APP-2/3/5/6 results, and remaining APP-4 live TopicBus work.

## Related Plan

- Root `plan.md`
- Task mapping:
  - `APP-1`: Flutter 3.41.9 toolchain installed and verified.
  - `APP-2`: Flutter app shell created.
  - `APP-3`: Initial bridge contract created.
  - `APP-5`: Initial UI screens implemented.
  - `APP-6`: Platform capability policy surfaced in UI and config.
  - `APP-7`: Validation performed for this slice.

## Related Requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related Specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Lessons Impact

updated

## Related Lessons

- [../lessons/flutter-windows-sdk-shared-bat-git.md](../lessons/flutter-windows-sdk-shared-bat-git.md)

## Searchable Lessons Summary

- Symptom: `flutter --version` or `flutter doctor` hangs on Windows with no output.
- Trigger: downloaded Flutter SDK contains `bin/internal/shared.bat` using `$git rev-parse HEAD`.
- Quick check: inspect `shared.bat` and test `cmd /d /c "for /f %r in ('PUSHD <flutterRoot> ^& $git rev-parse HEAD') do @echo %r"`.

## Requirements Impact

none

The work implements the already-documented cross-platform app direction.

## Specs Impact

none

The bridge contract follows the existing spec draft and does not change TopicBus, Stream, File, Server, Proto, SDK, or SubProto wire contracts.

## Validation

- `flutter doctor -v`
  - Flutter, Windows, Android, Chrome, Visual Studio, and connected devices passed.
  - Network resource checks for Maven and GitHub timed out.
- `flutter analyze`
  - passed.
- `flutter test`
  - passed.
- `flutter build windows`
  - passed; output `app/build/windows/x64/runner/Release/myflowhub_clipboard.exe`.
- `flutter build web`
  - passed; output `app/build/web`.
- `flutter build apk --debug`
  - not completed; Gradle/Android build exceeded the 15-minute command timeout in this environment and the spawned Flutter/Gradle processes were stopped.
- iOS build
  - not run because this validation host is Windows.
- Web smoke:
  - local preview served at `http://127.0.0.1:58341`.
  - Chrome headless screenshot generated at ignored path `app/build/web-preview-1280.png`.
- `GOWORK=off go test ./... -count=1`
  - passed.
- `GOWORK=off go build -o build/clipboardnode.exe ./cmd/clipboardnode`
  - passed.
- `git diff --check`
  - passed.

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

- Flutter remains the app shell.
- The UI uses a preview bridge until APP-4 wires live MyFlowHub TopicBus.
- Clipboard text is accepted in the manual-send view, but status/activity DTOs retain metadata only.
- Desktop and mobile capability differences are explicit UI state, not hidden implementation details.

## Remaining Work

- APP-4 live MyFlowHub TopicBus adapter.
- Native platform clipboard/share adapters behind the Flutter bridge.
- App packaging, app icons, tray/autostart, mobile share extension/foreground policies.

## Rollback

- Remove `app/`.
- Remove `bridge/`.
- Revert `core/runtime/config.go`, `README.md`, and `plan.md` changes from this slice.
