import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:myflowhub_clipboard/main.dart';
import 'package:myflowhub_clipboard/app/theme/app_theme.dart';
import 'package:myflowhub_clipboard/core/bridge/engine_contract.dart';
import 'package:myflowhub_clipboard/core/bridge/preview_engine_bridge.dart';

ClipboardNodeApp previewApp() {
  return ClipboardNodeApp(bridgeFactory: () => PreviewEngineBridge());
}

void main() {
  test('preview bridge stores body history and honors history limit', () async {
    final bridge = PreviewEngineBridge();
    addTearDown(bridge.dispose);

    await bridge.updateSettings(
      ClipboardSettings.defaults().copyWith(enabled: true, historyLimit: 2),
    );
    await bridge.connect();
    await bridge.sendText('alpha history body');
    await bridge.sendText('beta history body');
    await bridge.sendText('gamma history body');

    expect(bridge.currentState.history.map((entry) => entry.text), [
      'gamma history body',
      'beta history body',
    ]);

    await bridge.restoreHistory(bridge.currentState.history.last);
    expect(bridge.currentState.history.map((entry) => entry.text), [
      'beta history body',
      'gamma history body',
    ]);
    expect(bridge.currentState.history.first.id, startsWith('restore-'));

    await bridge.updateSettings(
      bridge.currentState.settings.copyWith(
        historyRetention: HistoryRetention.metadata,
      ),
    );
    expect(bridge.currentState.history, isEmpty);
  });

  test('preview bridge normalizes multiple topic routes', () async {
    final bridge = PreviewEngineBridge();
    addTearDown(bridge.dispose);

    await bridge.updateSettings(
      ClipboardSettings.defaults().copyWith(
        enabled: true,
        topic: '',
        topics: const [
          TopicSyncConfig(
            topic: ' clipboard.text ',
            syncToLocal: true,
            syncFromLocal: true,
          ),
          TopicSyncConfig(
            topic: 'clipboard.audit',
            syncToLocal: false,
            syncFromLocal: true,
          ),
        ],
      ),
    );

    expect(bridge.currentState.settings.topic, 'clipboard.text');
    expect(bridge.currentState.settings.topics.map((route) => route.topic), [
      'clipboard.text',
      'clipboard.audit',
    ]);
    expect(bridge.currentState.settings.topics.last.syncToLocal, isFalse);
    expect(bridge.currentState.settings.topics.last.syncFromLocal, isTrue);
  });

  test('body history ignores pending activity text', () {
    final state = ClipboardEngineState.initial(
      const PlatformCapability(
        platformLabel: 'test',
        automaticWatch: false,
        manualSend: true,
        autoApply: false,
        shareSheet: false,
        notes: [],
      ),
    );
    final history = appendClipboardHistory(
      state,
      ClipboardActivity(
        id: 'pending-1',
        kind: ActivityKind.pending,
        title: 'remote_pending',
        detail: 'TopicBus: clipboard.text',
        topic: 'clipboard.text',
        deviceLabel: 'remote',
        byteSize: 12,
        hashPrefix: 'abcdef123456',
        timestamp: DateTime.fromMillisecondsSinceEpoch(1700000000000),
      ),
      'not yet local clipboard',
    );

    expect(history, isEmpty);
  });

  test('parses persisted history payload and restore json', () {
    final settings = ClipboardSettings.defaults().copyWith(historyLimit: 2);
    final history = parseClipboardHistoryEntries([
      {
        'id': 'evt-3',
        'kind': 'applied',
        'text': 'gamma',
        'topic': 'clipboard.text',
        'device_label': 'Desktop',
        'byte_size': 5,
        'hash_prefix': 'abcdef123456',
        'timestamp_ms': 1700000000003,
      },
      {
        'id': 'evt-2',
        'kind': 'published',
        'text': 'beta',
        'topic': 'clipboard.audit',
        'device_label': 'Desktop',
        'byte_size': 4,
        'hash_prefix': '123456abcdef',
        'timestamp_ms': 1700000000002,
      },
      {
        'id': 'evt-dup',
        'kind': 'published',
        'text': 'beta',
        'timestamp_ms': 1700000000001,
      },
    ], settings);

    expect(history.map((entry) => entry.text), ['gamma', 'beta']);
    expect(history.first.kind, ActivityKind.applied);
    expect(history.first.topic, 'clipboard.text');
    expect(history.last.topic, 'clipboard.audit');

    final json = history.first.toJson();
    expect(json['text'], 'gamma');
    expect(json['kind'], 'applied');
    expect(json['timestamp_ms'], 1700000000003);

    expect(
      parseClipboardHistoryEntries([
        json,
      ], settings.copyWith(historyRetention: HistoryRetention.metadata)),
      isEmpty,
    );
  });

  testWidgets('uses stable switch colors and CJK-capable font fallback', (
    tester,
  ) async {
    await tester.pumpWidget(previewApp());

    final theme = Theme.of(tester.element(find.byType(Scaffold)));
    final switchTheme = theme.switchTheme;

    expect(
      switchTheme.thumbColor?.resolve({WidgetState.selected}),
      Colors.white,
    );
    expect(switchTheme.thumbColor?.resolve({}), AppColors.switchOffThumb);
    expect(switchTheme.trackColor?.resolve({}), AppColors.switchOffTrack);
    expect(
      switchTheme.trackColor?.resolve({WidgetState.hovered}),
      AppColors.tealSoft,
    );
    expect(switchTheme.trackOutlineColor?.resolve({}), AppColors.border);
    expect(switchTheme.trackOutlineWidth?.resolve({}), 1);
    expect(switchTheme.materialTapTargetSize, MaterialTapTargetSize.shrinkWrap);
    expect(switchTheme.splashRadius, 14);
    expect(switchTheme.padding, EdgeInsets.zero);
    expect(
      switchTheme.overlayColor?.resolve({WidgetState.hovered}),
      AppColors.teal.withValues(alpha: 0.12),
    );
    expect(
      switchTheme.overlayColor?.resolve({WidgetState.focused}),
      AppColors.teal.withValues(alpha: 0.14),
    );
    expect(
      theme.textTheme.bodyMedium?.fontFamilyFallback,
      contains('Microsoft YaHei UI'),
    );
    expect(
      theme.textTheme.bodyMedium?.fontFamilyFallback?.first,
      'Microsoft YaHei UI',
    );
  });

  testWidgets('shows ClipboardNode shell and safe default state', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1280, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(previewApp());

    expect(find.text('ClipboardNode'), findsWidgets);
    expect(find.text('总览'), findsWidgets);
    expect(find.text('历史'), findsWidgets);
    expect(find.text('日志'), findsWidgets);
    expect(find.text('同步关闭'), findsOneWidget);
    expect(find.text('clipboard.text'), findsWidgets);
    expect(find.text('平台能力'), findsOneWidget);
    expect(find.text('暂无日志'), findsOneWidget);
    expect(find.text('安全边界'), findsNothing);
  });

  testWidgets('opens clipboard history and log sections', (tester) async {
    tester.view.physicalSize = const Size(1280, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(previewApp());

    await tester.tap(find.text('历史'));
    await tester.pumpAndSettle();
    expect(find.text('剪贴板历史'), findsWidgets);
    expect(find.text('暂无剪贴板历史'), findsOneWidget);

    await tester.tap(find.text('日志').first);
    await tester.pumpAndSettle();
    expect(find.text('日志'), findsWidgets);
    expect(find.text('暂无日志'), findsOneWidget);
  });

  testWidgets('renders clipboard body history after sending text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1280, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(previewApp());

    await tester.tap(find.byIcon(Icons.tune_outlined).first);
    await tester.pumpAndSettle();
    expect(find.text('Topic 订阅'), findsOneWidget);
    expect(find.text('到本机'), findsOneWidget);
    expect(find.text('从本机'), findsOneWidget);
    expect(find.text('保存正文历史'), findsOneWidget);
    expect(find.text('历史条数'), findsOneWidget);
    expect(find.text('256'), findsOneWidget);

    await tester.tap(find.text('启用同步'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilledButton, '连接'));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(Icons.send_outlined).first);
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), '正文历史 widget body');
    await tester.tap(find.widgetWithText(FilledButton, '发送到订阅'));
    await tester.pumpAndSettle();

    await tester.tap(find.text('历史'));
    await tester.pumpAndSettle();

    expect(find.text('正文历史 widget body'), findsWidgets);
  });

  testWidgets('connects with background auth and rejects send when disabled', (
    tester,
  ) async {
    await tester.pumpWidget(previewApp());

    await tester.tap(find.widgetWithText(FilledButton, '连接'));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
    expect(find.text('已认证'), findsWidgets);
    expect(find.text('注册'), findsNothing);
    expect(find.text('登录'), findsNothing);

    await tester.tap(find.byIcon(Icons.send_outlined));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), 'hello from widget test');
    await tester.tap(find.widgetWithText(FilledButton, '发送到订阅'));
    await tester.pumpAndSettle();

    expect(find.text('请先启用剪贴板同步'), findsOneWidget);
    expect(find.text('hello from widget test'), findsOneWidget);
  });

  testWidgets('disconnect clears preview login state in the background', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1280, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(previewApp());
    await tester.tap(find.widgetWithText(FilledButton, '连接'));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.pumpAndSettle();
    expect(find.text('已认证'), findsWidgets);

    await tester.tap(find.widgetWithText(OutlinedButton, '断开'));
    await tester.pumpAndSettle();

    expect(find.text('未连接'), findsWidgets);
    expect(find.text('登录态已清理'), findsOneWidget);
  });

  testWidgets('updates parent endpoint from settings', (tester) async {
    tester.view.physicalSize = const Size(1280, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(previewApp());
    await tester.tap(find.byIcon(Icons.tune_outlined).first);
    await tester.pumpAndSettle();

    expect(find.text('父节点'), findsOneWidget);
    await tester.enterText(
      find.widgetWithText(TextField, '127.0.0.1:9000'),
      ' 10.0.0.2:9000 ',
    );
    await tester.tap(find.widgetWithText(FilledButton, '保存设置'));
    await tester.pumpAndSettle();

    expect(find.text('10.0.0.2:9000'), findsWidgets);
  });
}
