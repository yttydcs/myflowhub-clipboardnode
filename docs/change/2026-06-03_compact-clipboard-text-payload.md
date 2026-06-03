# 2026-06-03_compact-clipboard-text-payload

## 变更背景 / 目标

用户确认 ClipboardNode 不需要兼容旧的 verbose `clipboard.text.v1` payload。本次目标是把小文本同步 payload 收窄为只包含应用层必需字段，让手工 TopicBus 测试和节点间文本同步不再需要提前构造 `size`、`sha256`、`content_type`、`encoding`、`ts` 等派生元数据。

## 具体变更内容

- 将 `ClipboardTextEventV1` 序列化字段改为 compact JSON:

```json
{
  "v": 1,
  "id": "uuid",
  "from": 12,
  "instance": "runtime-uuid",
  "device": "win-laptop",
  "text": "hello"
}
```

- `Size` 和 `SHA256` 仍保留为 Go runtime 派生字段，但使用 `json:"-"`，不再进入 `clipboard.text.v1` payload。
- 新增 text event normalize path，在 marshal/parse 时裁剪身份字段、校验版本/事件 ID/来源/文本，并从 `text` 本地计算 UTF-8 byte size 和 SHA-256。
- 删除 text payload 对 `content_type`、`encoding`、`size`、`sha256`、`ts` 的远端输入依赖。
- 保持 `clipboard.transfer.v1` manifest 不变，large/oversize transfer 仍走 manifest 设计。
- 更新 runtime 和 fixture 测试，覆盖 compact payload parse、publish、oversize rejection、local-origin ignore 和派生 digest。
- 更新稳定 requirements/specs，把 compact text payload 作为当前 ClipboardNode 应用契约。

## Requirements impact

updated

更新 [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)，明确小文本 TopicBus 事件只携带 compact 身份字段和 `text`，byte size 与 SHA-256 由 ClipboardNode 本地计算，用于校验、状态和 loop suppression。

## Specs impact

updated

更新 [../specs/clipboard-sync.md](../specs/clipboard-sync.md)，同步 local publish、remote receive、接口草案、错误处理、性能测试和扩展性说明。MyFlowHub TopicBus、Auth、Server、Proto、SDK、SubProto wire contract 未改变。

## Lessons impact

none

本次是受控 contract 收窄，没有新增昂贵排查路径或可复用事故规则；不新增 `docs/lessons`。

## Related requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Related lessons

- none

## 对应 plan.md 任务映射

- `T1 - Runtime compact payload`: `core/runtime/event.go`
- `T2 - Tests and fixtures`: `core/runtime/event_test.go`, `core/runtime/runtime_test.go`
- `T3 - Documentation and archive`: `docs/requirements/clipboard-sync.md`, `docs/specs/clipboard-sync.md`, this change archive, `docs/change/README.md`

## 经验 / 教训摘要

- 系统剪贴板只保留正文，无法保留 `event_id` 或 origin metadata。远端写入后 watcher 看到的本地变化必须靠 runtime 的 suppress hash 识别，不应依赖剪贴板内容里携带原事件元数据。
- `event_id` 负责同一事件重复投递去重，`from` / `instance` 负责过滤自身来源，`SHA256(text)` 负责 remote apply 后的本地 watcher 回环抑制。
- 既然不兼容旧 payload，解析器应直接拒绝旧 verbose shape 中缺失 compact 必填字段的消息，而不是同时维护两套 contract。

## 可复用排查线索

- 症状: 手工发布 `clipboard.text.v1` 时需要填写 `size`、`sha256`、`content_type` 或 `encoding`。
- 触发条件: 测试样例或工具仍按旧 verbose text payload 构造消息。
- 关键词: `clipboard.text.v1`, `v`, `id`, `from`, `instance`, `text`, `ParseClipboardTextEventV1`.
- 快速检查: payload 只保留 `v/id/from/instance/device/text`；接收侧状态里的 size/hash prefix 应由 runtime 根据 `text` 派生。

## 关键设计决策与权衡

- 保持事件名 `clipboard.text.v1` 不变，只改变 ClipboardNode 应用 payload；TopicBus envelope 和 MyFlowHub wire protocol 不变。
- 不保留旧 verbose parser，因为用户明确说不用考虑兼容。这样能让当前 contract 单一，测试和手工发布也更直接。
- 继续保留 runtime `Size` / `SHA256` 字段，避免下游状态、pending、decision 和 loop suppression 逻辑失去已有元数据。
- 不调整 `clipboard.transfer.v1` manifest，因为 transfer manifest 本来就是 metadata/reference payload，和小文本 inline body 的简化目标不同。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./core/runtime -count=1`: passed.
- `$env:GOWORK='off'; go test ./core/runtime ./core/myflowhub ./bridge -count=1`: passed.
- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `git diff --check`: passed. PowerShell startup produced unrelated conda noise; Git reported CRLF conversion warnings only.

## 潜在影响

- 旧节点或旧手工脚本仍发布 verbose text payload 时，新 parser 不再兼容。
- 接收端 size/hash 必须以本地计算结果为准，不再信任远端 payload 声明。
- 如果平台剪贴板在写入后改变文本字节，hash-based suppress loop 仍可能把变形后的内容视为新的本地变化；这是既有剪贴板语义限制。

## 回滚方案

- Revert `core/runtime/event.go` 恢复 verbose text event JSON fields 和远端 size/hash 校验。
- Revert `core/runtime/event_test.go`、`core/runtime/runtime_test.go` 恢复旧 payload fixture。
- Revert `docs/requirements/clipboard-sync.md`、`docs/specs/clipboard-sync.md`、this archive and `docs/change/README.md` if the compact contract is paused.

## 子Agent执行轨迹

Stage 3.2 and 3.3 assessed parallelism. No sub-agent was dispatched because the change is a single small runtime contract with tightly coupled tests and docs; main agent retained implementation, validation, review, and archive ownership.
