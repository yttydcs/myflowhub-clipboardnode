import 'package:flutter/material.dart';

import '../core/bridge/preview_engine_bridge.dart';
import '../core/controller/clipboard_controller.dart';
import '../features/shell/clipboard_shell.dart';
import 'theme/app_theme.dart';

class ClipboardNodeApp extends StatefulWidget {
  const ClipboardNodeApp({super.key});

  @override
  State<ClipboardNodeApp> createState() => _ClipboardNodeAppState();
}

class _ClipboardNodeAppState extends State<ClipboardNodeApp> {
  late final ClipboardController _controller;

  @override
  void initState() {
    super.initState();
    _controller = ClipboardController(PreviewEngineBridge());
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'ClipboardNode',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.light(),
      home: ClipboardShell(controller: _controller),
    );
  }
}
