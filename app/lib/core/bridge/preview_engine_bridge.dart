import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';

import 'engine_bridge.dart';
import 'engine_contract.dart';

class PreviewEngineBridge implements ClipboardEngineBridge {
  PreviewEngineBridge() : _state = ClipboardEngineState.initial(_capability());

  static PlatformCapability _capability() {
    if (kIsWeb) {
      return const PlatformCapability(
        platformLabel: 'Web preview',
        automaticWatch: false,
        manualSend: true,
        autoApply: false,
        shareSheet: false,
        notes: ['Web 预览仅用于界面验证', '实际剪贴板能力由设备适配器提供'],
      );
    }
    switch (defaultTargetPlatform) {
      case TargetPlatform.android:
      case TargetPlatform.iOS:
        return const PlatformCapability(
          platformLabel: 'Mobile',
          automaticWatch: false,
          manualSend: true,
          autoApply: false,
          shareSheet: true,
          notes: ['移动端优先使用手动发送与系统分享入口', '前台接收后由用户确认应用'],
        );
      case TargetPlatform.windows:
      case TargetPlatform.linux:
      case TargetPlatform.macOS:
        return const PlatformCapability(
          platformLabel: 'Desktop',
          automaticWatch: true,
          manualSend: true,
          autoApply: true,
          shareSheet: false,
          notes: ['桌面端可开启自动监听', '自动应用默认关闭'],
        );
      case TargetPlatform.fuchsia:
        return const PlatformCapability(
          platformLabel: 'Unsupported',
          automaticWatch: false,
          manualSend: false,
          autoApply: false,
          shareSheet: false,
          notes: ['当前平台尚未提供剪贴板适配器'],
        );
    }
  }

  final StreamController<ClipboardEngineState> _states =
      StreamController<ClipboardEngineState>.broadcast(sync: true);
  ClipboardEngineState _state;

  @override
  ClipboardEngineState get currentState => _state;

  @override
  Stream<ClipboardEngineState> get states => _states.stream;

  void _emit(ClipboardEngineState next) {
    _state = next;
    if (!_states.isClosed) {
      _states.add(next);
    }
  }

  @override
  Future<void> connect() async {
    final parentEndpoint = _state.settings.parentEndpoint.trim();
    if (parentEndpoint.isEmpty) {
      throw StateError('请先填写父节点地址');
    }
    _emit(
      _state.copyWith(
        busy: true,
        connected: false,
        loggedIn: false,
        clearNodeId: true,
        hubEndpoint: parentEndpoint,
        authStage: '连接父节点',
        lastError: '',
      ),
    );
    await Future<void>.delayed(const Duration(milliseconds: 180));
    _emit(
      _state.copyWith(
        connected: true,
        hubEndpoint: parentEndpoint,
        authStage: '后台注册',
      ),
    );
    await Future<void>.delayed(const Duration(milliseconds: 160));
    _emit(_state.copyWith(authStage: '后台登录'));
    await Future<void>.delayed(const Duration(milliseconds: 160));
    _emit(
      _state.copyWith(
        connected: true,
        loggedIn: true,
        busy: false,
        hubEndpoint: parentEndpoint,
        authStage: '已认证',
        nodeId: 120,
      ),
    );
  }

  @override
  Future<void> disconnect() async {
    _emit(
      _state.copyWith(
        connected: false,
        loggedIn: false,
        busy: false,
        clearNodeId: true,
        authStage: '登录态已清理',
        lastError: '',
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
    validateHistoryLimit(settings.historyLimit);
    final normalizedDeviceId = settings.deviceId.trim().isEmpty
        ? 'local-device'
        : settings.deviceId.trim();
    final normalizedDisplayName = settings.displayName.trim().isEmpty
        ? normalizedDeviceId
        : settings.displayName.trim();
    final next = settings.copyWith(
      parentEndpoint: normalizedParentEndpoint,
      topic: normalizedTopic,
      deviceId: normalizedDeviceId,
      displayName: normalizedDisplayName,
      deviceLabel: normalizedDisplayName,
    );
    _emit(
      _state.copyWith(
        hubEndpoint: normalizedParentEndpoint,
        settings: next,
        history: trimClipboardHistory(_state.history, next),
        lastError: '',
      ),
    );
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
    final byteSize = utf8.encode(text).length;
    final now = DateTime.now();
    final hashPrefix = now.microsecondsSinceEpoch
        .toRadixString(16)
        .padLeft(12, '0');
    if (byteSize > _state.settings.maxInlineBytes) {
      final transfer = TransferStatus(
        id: 'preview-transfer-${now.microsecondsSinceEpoch}',
        state: 'unsupported',
        byteSize: byteSize,
        hashPrefix: hashPrefix.substring(hashPrefix.length - 12),
        detail: 'clipboard.transfer.v1',
      );
      _emit(
        _state.copyWith(
          transferStatus: transfer,
          lastError: '文本超过 TopicBus 内联上限，当前未配置大内容传输',
        ),
      );
      return;
    }
    final activity = ClipboardActivity(
      id: 'preview-${now.microsecondsSinceEpoch}',
      kind: ActivityKind.published,
      title: '已发布文本',
      detail: 'TopicBus 本地发布状态',
      deviceLabel: _state.settings.displayName,
      byteSize: byteSize,
      hashPrefix: hashPrefix.substring(hashPrefix.length - 12),
      timestamp: now,
    );
    _emit(
      _state.copyWith(
        activities: [
          activity,
          ..._state.activities,
        ].take(20).toList(growable: false),
        history: appendClipboardHistory(_state, activity, text),
        lastError: '',
      ),
    );
  }

  @override
  Future<void> readClipboard() async {
    throw StateError('预览模式不能读取系统剪贴板');
  }

  @override
  Future<void> applyPending(String eventId) async {
    if (_state.pendingEvent?.eventId != eventId) {
      throw StateError('没有可应用的待接收事件');
    }
    _emit(_state.copyWith(clearPendingEvent: true, lastError: ''));
  }

  @override
  Future<void> clearRecent() async {
    _emit(
      _state.copyWith(
        activities: const [],
        history: const [],
        clearPendingEvent: true,
        clearTransferStatus: true,
        lastError: '',
      ),
    );
  }

  @override
  Future<void> dispose() async {
    if (_state.connected || _state.loggedIn || _state.busy) {
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
    await _states.close();
  }
}
