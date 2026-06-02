# 2026-06-02_tag-release-ci

## 变更背景 / 目标

现有 CI/CD 已经能在 `master` push 后构建全平台 debug 资产并刷新
`debug-latest` prerelease。用户希望保留这个 debug 预览通道，同时在推送
版本 tag 时自动构建新的 Release 版本。

本 workflow 先实现过一个中间版本：`v*` tag 复用 debug artifacts 发布
GitHub Release。用户随后明确要求这不应只是 unsigned preview/debug
assets，因此本次最终方案回滚该中间态，并改为独立 `release.yml`
release-mode 流水线。

## 具体变更内容

- `.github/workflows/debug-latest.yml` 回到 debug-only：
  - `push.branches: master`
  - `push.tags-ignore: "**"`
  - 保留 `publish-debug-latest`
  - 移除 `publish-tag-release`
- 新增 `.github/workflows/release.yml`：
  - `push.tags: v*`
  - `workflow_dispatch` dry-run input `release_tag`
  - `prepare-release` 校验 tag 必须匹配 `vX.Y.Z`
  - manual dispatch 只验证 build path，不发布 GitHub Release
  - tag push 才运行 `publish-release`
- 新增 release-mode 平台资产：
  - Windows release zip、Windows Go CLI、Windows bridge helper
  - Linux release tarball
  - macOS release zip
  - Android release APK/AAB
  - iOS release IPA
  - Web release zip
  - SHA-256 checksum file
- `app/android/app/build.gradle.kts` 新增 release signing 配置：
  - 优先读取 `ANDROID_KEYSTORE_PATH`、`ANDROID_KEYSTORE_PASSWORD`、`ANDROID_KEY_ALIAS`、`ANDROID_KEY_PASSWORD`
  - 支持 `MYFLOWHUB_ANDROID_*` Gradle property 别名
  - 没有 release signing 时只允许本地/manual dry-run fallback 到 debug signing
  - tag release 在 workflow 内强制检查 Android signing secrets
- release workflow 明确签名/公证 secrets contract：
  - Windows PFX code signing
  - Android keystore signing
  - macOS Developer ID signing and notarization
  - iOS distribution certificate and provisioning profile export
- README 更新 `Release Channels`：
  - `debug-latest` 是 debug preview prerelease
  - `vX.Y.Z` tag 由独立 release workflow 发布 release-mode assets
  - manual `release.yml` run 不发布 release
  - store upload 和 installer generation 仍不在当前范围内

## Requirements impact

none

本次只改变 CI/CD 发布自动化与打包签名入口，不改变 ClipboardNode 产品需求、
同步行为、隐私模型或 MyFlowHub 协议复用边界。

## Specs impact

none

本次不改变 runtime、bridge、TopicBus payload、platform adapter 或
MyFlowHub 子协议技术契约。

## Lessons impact

none

未出现新的可复用故障模式；继续沿用既有 CI/native-exit 和 gomobile mobile
bindings lessons。

## Related requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Related lessons

- [../lessons/debug-latest-ci-native-exit-flutter-material.md](../lessons/debug-latest-ci-native-exit-flutter-material.md)
- [../lessons/gomobile-mobile-bindings.md](../lessons/gomobile-mobile-bindings.md)

## 对应 plan.md 任务映射

- `CI-REL-1` - `CI-REL-4`: 第一版 tag release 实现与归档；已被用户澄清后的 release-mode 方案 supersede。
- `CI-REL-5`: `.github/workflows/debug-latest.yml`, `.github/workflows/release.yml`
- `CI-REL-6`: `.github/workflows/release.yml` desktop/web/Go release jobs
- `CI-REL-7`: `app/android/app/build.gradle.kts`, `.github/workflows/release.yml`
- `CI-REL-8`: `.github/workflows/release.yml` Apple release jobs
- `CI-REL-9`: `README.md`, `docs/change/2026-06-02_tag-release-ci.md`
- `CI-REL-10`: workflow assertions, `git diff --check`, Go tests, hosted dry-run validation

## 经验 / 教训摘要

- `debug-latest` 是 workflow 自己移动的 debug tag，不能把它和稳定版本发布放在同一触发路径里。
- 复用 debug artifacts 创建稳定 GitHub Release 会误导用户，因此正式版本要独立 release workflow 和 `*-release` asset names。
- 手动 dry-run 和真实 tag release 的安全边界不同：dry-run 可以验证 release build path，tag release 必须要求签名 secrets。
- Android release signing 不应永久硬编码 debug key；可以保留本地 fallback，但 tag release 必须在 workflow 层阻断缺失 secrets 的发布。
- iOS IPA 需要 distribution certificate、provisioning profile、Team ID 和 export options；不能用 simulator debug build 替代生产 IPA。

## 可复用排查线索

- 症状: 推送 `v1.2.3` 后没有生成 release。
- 触发条件:
  - tag 不匹配 `vX.Y.Z`
  - 平台 release build 失败
  - 签名 secrets 缺失
  - `publish-release` 因 `prepare-release.outputs.publish_release != true` 被跳过
- 关键词:
  - `release.yml`
  - `prepare-release`
  - `publish-release`
  - `PUBLISH_RELEASE`
  - `Missing Android release signing secrets`
  - `Missing macOS signing/notarization secrets`
  - `Missing iOS signing/export secrets`
  - `myflowhub-clipboardnode-release-checksums.txt`
- 快速检查:
  - 查看 Actions run 是否由 `refs/tags/vX.Y.Z` 触发。
  - 查看 `prepare-release` 输出的 `publish_release` 是否为 `true`。
  - 在平台日志中搜索 `Missing ... secrets`。
  - 在 publish 日志中搜索 `Missing required release asset`。
  - 确认 `debug-latest.yml` 没有 `publish-tag-release` job。

## 关键设计决策与权衡

- 选择独立 `release.yml`，避免把 debug 和 release artifact paths 混在同一个超长 workflow 中。
- tag pattern 先用 `v*` 触发，再在 `prepare-release` 用脚本校验 `vX.Y.Z`，避免 GitHub Actions glob 无法表达严格 SemVer。
- manual `workflow_dispatch` 不发布 release，降低 CI 验证时误发版本的风险。
- Windows/macOS/iOS tag release 对签名 secrets 采取 fail-fast；manual dry-run 不强制，以便分支验证 workflow 结构。
- Android Gradle 层保留 debug signing fallback，是为了本地或 manual dry-run 能跑通 release build path；真实 tag release 由 workflow 阻断无签名发布。
- 当前 workflow 不做 store upload、MSI/DMG installer、Play Store/TestFlight 发布；这些可以在 signed assets 稳定后增加。

## 测试与验证方式 / 结果

- Python/PyYAML 解析 `.github/workflows/debug-latest.yml`: 通过。
  - `push.branches=[master]`
  - `push.tags-ignore=[**]`
  - 无 `publish-tag-release`
- Python/PyYAML 解析 `.github/workflows/release.yml`: 通过。
  - `push.tags=[v*]`
  - 存在 `workflow_dispatch`
  - jobs: `prepare-release`, `build-windows-release`, `build-linux-release`, `build-macos-release`, `build-android-release`, `build-ios-release`, `build-web-release`, `publish-release`
- Workflow assertions: 通过。
  - `publish-release` 只在 `needs.prepare-release.outputs.publish_release == 'true'` 时运行
  - `publish-release.needs` 覆盖所有 release build jobs
  - release assets list 覆盖 Windows/Linux/macOS/Android/iOS/Web/Go/checksums
  - workflow 包含 Windows/Android/macOS/iOS signing secret gates
- Android Gradle signing assertions: 通过。
  - release signing 支持 env/Gradle property
  - 缺失 release signing 时只保留 local/dry-run debug fallback
- `git diff --check`: 通过。
- `$env:GOWORK='off'; go test ./... -count=1`: 通过。
- Android release APK dry-run build: 通过。
  - Command: `flutter build apk --release --build-name 0.0.0 --build-number 1`
  - Result: produced `app/build/app/outputs/flutter-apk/app-release.apk`
  - Purpose: prove the Gradle Kotlin DSL release signing config parses and the dry-run debug-signing fallback can build a release APK locally.

待补充验证:

- hosted `workflow_dispatch` dry-run for `release.yml` after `release.yml` exists on `master`

未在本地证明:

- 真实 tag push 后的 GitHub Release 创建需要用户批准推送 `vX.Y.Z` tag。
- Windows/macOS/iOS production signing/notarization/export 需要私有证书和 GitHub Secrets。
- Apple 平台 release build 只能在 macOS runner 上验证，本地 Windows SDK 不提供 iOS/macOS build subcommands。
- Hosted `workflow_dispatch` dry-run 已尝试执行：
  - `gh workflow run release.yml --repo yttydcs/myflowhub-clipboardnode --ref chore/tag-release-ci -f release_tag=v0.0.0`
  - GitHub 返回 `HTTP 404: workflow release.yml not found on the default branch`
  - 原因是 `release.yml` 是本分支新增 workflow，尚未存在于默认分支；需要合并到 `master` 后再 dispatch 验证。

## 潜在影响

- `master` push 只刷新 `debug-latest` prerelease。
- `vX.Y.Z` tag push 会触发完整 release-mode build 和稳定 GitHub Release 发布。
- 缺少 signing secrets 时，真实 tag release 会失败，不会发布假生产包。
- Manual `release.yml` run 可用于验证 build path，但不会创建或更新 GitHub Release。

## 回滚方案

- 删除 `.github/workflows/release.yml`。
- 将 `app/android/app/build.gradle.kts` release signing 配置恢复为 debug signing。
- README 恢复为只描述 `debug-latest`。
- 如果已经创建了错误的版本 release，可在 GitHub Release 中删除该 release/tag 或重新推送正确 tag 后重跑。
- 如果只需回退到第一版 tag release，可重新添加 `debug-latest.yml` 的 `v*` trigger 和 `publish-tag-release`，但这会回到 debug assets release，不符合用户澄清后的目标。

## 子Agent执行轨迹

未派发子Agent；主 agent 完成需求/架构修订、计划、实现、验证、review 和归档。
