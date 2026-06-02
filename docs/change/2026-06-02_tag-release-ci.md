# 2026-06-02_tag-release-ci

## 变更背景 / 目标

现有 CI/CD 已经能在 `master` push 后构建全平台 debug 资产并刷新 `debug-latest` prerelease。为了支持按版本分发，本次新增 tag-driven GitHub Release：推送版本 tag 时自动构建并发布对应 Release，同时保留 `debug-latest` 预览通道。

## 具体变更内容

- `.github/workflows/debug-latest.yml` 的 `push` trigger 从忽略所有 tag 调整为：
  - `branches: master`
  - `tags: v*`
- 保留 `publish-debug-latest` 条件：只在 `push` 到 `refs/heads/master` 时刷新 `debug-latest`。
- 新增 `publish-tag-release`：
  - 只在 `push` 到 `refs/tags/v*` 时运行；
  - 依赖 Go CLI、Windows、Linux、macOS、Android、iOS simulator、Web 全部 build job；
  - 下载并校验与 `debug-latest` 相同的 8 个资产；
  - 根据 `GITHUB_REF_NAME` 创建或更新对应 GitHub Release；
  - 对已有 release 显式 patch 为 `draft=false`、`prerelease=false`、`make_latest=true`；
  - 使用 `gh release upload --clobber` 覆盖上传本次构建资产。
- README 将 `Debug Preview` 扩展为 `Release Channels`，说明：
  - `master` push 刷新 `debug-latest`；
  - 推送 `v*` tag 发布对应 GitHub Release；
  - 当前资产仍是 unsigned preview/debug artifacts，不是生产签名分发包。

## Requirements impact

none

本次只改变 CI/CD 发布自动化，不改变 ClipboardNode 产品需求、同步行为、隐私模型或 MyFlowHub 协议复用边界。

## Specs impact

none

本次不改变 runtime、bridge、TopicBus payload、platform adapter 或 MyFlowHub 子协议技术契约。

## Lessons impact

none

未出现新的可复用故障模式；继续沿用既有 `debug-latest` CI native exit lesson。

## Related requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Related lessons

- [../lessons/debug-latest-ci-native-exit-flutter-material.md](../lessons/debug-latest-ci-native-exit-flutter-material.md)

## 对应 plan.md 任务映射

- `CI-REL-1`: `.github/workflows/debug-latest.yml`
- `CI-REL-2`: `README.md`
- `CI-REL-3`: workflow YAML parse、release gate assertions、`git diff --check`、Go tests
- `CI-REL-4`: `docs/change/2026-06-02_tag-release-ci.md`、`docs/change/README.md`

## 经验 / 教训摘要

- `debug-latest` 是 workflow 自己移动的 tag，不能把所有 tag push 都当成正式发布触发条件。
- `v*` tag filter 和 `startsWith(github.ref, 'refs/tags/v')` job condition 双重限制可以避免 movable debug tag 触发稳定 release。
- 同一个 all-platform build workflow 可以同时服务 debug prerelease 和 version release，减少构建脚本漂移。
- 如果已有 tag release 曾被标记为 prerelease，重跑发布路径需要显式清理 release 状态，而不只覆盖 notes/assets。
- 当前仓库还没有生产签名、公证、商店分发或 installer 流程，因此 tag release 先发布已验证的 unsigned debug/preview 资产。

## 可复用排查线索

- 症状: 推送 `v1.2.3` 后没有生成 release。
- 触发条件: tag 不匹配 `v*`、build job 失败、或 `publish-tag-release` 被条件跳过。
- 关键词: `publish-tag-release`, `refs/tags/v`, `GITHUB_REF_NAME`, `--verify-tag`, `make_latest`, `debug-latest`.
- 快速检查:
  - 查看 Actions run 是否由 `refs/tags/v...` 触发。
  - 查看 `publish-tag-release` 是否等待了全部 build job。
  - 在 publish 日志中搜索 `Missing required release asset`。
  - 确认 tag release 不是被 `debug-latest` 的 tag 移动触发。

## 关键设计决策与权衡

- 保留单一 workflow，复用现有 artifact 生产路径，避免单独 `release.yml` 和 `debug-latest.yml` 资产清单漂移。
- tag 范围采用 `v*`，这是 GitHub Actions filter 支持下的保守默认；若后续需要严格 SemVer，可再细化命名约束或增加脚本校验。
- 版本 release 标记为 stable GitHub Release，但资产说明保留 unsigned/debug caveat，避免误导为生产签名包。
- `debug-latest` 继续 prerelease，`v*` release 走独立 concurrency group，避免互相取消。

## 测试与验证方式 / 结果

- Python/PyYAML 解析 `.github/workflows/debug-latest.yml`: 通过，确认 `push.branches=[master]`、`push.tags=[v*]`、新增 `publish-tag-release` job。
- Workflow condition assertions: 通过。
  - `publish-debug-latest` 仍限制为 `refs/heads/master`。
  - `publish-tag-release` 限制为 `refs/tags/v`。
  - release job 使用 `draft=false`、`prerelease=false`、`make_latest=true` 和 `--verify-tag --latest`。
- Publish dependency assertion: 通过，`publish-tag-release.needs` 与 `publish-debug-latest.needs` 一致。
- `git diff --check`: 通过，只有 Windows CRLF 提示。
- `$env:GOWORK='off'; go test ./... -count=1`: 通过。

未在本地证明：

- 真实 tag push 后的 GitHub hosted run 和 release 创建需要合并后推送 `v*` tag 才能验证。
- 本次未运行 Flutter build，因为没有改 Flutter/runtime 代码，CI tag run 会复用全平台 hosted build 作为最终 proof。

## 潜在影响

- 推送 `v*` tag 会触发完整全平台构建和一个稳定 GitHub Release 发布 job。
- `debug-latest` 可继续由 `master` push 刷新，不会因为 workflow 移动 tag 而触发 version release。
- 版本 release 当前仍是 unsigned debug/preview 资产，下载者不能把它视作已签名生产分发。

## 回滚方案

- 将 `.github/workflows/debug-latest.yml` trigger 恢复为 `tags-ignore: "**"`。
- 删除 `publish-tag-release` job。
- README 恢复为只描述 `debug-latest`。
- 删除本 change archive 或在后续 archive 中记录回滚。
- 如果已经创建了错误的版本 release，可在 GitHub Release 中删除该 release/tag 或重新推送正确 tag 后重跑。

## 子Agent执行轨迹

未派发子Agent；主 agent 完成需求/架构分析、计划、实现、验证、review 和归档。
