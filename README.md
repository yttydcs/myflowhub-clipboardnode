# MyFlowHub-ClipboardNode

Independent MyFlowHub node application for clipboard synchronization.

## Current Status

This repository contains the live sync core, desktop engine bridge, mobile
bridge foundation, and the cross-platform Flutter application shell:

- `app` contains the Flutter UI shell for desktop, mobile, and web. Desktop
  targets use the local Go engine bridge when the packaged helper is available;
  Android/iOS use a platform channel with an explicit stub fallback when the
  gomobile binding is not packaged; Web can use an explicit localhost bridge
  and otherwise remains preview/diagnostic.
- `bridge` defines the JSON command/event contract shared by the Flutter UI and
  the Go engine.
- `core/myflowhub` connects through MyFlowHub SDK/Auth/TopicBus and keeps
  TopicBus payloads as ClipboardNode application JSON.
- `core/engine` wires the transport, auth lifecycle, clipboard adapter, and
  shared runtime.
- `core/runtime` validates `clipboard.text.v1` payloads, publishes local text
  changes through a TopicBus interface, stores bounded pending remote metadata
  when `auto_apply` is disabled, applies pending events on user action,
  deduplicates event IDs, suppresses clipboard write loops, and publishes
  `clipboard.transfer.v1` metadata manifests only when an opaque transfer
  provider/reference is configured.
- `core/configstore` persists only non-sensitive settings. Clipboard text and event bodies are never written to config.
- `windows` provides a Win32 `CF_UNICODETEXT` adapter with bounded reads and a polling watcher; `platform` selects Windows, Linux, or macOS clipboard adapters.
- `cmd/clipboardnode` runs the foreground CLI engine and supports one-shot manual send.
- `cmd/clipboardnode-bridge` is the desktop stdio JSON bridge used by Flutter
  and can also expose an opt-in localhost HTTP/SSE bridge for Flutter Web.
- `nodemobile` exports the shared Go engine for Android/iOS gomobile bindings.

Sync is disabled by default. Small text sync is online-only and best-effort
through TopicBus. TopicBus publish is not a remote-apply ACK. Oversize content
is not split into TopicBus chunks; it is rejected unless a transfer manifest
provider/reference is configured.

## Release Channels

### Debug Preview

The latest automated all-platform debug build is published as a prerelease:

```text
https://github.com/yttydcs/myflowhub-clipboardnode/releases/tag/debug-latest
```

Each `master` push refreshes the movable `debug-latest` tag after all platform
debug jobs pass and uploads:

For a Windows desktop quick start, download
`myflowhub-clipboardnode-windows-debug.zip`, extract the whole zip, then
double-click `ClipboardNode.exe`. The desktop zip already includes the local Go
bridge helper used by the UI; no separate helper download is required. Keep the
extracted files together because the Flutter desktop app needs the bundled DLLs
and `data/` directory next to the executable.

- `myflowhub-clipboardnode-windows-debug.zip`: self-contained Flutter Windows debug runner directory with `ClipboardNode.exe` and the desktop bridge helper.
- `myflowhub-clipboardnode-linux-debug.tar.gz`: Flutter Linux debug bundle.
- `myflowhub-clipboardnode-macos-debug.zip`: unsigned Flutter macOS debug `.app`.
- `myflowhub-clipboardnode-android-debug.apk`: Flutter Android debug APK.
- `myflowhub-clipboardnode-ios-simulator-debug.zip`: unsigned Flutter iOS simulator debug `.app`.
- `myflowhub-clipboardnode-web-debug.zip`: Flutter Web debug bundle for hosting/browser preview; it is not the Windows desktop quick-start package.
- `clipboardnode-windows-amd64.exe`: Go CLI diagnostic helper for Windows, not the desktop UI.
- `clipboardnode-bridge-windows-amd64.exe`: Go stdio bridge helper for desktop UI integration; users normally do not start it directly.

Android debug builds generate `app/android/app/libs/myflowhub.aar` from
`nodemobile` before building the APK. If a developer builds locally without the
AAR, the APK still builds with an explicit stub bridge and reports that the
native binding is missing.

iOS simulator debug builds generate `app/ios/Frameworks/Nodemobile.xcframework`
on macOS before building. Without that generated framework, the app still
builds with an explicit stub bridge and reports that the native binding is
missing.

Manual `debug-latest` workflow runs and pull requests still build the same
artifacts in GitHub Actions, but only `master` pushes update the prerelease.
The movable `debug-latest` tag is ignored by the version release workflow.

### Version Releases

Pushing a version tag that matches `vX.Y.Z`, such as `v1.2.3`, runs
`.github/workflows/release.yml`. That workflow builds release-mode assets and
publishes a stable GitHub Release for the exact tag after every enabled platform
job succeeds. Platforms that require production signing secrets are skipped when
their required secret set is incomplete; unsigned production assets are not
published for those platforms.

Release assets may include:

For a Windows desktop quick start, download
`myflowhub-clipboardnode-windows-release.zip`, extract the whole zip, then
double-click `ClipboardNode.exe`. The desktop zip already includes the local Go
bridge helper used by the UI; no separate helper download is required. Keep the
extracted files together because the Flutter desktop app needs the bundled DLLs
and `data/` directory next to the executable.

- `myflowhub-clipboardnode-windows-release.zip`: self-contained Flutter Windows release runner directory with `ClipboardNode.exe` and the desktop bridge helper.
- `myflowhub-clipboardnode-linux-release.tar.gz`: Flutter Linux release bundle with the desktop bridge helper.
- `myflowhub-clipboardnode-macos-release.zip`: Flutter macOS release `.app`, signed and notarized when tag-release secrets are configured.
- `myflowhub-clipboardnode-android-release.apk`: Android release APK.
- `myflowhub-clipboardnode-android-release.aab`: Android release app bundle.
- `myflowhub-clipboardnode-ios-release.ipa`: iOS release IPA exported with a distribution certificate and provisioning profile.
- `myflowhub-clipboardnode-web-release.zip`: Flutter Web release bundle for web hosting; it is not the Windows desktop quick-start package.
- `clipboardnode-windows-amd64.exe`: Windows Go CLI diagnostic helper, not the desktop UI.
- `clipboardnode-bridge-windows-amd64.exe`: Windows Go stdio bridge helper; users normally do not start it directly.
- `myflowhub-clipboardnode-release-checksums.txt`: SHA-256 checksums for release assets.

Manual `release.yml` workflow runs are dry-runs: they validate release build
paths with a supplied `release_tag` input but do not publish a GitHub Release.

Real tag releases require signing secrets for platforms that publish signed
production binaries. If a required secret set is missing on a `vX.Y.Z` tag push,
that platform job is skipped and the release notes record the skip reason.

Required Android secrets:

- `ANDROID_KEYSTORE_BASE64`
- `ANDROID_KEYSTORE_PASSWORD`
- `ANDROID_KEY_ALIAS`
- `ANDROID_KEY_PASSWORD`

Required Windows signing secrets:

- `WINDOWS_CODESIGN_PFX_BASE64`
- `WINDOWS_CODESIGN_PFX_PASSWORD`
- `WINDOWS_CODESIGN_TIMESTAMP_URL` (optional; defaults to DigiCert timestamping)

Required macOS signing/notarization secrets:

- `MACOS_DEVELOPER_ID_CERT_BASE64`
- `MACOS_DEVELOPER_ID_CERT_PASSWORD`
- `MACOS_DEVELOPER_IDENTITY`
- `APPLE_NOTARY_KEY_ID`
- `APPLE_NOTARY_ISSUER_ID`
- `APPLE_NOTARY_KEY_BASE64`

Required iOS signing/export secrets:

- `IOS_DISTRIBUTION_CERT_BASE64`
- `IOS_DISTRIBUTION_CERT_PASSWORD`
- `IOS_PROVISIONING_PROFILE_BASE64`
- `IOS_DEVELOPMENT_TEAM`
- `IOS_EXPORT_METHOD` (optional; defaults to `ad-hoc`)
- `IOS_CODE_SIGN_IDENTITY` (optional; defaults to `Apple Distribution`)

Store uploads and installer generation are not part of the current release
workflow.

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
go build -o build/clipboardnode-bridge.exe ./cmd/clipboardnode-bridge
git diff --check
```

Or run:

```powershell
$flutterRoot = "D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter"
.\scripts\validate.ps1 -FlutterRoot $flutterRoot
```

`validate.ps1` fails when Flutter is unavailable unless `-SkipFlutter` is
passed explicitly for a Go-only validation run.

Flutter validation uses the local Flutter SDK selected during this workflow:

```powershell
$flutterRoot = "D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter"
$flutter = "$flutterRoot\bin\flutter.bat"
$env:PUB_CACHE = "$flutterRoot\.pub-cache"
cd app
& $flutter analyze
& $flutter test
& $flutter build windows --debug
& $flutter build apk --debug
& $flutter build web --debug
```

Build the optional Android gomobile AAR before an APK when validating true
mobile engine integration:

```powershell
.\scripts\build_aar.ps1 -OutFile app/android/app/libs/myflowhub.aar
cd app
& "D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat" build apk --debug
```

Run a local two-node MyFlowHub smoke test with a temporary Hub and two
ClipboardNode bridge processes:

```powershell
.\scripts\smoke_localhub_two_nodes.ps1 -ServerRoot D:\project\MyFlowHub3\repo\MyFlowHub-Server
```

The smoke keeps `auto_watch=false` and `auto_apply=false`, so it verifies
login, subscribe, TopicBus publish, and remote pending metadata without reading
from or writing to the system clipboard implicitly.

Build the iOS gomobile XCFramework on macOS before validating the live iOS
path:

```bash
./scripts/build_ios_xcframework.sh iossimulator app/ios/Frameworks/Nodemobile.xcframework
cd app
flutter build ios --debug --simulator --no-codesign
```

Run the Web bridge only on localhost with an explicit token:

```powershell
$token = [guid]::NewGuid().ToString("N")
.\build\clipboardnode-bridge.exe --web-listen 127.0.0.1:18291 --web-token $token
cd app
flutter build web --debug --dart-define=CLIPBOARDNODE_WEB_BRIDGE=http://127.0.0.1:18291 --dart-define=CLIPBOARDNODE_WEB_TOKEN=$token
```

Linux, macOS, Android, and iOS simulator debug builds are validated by GitHub
Actions on their matching hosted runners.

