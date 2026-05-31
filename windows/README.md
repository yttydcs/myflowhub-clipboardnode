# Windows Host Skeleton

The first Windows host skeleton provides:

- a Win32 `CF_UNICODETEXT` clipboard adapter without third-party dependencies;
- a polling watcher based on `GetClipboardSequenceNumber`;
- bounded clipboard reads and explicit errors;
- a headless command that loads the disabled-by-default JSON config.

The host intentionally fails when `enabled=true` until the MyFlowHub SDK login and TopicBus transport adapter are wired in a follow-up task. The shared runtime and fake TopicBus tests establish the clipboard sync contract without claiming that a live hub integration already exists.
