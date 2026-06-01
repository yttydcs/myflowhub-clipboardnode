import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:myflowhub_clipboard/main.dart';
import 'package:myflowhub_clipboard/app/theme/app_theme.dart';
import 'package:myflowhub_clipboard/core/bridge/preview_engine_bridge.dart';

ClipboardNodeApp previewApp() {
  return ClipboardNodeApp(bridgeFactory: () => PreviewEngineBridge());
}

void main() {
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
    expect(switchTheme.trackOutlineColor?.resolve({}), AppColors.border);
    expect(switchTheme.trackOutlineWidth?.resolve({}), 1);
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
    expect(find.text('同步关闭'), findsOneWidget);
    expect(find.text('clipboard.text.v1'), findsOneWidget);
    expect(find.text('平台能力'), findsOneWidget);
    expect(find.text('暂无活动'), findsOneWidget);
    expect(find.text('安全边界'), findsNothing);
  });

  testWidgets(
    'connects with background auth and sends metadata-only activity',
    (tester) async {
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
      await tester.tap(find.widgetWithText(FilledButton, '发送到 Topic'));
      await tester.pumpAndSettle();

      expect(find.text('请先启用剪贴板同步'), findsOneWidget);
      expect(find.text('hello from widget test'), findsOneWidget);
    },
  );

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
