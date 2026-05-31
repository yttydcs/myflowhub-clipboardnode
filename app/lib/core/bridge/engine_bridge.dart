import 'engine_contract.dart';

abstract interface class ClipboardEngineBridge {
  ClipboardEngineState get currentState;

  Stream<ClipboardEngineState> get states;

  Future<void> connect();

  Future<void> disconnect();

  Future<void> updateSettings(ClipboardSettings settings);

  Future<void> sendText(String text);

  Future<void> clearRecent();

  Future<void> dispose();
}
