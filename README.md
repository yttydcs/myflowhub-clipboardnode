# MyFlowHub-ClipboardNode

Independent MyFlowHub node application for clipboard synchronization.

## Current Status

This repository contains the MVP headless core and Windows host skeleton:

- `core/runtime` validates `clipboard.text.v1` payloads, publishes local text changes through a TopicBus interface, applies remote text events, deduplicates event IDs, and suppresses clipboard write loops.
- `core/configstore` persists only non-sensitive settings. Clipboard text and event bodies are never written to config.
- `windows` provides a Win32 `CF_UNICODETEXT` adapter with bounded reads and a polling watcher.
- `cmd/clipboardnode` loads config and verifies the Windows adapter, but intentionally refuses `enabled=true` until the live MyFlowHub SDK TopicBus transport is wired.

Sync is disabled by default. The first phase is text-only, online-only, and best-effort through TopicBus.

## Scope

- Runs as its own node instead of being embedded in MyFlowHub-Win or Server.
- Uses existing MyFlowHub protocols, with the first phase expected to use TopicBus for small text clipboard events.
- Keeps platform clipboard adapters in the host layers and shared sync logic in `core/`.

## Repository Shape

```text
core/          shared runtime, config, topic event model, dedupe logic
windows/       Windows host and clipboard adapter
android/       Android host and clipboard adapter
nodemobile/    Android gomobile bridge
docs/          requirements, specs, plans, changes, lessons
scripts/       build and maintenance scripts
```

Implementation changes should be made from a dedicated worktree under `D:/project/MyFlowHub3/worktrees/`.

## Validation

This repo lives under a parent workspace that may contain a `go.work`; validate it as an independent module:

```powershell
$env:GOWORK = "off"
go test ./... -count=1
go build -o build/clipboardnode.exe ./cmd/clipboardnode
git diff --check
```

Or run:

```powershell
.\scripts\validate.ps1
```

