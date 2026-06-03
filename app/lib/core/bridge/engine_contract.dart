import 'dart:convert';

enum ActivityKind { published, applied, pending, ignored, error }

enum HistoryRetention { none, metadata }

class PlatformCapability {
  const PlatformCapability({
    required this.platformLabel,
    required this.automaticWatch,
    required this.manualSend,
    required this.autoApply,
    required this.shareSheet,
    required this.notes,
  });

  final String platformLabel;
  final bool automaticWatch;
  final bool manualSend;
  final bool autoApply;
  final bool shareSheet;
  final List<String> notes;
}

class ClipboardSettings {
  const ClipboardSettings({
    required this.enabled,
    required this.parentEndpoint,
    required this.topic,
    required this.deviceId,
    required this.displayName,
    required this.deviceLabel,
    required this.maxInlineBytes,
    required this.autoWatch,
    required this.autoApply,
    required this.historyRetention,
    required this.transferProvider,
    required this.transferRef,
  });

  factory ClipboardSettings.defaults() {
    return const ClipboardSettings(
      enabled: false,
      parentEndpoint: '127.0.0.1:9000',
      topic: 'clipboard/shared',
      deviceId: 'local-device',
      displayName: 'local-device',
      deviceLabel: 'local-device',
      maxInlineBytes: 65536,
      autoWatch: false,
      autoApply: false,
      historyRetention: HistoryRetention.metadata,
      transferProvider: '',
      transferRef: '',
    );
  }

  final bool enabled;
  final String parentEndpoint;
  final String topic;
  final String deviceId;
  final String displayName;
  final String deviceLabel;
  final int maxInlineBytes;
  final bool autoWatch;
  final bool autoApply;
  final HistoryRetention historyRetention;
  final String transferProvider;
  final String transferRef;

  ClipboardSettings copyWith({
    bool? enabled,
    String? parentEndpoint,
    String? topic,
    String? deviceId,
    String? displayName,
    String? deviceLabel,
    int? maxInlineBytes,
    bool? autoWatch,
    bool? autoApply,
    HistoryRetention? historyRetention,
    String? transferProvider,
    String? transferRef,
  }) {
    return ClipboardSettings(
      enabled: enabled ?? this.enabled,
      parentEndpoint: parentEndpoint ?? this.parentEndpoint,
      topic: topic ?? this.topic,
      deviceId: deviceId ?? this.deviceId,
      displayName: displayName ?? this.displayName,
      deviceLabel: deviceLabel ?? this.deviceLabel,
      maxInlineBytes: maxInlineBytes ?? this.maxInlineBytes,
      autoWatch: autoWatch ?? this.autoWatch,
      autoApply: autoApply ?? this.autoApply,
      historyRetention: historyRetention ?? this.historyRetention,
      transferProvider: transferProvider ?? this.transferProvider,
      transferRef: transferRef ?? this.transferRef,
    );
  }

  Map<String, Object> toJson() {
    return {
      'enabled': enabled,
      'parent_endpoint': parentEndpoint,
      'topic': topic,
      'device_id': deviceId,
      'display_name': displayName,
      'device_label': deviceLabel,
      'max_inline_bytes': maxInlineBytes,
      'auto_watch': autoWatch,
      'auto_apply': autoApply,
      'history_retention': historyRetention.name,
      'transfer_provider': transferProvider,
      'transfer_ref': transferRef,
    };
  }
}

class ClipboardActivity {
  const ClipboardActivity({
    required this.id,
    required this.kind,
    required this.title,
    required this.detail,
    required this.deviceLabel,
    required this.byteSize,
    required this.hashPrefix,
    required this.timestamp,
  });

  final String id;
  final ActivityKind kind;
  final String title;
  final String detail;
  final String deviceLabel;
  final int byteSize;
  final String hashPrefix;
  final DateTime timestamp;
}

class ClipboardEngineState {
  const ClipboardEngineState({
    required this.connected,
    required this.loggedIn,
    required this.busy,
    required this.previewMode,
    required this.hubEndpoint,
    required this.authStage,
    required this.nodeId,
    required this.settings,
    required this.capability,
    required this.activities,
    required this.pendingEvent,
    required this.transferStatus,
    required this.lastError,
  });

  factory ClipboardEngineState.initial(PlatformCapability capability) {
    return ClipboardEngineState(
      connected: false,
      loggedIn: false,
      busy: false,
      previewMode: true,
      hubEndpoint: '127.0.0.1:9000',
      authStage: '等待连接',
      nodeId: null,
      settings: ClipboardSettings.defaults(),
      capability: capability,
      activities: const [],
      pendingEvent: null,
      transferStatus: null,
      lastError: '',
    );
  }

  final bool connected;
  final bool loggedIn;
  final bool busy;
  final bool previewMode;
  final String hubEndpoint;
  final String authStage;
  final int? nodeId;
  final ClipboardSettings settings;
  final PlatformCapability capability;
  final List<ClipboardActivity> activities;
  final PendingClipboardEvent? pendingEvent;
  final TransferStatus? transferStatus;
  final String lastError;

  ClipboardEngineState copyWith({
    bool? connected,
    bool? loggedIn,
    bool? busy,
    bool? previewMode,
    String? hubEndpoint,
    String? authStage,
    int? nodeId,
    bool clearNodeId = false,
    ClipboardSettings? settings,
    PlatformCapability? capability,
    List<ClipboardActivity>? activities,
    PendingClipboardEvent? pendingEvent,
    bool clearPendingEvent = false,
    TransferStatus? transferStatus,
    bool clearTransferStatus = false,
    String? lastError,
  }) {
    return ClipboardEngineState(
      connected: connected ?? this.connected,
      loggedIn: loggedIn ?? this.loggedIn,
      busy: busy ?? this.busy,
      previewMode: previewMode ?? this.previewMode,
      hubEndpoint: hubEndpoint ?? this.hubEndpoint,
      authStage: authStage ?? this.authStage,
      nodeId: clearNodeId ? null : nodeId ?? this.nodeId,
      settings: settings ?? this.settings,
      capability: capability ?? this.capability,
      activities: activities ?? this.activities,
      pendingEvent: clearPendingEvent
          ? null
          : pendingEvent ?? this.pendingEvent,
      transferStatus: clearTransferStatus
          ? null
          : transferStatus ?? this.transferStatus,
      lastError: lastError ?? this.lastError,
    );
  }
}

class PendingClipboardEvent {
  const PendingClipboardEvent({
    required this.eventId,
    required this.byteSize,
    required this.hashPrefix,
  });

  final String eventId;
  final int byteSize;
  final String hashPrefix;
}

class TransferStatus {
  const TransferStatus({
    required this.id,
    required this.state,
    required this.byteSize,
    required this.hashPrefix,
    required this.detail,
  });

  final String id;
  final String state;
  final int byteSize;
  final String hashPrefix;
  final String detail;
}

abstract final class EngineActions {
  static const connect = 'connect';
  static const setConfig = 'set_config';
  static const sendText = 'send_text';
  static const readClipboard = 'read_clipboard';
  static const applyEvent = 'apply_event';
  static const clearRecent = 'clear_recent';
  static const shutdown = 'shutdown';
}

abstract final class EngineEvents {
  static const statusChanged = 'status.changed';
  static const activityUpdated = 'activity.updated';
  static const transferUpdated = 'transfer.updated';
  static const clipboardReceived = 'clipboard.received';
  static const error = 'error';
}

class EngineCommand {
  const EngineCommand({
    required this.id,
    required this.action,
    this.data = const {},
  });

  final String id;
  final String action;
  final Map<String, Object?> data;

  String encode() {
    return jsonEncode({'id': id, 'action': action, 'data': data});
  }
}

class EngineEvent {
  const EngineEvent({required this.name, this.data = const {}});

  final String name;
  final Map<String, Object?> data;

  String encode() {
    return jsonEncode({'name': name, 'data': data});
  }
}
