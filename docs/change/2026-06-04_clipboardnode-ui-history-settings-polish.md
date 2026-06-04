# 2026-06-04_clipboardnode-ui-history-settings-polish

## 变更背景 / 目标

本轮收尾覆盖本地桌面预览中暴露的 ClipboardNode UI 和历史设置问题：

- 本地旧配置仍为 `history_retention=metadata`，导致“剪贴板历史”不保存正文。
- 设置页“历史条数”输入框占据整行，和本机策略项视觉不一致。
- 面板标题栏过窄，标题和分隔线视觉不齐。
- Switch hover state layer 默认半径过大，鼠标经过时高亮画到控件外侧；只调低透明度后又不明显。

目标是在不改变 TopicBus 协议、状态/日志隐私边界和持久化策略的前提下，让正文历史默认生效，并让本机策略 UI 更稳定、可预览。

## 具体变更内容

- `core/configstore`
  - 加入 legacy history 默认迁移：当旧配置为 `history_retention=metadata` 且没有显式 `history_limit` 时，加载时迁移为 `history_retention=body` 和 `history_limit=256`。
  - 保留显式 metadata 配置：如果用户已经保存过 `history_limit`，不会强行改回 body。
- Flutter bridge/state
  - `appendClipboardHistory` 按非空历史 id 去重，避免同一事件重复出现。
  - `PreviewEngineBridge` 增加单调序号，避免快速连续发送时预览事件 id 碰撞。
- Flutter settings UI
  - “历史条数”改为横向策略行：左侧图标/标题，右侧窄输入框，输入值右对齐并保留 `条` 后缀。
  - `_Panel` 标题栏恢复固定高度，分隔线改为满宽 1px，避免标题被压扁和线条不齐。
  - top bar 标题块使用固定高度和单行省略，减少标题/Topic 视觉漂移。
- Flutter switch theme
  - `SwitchThemeData.splashRadius` 设为 `14`，`materialTapTargetSize` 设为 `shrinkWrap`，`padding` 设为 `EdgeInsets.zero`，限制 hover state layer 外扩。
  - hover/focus/pressed 保留可见反馈：轨道 hover 变淡青色，overlay alpha 分别为 0.12 / 0.14 / 0.18。
- Tests
  - 增加 legacy config 迁移测试。
  - 增加显式 metadata history 保留测试。
  - 增加 switch hover 半径、padding、overlay 和 hover track 断言。

## Requirements impact

none

## Specs impact

none

## Lessons impact

updated

## Related requirements

- `docs/requirements/clipboard-sync.md`

## Related specs

- `docs/specs/clipboard-sync.md`

## Related lessons

- `docs/lessons/flutter-switch-hover-state-layer.md`

## 对应 plan.md 任务映射

- `UI-HISTORY-01`: ensure local body history defaults are effective for old local configs.
- `UI-HISTORY-02`: polish history settings control layout and panel title/divider alignment.
- `UI-SWITCH-01`: constrain Switch hover state layer radius while preserving visible hover feedback.
- `TEST-01`: cover config migration, preview history id stability, and switch theme behavior.

## 经验 / 教训摘要

- Flutter `Switch` hover 外扩不是单纯透明度问题；默认 `splashRadius` 控制 state layer 半径，只调 `overlayColor` 会在“大而淡”和“小但无反馈”之间反复。
- 需要同时控制 `splashRadius`、`materialTapTargetSize` / `padding` 和 overlay alpha，才能让 hover 既可见又不画到控件外。
- 用户看到“仅记录元数据”时，可能不是新默认失效，而是旧本地配置仍保存 metadata；需要区分“缺省迁移”和“用户显式选择”。

## 可复用排查线索

- 症状：
  - Switch hover 高亮跑到控件外侧。
  - 降低 hover alpha 后用户反馈“看不清”。
  - 剪贴板历史空态显示“当前未保存正文”。
- 触发条件：
  - Flutter Material Switch 使用默认 `splashRadius`。
  - 本地配置文件仍是旧版 `history_retention=metadata` 且缺少 `history_limit`。
- 关键词：
  - `SwitchThemeData.splashRadius`
  - `SwitchThemeData.overlayColor`
  - `history_retention=metadata`
  - `history_limit`
- 快速检查：
  - 查看 `Theme.of(context).switchTheme.splashRadius` 是否显式设置。
  - 读取 `%APPDATA%/MyFlowHub/ClipboardNode/config.json` 的 `history_retention` 和 `history_limit`。

## 关键设计决策与权衡

- 不把 hover 关掉：用户需要可见交互反馈，最终保留 hover/focus/pressed alpha。
- 不只调透明度：state layer 外扩由半径导致，透明度只能缓解视觉强度，不能解决范围问题。
- 不持久化正文历史：正文历史仍只存在于 Flutter UI 内存状态中，配置迁移只改变历史 retention 默认行为。
- 不改 requirements/specs：既有文档已要求 body history 默认 256 且可配置，本轮实现该既有要求并做 UI polish。

## 测试与验证方式 / 结果

- `GOWORK=off go test ./core/configstore ./cmd/clipboardnode-bridge ./bridge ./core/runtime -count=1`: passed during implementation.
- `flutter analyze`: passed.
- `flutter test`: passed.
- `flutter build windows --debug`: passed.
- `git diff --check`: passed, with existing CRLF warning noise only.
- Local Windows desktop app launched from `app/build/windows/x64/runner/Debug/ClipboardNode.exe` for preview.

## 潜在影响

- Old local config files that implicitly inherited metadata-only history now load as body history by default. Explicit metadata configs with a saved `history_limit` remain metadata.
- Switch tap target is shrink-wrapped at the control level; `SwitchListTile` still provides a row-level click target.

## 回滚方案

- Revert `app/lib/app/theme/app_theme.dart` switch theme changes to restore default Material hover behavior.
- Revert `_HistoryLimitControl` and `_PanelDivider` changes to return to the previous settings/panel layout.
- Revert `core/configstore` migration if old implicit metadata configs should remain metadata-only.
- Revert `appendClipboardHistory` id de-duplication and preview bridge sequence only if duplicate history behavior is desired for debugging.

## 子Agent执行轨迹

- none
