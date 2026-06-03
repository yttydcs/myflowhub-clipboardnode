# 2026-06-02_clipboardnode-startup-lifecycle

## 变更背景 / 目标

用户反馈 ClipboardNode 连接后不可用，UI 显示 `subscribe clipboard topic: context deadline exceeded`，同时 MyFlowHub-Win 控制台树中找不到该节点。排查确认本地 auth snapshot 保存了 `node_id=14` 和 `logged_in=true`，但新进程连接后可能直接信任持久化登录态并跳过登录，导致当前 TCP 会话没有重新绑定 node 14。

本次目标是修复 ClipboardNode 本地启动生命周期，避免半连接状态，并让节点注册/登录时提供可读显示名。

## 具体变更内容

- `core/myflowhub.Client` 加载 auth snapshot 时强制将 `LoggedIn` 归一为 `false`，保留 `device_id`、`node_id`、`hub_id` 等可复用身份字段。
- `core/myflowhub.Client.Close` 将 `logged_in=false` 写回 snapshot，避免下次进程启动误判旧会话仍在线。
- `core/myflowhub.Client` 的 register/login payload 补充 `display_name`，使用修剪后的 `deviceID`。
- `core/engine.Engine.Start` 在 `Connect` 成功后增加失败清理 guard；认证、runtime 创建或 TopicBus subscribe 失败时 best-effort 关闭 transport，成功启动后释放 guard。
- 增加单元测试覆盖 auth payload、snapshot stale-login 归一、Close 持久化登出，以及 Engine 启动失败后的 transport 清理。

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

- `docs/lessons/startup-subscribe-timeout-half-connected.md`

## 对应 plan.md 任务映射

- T1 - Startup Failure Cleanup: `core/engine/engine.go`, `core/engine/engine_test.go`
- T2 - Auth Session Snapshot and Display Name: `core/myflowhub/client.go`, `core/myflowhub/client_test.go`
- T3 - Validation: targeted and full Go tests
- T4 - Docs Archive: this file plus the lesson entry

## 经验 / 教训摘要

`logged_in` 是当前连接会话状态，不应作为可跨进程复用的事实持久化。auth snapshot 可以保存 node identity，但新 TCP 会话必须重新 login，才能让远端 Hub connection manager 绑定该 node 并被 management 列表发现。

## 可复用排查线索

- 症状：`subscribe clipboard topic: context deadline exceeded`
- 症状：本地 snapshot 有 node id，但 Win 控制台树找不到该 node
- 快速检查：
  - 查看 `%APPDATA%/myflowhub/ClipboardNode/myflowhub/auth_snapshot.json`
  - 查看 `logged_in` 是否来自旧 snapshot
  - 确认新连接是否真正发送了 auth login
  - 确认启动失败后 `transport.Status().Connected` 是否回到 false
- 关键词：ClipboardNode, node 14, auth_snapshot, logged_in, TopicBus subscribe timeout, management ListNodes

## 关键设计决策与权衡

- 没有修改 TopicBus、Auth、Management、SDK、Server 或 SubProto wire contract，符合 ClipboardNode 只复用现有协议的边界。
- 没有通过增加 subscribe timeout 掩盖问题；无响应仍应作为 transport failure 显式暴露。
- 没有自动重试 subscribe；后续如果需要 reconnect/backoff，应作为独立行为变更设计。
- cleanup guard 忽略启动失败路径上的 Close 错误，保留原始失败原因；正常 Stop 路径仍返回 Close 错误。

## 测试与验证方式 / 结果

- `GOWORK=off go test ./core/engine ./core/myflowhub ./core/runtime -count=1`：通过
- `GOWORK=off go test ./... -count=1`：通过
- 使用修复后的 `build/clipboardnode-bridge.exe` 在临时目录加载复制的 `%APPDATA%\MyFlowHub\ClipboardNode\config.json` 和 `node_keys.json`，执行 `connect` 后返回 `connected=true`、`logged_in=true`、`node_id=14`、`subscribed=true`：通过
- 使用当前 MCP `node_id=15` 查询 Hub 1 的 `myflowhub_management_list_subtree`，返回节点 `8 (NAS MyFlowHub MCP)`、`15 (AI MCP)`、`14 (local-device)`、`1`，确认修复后的 ClipboardNode 会重新出现在 management 树中：通过

## 潜在影响

- 进程重启后即使 snapshot 中曾有 `logged_in=true`，也会重新 login；这是期望行为。
- `Close` 现在会写 auth snapshot，极端情况下可能暴露本地文件写入错误；这比静默失败更安全。
- 远端 Hub 如果仍不返回 TopicBus subscribe response，ClipboardNode 会明确失败并断开，而不是保持半连接。

## 回滚方案

回滚 `core/engine/engine.go`、`core/myflowhub/client.go` 及对应测试变更即可恢复旧行为。若回滚，旧 snapshot 仍可能让新进程跳过登录并复现半连接问题。

## 子Agent执行轨迹

未使用子Agent；T1/T2 写集小且共享启动/auth 上下文，由主Agent完成实现、测试和复核。
