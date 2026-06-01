import Flutter
import Foundation
import UIKit

#if canImport(Nodemobile)
import Nodemobile
#endif

final class MobileEngineChannel {
  private let channel: FlutterMethodChannel

  init(controller: FlutterViewController) {
    channel = FlutterMethodChannel(
      name: "com.yttydcs.myflowhub.clipboardnode/engine",
      binaryMessenger: controller.binaryMessenger
    )
    channel.setMethodCallHandler(handle)
  }

  private func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    do {
      switch call.method {
      case "start":
        let config = try configJSON(call.arguments)
        result(try start(configJSON: config, workDir: workDir()))
      case "updateConfig":
        result(try updateConfig(configJSON: try configJSON(call.arguments)))
      case "stop":
        result(try stop())
      case "sendText":
        result(try sendText(text: stringArgument(call.arguments, "text")))
      case "readClipboard":
        let text = UIPasteboard.general.string ?? ""
        if text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
          throw EngineError("clipboard does not contain text")
        }
        _ = setClipboardText(text)
        result(try readClipboard())
      case "applyEvent":
        result(try applyEvent(eventID: stringArgument(call.arguments, "event_id")))
      case "status":
        result(status())
      default:
        result(FlutterMethodNotImplemented)
      }
    } catch {
      result(FlutterError(code: "clipboardnode_engine", message: error.localizedDescription, details: nil))
    }
  }

  private func configJSON(_ arguments: Any?) throws -> String {
    let object = arguments as? [String: Any] ?? [:]
    let data = try JSONSerialization.data(withJSONObject: object, options: [])
    return String(data: data, encoding: .utf8) ?? "{}"
  }

  private func stringArgument(_ arguments: Any?, _ key: String) throws -> String {
    let object = arguments as? [String: Any] ?? [:]
    return object[key] as? String ?? ""
  }

  private func workDir() throws -> String {
    guard let dir = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first else {
      throw EngineError("application support directory is unavailable")
    }
    try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
    return dir.path
  }

  private func start(configJSON: String, workDir: String) throws -> String {
    #if canImport(Nodemobile)
    var error: NSError?
    let value = MFHNodemobileStart(configJSON, workDir, &error)
    if let error { throw error }
    return value
    #else
    throw EngineError("iOS live engine requires Nodemobile.xcframework")
    #endif
  }

  private func updateConfig(configJSON: String) throws -> String {
    #if canImport(Nodemobile)
    var error: NSError?
    let value = MFHNodemobileUpdateConfig(configJSON, &error)
    if let error { throw error }
    return value
    #else
    return status()
    #endif
  }

  private func stop() throws -> String {
    #if canImport(Nodemobile)
    var error: NSError?
    let value = MFHNodemobileStop(&error)
    if let error { throw error }
    return value
    #else
    return status()
    #endif
  }

  private func sendText(text: String) throws -> String {
    #if canImport(Nodemobile)
    var error: NSError?
    let value = MFHNodemobileSendText(text, &error)
    if let error { throw error }
    return value
    #else
    throw EngineError("iOS live engine requires Nodemobile.xcframework")
    #endif
  }

  private func readClipboard() throws -> String {
    #if canImport(Nodemobile)
    var error: NSError?
    let value = MFHNodemobileReadClipboard(&error)
    if let error { throw error }
    return value
    #else
    throw EngineError("iOS live engine requires Nodemobile.xcframework")
    #endif
  }

  private func applyEvent(eventID: String) throws -> String {
    #if canImport(Nodemobile)
    var error: NSError?
    let value = MFHNodemobileApplyEvent(eventID, &error)
    if let error { throw error }
    return value
    #else
    throw EngineError("iOS live engine requires Nodemobile.xcframework")
    #endif
  }

  private func setClipboardText(_ text: String) -> String {
    #if canImport(Nodemobile)
    MFHNodemobileSetClipboardText(text)
    return status()
    #else
    return status()
    #endif
  }

  private func status() -> String {
    #if canImport(Nodemobile)
    return MFHNodemobileStatus()
    #else
    return "{\"running\":false,\"last_error\":\"iOS live engine requires Nodemobile.xcframework\"}"
    #endif
  }
}

private struct EngineError: LocalizedError {
  let message: String

  init(_ message: String) {
    self.message = message
  }

  var errorDescription: String? {
    message
  }
}
