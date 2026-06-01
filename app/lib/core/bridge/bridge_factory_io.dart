import 'package:flutter/foundation.dart';

import 'engine_bridge.dart';
import 'live_engine_bridge.dart';
import 'mobile_engine_bridge.dart';

ClipboardEngineBridge createPlatformBridge() {
  if (LiveEngineBridge.isSupportedHost) {
    return LiveEngineBridge();
  }
  if (!kIsWeb) {
    return MobileEngineBridge();
  }
  throw StateError('unexpected web platform in IO bridge factory');
}
