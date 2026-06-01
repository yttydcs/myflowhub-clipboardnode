# Gomobile Mobile Bindings

## Summary

Android and iOS live mobile ClipboardNode support depends on generated gomobile bindings. Stub app builds are useful fallback checks, but they are not proof of live mobile completion. Android needs a pinned AAR with verified Java class names and minSdk alignment. iOS needs a pinned XCFramework generated and validated on macOS/Xcode.

## Lookup Hints

- `gomobile bind`
- `myflowhub.aar`
- `Nodemobile.xcframework`
- `com.myflowhub.gomobile.nodemobile.Nodemobile`
- `minSdk 26`
- `canImport(Nodemobile)`
- `iOS gomobile binding requires macOS and Xcode`

## Symptoms

- Android APK builds but native channel reports a stub or cannot resolve `Nodemobile`.
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
- App `minSdk` is lower than the generated AAR requirement.
- iOS XCFramework is absent or generated with unexpected module/symbol names.

## Root Cause

gomobile output is generated platform binding code, not a stable checked-in API surface. The app must either package the generated artifact and prove the expected symbol/class layout, or explicitly fall back to a stub and report that live binding is required.

## Investigation Trail

1. Pin `golang.org/x/mobile/cmd/gomobile` in build scripts.
2. Build Android AAR into `app/android/app/libs/myflowhub.aar`.
3. Inspect the AAR with `jar tf` and confirm `com/myflowhub/gomobile/nodemobile/Nodemobile.class`.
4. Align Android `minSdk` to the AAR requirement.
5. Build the APK with the generated AAR present.
6. For iOS, generate `app/ios/Frameworks/Nodemobile.xcframework` on macOS and validate Swift import/symbol names through CI.

## Resolution

- Add pinned `scripts/build_aar.ps1` and `scripts/build_aar.sh`.
- Add pinned `scripts/build_ios_xcframework.sh` and a PowerShell wrapper that fails clearly outside macOS/Xcode.
- Keep generated AAR/XCFramework ignored.
- Make CI build gomobile artifacts before mobile app builds.
- Add explicit Android/iOS stub fallback messaging when bindings are absent.

## Prevention / Guardrails

- Do not treat a stub mobile build as live mobile completion.
- Pin gomobile versions in scripts and CI.
- Verify generated Android class names after AAR build.
- Keep Android app minSdk compatible with the generated AAR.
- Record iOS live proof only from macOS/Xcode build evidence.

## Related Requirements / Specs / Changes

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)
- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)
- [../change/2026-06-02_clipboard-full-platform-sync.md](../change/2026-06-02_clipboard-full-platform-sync.md)
