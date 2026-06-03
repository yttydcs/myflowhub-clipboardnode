# 2026-06-04_clipboardnode-header-alignment

## 变更背景 / 目标

用户指出 ClipboardNode 桌面界面顶部品牌文字看起来偏斜，且右侧“总览”标题区域下方横线没有和左侧品牌区域下边线对齐。

本次目标是修正桌面 shell 顶部视觉对齐，让左侧品牌 header 和右侧内容 top bar 共享一致高度与边框位置。

## 具体变更内容

- 在 `app/lib/features/shell/clipboard_shell.dart` 中新增 `_desktopHeaderHeight = 72`。
- 左侧 `_SideNav` 的品牌区从 `Padding + Divider` 改为固定高度的 decorated header。
- 右侧 `_TopBar` 从 `minHeight + vertical padding` 改为同一个固定高度。
- `_BrandMark` 内容使用居中布局，并让文字列以内容高度参与居中。

## Requirements impact

none

## Specs impact

none

## Lessons impact

none

## Related requirements

- `docs/requirements/clipboard-sync.md`

## Related specs

- `docs/specs/clipboard-sync.md`

## Related lessons

- none

## 对应 plan.md 任务映射

- Follow-up - Header Alignment UI Fix

## 经验 / 教训摘要

同一行视觉 header 不应由两套独立 padding 和 divider 拼接。共享固定高度和边框来源可以避免 1px 分隔线错位，也更容易保持品牌区与内容区的垂直居中。

## 可复用排查线索

- 症状：左侧品牌文字看起来偏斜
- 症状：右侧顶部横线和左侧下边线不对齐
- 关键词：ClipboardNode, `_SideNav`, `_TopBar`, `_BrandMark`, header alignment, Divider

## 测试与验证方式 / 结果

- `dart format app/lib/features/shell/clipboard_shell.dart`：通过
- `flutter analyze`：通过
- `flutter test`：通过，5 tests
- `git diff --check`：通过

## 潜在影响与回滚方案

仅影响桌面宽屏 shell 顶部布局。回滚 `app/lib/features/shell/clipboard_shell.dart` 中 header 高度和 `_BrandMark` 居中调整即可恢复旧布局。

## 子Agent执行轨迹

未使用子Agent；改动范围为单文件 UI 布局微调。
