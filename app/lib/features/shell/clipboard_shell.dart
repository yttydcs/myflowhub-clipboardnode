import 'dart:convert';

import 'package:flutter/material.dart';

import '../../app/theme/app_theme.dart';
import '../../core/bridge/engine_contract.dart';
import '../../core/controller/clipboard_controller.dart';

enum ClipboardSection {
  overview(Icons.dashboard_outlined, Icons.dashboard, '总览', '总览'),
  send(Icons.send_outlined, Icons.send, '发送', '发送'),
  history(
    Icons.content_paste_search_outlined,
    Icons.content_paste_search,
    '历史',
    '剪贴板历史',
  ),
  settings(Icons.tune_outlined, Icons.tune, '设置', '设置'),
  activity(Icons.receipt_long_outlined, Icons.receipt_long, '日志', '日志');

  const ClipboardSection(this.icon, this.selectedIcon, this.label, this.title);

  final IconData icon;
  final IconData selectedIcon;
  final String label;
  final String title;
}

class ClipboardShell extends StatefulWidget {
  const ClipboardShell({super.key, required this.controller});

  final ClipboardController controller;

  @override
  State<ClipboardShell> createState() => _ClipboardShellState();
}

class _ClipboardShellState extends State<ClipboardShell> {
  ClipboardSection _section = ClipboardSection.overview;

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: widget.controller,
      builder: (context, _) {
        final state = widget.controller.state;
        final wide = MediaQuery.sizeOf(context).width >= 920;
        final content = _SectionContent(
          section: _section,
          controller: widget.controller,
          state: state,
        );

        return Scaffold(
          body: SafeArea(
            child: Row(
              children: [
                if (wide) _SideNav(section: _section, onChanged: _setSection),
                Expanded(
                  child: Column(
                    children: [
                      _TopBar(
                        state: state,
                        controller: widget.controller,
                        section: _section,
                      ),
                      if (state.lastError.isNotEmpty)
                        _InlineError(message: state.lastError),
                      Expanded(
                        child: SingleChildScrollView(
                          padding: const EdgeInsets.fromLTRB(20, 18, 20, 24),
                          child: Center(
                            child: ConstrainedBox(
                              constraints: const BoxConstraints(maxWidth: 1180),
                              child: AnimatedSwitcher(
                                duration: const Duration(milliseconds: 160),
                                child: KeyedSubtree(
                                  key: ValueKey(_section),
                                  child: content,
                                ),
                              ),
                            ),
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          bottomNavigationBar: wide
              ? null
              : _BottomNav(section: _section, onChanged: _setSection),
        );
      },
    );
  }

  void _setSection(ClipboardSection next) {
    setState(() => _section = next);
  }
}

class _SectionContent extends StatelessWidget {
  const _SectionContent({
    required this.section,
    required this.controller,
    required this.state,
  });

  final ClipboardSection section;
  final ClipboardController controller;
  final ClipboardEngineState state;

  @override
  Widget build(BuildContext context) {
    switch (section) {
      case ClipboardSection.overview:
        return _OverviewSection(state: state);
      case ClipboardSection.send:
        return _SendSection(controller: controller, state: state);
      case ClipboardSection.history:
        return _ClipboardHistorySection(controller: controller, state: state);
      case ClipboardSection.settings:
        return _SettingsSection(controller: controller, state: state);
      case ClipboardSection.activity:
        return _LogSection(controller: controller, state: state);
    }
  }
}

class _SideNav extends StatelessWidget {
  const _SideNav({required this.section, required this.onChanged});

  final ClipboardSection section;
  final ValueChanged<ClipboardSection> onChanged;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 236,
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(right: BorderSide(color: AppColors.border)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Container(
            height: 72,
            padding: const EdgeInsets.symmetric(horizontal: 20),
            decoration: const BoxDecoration(
              border: Border(bottom: BorderSide(color: AppColors.border)),
            ),
            child: const Center(child: _BrandMark()),
          ),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.all(12),
              children: [
                for (final item in ClipboardSection.values)
                  _NavButton(
                    icon: item == section ? item.selectedIcon : item.icon,
                    label: item.label,
                    selected: item == section,
                    onPressed: () => onChanged(item),
                  ),
              ],
            ),
          ),
          const Padding(padding: EdgeInsets.all(16), child: _ProtocolBadge()),
        ],
      ),
    );
  }
}

class _BottomNav extends StatelessWidget {
  const _BottomNav({required this.section, required this.onChanged});

  final ClipboardSection section;
  final ValueChanged<ClipboardSection> onChanged;

  @override
  Widget build(BuildContext context) {
    return NavigationBar(
      selectedIndex: ClipboardSection.values.indexOf(section),
      onDestinationSelected: (index) =>
          onChanged(ClipboardSection.values[index]),
      destinations: [
        for (final item in ClipboardSection.values)
          NavigationDestination(
            icon: Icon(item.icon),
            selectedIcon: Icon(item.selectedIcon),
            label: item.label,
          ),
      ],
    );
  }
}

class _TopBar extends StatelessWidget {
  const _TopBar({
    required this.state,
    required this.controller,
    required this.section,
  });

  final ClipboardEngineState state;
  final ClipboardController controller;
  final ClipboardSection section;

  @override
  Widget build(BuildContext context) {
    final title = Theme.of(context).textTheme.titleLarge;
    return Container(
      constraints: const BoxConstraints(minHeight: 72),
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Text(section.title, style: title),
                const SizedBox(height: 4),
                Text(
                  state.settings.topic,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
          const SizedBox(width: 12),
          _StatusPill(
            icon: state.loggedIn
                ? Icons.verified_user_outlined
                : state.connected
                ? Icons.hub
                : Icons.hub_outlined,
            label: state.loggedIn
                ? '已认证'
                : state.connected
                ? state.authStage
                : '未连接',
            tone: state.loggedIn
                ? _Tone.success
                : state.connected
                ? _Tone.warning
                : _Tone.muted,
          ),
          const SizedBox(width: 8),
          _StatusPill(
            icon: state.settings.enabled
                ? Icons.sync_outlined
                : Icons.sync_disabled_outlined,
            label: state.settings.enabled ? '同步开启' : '同步关闭',
            tone: state.settings.enabled ? _Tone.success : _Tone.warning,
          ),
          const SizedBox(width: 12),
          if (state.connected)
            OutlinedButton.icon(
              onPressed: state.busy ? null : controller.disconnect,
              icon: const Icon(Icons.link_off, size: 18),
              label: const Text('断开'),
            )
          else
            FilledButton.icon(
              onPressed: state.busy ? null : controller.connect,
              icon: state.busy
                  ? const SizedBox.square(
                      dimension: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.link, size: 18),
              label: const Text('连接'),
            ),
        ],
      ),
    );
  }
}

class _OverviewSection extends StatelessWidget {
  const _OverviewSection({required this.state});

  final ClipboardEngineState state;

  @override
  Widget build(BuildContext context) {
    final columns = MediaQuery.sizeOf(context).width >= 960 ? 4 : 2;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _MetricGrid(
          columns: columns,
          items: [
            _MetricItem(
              icon: Icons.hub_outlined,
              label: 'Hub',
              value: state.connected ? state.hubEndpoint : '未连接',
              tone: state.connected ? _Tone.success : _Tone.muted,
            ),
            _MetricItem(
              icon: Icons.badge_outlined,
              label: 'Node',
              value: state.nodeId?.toString() ?? state.authStage,
              tone: state.loggedIn ? _Tone.success : _Tone.muted,
            ),
            _MetricItem(
              icon: Icons.text_fields,
              label: 'Inline',
              value: '${state.settings.maxInlineBytes} B',
              tone: _Tone.info,
            ),
            _MetricItem(
              icon: Icons.devices_outlined,
              label: 'Platform',
              value: state.capability.platformLabel,
              tone: state.capability.automaticWatch
                  ? _Tone.success
                  : _Tone.warning,
            ),
          ],
        ),
        const SizedBox(height: 18),
        _ResponsiveTwoColumn(
          first: _PolicyPanel(state: state),
          second: _QueuePanel(state: state),
        ),
      ],
    );
  }
}

class _PolicyPanel extends StatelessWidget {
  const _PolicyPanel({required this.state});

  final ClipboardEngineState state;

  @override
  Widget build(BuildContext context) {
    final capability = state.capability;
    return _Panel(
      title: '平台能力',
      icon: Icons.tune_outlined,
      child: Column(
        children: [
          _CapabilityRow(
            icon: Icons.visibility_outlined,
            label: '自动监听',
            active: capability.automaticWatch && state.settings.autoWatch,
            available: capability.automaticWatch,
          ),
          _CapabilityRow(
            icon: Icons.touch_app_outlined,
            label: '手动发送',
            active: capability.manualSend,
            available: capability.manualSend,
          ),
          _CapabilityRow(
            icon: Icons.assignment_turned_in_outlined,
            label: '自动应用',
            active: capability.autoApply && state.settings.autoApply,
            available: capability.autoApply,
          ),
          _CapabilityRow(
            icon: Icons.ios_share_outlined,
            label: '系统分享',
            active: capability.shareSheet,
            available: capability.shareSheet,
          ),
          const SizedBox(height: 10),
          for (final note in capability.notes)
            _CompactNote(icon: Icons.info_outline, text: note),
        ],
      ),
    );
  }
}

class _QueuePanel extends StatelessWidget {
  const _QueuePanel({required this.state});

  final ClipboardEngineState state;

  @override
  Widget build(BuildContext context) {
    final pending = state.pendingEvent;
    final transfer = state.transferStatus;
    final latest = state.activities.firstOrNull;
    return _Panel(
      title: pending == null && transfer == null ? '最近日志' : '接收队列',
      icon: Icons.timeline_outlined,
      child: pending == null && transfer == null && latest == null
          ? const _EmptyState(
              icon: Icons.inbox_outlined,
              title: '暂无日志',
              detail: 'metadata only',
            )
          : Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                if (pending != null) ...[
                  _PendingTile(pending: pending, onApply: null),
                  if (transfer != null || latest != null) const Divider(),
                ],
                if (transfer != null) ...[
                  _TransferTile(status: transfer),
                  if (latest != null) const Divider(),
                ],
                if (latest != null) _ActivityTile(activity: latest),
              ],
            ),
    );
  }
}

class _SendSection extends StatefulWidget {
  const _SendSection({required this.controller, required this.state});

  final ClipboardController controller;
  final ClipboardEngineState state;

  @override
  State<_SendSection> createState() => _SendSectionState();
}

class _SendSectionState extends State<_SendSection> {
  final TextEditingController _text = TextEditingController();

  @override
  void dispose() {
    _text.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final byteSize = utf8.encode(_text.text).length;
    final overLimit = byteSize > widget.state.settings.maxInlineBytes;
    return _Panel(
      title: '手动发送',
      icon: Icons.send_outlined,
      trailing: _StatusPill(
        icon: overLimit ? Icons.error_outline : Icons.text_fields,
        label: '$byteSize B',
        tone: overLimit ? _Tone.error : _Tone.info,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TextField(
            controller: _text,
            minLines: 10,
            maxLines: 18,
            maxLength: widget.state.settings.maxInlineBytes,
            onChanged: (_) => setState(() {}),
            decoration: const InputDecoration(
              hintText: 'Paste text to publish',
              counterText: '',
              alignLabelWithHint: true,
            ),
          ),
          const SizedBox(height: 14),
          Wrap(
            spacing: 10,
            runSpacing: 10,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              FilledButton.icon(
                onPressed: overLimit
                    ? null
                    : () => widget.controller.sendText(_text.text),
                icon: const Icon(Icons.send, size: 18),
                label: const Text('发送到 Topic'),
              ),
              OutlinedButton.icon(
                onPressed: widget.state.capability.manualSend
                    ? widget.controller.readClipboard
                    : null,
                icon: const Icon(Icons.content_paste_search, size: 18),
                label: const Text('读取并发送'),
              ),
              OutlinedButton.icon(
                onPressed: _text.text.isEmpty
                    ? null
                    : () => setState(_text.clear),
                icon: const Icon(Icons.clear, size: 18),
                label: const Text('清空'),
              ),
              _StatusPill(
                icon: widget.state.settings.enabled
                    ? Icons.sync_outlined
                    : Icons.sync_disabled_outlined,
                label: widget.state.settings.enabled ? 'TopicBus 就绪' : '同步关闭',
                tone: widget.state.settings.enabled
                    ? _Tone.success
                    : _Tone.warning,
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _SettingsSection extends StatefulWidget {
  const _SettingsSection({required this.controller, required this.state});

  final ClipboardController controller;
  final ClipboardEngineState state;

  @override
  State<_SettingsSection> createState() => _SettingsSectionState();
}

class _SettingsSectionState extends State<_SettingsSection> {
  late final TextEditingController _parentEndpoint;
  late final TextEditingController _topic;
  late final TextEditingController _deviceLabel;
  late final TextEditingController _maxInlineBytes;
  late final TextEditingController _transferProvider;
  late final TextEditingController _transferRef;

  @override
  void initState() {
    super.initState();
    final settings = widget.state.settings;
    _parentEndpoint = TextEditingController(text: settings.parentEndpoint);
    _topic = TextEditingController(text: settings.topic);
    _deviceLabel = TextEditingController(text: settings.deviceLabel);
    _maxInlineBytes = TextEditingController(
      text: settings.maxInlineBytes.toString(),
    );
    _transferProvider = TextEditingController(text: settings.transferProvider);
    _transferRef = TextEditingController(text: settings.transferRef);
  }

  @override
  void didUpdateWidget(covariant _SettingsSection oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.state.settings != widget.state.settings) {
      _syncController(_parentEndpoint, widget.state.settings.parentEndpoint);
      _syncController(_topic, widget.state.settings.topic);
      _syncController(_deviceLabel, widget.state.settings.deviceLabel);
      _syncController(
        _maxInlineBytes,
        widget.state.settings.maxInlineBytes.toString(),
      );
      _syncController(
        _transferProvider,
        widget.state.settings.transferProvider,
      );
      _syncController(_transferRef, widget.state.settings.transferRef);
    }
  }

  @override
  void dispose() {
    _parentEndpoint.dispose();
    _topic.dispose();
    _deviceLabel.dispose();
    _maxInlineBytes.dispose();
    _transferProvider.dispose();
    _transferRef.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final settings = widget.state.settings;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _Panel(
          title: '同步配置',
          icon: Icons.settings_outlined,
          child: Column(
            children: [
              SwitchListTile(
                value: settings.enabled,
                onChanged: widget.controller.setSyncEnabled,
                title: const Text('启用同步'),
                secondary: const Icon(Icons.sync_outlined),
                contentPadding: EdgeInsets.zero,
              ),
              const Divider(),
              const SizedBox(height: 14),
              _FormGrid(
                children: [
                  TextField(
                    controller: _parentEndpoint,
                    decoration: const InputDecoration(
                      labelText: '父节点',
                      prefixIcon: Icon(Icons.hub_outlined),
                      hintText: '127.0.0.1:9000',
                    ),
                  ),
                  TextField(
                    controller: _topic,
                    decoration: const InputDecoration(
                      labelText: 'Topic',
                      prefixIcon: Icon(Icons.tag_outlined),
                    ),
                  ),
                  TextField(
                    controller: _deviceLabel,
                    decoration: const InputDecoration(
                      labelText: '设备标签',
                      prefixIcon: Icon(Icons.devices_outlined),
                    ),
                  ),
                  TextField(
                    controller: _maxInlineBytes,
                    keyboardType: TextInputType.number,
                    decoration: const InputDecoration(
                      labelText: '内联上限',
                      prefixIcon: Icon(Icons.data_object_outlined),
                      suffixText: 'bytes',
                    ),
                  ),
                  TextField(
                    controller: _transferProvider,
                    decoration: const InputDecoration(
                      labelText: '传输 Provider',
                      prefixIcon: Icon(Icons.move_to_inbox_outlined),
                    ),
                  ),
                  TextField(
                    controller: _transferRef,
                    decoration: const InputDecoration(
                      labelText: '传输引用',
                      prefixIcon: Icon(Icons.key_outlined),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 14),
              Align(
                alignment: Alignment.centerLeft,
                child: FilledButton.icon(
                  onPressed: _saveSettings,
                  icon: const Icon(Icons.save_outlined, size: 18),
                  label: const Text('保存设置'),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 18),
        _Panel(
          title: '本机策略',
          icon: Icons.policy_outlined,
          child: Column(
            children: [
              SwitchListTile(
                value: settings.autoWatch,
                onChanged: widget.state.capability.automaticWatch
                    ? (value) => _update(settings.copyWith(autoWatch: value))
                    : null,
                title: const Text('自动监听剪贴板'),
                secondary: const Icon(Icons.visibility_outlined),
                contentPadding: EdgeInsets.zero,
              ),
              SwitchListTile(
                value: settings.autoApply,
                onChanged: widget.state.capability.autoApply
                    ? (value) => _update(settings.copyWith(autoApply: value))
                    : null,
                title: const Text('自动应用远端文本'),
                secondary: const Icon(Icons.assignment_turned_in_outlined),
                contentPadding: EdgeInsets.zero,
              ),
              const Divider(),
              RadioGroup<HistoryRetention>(
                groupValue: settings.historyRetention,
                onChanged: (value) {
                  if (value != null) {
                    _update(settings.copyWith(historyRetention: value));
                  }
                },
                child: const Column(
                  children: [
                    ListTile(
                      title: Text('保留日志元数据'),
                      leading: Icon(Icons.list_alt_outlined),
                      trailing: Radio<HistoryRetention>(
                        value: HistoryRetention.metadata,
                      ),
                      contentPadding: EdgeInsets.zero,
                    ),
                    ListTile(
                      title: Text('不保留日志历史'),
                      leading: Icon(Icons.delete_outline),
                      trailing: Radio<HistoryRetention>(
                        value: HistoryRetention.none,
                      ),
                      contentPadding: EdgeInsets.zero,
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  void _syncController(TextEditingController controller, String value) {
    if (controller.text != value) {
      controller.text = value;
    }
  }

  Future<void> _saveSettings() async {
    final parsedLimit = int.tryParse(_maxInlineBytes.text.trim());
    if (parsedLimit == null) {
      await widget.controller.updateSettings(
        widget.state.settings.copyWith(maxInlineBytes: -1),
      );
      return;
    }
    await _update(
      widget.state.settings.copyWith(
        parentEndpoint: _parentEndpoint.text,
        topic: _topic.text,
        deviceLabel: _deviceLabel.text.trim().isEmpty
            ? 'local-device'
            : _deviceLabel.text.trim(),
        maxInlineBytes: parsedLimit,
        transferProvider: _transferProvider.text.trim(),
        transferRef: _transferRef.text.trim(),
      ),
    );
  }

  Future<void> _update(ClipboardSettings settings) {
    return widget.controller.updateSettings(settings);
  }
}

class _ClipboardHistorySection extends StatelessWidget {
  const _ClipboardHistorySection({
    required this.controller,
    required this.state,
  });

  final ClipboardController controller;
  final ClipboardEngineState state;

  @override
  Widget build(BuildContext context) {
    return _Panel(
      title: '剪贴板历史',
      icon: Icons.content_paste_search_outlined,
      trailing: OutlinedButton.icon(
        onPressed: state.activities.isEmpty ? null : controller.clearRecent,
        icon: const Icon(Icons.delete_sweep_outlined, size: 18),
        label: const Text('清空历史'),
      ),
      child:
          state.activities.isEmpty &&
              state.pendingEvent == null &&
              state.transferStatus == null
          ? const _EmptyState(
              icon: Icons.content_paste_off_outlined,
              title: '暂无剪贴板历史',
              detail: '仅记录元数据，不保存正文',
            )
          : Column(
              children: [
                if (state.pendingEvent != null) ...[
                  _PendingTile(
                    pending: state.pendingEvent!,
                    onApply: () =>
                        controller.applyPending(state.pendingEvent!.eventId),
                  ),
                  if (state.transferStatus != null ||
                      state.activities.isNotEmpty)
                    const Divider(),
                ],
                if (state.transferStatus != null) ...[
                  _TransferTile(status: state.transferStatus!),
                  if (state.activities.isNotEmpty) const Divider(),
                ],
                for (final activity in state.activities) ...[
                  _ClipboardHistoryTile(activity: activity),
                  if (activity != state.activities.last) const Divider(),
                ],
              ],
            ),
    );
  }
}

class _LogSection extends StatelessWidget {
  const _LogSection({required this.controller, required this.state});

  final ClipboardController controller;
  final ClipboardEngineState state;

  @override
  Widget build(BuildContext context) {
    return _Panel(
      title: '日志',
      icon: Icons.receipt_long_outlined,
      trailing: OutlinedButton.icon(
        onPressed: state.activities.isEmpty ? null : controller.clearRecent,
        icon: const Icon(Icons.delete_sweep_outlined, size: 18),
        label: const Text('清空'),
      ),
      child:
          state.activities.isEmpty &&
              state.pendingEvent == null &&
              state.transferStatus == null
          ? const _EmptyState(
              icon: Icons.inbox_outlined,
              title: '暂无日志',
              detail: 'metadata only',
            )
          : Column(
              children: [
                if (state.pendingEvent != null) ...[
                  _PendingTile(
                    pending: state.pendingEvent!,
                    onApply: () =>
                        controller.applyPending(state.pendingEvent!.eventId),
                  ),
                  if (state.transferStatus != null ||
                      state.activities.isNotEmpty)
                    const Divider(),
                ],
                if (state.transferStatus != null) ...[
                  _TransferTile(status: state.transferStatus!),
                  if (state.activities.isNotEmpty) const Divider(),
                ],
                for (final activity in state.activities) ...[
                  _ActivityTile(activity: activity),
                  if (activity != state.activities.last) const Divider(),
                ],
              ],
            ),
    );
  }
}

class _Panel extends StatelessWidget {
  const _Panel({
    required this.title,
    required this.icon,
    required this.child,
    this.trailing,
  });

  final String title;
  final IconData icon;
  final Widget child;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      clipBehavior: Clip.antiAlias,
      shape: RoundedRectangleBorder(
        side: const BorderSide(color: AppColors.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 12),
            child: Row(
              children: [
                Icon(icon, size: 20, color: AppColors.teal),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    title,
                    style: Theme.of(context).textTheme.titleMedium,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                ?trailing,
              ],
            ),
          ),
          const Divider(),
          Padding(padding: const EdgeInsets.all(16), child: child),
        ],
      ),
    );
  }
}

class _MetricGrid extends StatelessWidget {
  const _MetricGrid({required this.columns, required this.items});

  final int columns;
  final List<_MetricItem> items;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final gap = 12.0;
        final itemWidth =
            (constraints.maxWidth - gap * (columns - 1)) / columns;
        return Wrap(
          spacing: gap,
          runSpacing: gap,
          children: [
            for (final item in items)
              SizedBox(
                width: itemWidth.clamp(190, constraints.maxWidth),
                child: item,
              ),
          ],
        );
      },
    );
  }
}

class _MetricItem extends StatelessWidget {
  const _MetricItem({
    required this.icon,
    required this.label,
    required this.value,
    required this.tone,
  });

  final IconData icon;
  final String label;
  final String value;
  final _Tone tone;

  @override
  Widget build(BuildContext context) {
    final colors = _toneColors(tone);
    return Container(
      height: 112,
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: AppColors.surface,
        border: Border.all(color: AppColors.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 34,
                height: 34,
                decoration: BoxDecoration(
                  color: colors.background,
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Icon(icon, color: colors.foreground, size: 19),
              ),
              const Spacer(),
              Container(
                width: 8,
                height: 8,
                decoration: BoxDecoration(
                  color: colors.foreground,
                  shape: BoxShape.circle,
                ),
              ),
            ],
          ),
          const Spacer(),
          Text(label, style: Theme.of(context).textTheme.bodySmall),
          const SizedBox(height: 4),
          Text(
            value,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: Theme.of(context).textTheme.titleMedium,
          ),
        ],
      ),
    );
  }
}

class _ResponsiveTwoColumn extends StatelessWidget {
  const _ResponsiveTwoColumn({required this.first, required this.second});

  final Widget first;
  final Widget second;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth < 760) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [first, const SizedBox(height: 18), second],
          );
        }
        return Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(child: first),
            const SizedBox(width: 18),
            Expanded(child: second),
          ],
        );
      },
    );
  }
}

class _FormGrid extends StatelessWidget {
  const _FormGrid({required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final wide = constraints.maxWidth >= 760;
        final itemWidth = wide
            ? (constraints.maxWidth - 12) / 2
            : constraints.maxWidth;
        return Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            for (final child in children)
              SizedBox(width: itemWidth, child: child),
          ],
        );
      },
    );
  }
}

class _ClipboardHistoryTile extends StatelessWidget {
  const _ClipboardHistoryTile({required this.activity});

  final ClipboardActivity activity;

  @override
  Widget build(BuildContext context) {
    final tone = switch (activity.kind) {
      ActivityKind.published => _Tone.success,
      ActivityKind.applied => _Tone.info,
      ActivityKind.pending => _Tone.warning,
      ActivityKind.ignored => _Tone.muted,
      ActivityKind.error => _Tone.error,
    };
    final colors = _toneColors(tone);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 10),
      child: Row(
        children: [
          Container(
            width: 42,
            height: 42,
            decoration: BoxDecoration(
              color: colors.background,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Icon(
              _activityIcon(activity.kind),
              color: colors.foreground,
              size: 21,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  _historyTitle(activity.kind),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const SizedBox(height: 3),
                Text(
                  '${_formatHistoryTime(activity.timestamp)} · ${activity.deviceLabel} · ${activity.byteSize} B',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
          const SizedBox(width: 12),
          Container(
            constraints: const BoxConstraints(maxWidth: 120),
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            decoration: BoxDecoration(
              color: AppColors.surfaceMuted,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Text(
              activity.hashPrefix.isEmpty ? '-' : activity.hashPrefix,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                fontFeatures: const [FontFeature.tabularFigures()],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ActivityTile extends StatelessWidget {
  const _ActivityTile({required this.activity});

  final ClipboardActivity activity;

  @override
  Widget build(BuildContext context) {
    final tone = switch (activity.kind) {
      ActivityKind.published => _Tone.success,
      ActivityKind.applied => _Tone.info,
      ActivityKind.pending => _Tone.warning,
      ActivityKind.ignored => _Tone.warning,
      ActivityKind.error => _Tone.error,
    };
    final colors = _toneColors(tone);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 10),
      child: Row(
        children: [
          Container(
            width: 38,
            height: 38,
            decoration: BoxDecoration(
              color: colors.background,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Icon(
              _activityIcon(activity.kind),
              color: colors.foreground,
              size: 20,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  activity.title,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const SizedBox(height: 3),
                Text(
                  '${activity.detail} · ${activity.deviceLabel} · ${activity.byteSize} B',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
          const SizedBox(width: 12),
          Text(
            activity.hashPrefix,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
              fontFeatures: const [FontFeature.tabularFigures()],
            ),
          ),
        ],
      ),
    );
  }
}

class _PendingTile extends StatelessWidget {
  const _PendingTile({required this.pending, required this.onApply});

  final PendingClipboardEvent pending;
  final VoidCallback? onApply;

  @override
  Widget build(BuildContext context) {
    final colors = _toneColors(_Tone.warning);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 10),
      child: Row(
        children: [
          Container(
            width: 38,
            height: 38,
            decoration: BoxDecoration(
              color: colors.background,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Icon(
              Icons.pending_actions,
              color: colors.foreground,
              size: 20,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('待应用远端文本', style: Theme.of(context).textTheme.titleMedium),
                const SizedBox(height: 3),
                Text(
                  '${pending.eventId} · ${pending.byteSize} B',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
          const SizedBox(width: 12),
          if (onApply == null)
            Text(
              pending.hashPrefix,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                fontFeatures: const [FontFeature.tabularFigures()],
              ),
            )
          else
            OutlinedButton.icon(
              onPressed: onApply,
              icon: const Icon(Icons.assignment_turned_in_outlined, size: 18),
              label: const Text('应用'),
            ),
        ],
      ),
    );
  }
}

class _TransferTile extends StatelessWidget {
  const _TransferTile({required this.status});

  final TransferStatus status;

  @override
  Widget build(BuildContext context) {
    final unsupported = status.state == 'unsupported';
    final tone = unsupported ? _Tone.error : _Tone.info;
    final colors = _toneColors(tone);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 10),
      child: Row(
        children: [
          Container(
            width: 38,
            height: 38,
            decoration: BoxDecoration(
              color: colors.background,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Icon(
              unsupported ? Icons.block : Icons.move_to_inbox_outlined,
              color: colors.foreground,
              size: 20,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('大内容传输', style: Theme.of(context).textTheme.titleMedium),
                const SizedBox(height: 3),
                Text(
                  '${status.state} · ${status.detail} · ${status.byteSize} B',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
          const SizedBox(width: 12),
          Text(
            status.hashPrefix,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
              fontFeatures: const [FontFeature.tabularFigures()],
            ),
          ),
        ],
      ),
    );
  }
}

class _CapabilityRow extends StatelessWidget {
  const _CapabilityRow({
    required this.icon,
    required this.label,
    required this.active,
    required this.available,
  });

  final IconData icon;
  final String label;
  final bool active;
  final bool available;

  @override
  Widget build(BuildContext context) {
    final tone = !available
        ? _Tone.muted
        : active
        ? _Tone.success
        : _Tone.warning;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 7),
      child: Row(
        children: [
          Icon(icon, size: 19, color: _toneColors(tone).foreground),
          const SizedBox(width: 10),
          Expanded(child: Text(label)),
          _StatusPill(
            icon: active
                ? Icons.check
                : available
                ? Icons.pause
                : Icons.block,
            label: active
                ? '开启'
                : available
                ? '可用'
                : '不可用',
            tone: tone,
          ),
        ],
      ),
    );
  }
}

class _CompactNote extends StatelessWidget {
  const _CompactNote({required this.icon, required this.text});

  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          Icon(icon, size: 18, color: AppColors.inkMuted),
          const SizedBox(width: 10),
          Expanded(child: Text(text, overflow: TextOverflow.ellipsis)),
        ],
      ),
    );
  }
}

class _StatusPill extends StatelessWidget {
  const _StatusPill({
    required this.icon,
    required this.label,
    required this.tone,
  });

  final IconData icon;
  final String label;
  final _Tone tone;

  @override
  Widget build(BuildContext context) {
    final colors = _toneColors(tone);
    return Container(
      height: 34,
      padding: const EdgeInsets.symmetric(horizontal: 10),
      decoration: BoxDecoration(
        color: colors.background,
        borderRadius: BorderRadius.circular(17),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 16, color: colors.foreground),
          const SizedBox(width: 6),
          Text(
            label,
            style: Theme.of(context).textTheme.labelMedium?.copyWith(
              color: colors.foreground,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}

class _InlineError extends StatelessWidget {
  const _InlineError({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      color: AppColors.coralSoft,
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
      child: Row(
        children: [
          const Icon(Icons.error_outline, size: 18, color: AppColors.coral),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              message,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                color: AppColors.coral,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _BrandMark extends StatelessWidget {
  const _BrandMark();

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 38,
          height: 38,
          decoration: BoxDecoration(
            color: AppColors.ink,
            borderRadius: BorderRadius.circular(8),
          ),
          child: const Icon(
            Icons.content_paste_go,
            color: Colors.white,
            size: 21,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'ClipboardNode',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              Text('MyFlowHub', style: Theme.of(context).textTheme.bodySmall),
            ],
          ),
        ),
      ],
    );
  }
}

class _ProtocolBadge extends StatelessWidget {
  const _ProtocolBadge();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.surfaceMuted,
        borderRadius: BorderRadius.circular(8),
      ),
      child: const Row(
        children: [
          Icon(Icons.topic_outlined, color: AppColors.navy, size: 18),
          SizedBox(width: 8),
          Expanded(
            child: Text(
              'clipboard.text.v1',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }
}

class _NavButton extends StatelessWidget {
  const _NavButton({
    required this.icon,
    required this.label,
    required this.selected,
    required this.onPressed,
  });

  final IconData icon;
  final String label;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Material(
        color: selected ? AppColors.tealSoft : Colors.transparent,
        borderRadius: BorderRadius.circular(7),
        child: InkWell(
          borderRadius: BorderRadius.circular(7),
          onTap: onPressed,
          child: SizedBox(
            height: 44,
            child: Row(
              children: [
                const SizedBox(width: 12),
                Icon(
                  icon,
                  size: 20,
                  color: selected ? AppColors.teal : AppColors.inkMuted,
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    label,
                    style: TextStyle(
                      color: selected ? AppColors.teal : AppColors.ink,
                      fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({
    required this.icon,
    required this.title,
    required this.detail,
  });

  final IconData icon;
  final String title;
  final String detail;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 160,
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, color: AppColors.inkMuted, size: 34),
            const SizedBox(height: 10),
            Text(title, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 4),
            Text(detail, style: Theme.of(context).textTheme.bodySmall),
          ],
        ),
      ),
    );
  }
}

enum _Tone { success, warning, error, muted, info }

class _ToneColors {
  const _ToneColors({required this.foreground, required this.background});

  final Color foreground;
  final Color background;
}

_ToneColors _toneColors(_Tone tone) {
  return switch (tone) {
    _Tone.success => const _ToneColors(
      foreground: AppColors.teal,
      background: AppColors.tealSoft,
    ),
    _Tone.warning => const _ToneColors(
      foreground: AppColors.amber,
      background: AppColors.amberSoft,
    ),
    _Tone.error => const _ToneColors(
      foreground: AppColors.coral,
      background: AppColors.coralSoft,
    ),
    _Tone.muted => const _ToneColors(
      foreground: AppColors.inkMuted,
      background: AppColors.surfaceMuted,
    ),
    _Tone.info => const _ToneColors(
      foreground: AppColors.navy,
      background: AppColors.navySoft,
    ),
  };
}

IconData _activityIcon(ActivityKind kind) {
  return switch (kind) {
    ActivityKind.published => Icons.north_east,
    ActivityKind.applied => Icons.south_west,
    ActivityKind.pending => Icons.pending_actions,
    ActivityKind.ignored => Icons.block,
    ActivityKind.error => Icons.error_outline,
  };
}

String _historyTitle(ActivityKind kind) {
  return switch (kind) {
    ActivityKind.published => '已发送到 Topic',
    ActivityKind.applied => '已应用到本机剪贴板',
    ActivityKind.pending => '等待处理的剪贴板内容',
    ActivityKind.ignored => '已忽略的剪贴板事件',
    ActivityKind.error => '剪贴板同步失败',
  };
}

String _formatHistoryTime(DateTime timestamp) {
  final local = timestamp.toLocal();
  String two(int value) => value.toString().padLeft(2, '0');
  return '${two(local.hour)}:${two(local.minute)}:${two(local.second)}';
}
