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
    _emit(_state.copyWith(busy: true, lastError: ''));
    await Future<void>.delayed(const Duration(milliseconds: 420));
    _emit(
      _state.copyWith(
        connected: true,
        loggedIn: true,
        busy: false,
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
        lastError: '',
      ),
    );
  }

  @override
  Future<void> updateSettings(ClipboardSettings settings) async {
    final normalizedTopic = settings.topic.trim();
    if (settings.enabled && normalizedTopic.isEmpty) {
      throw StateError('启用同步时必须填写 Topic');
    }
    if (settings.maxInlineBytes <= 0) {
      throw StateError('内联文本上限必须大于 0');
    }
    _emit(
      _state.copyWith(
        settings: settings.copyWith(topic: normalizedTopic),
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
    if (byteSize > _state.settings.maxInlineBytes) {
      throw StateError('文本超过 TopicBus 内联上限，请使用后续的大内容传输入口');
    }
    final now = DateTime.now();
    final hashPrefix = now.microsecondsSinceEpoch
        .toRadixString(16)
        .padLeft(12, '0');
    final activity = ClipboardActivity(
      id: 'preview-${now.microsecondsSinceEpoch}',
      kind: ActivityKind.published,
      title: '已发布文本',
      detail: 'TopicBus 本地发布状态',
      deviceLabel: _state.settings.deviceLabel,
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
        lastError: '',
      ),
    );
  }

  @override
  Future<void> clearRecent() async {
    _emit(_state.copyWith(activities: const [], lastError: ''));
  }

  @override
  Future<void> dispose() async {
    await _states.close();
  }
}
