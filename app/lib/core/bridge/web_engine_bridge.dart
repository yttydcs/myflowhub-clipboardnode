// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:async';
import 'dart:convert';
import 'dart:html' as html;

import 'engine_bridge.dart';
import 'engine_contract.dart';

class WebEngineBridge implements ClipboardEngineBridge {
  WebEngineBridge({
    String endpoint = const String.fromEnvironment('CLIPBOARDNODE_WEB_BRIDGE'),
    String token = const String.fromEnvironment('CLIPBOARDNODE_WEB_TOKEN'),
  }) : _endpoint = _normalizeEndpoint(endpoint),
       _token = token.trim(),
       _state = ClipboardEngineState.initial(_capability()).copyWith(
         previewMode: false,
         lastError: token.trim().isEmpty
             ? 'Web live bridge requires token'
             : '',
       );

  static bool get isConfigured {
    return const String.fromEnvironment(
      'CLIPBOARDNODE_WEB_BRIDGE',
    ).trim().isNotEmpty;
  }

  static String _normalizeEndpoint(String endpoint) {
    return endpoint.trim().replaceAll(RegExp(r'/+$'), '');
  }

  static PlatformCapability _capability() {
    return const PlatformCapability(
      platformLabel: 'Web local bridge',
      automaticWatch: false,
      manualSend: true,
      autoApply: false,
      shareSheet: false,
      notes: ['Web 通过本机 localhost bridge 连接 Go engine', '浏览器剪贴板读取需要用户手势'],
    );
  }

  final String _endpoint;
  final String _token;
  final StreamController<ClipboardEngineState> _states =
      StreamController<ClipboardEngineState>.broadcast(sync: true);
  final Map<String, Completer<void>> _pending = {};
  ClipboardEngineState _state;
  html.EventSource? _events;
  int _seq = 0;

  @override
  ClipboardEngineState get currentState => _state;

  @override
  Stream<ClipboardEngineState> get states => _states.stream;

  @override
  Future<void> connect() async {
    _validateConfigured();
    _emit(
      _state.copyWith(busy: true, authStage: '连接本机 Web bridge', lastError: ''),
    );
    _ensureEvents();
    await _send(EngineActions.setConfig, _state.settings.toJson());
    await _send(EngineActions.connect);
  }

  @override
  Future<void> disconnect() async {
    if (_events == null) {
      _emitDisconnected();
      return;
    }
    await _send(EngineActions.shutdown);
    _events?.close();
    _events = null;
    _emitDisconnected();
  }

  @override
  Future<void> updateSettings(ClipboardSettings settings) async {
    final endpoint = settings.parentEndpoint.trim();
    final topic = settings.topic.trim();
    if (endpoint.isEmpty) {
      throw StateError('父节点地址不能为空');
    }
    if (settings.enabled && topic.isEmpty) {
      throw StateError('启用同步时必须填写 Topic');
    }
    if (settings.maxInlineBytes <= 0) {
      throw StateError('内联文本上限必须大于 0');
    }
    final next = settings.copyWith(parentEndpoint: endpoint, topic: topic);
    _emit(_state.copyWith(settings: next, hubEndpoint: endpoint));
    if (_events != null) {
      await _send(EngineActions.setConfig, next.toJson());
    }
  }

  @override
  Future<void> sendText(String text) async {
    if (text.trim().isEmpty) {
      throw StateError('待发送文本不能为空');
    }
    await _send(EngineActions.sendText, {'text': text});
  }

  @override
  Future<void> readClipboard() async {
    await _send(EngineActions.readClipboard);
  }

  @override
  Future<void> applyPending(String eventId) async {
    if (eventId.trim().isEmpty) {
      throw StateError('待应用事件 ID 不能为空');
    }
    await _send(EngineActions.applyEvent, {'event_id': eventId.trim()});
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
    if (_events != null) {
      await _send(EngineActions.clearRecent);
    }
  }

  @override
  Future<void> dispose() async {
    _events?.close();
    _events = null;
    for (final completer in _pending.values) {
      if (!completer.isCompleted) {
        completer.completeError(StateError('web bridge disposed'));
      }
    }
    _pending.clear();
    await _states.close();
  }

  void _validateConfigured() {
    if (_endpoint.isEmpty) {
      throw StateError('Web live bridge endpoint is not configured');
    }
    if (!_endpoint.startsWith('http://127.0.0.1') &&
        !_endpoint.startsWith('http://localhost') &&
        !_endpoint.startsWith('http://[::1]')) {
      throw StateError('Web live bridge must use localhost');
    }
    if (_token.isEmpty) {
      throw StateError('Web live bridge token is required');
    }
  }

  void _ensureEvents() {
    if (_events != null) {
      return;
    }
    final url = '$_endpoint/events?token=${Uri.encodeQueryComponent(_token)}';
    final source = html.EventSource(url);
    source.onMessage.listen((event) {
      final data = event.data;
      if (data is String) {
        _handleLine(data);
      }
    });
    source.onError.listen((_) {
      _emit(
        _state.copyWith(
          busy: false,
          lastError: 'Web bridge event stream failed',
        ),
      );
    });
    _events = source;
  }

  Future<void> _send(String action, [Map<String, Object?> data = const {}]) {
    final id = 'web-${++_seq}';
    final completer = Completer<void>();
    _pending[id] = completer;
    final command = EngineCommand(id: id, action: action, data: data).encode();
    html.HttpRequest.request(
          '$_endpoint/command',
          method: 'POST',
          requestHeaders: {
            'Content-Type': 'application/json',
            'X-ClipboardNode-Token': _token,
          },
          sendData: command,
        )
        .then((response) async {
          final raw = response.responseText;
          if (raw != null && raw.trim().isNotEmpty) {
            final decoded = jsonDecode(raw);
            if (decoded is Map<String, Object?>) {
              final status = decoded['status'];
              if (status is Map<String, Object?>) {
                _applyStatus(status);
              }
              if (decoded['ok'] == false) {
                final message =
                    decoded['error'] as String? ?? 'web bridge command failed';
                _complete(id, StateError(message));
                return;
              }
            }
          }
          await _fetchStatus();
          _complete(id, null);
        })
        .catchError((Object error) {
          _complete(id, error);
        });
    return completer.future.timeout(
      const Duration(seconds: 10),
      onTimeout: () {
        _pending.remove(id);
        throw TimeoutException('web bridge command timed out: $action');
      },
    );
  }

  Future<void> _fetchStatus() async {
    final response = await html.HttpRequest.request(
      '$_endpoint/status',
      method: 'GET',
      requestHeaders: {'X-ClipboardNode-Token': _token},
    );
    final text = response.responseText;
    if (text == null || text.trim().isEmpty) {
      return;
    }
    final decoded = jsonDecode(text);
    if (decoded is Map<String, Object?>) {
      _applyStatus(decoded);
    }
  }

  void _handleLine(String line) {
    final decoded = jsonDecode(line);
    if (decoded is! Map<String, Object?>) {
      return;
    }
    final id = decoded['id'] as String?;
    final ok = decoded['ok'] != false;
    final error = decoded['error'] as String? ?? '';
    if (!ok && error.isNotEmpty) {
      _complete(id, StateError(error));
      _emit(_state.copyWith(busy: false, lastError: error));
      return;
    }
    final event = decoded['name'] as String? ?? '';
    final rawData = decoded['data'];
    final data = rawData is Map<String, Object?>
        ? rawData
        : <String, Object?>{};
    switch (event) {
      case EngineEvents.statusChanged:
        _applyStatus(data);
      case EngineEvents.activityUpdated:
        _applyActivity(data);
      case EngineEvents.transferUpdated:
        _applyTransfer(data);
      case EngineEvents.error:
        _emit(_state.copyWith(busy: false, lastError: error));
    }
    _complete(id, null);
  }

  void _applyStatus(Map<String, Object?> data) {
    final settings = _state.settings.copyWith(
      enabled: data['enabled'] as bool? ?? _state.settings.enabled,
      parentEndpoint:
          data['parent_endpoint'] as String? ?? _state.settings.parentEndpoint,
      topic: data['topic'] as String? ?? _state.settings.topic,
      deviceLabel:
          data['device_label'] as String? ?? _state.settings.deviceLabel,
      autoWatch: data['auto_watch'] as bool? ?? _state.settings.autoWatch,
      autoApply: data['auto_apply'] as bool? ?? _state.settings.autoApply,
      transferProvider:
          data['transfer_provider'] as String? ??
          _state.settings.transferProvider,
      transferRef:
          data['transfer_ref'] as String? ?? _state.settings.transferRef,
    );
    final pendingEventId = data['pending_event_id'] as String? ?? '';
    final pending = pendingEventId.isEmpty
        ? null
        : PendingClipboardEvent(
            eventId: pendingEventId,
            byteSize: (data['pending_size'] as num?)?.toInt() ?? 0,
            hashPrefix: data['pending_hash_prefix'] as String? ?? '',
          );
    final loggedIn = data['logged_in'] as bool? ?? false;
    _emit(
      _state.copyWith(
        connected: data['connected'] as bool? ?? false,
        loggedIn: loggedIn,
        busy: false,
        previewMode: false,
        hubEndpoint: settings.parentEndpoint,
        authStage: loggedIn ? '已认证' : '等待认证',
        nodeId: (data['node_id'] as num?)?.toInt(),
        settings: settings,
        pendingEvent: pending,
        clearPendingEvent: pending == null,
        lastError: data['last_error'] as String? ?? '',
      ),
    );
  }

  void _applyActivity(Map<String, Object?> data) {
    final kind = switch (data['kind'] as String? ?? 'ignored') {
      'published' => ActivityKind.published,
      'applied' => ActivityKind.applied,
      'pending' => ActivityKind.pending,
      'error' => ActivityKind.error,
      _ => ActivityKind.ignored,
    };
    final activity = ClipboardActivity(
      id:
          data['id'] as String? ??
          'web-${DateTime.now().microsecondsSinceEpoch}',
      kind: kind,
      title: data['title'] as String? ?? kind.name,
      detail: data['detail'] as String? ?? 'TopicBus',
      deviceLabel:
          data['device_label'] as String? ?? _state.settings.deviceLabel,
      byteSize: (data['byte_size'] as num?)?.toInt() ?? 0,
      hashPrefix: data['hash_prefix'] as String? ?? '',
      timestamp: DateTime.now(),
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

  void _applyTransfer(Map<String, Object?> data) {
    final id = data['id'] as String? ?? '';
    if (id.isEmpty) {
      return;
    }
    _emit(
      _state.copyWith(
        transferStatus: TransferStatus(
          id: id,
          state: data['state'] as String? ?? 'pending',
          byteSize: (data['byte_size'] as num?)?.toInt() ?? 0,
          hashPrefix: data['hash_prefix'] as String? ?? '',
          detail: data['detail'] as String? ?? 'clipboard.transfer.v1',
        ),
      ),
    );
  }

  void _complete(String? id, Object? error) {
    if (id == null) {
      return;
    }
    final completer = _pending.remove(id);
    if (completer == null || completer.isCompleted) {
      return;
    }
    if (error == null) {
      completer.complete();
    } else {
      completer.completeError(error);
    }
  }

  void _emitDisconnected() {
    _emit(
      _state.copyWith(
        connected: false,
        loggedIn: false,
        busy: false,
        clearNodeId: true,
        authStage: '登录态已清理',
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
