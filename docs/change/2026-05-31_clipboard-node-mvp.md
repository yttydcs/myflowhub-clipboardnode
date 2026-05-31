# 2026-05-31_clipboard-node-mvp

## 变更背景 / 目标

为 MyFlowHub 剪贴板同步能力新建独立 `MyFlowHub-ClipboardNode` 仓库，并实现第一阶段 MVP：使用 TopicBus 承载在线小文本剪贴板事件，保持默认禁用、文本大小有界、无剪贴板正文日志或配置持久化。

## 具体变更内容

- 新增 Go module `github.com/yttydcs/myflowhub-clipboardnode`。
- 新增 `core/clipboard`，定义平台剪贴板适配器接口和文本事件结构。
- 新增 `core/runtime`，实现 `clipboard.text.v1` 事件模型、校验、hash、JSON 编解码、TopicBus 客户端接口、启停/配置切换、重订阅、事件 ID 去重、本地来源过滤和远端写入回环抑制。
- 新增 `core/configstore`，用 JSON 保存非敏感配置，缺失配置返回 disabled 默认值。
- 新增 `windows` Win32 `CF_UNICODETEXT` 适配器和轮询 watcher；读取有上限，错误通过事件上报，不保存正文历史。
- 新增 `cmd/clipboardnode` headless 宿主骨架；在 live SDK TopicBus 传输接入前，`enabled=true` 会明确失败。
- 新增 `scripts/validate.ps1`，在独立 module 模式下运行测试、构建和 `git diff --check`。
- `.gitignore` 增加 `.ace-tool/`，避免本地索引工具产物进入仓库。

## Requirements impact

none. 本轮实现已覆盖既有 `docs/requirements/clipboard-sync.md` 的 MVP 范围，没有新增或废弃需求。

## Specs impact

none. 实现遵循 `docs/specs/clipboard-sync.md` 的模块边界和事件契约；live SDK TopicBus 传输仍按规格保留为后续宿主接入点。

## Lessons impact

none. 本轮未产生需要长期排查复用的生产问题；`GOWORK=off` 的独立仓库验证方式已记录在 README 和本归档中。

## Related requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Related lessons

- none

## 对应 plan.md 任务映射

- `CLIP-1`: `core/runtime` 事件模型、校验、hash、JSON 编解码和单元测试。
- `CLIP-2`: `core/runtime` 编排逻辑、fake TopicBus/fake clipboard 测试、启停与回环抑制。
- `CLIP-3`: `windows` 剪贴板适配器与 `cmd/clipboardnode` 宿主骨架。
- `CLIP-4`: `core/configstore` JSON 配置持久化和测试。
- `CLIP-5`: README、变更归档、验证脚本、`.gitignore` 仓库卫生。

## 经验 / 教训摘要

- 新仓库位于父级 `go.work` 覆盖范围内时，独立验证需显式设置 `GOWORK=off`。
- 禁用同步时必须先停止剪贴板 watcher，再做 TopicBus 退订；即使退订失败，也不能继续读取系统剪贴板。
- TopicBus 传输应保持接口化，避免在核心剪贴板规则稳定前把 SDK 登录和 UI 生命周期耦合进核心 runtime。

## 可复用排查线索

- 症状：`go test ./...` 报 `directory prefix . does not contain modules listed in go.work`。
- 触发条件：新独立仓库在父级 workspace 目录下，但还未加入父级 `go.work`。
- 关键词：`GOWORK=off`, `go.work`, `directory prefix`, `not one of the workspace modules`。
- 快速检查：在仓库根目录执行 `$env:GOWORK="off"; go test ./... -count=1`。

## 关键设计决策与权衡

- 使用 TopicBus 承载 `clipboard.text.v1` 小文本事件，保留 Stream/File 作为未来大内容路径。
- 核心 runtime 只依赖接口，不直接依赖 MyFlowHub SDK 或 Win32 API。
- Windows watcher 采用轮询 `GetClipboardSequenceNumber` 的最小实现，先保证可构建和边界明确，后续可替换为消息循环方案。
- 配置只保存 `enabled`、`topic`、`max_inline_bytes`、`device_label`，不保存剪贴板正文、事件正文或历史。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./... -count=1`: passed.
- `$env:GOWORK='off'; go build -o 'build/clipboardnode.exe' ./cmd/clipboardnode`: passed.
- `git diff --check`: passed.

## 潜在影响

- 当前宿主骨架尚未接入 live MyFlowHub SDK 登录和 TopicBus publish/subscribe，启用真实同步仍需后续任务。
- Windows watcher 使用轮询方式，后续如需要更低功耗或更及时响应，可改为消息窗口监听剪贴板更新。
- TopicBus 无 delivery ACK 和 offline replay，用户界面不能宣称远端已应用。

## 回滚方案

- 回滚本次提交即可移除 `core/`、`windows/`、`cmd/`、`scripts/` 及对应 README/change 更新。
- 若只需禁用运行能力，保持默认 `enabled=false` 或删除配置文件即可阻止剪贴板读取和写入。

## 子Agent执行轨迹

未派发子Agent。原因：核心事件模型、runtime、配置和 Windows 骨架共享边界较多，且本轮未暴露合适的 sub-agent dispatch 工具；主 Agent 单独完成实现、审查和验证。
