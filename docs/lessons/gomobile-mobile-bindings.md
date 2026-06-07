# Gomobile Mobile Bindings

## Summary

Android and iOS live mobile ClipboardNode support depends on generated gomobile bindings. Stub app builds are useful fallback checks, but they are not proof of live mobile completion. Android needs a pinned AAR with verified Java class names and minSdk alignment. iOS needs a pinned XCFramework generated and validated on macOS/Xcode.

## Lookup Hints

- `gomobile bind`
- `myflowhub.aar`
- `Nodemobile.xcframework`
- `com.myflowhub.gomobile.nodemobile.Nodemobile`
- `javap -classpath classes.jar com.myflowhub.gomobile.nodemobile.Nodemobile`
- `takeLastAppliedText`
- `NoSuchMethodException`
- `InvocationTargetException`
- Android auto apply remote text
- `remote_applied`
- GitHub Android debug APK
- `minSdk 26`
- `canImport(Nodemobile)`
- `iOS gomobile binding requires macOS and Xcode`

## Symptoms

- Android APK builds but native channel reports a stub or cannot resolve `Nodemobile`.
- Android APK packages the AAR but the Kotlin reflection bridge fails at runtime with `NoSuchMethodException` for gomobile exports.
- GitHub-built Android APK connects and subscribes, but remote auto-apply does not update the Android system clipboard.
- Android UI reports only `InvocationTargetException`, hiding the underlying gomobile or connection cause.
- Android merge/build fails because the AAR minSdk is higher than the app minSdk.
- iOS builds in stub mode and reports that `Nodemobile.xcframework` is required.
- CI claims mobile success from Flutter-only builds even though no generated binding artifact was produced.

## Impact

- Mobile targets appear buildable but do not exercise the live Go engine.
- Runtime failures can be misread as app logic bugs when the binding artifact is simply absent or mismatched.
- iOS live proof can be overstated if tested only on Windows or non-Xcode environments.

## Trigger Conditions

- gomobile version is not pinned and generated package/module output drifts.
- AAR is not generated before `flutter build apk`.
- Kotlin resolver class names do not match generated Java package layout.
- Kotlin reflection method names do not match gomobile's generated Java names; exported Go functions are exposed as lowerCamel methods such as `start`, `applyEvent`, and `takeLastAppliedText`.
- Android remote auto-apply depends on a mobile-only text handoff: Go runtime accepts the remote event, writes through the gomobile manual clipboard, and Kotlin later polls `takeLastAppliedText` to write Android's system clipboard.
- Kotlin reflection can wrap real gomobile failures in `InvocationTargetException`; UI and logs need the unwrapped target exception to diagnose stale/mismatched AARs or runtime failures.
- App `minSdk` is lower than the generated AAR requirement.
- iOS XCFramework is absent or generated with unexpected module/symbol names.

## Root Cause

gomobile output is generated platform binding code, not a stable checked-in API surface. The app must either package the generated artifact and prove the expected symbol/class layout, or explicitly fall back to a stub and report that live binding is required.

## Investigation Trail

1. Pin `golang.org/x/mobile/cmd/gomobile` in build scripts.
2. Build Android AAR into `app/android/app/libs/myflowhub.aar`.
3. Inspect the AAR with `jar tf` and confirm `com/myflowhub/gomobile/nodemobile/Nodemobile.class`.
4. Extract `classes.jar` and run `javap -classpath classes.jar com.myflowhub.gomobile.nodemobile.Nodemobile` to confirm exported method names. Kotlin reflection should use lowerCamel names from `javap`, not Go/PascalCase names.
5. Align Android `minSdk` to the AAR requirement.
6. Build the APK with the generated AAR present.
7. For iOS, generate `app/ios/Frameworks/Nodemobile.xcframework` on macOS and validate Swift import/symbol names through CI.
8. If GitHub-built Android APK does not auto-apply remote text, check whether `takeLastAppliedText` is present, whether Android route `sync_to_local` and `auto_apply` are enabled, and whether Kotlin logs show an unwrapped gomobile cause rather than a bare `InvocationTargetException`.

## Resolution

- Add pinned `scripts/build_aar.ps1` and `scripts/build_aar.sh`.
- Add pinned `scripts/build_ios_xcframework.sh` and a PowerShell wrapper that fails clearly outside macOS/Xcode.
- Keep generated AAR/XCFramework ignored.
- Make CI build gomobile artifacts before mobile app builds.
- Add explicit Android/iOS stub fallback messaging when bindings are absent.
- Keep Android remote-applied text handoff mobile-local and body-safe: do not put clipboard bodies into status/config/logs.
- Unwrap Kotlin `InvocationTargetException` before surfacing MethodChannel errors.

## Prevention / Guardrails

- Do not treat a stub mobile build as live mobile completion.
- Pin gomobile versions in scripts and CI.
- Verify generated Android class names after AAR build.
- Verify generated Java method names with `javap` before wiring Kotlin reflection.
- When Android auto-apply fails on a GitHub APK, first verify the `debug-latest` run actually corresponds to the pushed commit and that the Android job uploaded a fresh `myflowhub.aar`.
- Preserve a focused test for the mobile applied-text handoff because this is not exercised by desktop bridge tests.
- Keep Android app minSdk compatible with the generated AAR.
- Record iOS live proof only from macOS/Xcode build evidence.

## Related Requirements / Specs / Changes

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)
- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)
- [../change/2026-06-02_clipboard-full-platform-sync.md](../change/2026-06-02_clipboard-full-platform-sync.md)
- [../change/2026-06-04_android-clipboard-topic-settings.md](../change/2026-06-04_android-clipboard-topic-settings.md)
- [../change/2026-06-07_android-auto-apply-remote.md](../change/2026-06-07_android-auto-apply-remote.md)
