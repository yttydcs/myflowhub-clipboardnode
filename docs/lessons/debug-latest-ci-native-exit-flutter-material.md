# Debug-latest CI Native Exit And Flutter Material Checks

## Summary

The first `debug-latest` GitHub Actions run published a prerelease even though `flutter test` reported one failed widget test. The PowerShell step continued after the failed native command, then built and uploaded artifacts. The failed widget test came from a newer Flutter assertion that rejects a `ListTile` under a colored `DecoratedBox` without a proper `Material` ancestor for ink/background painting.

## Lookup Hints

- `debug-latest` published after failed tests.
- `flutter test` shows `4 tests passed, 1 failed` but the job concludes success.
- PowerShell `$ErrorActionPreference = "Stop"`.
- `$LASTEXITCODE`.
- `ListTile background color or ink splashes may be invisible`.
- `DecoratedBox` / `Material` ancestor.
- Flutter stable drift from local SDK to CI SDK.

## Symptoms

- GitHub Actions run conclusion is `success`.
- The release job executes and uploads `myflowhub-clipboardnode-windows-debug.zip`.
- Build logs include `##[error]4 tests passed, 1 failed`.
- The failure details mention:
  - `ListTile background color or ink splashes may be invisible.`
  - `To fix this, wrap the ListTile in its own Material widget, or remove the background color from the intermediate DecoratedBox.`

## Impact

- A `debug-latest` prerelease can be refreshed from a commit whose test phase failed.
- Users may download a debug artifact that did not actually pass validation.
- The false-success signal hides the real UI compatibility problem unless logs are inspected.

## Trigger Conditions

- GitHub Actions Windows job runs validation commands inside a PowerShell script block.
- The script only sets `$ErrorActionPreference = "Stop"` and does not check `$LASTEXITCODE` after native commands.
- `flutter test` returns a non-zero exit code, but the next native command succeeds.
- CI Flutter version is newer than the local validated SDK and tightens widget assertions.

## Root Cause

PowerShell treats many native command failures differently from PowerShell exceptions. `$ErrorActionPreference` is not enough to stop the script after `go`, `flutter`, or other native executables fail. The workflow must check `$LASTEXITCODE` after each critical native command or use a wrapper that throws on non-zero exit.

The UI failure was caused by `_Panel` using a colored `Container` / `DecoratedBox` as an intermediate background around descendants that include `ListTile`. Newer Flutter assertions require `ListTile` ink/background effects to be painted on an appropriate `Material` ancestor instead of being hidden by the intermediate decoration.

## Investigation Trail

1. Watched GitHub Actions run `26717910962`; both jobs reported success and the release was uploaded.
2. Queried release assets and confirmed zip/exe were present.
3. Inspected the build job log and found `flutter test` reported one failed test.
4. Confirmed the step continued into `flutter build windows --debug` and packaging after the failed test.
5. Read the test failure text and identified the `ListTile` / `DecoratedBox` material ancestry assertion.
6. Compared local SDK `3.41.9` with CI SDK `3.44.0`, showing the CI run used a newer stable version.

## Resolution

- Add explicit `$LASTEXITCODE` checks after all critical native commands in the GitHub Actions PowerShell steps.
- Pin CI Flutter to the locally validated version until the project intentionally upgrades.
- Upgrade official GitHub Actions to Node 24-compatible major versions.
- Replace the `_Panel` colored `Container` wrapper with a shaped `Material` wrapper so descendant `ListTile` widgets have the right material ancestor.

## Prevention / Guardrails

- In PowerShell CI scripts, check `$LASTEXITCODE` after each critical native command.
- Do not trust a successful job conclusion after adding a new workflow until the log is inspected at least once.
- Pin SDK/tool versions for release-producing workflows.
- When Flutter widget tests fail only on newer CI SDKs, search for framework assertions before weakening tests.
- Prefer `Material` with `shape`, `color`, and `clipBehavior` for styled surfaces that can contain `ListTile`, `InkWell`, or similar Material ink widgets.

## Related Requirements / Specs / Changes

- [../change/2026-05-31_debug-latest-build.md](../change/2026-05-31_debug-latest-build.md)
