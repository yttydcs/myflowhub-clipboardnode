# 2026-06-04 Clipboard Body History

## 变更背景 / 目标

用户明确要求“历史”指剪贴板正文历史，而不是活动元数据，并要求历史长度可配置，默认保存 256 条。

本次变更将 ClipboardNode 的 History 页面改为展示实际文本正文，并将正文历史作为本地内存 UI 状态处理；日志/活动仍保持 metadata-only。

## 具体变更内容

- Runtime 配置新增 `history_limit`，默认 `256`，并新增 `history_retention=body` 默认模式。
- Runtime `Decision` 新增内存字段 `Text`，仅在成功的 inline 文本发布、接收 pending、自动应用和手动应用 pending 路径赋值。
- `Decision.Text` 使用 `json:"-"`，避免 gomobile 或默认 JSON 决策输出意外包含正文。
- Bridge settings/status 增加 `history_limit`；activity 增加可选 `text` 字段。
- `clipboardnode-bridge` 只在 `history_retention=body` 时把 `Decision.Text` 写入 activity `text`。
- Flutter 状态新增 `ClipboardHistoryEntry` 和独立 `state.history`，与 `state.activities` 分离。
- Preview/live/web/mobile bridge 都支持按 `history_limit` 裁剪正文历史，切换到 metadata/none 时清空正文历史。
- History 页面改为显示正文历史；Log 页面继续显示活动元数据。
- Settings 页面新增正文历史 retention 选项和“历史条数”输入框。
- `clear_recent` 在 UI 侧同时清理日志、正文历史、pending 和 transfer 状态；live/web 端跳过空 activity 回包，避免清空后再插入一条空日志。
- 更新 requirements/specs，将默认正文历史行为和隐私边界写入稳定文档。

## Requirements impact

updated

## Specs impact

updated

## Lessons impact

none

本次没有产生新的可复用故障排查模式；主要是明确的产品行为变更和隐私边界收口。

## Related requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Related lessons

- none

## 对应 plan.md 任务映射

- `T1`: Runtime retention config and decision text.
- `T2`: Bridge contract and privacy gating.
- `T3`: Flutter body history state and UI.
- `T4`: Docs, validation, archive.

## 经验 / 教训摘要

- Body history and activity/log metadata must be modeled as separate state lists. Reusing activity records for history makes it impossible to satisfy both “显示正文历史”和“日志不暴露正文”。
- Sensitive body text should be gated at the bridge boundary, not only hidden by UI. The UI still checks retention mode, but Go bridge is the stronger privacy boundary.
- Runtime decisions can carry in-memory text for local UI integration while using `json:"-"` to keep default serialized diagnostic/mobile decision output body-free.

## 可复用排查线索

- 症状：History 页面只显示 hash/size/device，没有正文。
- 触发条件：UI 复用 `state.activities` 作为 history 数据源。
- 关键词：`ClipboardHistoryEntry`, `history_limit`, `history_retention=body`, `activity.updated text`, `Decision.Text`.
- 快速检查：
  - `ClipboardSettings.defaults()` 应为 `HistoryRetention.body` 和 `256`。
  - `Status` JSON 不应包含 `text`。
  - `Activity` JSON 只有在 body retention 下才应包含 `text`。
  - History 页面应迭代 `state.history`，Log 页面应迭代 `state.activities`。

## 关键设计决策与权衡

- 默认正文历史为本地内存状态，不持久化到磁盘，满足默认 256 条可见历史，同时避免扩大持久化隐私风险。
- 保留 `metadata` 和 `none` retention 模式，允许用户关闭正文历史或只保留日志元数据。
- `history_limit` 设上限 `5000`，避免误配置造成无界内存保留。
- Oversize transfer 路径不记录正文历史，因为 transfer manifest 不携带正文，避免通过历史功能绕开大内容正文边界。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./core/runtime ./bridge ./cmd/clipboardnode-bridge -count=1`: passed.
- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `flutter pub get`: passed.
- `flutter analyze`: passed.
- `flutter test`: passed.
- `flutter build windows --debug`: passed.
- `git diff --check`: passed.
- Local desktop app launched from worktree debug build: PID `60220`.

## 潜在影响

- 默认行为从 metadata/no-body 变为本地内存正文历史 256 条，隐私敏感度提高，但可在设置中切换为 metadata 或 none。
- Live activity event 在 body retention 下会携带正文给本机 Flutter UI；该字段不进入 status、transfer 或 logs。
- Mobile native remote/apply 正文历史仍受 native decision 能否提供正文限制；本次保证手动 send 路径可用，fallback preview 路径完整可用。

## 回滚方案

- Revert `core/runtime` 中 `HistoryRetentionBody`、`HistoryLimit`、`Decision.Text` 相关更改。
- Revert bridge contract and `clipboardnode-bridge` activity text gating.
- Revert Flutter `ClipboardHistoryEntry`、`state.history`、History UI 和 Settings history limit 控件。
- Revert requirements/specs and this change archive/index entry.

## 子Agent执行轨迹

- No sub-agent dispatched. The runtime/bridge/UI changes share one privacy-sensitive contract, so the main agent retained ownership through implementation, validation, and review.
