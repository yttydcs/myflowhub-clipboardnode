# ClipboardNode App

Flutter shell for the cross-platform MyFlowHub clipboard application.

The app currently uses a preview bridge while the live MyFlowHub TopicBus
adapter is implemented. It already covers the core product surfaces:

- connection state;
- sync status;
- topic and local policy settings;
- manual text send;
- metadata-only activity.

Run validation from this directory with the workflow-selected Flutter SDK on
PATH:

```powershell
flutter analyze
flutter test
flutter build windows
flutter build web
```
