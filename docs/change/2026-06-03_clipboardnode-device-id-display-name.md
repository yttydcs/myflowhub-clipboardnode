# 2026-06-03_clipboardnode-device-id-display-name

## 变更背景 / 目标

用户反馈修改设备 ID 后会弹出 `authenticate myflowhub node: invalid signature`，并明确要求修改设备 ID 时清理对应登录数据，同时显示名称需要支持独立配置。

本次目标是将 ClipboardNode 的认证身份和展示名称拆开：`device_id` 用于 MyFlowHub auth identity 和签名，`display_name` 用于 UI 与 auth metadata。修改 `device_id` 时清理本地 auth snapshot，避免复用旧 node id；仅修改 `display_name` 时保留同一节点身份。

## 具体变更内容

- `core/runtime.Config` 新增 `device_id` 与 `display_name`，保留 `device_label` 作为 legacy alias。
- `NormalizeConfig` 将旧 `device_label` 配置迁移为 `device_id` / `display_name` fallback，并把 `device_label` 归一为展示名。
- bridge settings/status JSON 新增 `device_id` 与 `display_name`，stdio bridge 在 `set_config` 时先归一化再保存和更新 engine。
- `core/engine.Engine.UpdateConfig` 比较归一化后的 `device_id`，变化时停止 runtime 并调用 `ClearAuth`。
- `core/myflowhub.Client.EnsureIdentity` 在启动认证前检查保存的 snapshot identity；不匹配时清理 stale snapshot。
- auth register/login payload 使用 `device_id` 作为身份，使用 `display_name` 作为 metadata。
- Flutter 设置页拆分为 `设备 ID` 和 `显示名称` 两个字段，live/web/mobile/preview bridge 兼容解析新旧字段。
- 增加 Go 回归测试覆盖 config persistence、bridge contract、engine auth clearing、auth payload、stale snapshot clearing 和 display-name-only 行为。

## Requirements impact

updated

## Specs impact

updated

## Lessons impact

updated

## Related requirements

- `docs/requirements/clipboard-sync.md`

## Related specs

- `docs/specs/clipboard-sync.md`

## Related lessons

- `docs/lessons/device-id-auth-snapshot-mismatch.md`
- `docs/lessons/startup-subscribe-timeout-half-connected.md`

## 对应 plan.md 任务映射

- T1 - Config and Bridge Contract: `core/runtime/config.go`, `core/configstore/store_test.go`, `bridge/contract.go`, `bridge/contract_test.go`, `cmd/clipboardnode-bridge/main.go`
- T2 - Auth Clearing and Payloads: `core/engine/engine.go`, `core/engine/engine_test.go`, `core/myflowhub/client.go`, `core/myflowhub/client_test.go`
- T3 - Flutter Settings UI: `app/lib/core/bridge/*.dart`, `app/lib/features/shell/clipboard_shell.dart`
- T4 - Validation: Go tests, Flutter tests, bridge smoke
- T5 - Docs Archive: this change archive and the lesson entry

## 经验 / 教训摘要

设备身份和显示名称不能共用一个字段。`device_id` 是 auth identity，影响签名和 node reuse；`display_name` 是展示 metadata。修改身份时必须清理本地 auth snapshot，否则会把旧 node id 与新身份混用。

## 可复用排查线索

- 症状：`authenticate myflowhub node: invalid signature`
- 症状：修改设备 ID 后连接失败
- 症状：旧 `auth_snapshot.json` 中还有旧 `device_id` / `node_id`
- 触发条件：已注册设备再次修改 `device_id`
- 快速检查：
  - 查看 config 中 `device_id`、`display_name`、legacy `device_label`
  - 查看 `myflowhub/auth_snapshot.json`
  - 比较 snapshot `device_id` 是否等于 config `device_id`
  - 只改显示名称时确认 node id 没有被清掉
- 关键词：ClipboardNode, device_id, display_name, device_label, auth_snapshot, invalid signature

## 关键设计决策与权衡

- 没有修改 MyFlowHub Auth、TopicBus、Management、SDK、Server 或 SubProto wire contract。
- 没有自动删除 `node_keys.json`；该文件是密钥材料，当前 bug 只需要清理 stale auth snapshot。
- 旧 `device_label` 继续兼容，避免升级后旧 config 无法启动。
- 修改 `device_id` 后可能需要重新注册并等待 Hub 审批；这是正确的身份变更语义。

## 测试与验证方式 / 结果

- `GOWORK=off go test ./... -count=1`：通过
- `GOWORK=off go test -race ./core/myflowhub ./core/engine ./bridge -count=1`：通过
- `GOWORK=off go build ./cmd/clipboardnode-bridge`：通过
- `flutter analyze`：通过
- `flutter test`：通过，5 tests
- `scripts/validate.ps1 -FlutterBin D:\project\MyFlowHub3\.tmp\tools\flutter-sdk-3.41.9\flutter\bin\flutter.bat`：通过，包含 Go tests/builds、Flutter analyze/test 和 `git diff --check`
- Bridge smoke 使用临时 config/auth 目录：
  - `device_id` 从 `old-device` 改为 `new-device` 后，配置保存新 identity/display，`myflowhub/auth_snapshot.json` 被清空。
  - `same-device` 只改 `display_name` 后，配置保存新显示名，auth snapshot 保留同一 node identity。

## 潜在影响与回滚方案

- 修改 `device_id` 后会清理 auth snapshot 并触发重新注册/登录流程，可能需要 Hub 重新审批。
- 旧配置会在保存时写入新字段，并把 `device_label` 归一为显示名称。
- 回滚时恢复 `core/runtime`、`bridge`、`core/engine`、`core/myflowhub`、Flutter bridge/UI 及相关测试即可；回滚后修改设备 ID 仍可能复现 stale auth snapshot 与 invalid signature。

## 子Agent执行轨迹

未使用子Agent。T1-T3 的字段语义跨 Go 和 Dart 共享，写集小但耦合紧，由主Agent完成实现、验证和复核。
