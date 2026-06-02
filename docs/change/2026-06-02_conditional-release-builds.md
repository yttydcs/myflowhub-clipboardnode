# 2026-06-02_conditional-release-builds

## 变更背景 / 目标

第一版正式 release workflow 要求 tag release 时所有签名平台的 secrets 都存在。
当前仓库已经配置 Android release signing secrets，但 Windows、macOS、iOS 的
生产证书材料尚未配置。用户希望未配置的平台先不进行 build，避免阻塞
`v0.1.0` 首个版本发布，同时不能把 unsigned 产物伪装成生产 release。

## 具体变更内容

- `.github/workflows/release.yml`:
  - `prepare-release` 新增每个平台的 build capability outputs。
  - `prepare-release` 新增 `Detect configured release platforms` step。
  - Windows/macOS/Android/iOS release build jobs 增加 job-level `if` 条件。
  - tag release 时，签名 secret set 不完整的平台会被 skipped。
  - manual `workflow_dispatch` dry-run 仍启用所有平台 build path 验证，且不发布 Release。
  - `publish-release` 改用 `always()`，允许上游 job 被 skipped 后仍进入发布阶段。
  - `publish-release` 新增 enabled/skipped job result 校验，避免 enabled job 失败后误发布。
  - release assets、checksums、Release Notes 改为按实际产物动态生成。
- `README.md`:
  - 将版本发布说明从“所有 required assets 都必须存在”调整为“所有 enabled platform jobs 必须成功”。
  - 明确未配置签名 secrets 的平台会被跳过，不发布 unsigned production assets。

## Requirements impact

none

本次只调整 CI/CD 版本发布策略，不改变 ClipboardNode 产品能力、同步行为、
隐私模型、协议复用边界或平台功能目标。

## Specs impact

none

本次不改变 runtime、bridge、TopicBus payload、platform adapter、
gomobile binding 或 MyFlowHub 子协议技术契约。

## Lessons impact

none

本次路径直接延续既有 tag release CI 和 gomobile/mobile binding lessons；
没有暴露新的独立故障模式。GitHub API 偶发 TLS/EOF 重试属于环境网络噪声，
不足以新增项目 lesson。

## Related requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Related lessons

- [../lessons/debug-latest-ci-native-exit-flutter-material.md](../lessons/debug-latest-ci-native-exit-flutter-material.md)
- [../lessons/gomobile-mobile-bindings.md](../lessons/gomobile-mobile-bindings.md)

## 对应 plan.md 任务映射

- `REL-1`: `.github/workflows/release.yml` prepare-release platform capability outputs.
- `REL-2`: `.github/workflows/release.yml` signed-platform job gates.
- `REL-3`: `.github/workflows/release.yml` dynamic publish asset and checksum logic.
- `REL-4`: `README.md` configured-platform release documentation.

## 经验 / 教训摘要

- 未配置生产签名材料的平台不应构建并发布 production asset；跳过比失败整个 release 更适合分阶段首版发布。
- manual dry-run 和 tag release 的目标不同：dry-run 验证 build path，tag release 发布已配置平台的生产资产。
- 使用 skipped needs job 时，publish job 必须用 `always()` 并在脚本内显式校验 enabled job 的 result。
- 动态 release asset 列表必须同时驱动 upload、checksum 和 release notes，避免三处清单不一致。

## 可复用排查线索

- 症状:
  - 推送 `vX.Y.Z` 后某个平台没有出现在 Release assets 中。
  - `Publish GitHub Release` 被跳过或失败。
  - Release notes 中显示 `skipped: missing ...`。
- 触发条件:
  - Windows/macOS/Android/iOS 平台缺少对应 signing secrets。
  - enabled 平台 build job 失败。
  - `publish-release` 没有使用 `always()`，上游 skipped job 传递导致发布 job skipped。
- 关键词:
  - `Detect configured release platforms`
  - `build_windows`
  - `build_android`
  - `windows_status`
  - `skipped: missing`
  - `Validate release build results`
  - `known_assets`
  - `release_assets`
- 快速检查:
  - 查看 `Prepare release metadata` 的 platform build plan 表。
  - 查看 `gh secret list --repo yttydcs/myflowhub-clipboardnode` 是否包含目标平台 secret set。
  - 查看 `Publish GitHub Release` 的 `Validate release build results` step。
  - 对 manual dry-run，确认 `Publish GitHub Release` skipped 且 `gh release view v0.0.0` 不存在。

## 关键设计决策与权衡

- 选择在 `prepare-release` 集中计算平台能力，而不是在每个 job 内自行决定是否跳过，避免 publish 阶段无法知道跳过原因。
- Linux 和 Web 保持 always enabled，因为当前 workflow 没有签名 prerequisites。
- Windows/macOS/Android/iOS 采用 job-level skip，不再进入会产生误导产物的 build path。
- Manual dry-run 保持全平台 path 验证，以继续覆盖 macOS/iOS/Windows build 脚本结构。
- `publish-release` 不再维护静态 required asset 清单，而是从已下载 artifacts 中选择已知 release asset names。

## 测试与验证方式 / 结果

- Python/PyYAML workflow structural check: 通过。
- Bash syntax check for all `shell: bash` release workflow scripts: 通过，16 scripts。
- PowerShell parser syntax check for `shell: pwsh` scripts: 通过，2 scripts。
- Capability simulation:
  - tag release with Android-only secrets: Windows/macOS/iOS skipped, Linux/Android/Web enabled。
  - manual dry-run with no secrets: all platforms enabled。
- Publish asset simulation:
  - only Linux/Web/Android assets present 时，只上传这些 assets，并只为这些 assets 生成 checksum。
- `git diff --check`: 通过。
- Hosted GitHub Actions dry-run:
  - Run: <https://github.com/yttydcs/myflowhub-clipboardnode/actions/runs/26824592121>
  - Trigger: `workflow_dispatch` on `chore/conditional-release-builds`, `release_tag=v0.0.0`
  - Result: success
  - Successful jobs:
    - Prepare release metadata
    - Build Windows release
    - Build Linux release
    - Build macOS release
    - Build Android release
    - Build iOS release
    - Build Web release
  - `Publish GitHub Release`: skipped as expected for manual dry-run.
  - `gh release view v0.0.0`: absent, confirming no accidental release.

## 潜在影响

- `vX.Y.Z` tag release 在当前 Android-only signing 配置下，预计发布 Linux、Android、Web 资产。
- Windows/macOS/iOS 会在 tag release 的 platform build plan 中显示 skipped，直到对应 secrets 配置完成。
- 如果 enabled 平台 build 失败，publish 阶段会失败，不会发布部分错误产物。

## 回滚方案

- 恢复 `.github/workflows/release.yml` 中静态 required asset 清单和 hard secret validation。
- 恢复 README 旧的 all-platform release wording。
- 如需临时阻断 tag release，可删除/禁用 `release.yml` 的 `push.tags` 触发。

## 子Agent执行轨迹

未派发子Agent；主 agent 完成需求分析、架构调整、计划、实现、验证、review 和归档。

