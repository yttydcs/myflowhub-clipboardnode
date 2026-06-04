# Flutter Switch Hover State Layer

## Summary

When a Flutter Material `Switch` hover looks too large or bleeds outside the switch track, do not treat it as only an opacity problem. The visible hover area is controlled by the switch state layer radius, so `SwitchThemeData.splashRadius` must be tuned together with `overlayColor`.

## Lookup Hints

- Symptoms: Switch hover is too large, hover highlight draws outside the control, hover disappears after alpha is reduced.
- Keywords: `SwitchThemeData.splashRadius`, `SwitchThemeData.overlayColor`, `materialTapTargetSize`, `SwitchListTile`, hover state layer.
- Quick check: inspect `Theme.of(context).switchTheme.splashRadius` and `overlayColor` before changing color alpha only.

## Symptoms

- User reports that switch hover is visually too large.
- The hover state layer appears outside the switch track/thumb region.
- Lowering overlay alpha makes hover hard to see but does not fix the geometry.

## Impact

- Desktop settings UI looks unpolished and noisy.
- Hover feedback becomes either too prominent or too subtle if only color opacity is changed.
- Repeated tweaking can regress accessibility/feedback if the hover state is accidentally disabled.

## Trigger Conditions

- Flutter Material `Switch` or `SwitchListTile` is used with default state layer radius.
- A custom `overlayColor` is set without constraining `splashRadius`.
- Dense operational UI places switches near panel edges or separators, making external hover bleed obvious.

## Root Cause

Flutter paints a switch state layer around the thumb using `splashRadius`. The default radius can be visually larger than the switch track in dense desktop layouts. `overlayColor` only changes color strength; it does not reduce the painted state layer bounds.

## Investigation Trail

1. Initial hover complaint looked like a color opacity issue.
2. Reducing hover opacity removed the obvious blob but made hover feedback too weak.
3. Flutter SDK source confirmed `SwitchThemeData` exposes `materialTapTargetSize`, `splashRadius`, and `padding`.
4. The correct fix was to lower `splashRadius`, shrink the control padding, and then increase overlay alpha enough for visible feedback.

## Resolution

- Set `SwitchThemeData.splashRadius` to a smaller value such as `14` for dense desktop settings UI.
- Set `materialTapTargetSize` to `MaterialTapTargetSize.shrinkWrap` and `padding` to `EdgeInsets.zero` when row-level controls already provide sufficient hit area.
- Keep hover/focus/pressed overlay colors visible with tuned alpha values.
- Prefer a subtle track color change on hover so the state is visible even with a smaller state layer.

## Prevention / Guardrails

- Do not fix oversized switch hover by only reducing `overlayColor` alpha.
- Keep a widget/theme test that asserts `splashRadius`, `padding`, and hover/focus overlay values.
- For `SwitchListTile`, remember that the tile provides the larger click target; the switch state layer can remain visually compact.

## Related Docs

- `docs/change/2026-06-04_clipboardnode-ui-history-settings-polish.md`
- `docs/specs/clipboard-sync.md`
