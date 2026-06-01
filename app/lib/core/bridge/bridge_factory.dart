import 'engine_bridge.dart';

import 'bridge_factory_stub.dart'
    if (dart.library.io) 'bridge_factory_io.dart'
    if (dart.library.html) 'bridge_factory_web.dart';

ClipboardEngineBridge createDefaultBridge() => createPlatformBridge();
