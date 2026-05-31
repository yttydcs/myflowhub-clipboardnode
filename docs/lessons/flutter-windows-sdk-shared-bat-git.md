# Flutter Windows SDK shared.bat `$git` Hang

## Summary

On this Windows workspace, downloaded Flutter SDKs could install successfully and their bundled Dart SDKs worked, but `flutter --version`, `flutter doctor`, and direct Flutter tool startup hung with no useful output. The root cause was a Flutter SDK batch script line that invoked `$git rev-parse HEAD` instead of `git rev-parse HEAD`, forcing the tool to rebuild or start incorrectly.

## Lookup Hints

- Flutter CLI hangs.
- `flutter --version` no output.
- `flutter doctor -v` no output.
- `Building flutter tool...` followed by a stuck Dart process.
- `shared.bat`.
- `$git rev-parse HEAD`.
- `flutter.bat.lock`.
- Conda `Command Processor\AutoRun` noise can obscure the real failure.

## Symptoms

- `flutter --version` timed out.
- `flutter doctor -v` timed out.
- Direct `dart flutter_tools.snapshot --version` also hung.
- Dart SDK itself worked:
  - `dart --version` printed the expected Dart version.
- Flutter cache contained a stale `flutter.bat.lock`.
- `cmd /d /c "for /f %r in ('PUSHD <flutterRoot> ^& $git rev-parse HEAD') do @echo %r"` failed with:
  - `'rev-parse' is not recognized as an internal or external command`.

## Impact

- Blocked APP-1 toolchain gate.
- Prevented normal `flutter create`, `flutter analyze`, `flutter test`, and `flutter build` until the local SDK was repaired.

## Trigger Conditions

- Windows Flutter SDK extracted under the workspace temp tools directory.
- SDK file `bin/internal/shared.bat` contained:
  - `FOR /f %%r IN ('PUSHD %FLUTTER_ROOT% ^& $git rev-parse HEAD') DO (`
- The current machine also has `HKCU\Software\Microsoft\Command Processor\AutoRun` running a Conda hook, which adds noisy unrelated errors and can make command diagnosis harder.

## Root Cause

Flutter's Windows batch script tried to execute `$git rev-parse HEAD`. In `cmd`, `$git` is not expanded as an executable name, so the nested command is parsed incorrectly. That makes revision detection fail and causes the Flutter tool startup path to rebuild or hang.

## Investigation Trail

1. Verified `flutter` and `dart` were not originally on PATH.
2. Downloaded official Flutter stable SDKs and verified SHA256.
3. Confirmed bundled Dart worked with `dart --version`.
4. Observed Flutter CLI hanging with no stdout/stderr and idle Dart processes.
5. Checked `bin/internal/shared.bat`, `flutter_tools.stamp`, and Git revision.
6. Reproduced the failing command with `cmd /d /c` using `$git rev-parse HEAD`.
7. Patched the local SDK file outside the application repository from `$git rev-parse HEAD` to `git rev-parse HEAD`.
8. Re-ran `flutter --version` and `flutter doctor -v` successfully.

## Resolution

Patch the local Flutter SDK file:

```text
<flutterRoot>/bin/internal/shared.bat
```

Replace:

```bat
$git rev-parse HEAD
```

with:

```bat
git rev-parse HEAD
```

Then remove stale Flutter tool locks if needed:

```powershell
Remove-Item -LiteralPath "$flutterRoot\bin\cache\flutter.bat.lock" -Force -ErrorAction SilentlyContinue
```

Validated with:

```powershell
flutter --version
flutter doctor -v
```

## Prevention / Guardrails

- Keep Flutter SDK under a known local tools path and do not commit it.
- When Flutter CLI hangs but Dart works, inspect `bin/internal/shared.bat` before reinstalling repeatedly.
- Check for stale `flutter.bat.lock` and local Dart processes under the same Flutter SDK.
- Use `cmd /d` for diagnosis to avoid Command Processor AutoRun side effects.
- Record the selected Flutter SDK path in the workflow plan.

## Related Docs

- [../change/2026-05-31_clipboard-cross-platform-app-shell.md](../change/2026-05-31_clipboard-cross-platform-app-shell.md)
- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

