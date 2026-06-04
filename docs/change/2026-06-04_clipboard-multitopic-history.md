# 2026-06-04 Clipboard Multi-topic History

## Background

用户要求 ClipboardNode 重新划分“历史”和“日志”：历史只展示可恢复的剪贴板正文，发送、接收、pending、transfer 和错误等过程记录进入日志/队列；点击历史正文应恢复到本机剪贴板并置顶。

用户同时要求支持订阅多个 TopicBus topic，并允许每个 topic 独立控制“同步到本机”和“从本机同步出去”。新默认 TopicBus topic 为 `clipboard.text`，ClipboardNode 应用事件名仍为 `clipboard.text.v1`。

后续用户确认剪贴板正文历史也必须持久化，配置也必须持久化。最终实现将正文历史放入独立本地 `history.json`，配置继续通过 `config.json` 持久化，但配置、status、日志、transfer、pending metadata 均不保存正文。

## Changes

- Go runtime/config:
  - 新增 `DefaultTopic=clipboard.text`、`TopicRoute`、`Config.Topics` 和 bounded route normalization。
  - 保留 legacy `topic` 兼容；`topics` 存在时作为 canonical route list，`topic` 归一化为第一条 route。
  - 支持多 topic subscribe/unsubscribe/resubscribe，配置更新时按 topic set diff 订阅和取消订阅。
  - 本地文本和 transfer manifest 按 `sync_from_local=true` fan-out 发布。
  - 远端消息按 `sync_to_local` gate，unknown topic 和 local-disabled topic 只记录 metadata ignored decision。
  - pending status、pending queue、activity decision 携带 topic metadata。
  - remote pending 在 bridge 边界保持 metadata-only，只有本地发布和已应用文本进入 body history。
- Bridge contract:
  - `Settings` 和 `Status` 增加 `topics`。
  - `Status` 增加 `pending_topic`。
  - `Activity` 增加 `topic`，detail 显示 `TopicBus: <topic>`。
  - 新增 `HistoryEntry`、`restore_history` command 和 `history.updated` event，作为唯一专用正文历史列表通道。
- Local bridge history persistence:
  - 新增 `cmd/clipboardnode-bridge/history_store.go`，在 `config.json` 同目录保存 `history.json`。
  - `history_retention=body` 时追加、去重、裁剪并持久化正文历史。
  - `history_retention=metadata|none` 或用户清空最近记录时清空 persisted body history。
  - restore 历史正文后通过 `restore_history` 将该正文持久化提升到历史顶部。
  - web bridge 新增 `/history` localhost endpoint，作为 SSE 启动事件之外的加载兜底。
- Flutter bridge/state:
  - 新增 `TopicSyncConfig`、topic normalization 和 default `clipboard.text`。
  - live/web/mobile/preview settings/status 全部解析和序列化 `topics`。
  - 新增 `restoreHistory`，live/web/mobile 写入系统剪贴板，preview 只更新本地状态。
  - 正文历史按 text 去重，restore 后置顶；pending activity 不进入正文历史。
  - live/web 解析 `history.updated`，restore 成功写剪贴板后向 Go bridge 发送 `restore_history`，使置顶顺序持久化。
- Flutter UI:
  - Settings 页面从单 Topic 输入改为多 topic route editor。
  - 每个 topic row 提供“到本机”和“从本机”方向开关。
  - History 页面改为 body-only 可点击恢复条目。
  - Log/queue 页面继续显示 send/receive/pending/transfer/error metadata。
  - 默认 topic 展示更新为 `clipboard.text`，手动发送按钮文案改为“发送到订阅”。
- Docs/tests:
- 更新 requirements/specs，写入 topic route、default topic、history restore、persistent body history store 和 pending metadata-only 边界。
  - 增加 Go route normalization、fan-out、remote gating、多 topic subscribe/update/reconnect、partial subscribe cleanup 覆盖。
  - 增加 Flutter topic route normalization、history restore promotion、pending 不入正文历史和 UI 文案覆盖。

## Related Plan

- [../../plan.md](../../plan.md)

## Related Requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related Specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Lessons Impact

none

本次没有产生新的可复用故障排查模式；关键规则已进入 requirements/specs，审查中发现的 pending metadata-only 边界属于本次需求本身。

## Related Lessons

- none

## Searchable Lessons Summary

- Keywords: `TopicRoute`, `clipboard.text`, `sync_to_local`, `sync_from_local`, `pending_topic`, `history.json`, `history.updated`, `restore_history`, `restoreHistory`, `ClipboardHistoryEntry`, `ignored_topic_policy`, `ignored_local_policy`.
- Future lookup: if History shows pending receive bodies, verify bridge `decisionCanEnterBodyHistory` and Dart `appendClipboardHistory` pending filter. If persisted history does not reload, check `cmd/clipboardnode-bridge/history_store.go`, `emitHistory`, and web `/history`.

## Requirements Impact

updated

`docs/requirements/clipboard-sync.md` now defines multi-topic directional sync, default TopicBus topic `clipboard.text`, body history restore semantics, local persistent body history, persistent non-sensitive config, and log/history separation.

## Specs Impact

updated

`docs/specs/clipboard-sync.md` now defines `TopicRoute`, `topics` config/status contract, multi-topic subscribe/fan-out/gating flow, `pending_topic`, `history.json`, `restore_history`, `history.updated`, and pending metadata-only behavior.

## Validation

- `$env:GOWORK='off'; go test ./core/runtime ./core/configstore ./bridge ./cmd/clipboardnode-bridge -count=1`: passed.
- `$env:GOWORK='off'; go test ./bridge ./cmd/clipboardnode-bridge -count=1`: passed after adding persistent history store coverage.
- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat analyze` from `app`: passed.
- `D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat test` from `app`: passed.
- `git diff --check`: passed.
- `rg -n "[ \t]+$" docs\change\2026-06-04_clipboard-multitopic-history.md`: no trailing whitespace matches.

PowerShell continued to print unrelated `conda-script.py ... invalid choice: ''` / `Invoke-Expression ... empty string` noise after successful commands.

## Code Review Notes

- Cross-language contract checked for `topic`, `topics`, `pending_topic`, and activity `topic`.
- Cross-language persistent history contract checked for `HistoryEntry`, `restore_history`, and `history.updated`.
- Clipboard body text remains absent from status, config, transfer, and metadata-only pending activity; persisted body text is isolated to local `history.json` and `history.updated`.
- Explicit empty/duplicate topics fail validation.
- Multi-topic reconnect partial subscribe failure now cleans up already-subscribed partial routes.
- No sub-agent was dispatched because runtime, bridge, Dart parser, and UI fields share one coupled JSON contract.

## Rollback

- Revert `core/runtime` topic route model, route normalization, multi-topic subscribe/fan-out/gating, and pending topic status changes.
- Revert bridge `TopicRoute`, `topics`, `pending_topic`, and activity topic mappings.
- Revert `cmd/clipboardnode-bridge/history_store.go`, `history_store_test.go`, `restore_history`, `history.updated`, `/history`, and `history.json` wiring.
- Revert Flutter `TopicSyncConfig`, multi-topic settings UI, `restoreHistory`, body-only clickable history tile, and related tests.
- Revert requirements/specs updates plus this archive/index entry.
