import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:myflowhub_clipboard/main.dart';

void main() {
  testWidgets('shows ClipboardNode shell and safe default state', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1280, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(const ClipboardNodeApp());

    expect(find.text('ClipboardNode'), findsWidgets);
    expect(find.text('总览'), findsWidgets);
    expect(find.text('同步关闭'), findsOneWidget);
    expect(find.text('clipboard.text.v1'), findsOneWidget);
    expect(find.text('暂无活动'), findsOneWidget);
    expect(find.textContaining('私有 MyFlowHub 拓扑'), findsOneWidget);
  });

  testWidgets('connects preview bridge and sends metadata-only activity', (
    tester,
  ) async {
    await tester.pumpWidget(const ClipboardNodeApp());

    await tester.tap(find.widgetWithText(FilledButton, '连接'));
    await tester.pumpAndSettle();
    expect(find.text('已连接'), findsOneWidget);

    await tester.tap(find.byIcon(Icons.send_outlined));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), 'hello from widget test');
    await tester.tap(find.widgetWithText(FilledButton, '发送到 Topic'));
    await tester.pumpAndSettle();

    expect(find.text('请先启用剪贴板同步'), findsOneWidget);
    expect(find.text('hello from widget test'), findsOneWidget);
  });
}
