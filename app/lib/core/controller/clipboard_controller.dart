import 'dart:async';

import 'package:flutter/foundation.dart';

import '../bridge/engine_bridge.dart';
import '../bridge/engine_contract.dart';

class ClipboardController extends ChangeNotifier {
  ClipboardController(this._bridge) : _state = _bridge.currentState {
    _subscription = _bridge.states.listen((next) {
      _state = next;
      notifyListeners();
    });
  }

  final ClipboardEngineBridge _bridge;
  late final StreamSubscription<ClipboardEngineState> _subscription;
  ClipboardEngineState _state;

  ClipboardEngineState get state => _state;

  Future<void> connect() => _run(_bridge.connect);

  Future<void> disconnect() => _run(_bridge.disconnect);

  Future<void> updateSettings(ClipboardSettings settings) {
    return _run(() => _bridge.updateSettings(settings));
  }

  Future<void> setSyncEnabled(bool enabled) {
    return updateSettings(_state.settings.copyWith(enabled: enabled));
  }

  Future<void> sendText(String text) => _run(() => _bridge.sendText(text));

  Future<void> clearRecent() => _run(_bridge.clearRecent);

  Future<void> _run(Future<void> Function() action) async {
    try {
      await action();
    } on Object catch (error) {
      _state = _state.copyWith(lastError: _messageFor(error), busy: false);
      notifyListeners();
    }
  }

  String _messageFor(Object error) {
    if (error is StateError) {
      return error.message.toString();
    }
    return '操作失败：$error';
  }

  @override
  void dispose() {
    _subscription.cancel();
    _bridge.dispose();
    super.dispose();
  }
}
