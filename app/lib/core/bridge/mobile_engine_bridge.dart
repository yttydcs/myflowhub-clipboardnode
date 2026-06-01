import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';

import 'engine_bridge.dart';
import 'engine_contract.dart';
import 'preview_engine_bridge.dart';

class MobileEngineBridge implements ClipboardEngineBridge {
  MobileEngineBridge() : this._(PreviewEngineBridge());

  MobileEngineBridge._(this._fallback)
    : _state = _fallback.currentState.copyWith(previewMode: false);

  static const MethodChannel _channel = MethodChannel(
    'com.yttydcs.myflowhub.clipboardnode/engine',
  );

  final PreviewEngineBridge _fallback;
  final StreamController<ClipboardEngineState> _states =
      StreamController<ClipboardEngineState>.broadcast(sync: true);
  ClipboardEngineState _state;

  @override
  ClipboardEngineState get currentState => _state;

  @override
  Stream<ClipboardEngineState> get states => _states.stream;

  @override
  Future<void> connect() async {
    try {
      final raw = await _channel.invokeMethod<String>(
        'start',
        _state.settings.toJson(),
      );
      _applyStatus(raw);
    } on MissingPluginException {
      await _fallback.connect();
      _emit(_fallback.currentState.copyWith(previewMode: true));
    }
  }

  @override
  Future<void> disconnect() async {
    try {
      final raw = await _channel.invokeMethod<String>('stop');
      _applyStatus(raw);
    } on MissingPluginException {
      await _fallback.disconnect();
      _emit(_fallback.currentState.copyWith(previewMode: true));
    }
  }

  @override
  Future<void> updateSettings(ClipboardSettings settings) async {
    await _fallback.updateSettings(settings);
    final normalized = _fallback.currentState.settings;
    _emit(_state.copyWith(settings: normalized));
    if (!_state.connected && !_state.loggedIn) {
      return;
    }
    try {
      final raw = await _channel.invokeMethod<String>(
        'updateConfig',
        normalized.toJson(),
      );
      _applyStatus(raw);
    } on MissingPluginException {
      _emit(_fallback.currentState.copyWith(previewMode: true));
    }
  }

  @override
  Future<void> sendText(String text) async {
    try {
      final raw = await _channel.invokeMethod<String>('sendText', {
        'text': text,
      });
      _applyDecision(raw);
    } on MissingPluginException {
      await _fallback.sendText(text);
      _emit(_fallback.currentState.copyWith(previewMode: true));
    }
  }

  @override
  Future<void> readClipboard() async {
    try {
      final raw = await _channel.invokeMethod<String>('readClipboard');
      _applyDecision(raw);
    } on MissingPluginException {
      await _fallback.readClipboard();
      _emit(_fallback.currentState.copyWith(previewMode: true));
    }
  }

  @override
  Future<void> applyPending(String eventId) async {
    try {
      final raw = await _channel.invokeMethod<String>('applyEvent', {
        'event_id': eventId,
      });
      _applyDecision(raw);
    } on MissingPluginException {
      await _fallback.applyPending(eventId);
      _emit(_fallback.currentState.copyWith(previewMode: true));
    }
  }

  @override
  Future<void> clearRecent() async {
    _emit(
      _state.copyWith(
        activities: const [],
        clearPendingEvent: true,
        clearTransferStatus: true,
      ),
    );
  }

  @override
  Future<void> dispose() async {
    await _fallback.dispose();
    await _states.close();
  }

  void _applyStatus(String? raw) {
    if (raw == null || raw.trim().isEmpty) {
      return;
    }
    final decoded = jsonDecode(raw);
    if (decoded is! Map<String, Object?>) {
      return;
    }
    final running = decoded['running'] as bool?;
    final connected = decoded['Connected'] as bool? ?? running ?? false;
    final loggedIn = decoded['LoggedIn'] as bool? ?? false;
    final runtime = decoded['Runtime'];
    final runtimeData = runtime is Map<String, Object?>
        ? runtime
        : <String, Object?>{};
    final nodeId = decoded['NodeID'] as int?;
    final endpoint = decoded['ParentEndpoint'] as String?;
    _emit(
      _state.copyWith(
        connected: connected,
        loggedIn: loggedIn,
        previewMode: false,
        hubEndpoint: endpoint ?? _state.hubEndpoint,
        authStage: loggedIn ? '已认证' : '等待认证',
        nodeId: nodeId,
        settings: _state.settings.copyWith(
          enabled: runtimeData['Enabled'] as bool? ?? _state.settings.enabled,
          topic: runtimeData['Topic'] as String? ?? _state.settings.topic,
          autoWatch:
              runtimeData['AutoWatch'] as bool? ?? _state.settings.autoWatch,
          autoApply:
              runtimeData['AutoApply'] as bool? ?? _state.settings.autoApply,
        ),
        lastError:
            decoded['last_error'] as String? ??
            decoded['LastError'] as String? ??
            '',
      ),
    );
  }

  void _applyDecision(String? raw) {
    if (raw == null || raw.trim().isEmpty) {
      return;
    }
    final decoded = jsonDecode(raw);
    if (decoded is! Map<String, Object?>) {
      return;
    }
    final now = DateTime.now();
    final action = decoded['Action'] as String? ?? 'published';
    final kind = switch (action) {
      'remote_applied' => ActivityKind.applied,
      'remote_pending' ||
      'transfer_pending' ||
      'transfer_published' => ActivityKind.pending,
      'validation_failed' ||
      'transport_failed' ||
      'clipboard_write_failed' => ActivityKind.error,
      _ => ActivityKind.published,
    };
    final activity = ClipboardActivity(
      id:
          decoded['EventID'] as String? ??
          'mobile-${now.microsecondsSinceEpoch}',
      kind: kind,
      title: action,
      detail: 'TopicBus',
      deviceLabel: _state.settings.deviceLabel,
      byteSize: (decoded['Size'] as num?)?.toInt() ?? 0,
      hashPrefix: decoded['HashPrefix'] as String? ?? '',
      timestamp: now,
    );
    _emit(
      _state.copyWith(
        activities: [
          activity,
          ..._state.activities,
        ].take(20).toList(growable: false),
      ),
    );
  }

  void _emit(ClipboardEngineState next) {
    _state = next;
    if (!_states.isClosed) {
      _states.add(next);
    }
  }
}
