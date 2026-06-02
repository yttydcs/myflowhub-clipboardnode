# 2026-06-02_windows-unsigned-release-preview

## 变更背景 / 目标

当前 `v0.1.0` 已完成首个 tag release，但 Windows/macOS/iOS 生产签名材料尚未配置。
此前 conditional release workflow 会在缺少 Windows code-signing secrets 时跳过 Windows，
导致首版用户没有 Windows 包可试用。

用户选择方案 2：没有 Windows 代码签名证书时，仍构建 release-mode Windows 预览包，
但资产名称、README 和包内说明必须明确标记为 unsigned preview，不能伪装成签名生产包。

## 具体变更内容

- `.github/workflows/release.yml`:
  - tag release 时 Windows capability 固定启用，不再因缺少 `WINDOWS_CODESIGN_PFX_*` secrets 跳过。
  - Windows signing secrets 存在时，继续产出签名生产资产名：
    - `myflowhub-clipboardnode-windows-release.zip`
    - `clipboardnode-windows-amd64.exe`
    - `clipboardnode-bridge-windows-amd64.exe`
  - Windows signing secrets 缺失时，产出明确标记的 unsigned preview 资产名：
    - `myflowhub-clipboardnode-windows-unsigned-preview.zip`
    - `clipboardnode-windows-amd64-unsigned-preview.exe`
    - `clipboardnode-bridge-windows-amd64-unsigned-preview.exe`
  - Windows package 内新增 `README-WINDOWS.txt` unsigned preview 警告。
  - `known_assets` 增加 unsigned preview Windows names，使 publish 阶段能上传这些实际 release assets 并生成 checksums。
- `README.md`:
  - 区分 signed Windows production package 和 unsigned Windows preview package。
  - 明确 unsigned preview 是 release-mode build，但不是签名生产 Windows release。

## Requirements impact

none

本次只调整 CI/CD 发布资产策略，不改变 ClipboardNode 同步功能、平台运行时能力、
TopicBus payload、隐私模型、协议复用边界或 UI 功能范围。

## Specs impact

none

本次不改变 runtime、bridge、platform adapter、TopicBus app event、gomobile binding、
MyFlowHub subprotocol 或任何 wire contract。

## Lessons impact

none

本次没有暴露新的可复用故障模式。关键经验已在本 change archive 中记录；
未来排查仍可从 conditional release builds 和 tag release CI archives 进入。

## Related requirements

- [../requirements/clipboard-sync.md](../requirements/clipboard-sync.md)

## Related specs

- [../specs/clipboard-sync.md](../specs/clipboard-sync.md)

## Related lessons

- [../lessons/debug-latest-ci-native-exit-flutter-material.md](../lessons/debug-latest-ci-native-exit-flutter-material.md)
- [../lessons/gomobile-mobile-bindings.md](../lessons/gomobile-mobile-bindings.md)

## 对应 plan.md 任务映射

- `WIN-1`: `.github/workflows/release.yml` Windows build capability and status handling.
- `WIN-2`: `.github/workflows/release.yml` Windows signed/unsigned output name selection and package README warning.
- `WIN-3`: `.github/workflows/release.yml` publish asset discovery plus `README.md` user-facing documentation.
- `WIN-4`: `docs/change/2026-06-02_windows-unsigned-release-preview.md` and `docs/change/README.md` archive.

## 经验 / 教训摘要

- Windows 可以先发布 unsigned preview release-mode 包，但必须在 release asset name、README 和包内 README 三层显式标记。
- GitHub Actions artifact 外层名称可以保持稳定，真正面向 Release 的文件名必须由 publish 阶段扫描出来。
- Signed production asset names 必须只在 signing secrets 完整时使用，避免未来配置证书后再迁移用户文档或脚本。
- Manual dry-run 可以验证 Windows preview build path，但 `workflow_dispatch` 必须保持不发布 Release。

## 可复用排查线索

- 症状:
  - Windows 没有出现在 Release assets 中。
  - Windows asset 名称没有 `unsigned-preview`，但仓库缺少 Windows signing secrets。
  - `Publish GitHub Release` 在 manual dry-run 中没有 skipped。
- 触发条件:
  - `WINDOWS_CODESIGN_PFX_BASE64` 或 `WINDOWS_CODESIGN_PFX_PASSWORD` 未配置。
  - Windows package name 与 `known_assets` 不一致。
  - 只查看 GitHub Actions artifact 外层名称，而没有查看 artifact 内的 release asset 文件名。
- 关键词:
  - `WINDOWS_CODESIGN_PFX_BASE64`
  - `WINDOWS_CODESIGN_PFX_PASSWORD`
  - `unsigned-preview`
  - `README-WINDOWS.txt`
  - `known_assets`
  - `windows_status`
  - `myflowhub-clipboardnode-windows-release`
- 快速检查:
  - 查看 `Prepare release metadata` 的 Windows status 是否显示 unsigned preview。
  - 下载 `myflowhub-clipboardnode-windows-release` artifact，确认内部文件名包含 `unsigned-preview`。
  - 展开 `myflowhub-clipboardnode-windows-unsigned-preview.zip`，确认根目录 `README-WINDOWS.txt` 包含 unsigned preview 警告。
  - 对 manual dry-run，确认 `Publish GitHub Release` skipped 且 `gh release view v0.0.0` 不存在。

## 关键设计决策与权衡

- 选择保留 signed Windows production asset names 不变，避免证书配置完成后破坏既有下载命名。
- 选择将 unsigned preview 标记放进 release asset 文件名，而不是只写 Release Notes；文件名是用户下载前最稳定的安全提示。
- 选择保留 Windows GitHub Actions artifact 外层名称为 `myflowhub-clipboardnode-windows-release`，因为它只是 workflow 内部 artifact 名；publish 阶段按内部文件名生成真实 Release assets。
- macOS/iOS 行为保持不变：没有配置生产签名材料时，tag release 仍跳过，避免产生未签名生产语义的 Apple 平台包。

## 测试与验证方式 / 结果

- `git diff --check`: 通过。
- YAML structural checks: 通过。
- Bash syntax checks for release workflow scripts: 通过，16 scripts。
- PowerShell parser checks for release workflow scripts: 通过，2 scripts。
- Capability simulation:
  - tag release without Windows signing secrets: `build_windows=true`。
  - tag release with Windows signing secrets: `build_windows=true`。
- Publish asset simulation:
  - unsigned Windows preview names are included in `known_assets` and selected for upload when present.
- README assertions:
  - README contains signed Windows release package wording and unsigned Windows preview package wording.
- Hosted GitHub Actions dry-run:
  - Run: <https://github.com/yttydcs/myflowhub-clipboardnode/actions/runs/26831337705>
  - Trigger: `workflow_dispatch` on `chore/windows-unsigned-release-preview`, `release_tag=v0.0.0`
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
  - `gh release view v0.0.0`: absent, confirming no accidental Release was published.
- Windows artifact inspection:
  - Artifact outer name: `myflowhub-clipboardnode-windows-release`.
  - Internal release asset files:
    - `myflowhub-clipboardnode-windows-unsigned-preview.zip`
    - `clipboardnode-windows-amd64-unsigned-preview.exe`
    - `clipboardnode-bridge-windows-amd64-unsigned-preview.exe`
  - Expanded zip root contains `README-WINDOWS.txt` with the warning that the package is an unsigned preview and not a signed production Windows release.

## 潜在影响

- 后续 tag release 在当前 Windows signing secrets 缺失状态下，会额外发布 Windows unsigned preview assets。
- 用户在 Windows 上仍会看到 Unknown Publisher 或 SmartScreen warning。
- 配置 Windows code-signing secrets 后，同一 workflow 会自动恢复 signed production Windows asset names。

## 回滚方案

- 恢复 `.github/workflows/release.yml` 中 Windows capability 对 signing secret set 的 hard gate。
- 移除 unsigned preview Windows asset names 和 package README warning。
- 恢复 README 中 Windows unsigned preview 说明。
- 如已发布包含 unsigned preview 的 tag release，可删除对应 Windows preview assets 或补发新 patch tag。

## 子Agent执行轨迹

未派发子Agent；主 agent 完成需求分析、架构调整、计划、实现、验证、review 和归档。
