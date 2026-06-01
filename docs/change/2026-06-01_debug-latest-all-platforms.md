# 2026-06-01_debug-latest-all-platforms

## 变更背景 / 目标

`debug-latest` 首版只发布 Windows debug 包。为了让 ClipboardNode 的 Flutter UI 壳层在各设备上更早暴露平台构建问题，本次把自动化扩展为 Windows、Linux、macOS、Android、iOS simulator 和 Web 全平台 debug 构建，并继续复用同一个 prerelease 通道。

## 具体变更内容

- 新增 Flutter Linux 和 macOS 平台 host 项目，使对应 hosted runner 可以执行 `flutter build linux --debug` 和 `flutter build macos --debug`。
- 将 `.github/workflows/debug-latest.yml` 改为多 job 构建：
  - Go CLI 验证和 Windows helper 二进制构建；
  - Windows Flutter debug runner 打包；
  - Linux Flutter debug bundle 打包；
  - macOS unsigned debug `.app` 打包；
  - Android debug APK 打包；
  - iOS simulator unsigned debug `.app` 打包；
  - Web debug bundle 打包。
- `publish-debug-latest` 依赖所有构建 job，下载并校验全部资产后才移动 `debug-latest` tag 和覆盖上传 prerelease assets。
- README 更新为全平台 debug 预览资产说明。

## Requirements impact

none

本次只扩展发布自动化，符合现有 “cross-platform UI for desktop and mobile targets” 目标，不改变产品功能需求。

## Specs impact

none

本次不改变 ClipboardNode runtime、TopicBus 应用 payload、bridge contract 或 MyFlowHub 子协议边界。

## Lessons impact

none

未发现新的可复用故障模式；继续沿用既有 `debug-latest` native exit code lesson。

## Related requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Related lessons

- [../lessons/debug-latest-ci-native-exit-flutter-material.md](../lessons/debug-latest-ci-native-exit-flutter-material.md)
- [../lessons/flutter-windows-sdk-shared-bat-git.md](../lessons/flutter-windows-sdk-shared-bat-git.md)

## 对应 plan.md 任务映射

- `CI-8`: `app/linux`, `app/macos`, `app/.metadata`
- `CI-9`: `.github/workflows/debug-latest.yml`
- `CI-10`: `README.md`, `docs/change/2026-06-01_debug-latest-all-platforms.md`, `docs/change/README.md`
- `CI-11`: 本地验证、强制审查、push 和远端 Actions 检查

## 经验 / 教训摘要

- Flutter 的 Linux/macOS build 需要对应平台 host 目录，不能只在 workflow 中新增 build 命令。
- iOS debug 预览在 CI 中应构建 simulator app，并用 `--no-codesign` 避免把签名当成 debug 验证前置条件。
- 多平台资产路径差异较大，显式 job 比统一 matrix 更容易做路径校验和故障定位。
- Windows PowerShell native command 仍需保留 `$LASTEXITCODE` 检查，不能回退到只依赖 `$ErrorActionPreference`。

## 可复用排查线索

- 症状: `debug-latest` release 只有 Windows asset 或缺少某个平台。
- 触发条件: 对应 platform job 失败、artifact 名称变化、publish job 校验失败。
- 关键词: `debug-latest`, `merge-multiple`, `required_assets`, `myflowhub-clipboardnode-linux-debug.tar.gz`, `ios-simulator`.
- 快速检查:
  - 查看 `publish-debug-latest` 是否等待了全部 build job。
  - 在 publish 日志中搜索 `Missing required debug asset`。
  - 检查对应 build job 的打包路径是否和 Flutter 输出路径一致。

## 关键设计决策与权衡

- 保留一个 `debug-latest` 通道，而不是每个平台单独 release，方便测试者统一入口下载。
- Go CLI helper 独立为一个 job，避免每个 Flutter job 重复跑 Go 测试和跨编译。
- Go CLI 仍然只发布 Windows helper，因为当前 CLI host skeleton 仍依赖 Windows clipboard adapter；Linux/macOS 的本轮跨平台产物是 Flutter UI debug 包。
- Linux 使用 `.tar.gz` 保留 executable bit；Windows、macOS、iOS simulator 和 Web 使用 zip。
- Android 只上传 debug APK；生产签名和 app bundle 不属于 debug preview workflow。
- macOS/iOS 资产是 unsigned debug preview，后续生产分发应另建 release workflow。

## 测试与验证方式 / 结果

- `actions/setup-java` 最新 release tag 查询: `v5.2.0`，workflow 使用 `actions/setup-java@v5`。
- YAML 结构解析: 通过，确认 8 个 job 和 publish dependencies。
- `git diff --check`: 通过。
- 本地 Windows 可执行验证和远端 GitHub Actions 结果见本 workflow 后续 Stage 3.3 记录。

## 潜在影响

- 每次 `master` push 的 CI 成本和时间会增加，因为 macOS、Linux、Android、iOS simulator 和 Web 都会构建。
- GitHub Release assets 会从 Windows-only 增加到全平台，多平台下载入口需要用户按设备选择。
- 新生成的 Linux/macOS host 项目是 Flutter 默认壳层，未来平台集成、图标和权限需要按产品化需求继续完善。

## 回滚方案

- 将 `.github/workflows/debug-latest.yml` 恢复为上一版 Windows-only workflow。
- 删除 `app/linux`、`app/macos` 和 `.metadata` 中对应平台条目。
- 回滚 README 和本 change/index 更新。
- 如需清理远端资产，可在 GitHub Release 中删除新增平台 asset，或等待下一次 Windows-only publish 覆盖所需资产。

## 子Agent执行轨迹

未派发子Agent；主 agent 完成计划、实现、验证和归档。
