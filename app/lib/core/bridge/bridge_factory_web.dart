import 'engine_bridge.dart';
import 'preview_engine_bridge.dart';
import 'web_engine_bridge.dart';

ClipboardEngineBridge createPlatformBridge() {
  if (WebEngineBridge.isConfigured) {
    return WebEngineBridge();
  }
  return PreviewEngineBridge();
}
