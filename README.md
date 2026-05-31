# MyFlowHub-ClipboardNode

Independent MyFlowHub node application for clipboard synchronization.

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

