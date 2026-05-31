# 2026-05-31_debug-latest-build

## 变更背景 / 目标

为 ClipboardNode 增加 `debug-latest` 自动化构建发布能力，让 `master` 最新提交可以通过 GitHub Releases 直接下载 Windows debug 预览包。

## 具体变更内容

- 新增 `.github/workflows/debug-latest.yml`。
- 在 Windows runner 上执行 Go 测试、Go CLI 构建、Flutter 分析、Flutter 测试和 Flutter Windows debug 构建。
- 将完整 Flutter Windows debug runner 目录打包为 `myflowhub-clipboardnode-windows-debug.zip`。
- 同步上传 Go CLI 二进制 `clipboardnode-windows-amd64.exe`。
- 在 `master` push 成功后强制更新 `debug-latest` tag，创建或更新 `Debug (latest)` prerelease，并覆盖上传最新资产。
- README 增加 debug 预览下载入口和产物说明。

## Requirements impact

none

## Specs impact

none

## Lessons impact

updated

首个远端 run 暴露出 PowerShell native command 失败未中断 workflow，以及 Flutter 新版本 `ListTile` material ancestry 断言问题，已补充 reusable lesson。

## Related requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Related lessons

- [../lessons/flutter-windows-sdk-shared-bat-git.md](../lessons/flutter-windows-sdk-shared-bat-git.md)
- [../lessons/debug-latest-ci-native-exit-flutter-material.md](../lessons/debug-latest-ci-native-exit-flutter-material.md)

## 对应 plan.md 任务映射

- `CI-1`: `.github/workflows/debug-latest.yml`
- `CI-2`: `README.md`, `docs/change/2026-05-31_debug-latest-build.md`, `docs/change/README.md`
- `CI-3`: 本地验证与审查记录
- `CI-4`: 首跑后修复 PowerShell native command 退出码检查、Flutter 版本 pin、Actions major 版本。
- `CI-5`: 修复 Flutter 3.44 `ListTile` material ancestry 断言。
- `CI-6`: 增加 lessons 和归档修正。
- `CI-7`: 二次本地/远端验证。

## 经验 / 教训摘要

- Flutter Windows 发布不能只上传 `.exe`；debug 预览包必须包含 runner 目录中的 DLL、`data/` 和其他运行时文件。
- `debug-latest` 是可移动预览通道，不等价于签名生产版本。
- PowerShell 的 `$ErrorActionPreference = "Stop"` 不能替代 native command 的显式 `$LASTEXITCODE` 检查。
- Flutter stable 自动漂移会把本地没暴露的 widget 断言带到 CI；CI 需要 pin 已验证版本或主动兼容新版本断言。

## 可复用排查线索

- 症状: GitHub Release 没有刷新 `debug-latest`。
- 触发条件: workflow 不是 `push` 到 `refs/heads/master`，或 build job 失败。
- 关键词: `debug-latest`, `myflowhub-clipboardnode-windows-debug.zip`, `clipboardnode-windows-amd64.exe`, `Publish debug-latest`.
- 快速检查:
  - 查看 Actions 的 `Build Windows debug` job 是否完成。
  - 查看 `Publish debug-latest` job 是否因分支条件被跳过。
  - 查看 release assets 是否被 `gh release upload --clobber` 覆盖。

## 关键设计决策与权衡

- 选择单个 workflow 覆盖构建和发布，减少 artifact/job 之间的接口复杂度。
- 选择 Flutter stable channel，而不是写死本地工作区 SDK 路径，避免 CI 依赖环境私有路径。
- 只在 `master` push 发布，pull request 和手动运行只用于验证/下载 Actions artifact。
- Go action 缓存先关闭，因为当前 Go module 没有 `go.sum`，后续有依赖后可再启用缓存。
- 首跑后将 Flutter pin 到 `3.41.9`，并把 GitHub 官方 Actions 升级到 Node 24-compatible major 版本。

## 测试与验证方式 / 结果

- `GOWORK=off go test ./... -count=1`: 通过。
- `go build -o build/clipboardnode-windows-amd64.exe ./cmd/clipboardnode`: 通过。
- `flutter analyze`: 通过。
- `flutter test`: 通过，5 个 widget/bridge 测试通过。
- `flutter build windows --debug`: 通过，生成 `app/build/windows/x64/runner/Debug/myflowhub_clipboard.exe`。
- 本地模拟 packaging 脚本: 通过，生成 `myflowhub-clipboardnode-windows-debug.zip` 和 `clipboardnode-windows-amd64.exe`。
- `git diff --check`: 通过。
- `actionlint`: 本机未安装，未运行；workflow 已做 YAML 解析和命令路径人工审查。
- 首个远端 run `26717910962`: job conclusion 为 success，但日志显示 `flutter test` 4 通过 1 失败，原因是 native command 失败未中断 PowerShell step；该 run 的 release 资产不应作为最终验证结果。
- 修复后验证结果见后续提交和第二个远端 run。

## 潜在影响

- 首次推送后 GitHub Actions 会创建 `debug-latest` tag 和 prerelease。
- debug 包未签名，Windows 下载后可能需要用户确认运行。
- Flutter stable channel 变化可能在未来触发 CI 行为变化；需要后续根据项目稳定版本 pin。

## 回滚方案

- 删除 `.github/workflows/debug-latest.yml` 停止自动构建/发布。
- 如需撤销发布通道，可在 GitHub 删除 `debug-latest` release 和 tag。
- README 和本变更归档可随 workflow 删除一并回滚。

## 子Agent执行轨迹

未派发子Agent；主 agent 完成 worktree、计划、实现、验证与归档。
