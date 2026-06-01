# MyFlowHub-ClipboardNode

Independent MyFlowHub node application for clipboard synchronization.

## Current Status

This repository contains the headless sync core, Windows host skeleton, and the
first cross-platform Flutter application shell:

- `app` contains the Flutter UI shell for desktop, mobile, and web preview. It
  currently uses a preview engine bridge while the live MyFlowHub TopicBus
  adapter is implemented.
- `bridge` defines the JSON command/event contract shared by the Flutter UI and
  the Go engine.
- `core/runtime` validates `clipboard.text.v1` payloads, publishes local text changes through a TopicBus interface, applies remote text events, deduplicates event IDs, and suppresses clipboard write loops.
- `core/configstore` persists only non-sensitive settings. Clipboard text and event bodies are never written to config.
- `windows` provides a Win32 `CF_UNICODETEXT` adapter with bounded reads and a polling watcher.
- `cmd/clipboardnode` loads config and verifies the Windows adapter, but intentionally refuses `enabled=true` until the live MyFlowHub SDK TopicBus transport is wired.

Sync is disabled by default. The first phase is text-only, online-only, and best-effort through TopicBus.

## Debug Preview

The latest automated all-platform debug build is published as a prerelease:

```text
https://github.com/yttydcs/myflowhub-clipboardnode/releases/tag/debug-latest
```

Each `master` push refreshes the movable `debug-latest` tag after all platform jobs pass and uploads:

- `myflowhub-clipboardnode-windows-debug.zip`: full Flutter Windows debug runner directory.
- `myflowhub-clipboardnode-linux-debug.tar.gz`: Flutter Linux debug bundle.
- `myflowhub-clipboardnode-macos-debug.zip`: unsigned Flutter macOS debug `.app`.
- `myflowhub-clipboardnode-android-debug.apk`: Flutter Android debug APK.
- `myflowhub-clipboardnode-ios-simulator-debug.zip`: unsigned Flutter iOS simulator debug `.app`.
- `myflowhub-clipboardnode-web-debug.zip`: Flutter Web debug bundle.
- `clipboardnode-windows-amd64.exe`: Go CLI helper for Windows.

Manual workflow runs and pull requests still build the same artifacts in GitHub Actions, but only `master` pushes update the prerelease.
Debug artifacts are unsigned preview builds, not production distribution packages.

## Scope

- Runs as its own node instead of being embedded in MyFlowHub-Win or Server.
- Uses existing MyFlowHub protocols, with the first phase expected to use TopicBus for small text clipboard events.
- Keeps platform clipboard adapters in the host layers and shared sync logic in `core/`.

## Repository Shape

```text
app/           Flutter cross-platform application shell
bridge/        Go JSON bridge contract for UI <-> engine commands/events
core/          shared runtime, config, topic event model, dedupe logic
windows/       Windows host and clipboard adapter
docs/          requirements, specs, plans, changes, lessons
scripts/       build and maintenance scripts
```

Implementation changes should be made from a dedicated worktree under `D:/project/MyFlowHub3/worktrees/`.

## Validation

This repo lives under a parent workspace that may contain a `go.work`; validate it as an independent module:

```powershell
$env:GOWORK = "off"
go test ./... -count=1
go build -o build/clipboardnode.exe ./cmd/clipboardnode
git diff --check
```

Or run:

```powershell
.\scripts\validate.ps1
```

Flutter validation uses the local Flutter SDK selected during this workflow:

```powershell
$flutterRoot = "D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter"
$env:PATH = "$flutterRoot\bin;$env:PATH"
$env:PUB_CACHE = "$flutterRoot\.pub-cache"
cd app
flutter analyze
flutter test
flutter build windows --debug
flutter build web --debug
```

Linux, macOS, Android, and iOS simulator debug builds are validated by GitHub Actions on their matching hosted runners.

