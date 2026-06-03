# Lessons

Reusable lessons for ClipboardNode workflows.

## Current

- [device-id-auth-snapshot-mismatch.md](device-id-auth-snapshot-mismatch.md) - Changing ClipboardNode `device_id` must clear stale auth snapshot state; display-name-only edits should not reset node identity.
- [startup-subscribe-timeout-half-connected.md](startup-subscribe-timeout-half-connected.md) - Startup subscribe timeout with a local node id can come from stale persisted `logged_in=true`; re-login on fresh TCP sessions and close transport after failed startup.
- [gomobile-mobile-bindings.md](gomobile-mobile-bindings.md) - gomobile AAR/XCFramework live mobile proof requires pinned generation, artifact/class/module verification, Android minSdk alignment, and explicit stubs when bindings are absent.
- [web-localhost-bridge-errors.md](web-localhost-bridge-errors.md) - Web live mode requires explicit loopback bridge/token configuration and command/error contracts that encode `ok:false`.
- [debug-latest-ci-native-exit-flutter-material.md](debug-latest-ci-native-exit-flutter-material.md) - debug-latest can publish after failed tests if PowerShell native exits are unchecked; also covers Flutter `ListTile` Material ancestry assertions.
- [flutter-windows-sdk-shared-bat-git.md](flutter-windows-sdk-shared-bat-git.md) - Flutter CLI hangs on Windows; check `shared.bat`, `$git rev-parse HEAD`, and `flutter.bat.lock`.

