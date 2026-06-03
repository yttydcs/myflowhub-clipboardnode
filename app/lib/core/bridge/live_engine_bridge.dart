import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';

import 'engine_bridge.dart';
import 'engine_contract.dart';

class LiveEngineBridge implements ClipboardEngineBridge {
  LiveEngineBridge({String? executable})
    : _executable = executable,
      _state = ClipboardEngineState.initial(
        _capability(),
      ).copyWith(previewMode: false);

  static bool get isSupportedHost {
    if (kIsWeb) {
      return false;
    }
    return Platform.isWindows || Platform.isLinux || Platform.isMacOS;
  }

  static PlatformCapability _capability() {
    if (Platform.isWindows || Platform.isLinux || Platform.isMacOS) {
      return const PlatformCapability(
        platformLabel: 'Desktop',
        automaticWatch: true,
        manualSend: true,
        autoApply: true,
        shareSheet: false,
        notes: ['桌面端使用本地 Go engine', '自动监听和自动应用均由本机策略控制'],
      );
    }
    return const PlatformCapability(
      platformLabel: 'Unsupported',
      automaticWatch: false,
      manualSend: false,
      autoApply: false,
      shareSheet: false,
      notes: ['当前平台尚未接入 live engine'],
    );
  }

  final String? _executable;
  final StreamController<ClipboardEngineState> _states =
      StreamController<ClipboardEngineState>.broadcast(sync: true);
  final Map<String, Completer<void>> _pending = {};
  ClipboardEngineState _state;
  Process? _process;
  StreamSubscription<String>? _stdoutSub;
  StreamSubscription<List<int>>? _stderrSub;
  int _seq = 0;

  @override
  ClipboardEngineState get currentState => _state;

  @override
  Stream<ClipboardEngineState> get states => _states.stream;

  @override
  Future<void> connect() async {
    _emit(_state.copyWith(busy: true, authStage: '连接父节点', lastError: ''));
    await _ensureProcess();
    await _send(EngineActions.setConfig, _state.settings.toJson());
    await _send(EngineActions.connect);
  }

  @override
  Future<void> disconnect() async {
    if (_process == null) {
      _emit(
        _state.copyWith(
          connected: false,
          loggedIn: false,
          busy: false,
          clearNodeId: true,
          authStage: '登录态已清理',
        ),
      );
      return;
    }
    await _send(EngineActions.shutdown);
    await _closeProcess();
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

  @override
  Future<void> updateSettings(ClipboardSettings settings) async {
    final normalizedParentEndpoint = settings.parentEndpoint.trim();
    final normalizedTopic = settings.topic.trim();
    if (normalizedParentEndpoint.isEmpty) {
      throw StateError('父节点地址不能为空');
    }
    if (settings.enabled && normalizedTopic.isEmpty) {
      throw StateError('启用同步时必须填写 Topic');
    }
    if (settings.maxInlineBytes <= 0) {
      throw StateError('内联文本上限必须大于 0');
    }
    final next = settings.copyWith(
      parentEndpoint: normalizedParentEndpoint,
      topic: normalizedTopic,
    );
    _emit(
      _state.copyWith(settings: next, hubEndpoint: normalizedParentEndpoint),
    );
    if (_process != null) {
      await _send(EngineActions.setConfig, next.toJson());
    }
  }

  @override
  Future<void> sendText(String text) async {
    final normalizedText = text.trim();
    if (!_state.connected || !_state.loggedIn) {
      throw StateError('请先连接并登录 Hub');
    }
    if (!_state.settings.enabled) {
      throw StateError('请先启用剪贴板同步');
    }
    if (normalizedText.isEmpty) {
      throw StateError('待发送文本不能为空');
    }
    await _send(EngineActions.sendText, {'text': text});
  }

  @override
  Future<void> readClipboard() async {
    if (!_state.connected || !_state.loggedIn) {
      throw StateError('请先连接并登录 Hub');
    }
    if (!_state.settings.enabled) {
      throw StateError('请先启用剪贴板同步');
    }
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
    if (_process != null) {
      await _send(EngineActions.clearRecent);
    }
  }

  @override
  Future<void> dispose() async {
    await _closeProcess();
    for (final completer in _pending.values) {
      if (!completer.isCompleted) {
        completer.completeError(StateError('engine disposed'));
      }
    }
    _pending.clear();
    await _states.close();
  }

  Future<void> _ensureProcess() async {
    if (_process != null) {
      return;
    }
    try {
      _process = await Process.start(_resolveExecutable(), const []);
    } on Object catch (error) {
      _emit(_state.copyWith(busy: false, lastError: '启动本地 engine 失败：$error'));
      rethrow;
    }
    _stdoutSub = _process!.stdout
        .transform(utf8.decoder)
        .transform(const LineSplitter())
        .listen(_handleLine, onError: _handleProcessError);
    _stderrSub = _process!.stderr.listen((chunk) {
      final message = utf8.decode(chunk, allowMalformed: true).trim();
      if (message.isNotEmpty) {
        _emit(_state.copyWith(lastError: message));
      }
    });
    unawaited(
      _process!.exitCode.then((code) {
        _process = null;
        if (code != 0) {
          _emit(_state.copyWith(busy: false, lastError: '本地 engine 已退出：$code'));
        }
      }),
    );
  }

  String _resolveExecutable() {
    if (_executable != null && _executable.trim().isNotEmpty) {
      return _executable;
    }
    final name = Platform.isWindows
        ? 'clipboardnode-bridge.exe'
        : 'clipboardnode-bridge';
    final script = Platform.resolvedExecutable;
    final candidates = <String>[
      '${File(script).parent.path}${Platform.pathSeparator}$name',
      '${Directory.current.path}${Platform.pathSeparator}$name',
      name,
    ];
    if (Platform.isMacOS) {
      candidates.insert(
        0,
        '${File(script).parent.path}${Platform.pathSeparator}$name',
      );
    }
    for (final path in candidates) {
      if (path == name || File(path).existsSync()) {
        return path;
      }
    }
    return name;
  }

  Future<void> _send(String action, [Map<String, Object?> data = const {}]) {
    final process = _process;
    if (process == null) {
      return Future<void>.error(StateError('engine process is not running'));
    }
    final id = 'cmd-${++_seq}';
    final completer = Completer<void>();
    _pending[id] = completer;
    process.stdin.writeln(
      EngineCommand(id: id, action: action, data: data).encode(),
    );
    return completer.future.timeout(
      const Duration(seconds: 10),
      onTimeout: () {
        _pending.remove(id);
        throw TimeoutException('engine command timed out: $action');
      },
    );
  }

  void _handleLine(String line) {
    if (line.trim().isEmpty) {
      return;
    }
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
      deviceId:
          data['device_id'] as String? ??
          data['device_label'] as String? ??
          _state.settings.deviceId,
      displayName:
          data['display_name'] as String? ??
          data['device_label'] as String? ??
          _state.settings.displayName,
      deviceLabel:
          data['display_name'] as String? ??
          data['device_label'] as String? ??
          _state.settings.deviceLabel,
      autoWatch: data['auto_watch'] as bool? ?? _state.settings.autoWatch,
      autoApply: data['auto_apply'] as bool? ?? _state.settings.autoApply,
      transferProvider:
          data['transfer_provider'] as String? ??
          _state.settings.transferProvider,
      transferRef:
          data['transfer_ref'] as String? ?? _state.settings.transferRef,
    );
    final loggedIn = data['logged_in'] as bool? ?? false;
    final pendingEventId = data['pending_event_id'] as String? ?? '';
    final pending = pendingEventId.isEmpty
        ? null
        : PendingClipboardEvent(
            eventId: pendingEventId,
            byteSize: (data['pending_size'] as num?)?.toInt() ?? 0,
            hashPrefix: data['pending_hash_prefix'] as String? ?? '',
          );
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
    final id =
        data['id'] as String? ??
        'activity-${DateTime.now().microsecondsSinceEpoch}';
    final kind = switch (data['kind'] as String? ?? 'ignored') {
      'published' => ActivityKind.published,
      'applied' => ActivityKind.applied,
      'pending' => ActivityKind.pending,
      'error' => ActivityKind.error,
      _ => ActivityKind.ignored,
    };
    final activity = ClipboardActivity(
      id: id,
      kind: kind,
      title: data['title'] as String? ?? kind.name,
      detail: data['detail'] as String? ?? 'TopicBus',
      deviceLabel:
          data['device_label'] as String? ?? _state.settings.displayName,
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
        lastError: '',
      ),
    );
  }

  void _applyTransfer(Map<String, Object?> data) {
    final id = data['id'] as String? ?? '';
    if (id.isEmpty) {
      return;
    }
    final transfer = TransferStatus(
      id: id,
      state: data['state'] as String? ?? 'pending',
      byteSize: (data['byte_size'] as num?)?.toInt() ?? 0,
      hashPrefix: data['hash_prefix'] as String? ?? '',
      detail: data['detail'] as String? ?? 'clipboard.transfer.v1',
    );
    _emit(_state.copyWith(transferStatus: transfer, lastError: ''));
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

  void _handleProcessError(Object error) {
    _emit(
      _state.copyWith(busy: false, lastError: 'engine stream failed: $error'),
    );
  }

  Future<void> _closeProcess() async {
    final process = _process;
    _process = null;
    await _stdoutSub?.cancel();
    await _stderrSub?.cancel();
    _stdoutSub = null;
    _stderrSub = null;
    process?.kill();
  }

  void _emit(ClipboardEngineState next) {
    _state = next;
    if (!_states.isClosed) {
      _states.add(next);
    }
  }
}
