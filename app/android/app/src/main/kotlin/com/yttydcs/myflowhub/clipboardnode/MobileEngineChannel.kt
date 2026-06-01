package com.yttydcs.myflowhub.clipboardnode

import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import org.json.JSONObject

class MobileEngineChannel(
    flutterEngine: FlutterEngine,
    private val context: Context,
) : MethodChannel.MethodCallHandler {
    private val channel = MethodChannel(
        flutterEngine.dartExecutor.binaryMessenger,
        "com.yttydcs.myflowhub.clipboardnode/engine",
    )
    private val bridge: NodeBridge = try {
        GoNodeBridge()
    } catch (t: Throwable) {
        StubNodeBridge(t.message ?: t.toString())
    }

    init {
        channel.setMethodCallHandler(this)
    }

    fun handleIntent(intent: Intent?) {
        val sharedText = extractSharedText(intent) ?: return
        try {
            bridge.setClipboardText(sharedText)
        } catch (_: Throwable) {
            // The Flutter UI can still read the system clipboard manually if gomobile is unavailable.
        }
    }

    override fun onMethodCall(call: MethodCall, result: MethodChannel.Result) {
        try {
            when (call.method) {
                "start" -> {
                    val config = JSONObject(call.arguments as? Map<*, *> ?: emptyMap<Any, Any>()).toString()
                    result.success(bridge.start(config, context.filesDir.absolutePath))
                }
                "updateConfig" -> {
                    val config = JSONObject(call.arguments as? Map<*, *> ?: emptyMap<Any, Any>()).toString()
                    result.success(bridge.updateConfig(config))
                }
                "stop" -> result.success(bridge.stop())
                "sendText" -> {
                    val args = call.arguments as? Map<*, *> ?: emptyMap<Any, Any>()
                    result.success(bridge.sendText(args["text"] as? String ?: ""))
                }
                "readClipboard" -> {
                    val text = readSystemClipboard()
                    bridge.setClipboardText(text)
                    result.success(bridge.readClipboard())
                }
                "applyEvent" -> {
                    val args = call.arguments as? Map<*, *> ?: emptyMap<Any, Any>()
                    result.success(bridge.applyEvent(args["event_id"] as? String ?: ""))
                }
                "status" -> result.success(bridge.status())
                else -> result.notImplemented()
            }
        } catch (t: Throwable) {
            result.error("clipboardnode_engine", t.message ?: t.toString(), null)
        }
    }

    private fun readSystemClipboard(): String {
        val manager = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        val item = manager.primaryClip?.takeIf { it.itemCount > 0 }?.getItemAt(0)
            ?: throw IllegalStateException("clipboard does not contain text")
        return item.coerceToText(context)?.toString()
            ?: throw IllegalStateException("clipboard does not contain text")
    }

    private fun extractSharedText(intent: Intent?): String? {
        if (intent == null) {
            return null
        }
        if (intent.action != Intent.ACTION_SEND) {
            return null
        }
        if (!intent.type.orEmpty().startsWith("text/")) {
            return null
        }
        return intent.getStringExtra(Intent.EXTRA_TEXT)
    }
}

interface NodeBridge {
    fun start(configJson: String, workDir: String): String
    fun updateConfig(configJson: String): String
    fun stop(): String
    fun sendText(text: String): String
    fun readClipboard(): String
    fun applyEvent(eventId: String): String
    fun setClipboardText(text: String): String
    fun status(): String
}

class StubNodeBridge(private val initError: String) : NodeBridge {
    override fun start(configJson: String, workDir: String): String = status()
    override fun updateConfig(configJson: String): String = status()
    override fun stop(): String = status()
    override fun sendText(text: String): String {
        throw IllegalStateException(initError)
    }
    override fun readClipboard(): String {
        throw IllegalStateException(initError)
    }
    override fun applyEvent(eventId: String): String {
        throw IllegalStateException(initError)
    }
    override fun setClipboardText(text: String): String = status()
    override fun status(): String = JSONObject(
        mapOf(
            "running" to false,
            "last_error" to initError,
        ),
    ).toString()
}

class GoNodeBridge : NodeBridge {
    private val cls = resolveClass()

    override fun start(configJson: String, workDir: String): String {
        return invoke("Start", arrayOf(String::class.java, String::class.java), configJson, workDir)
    }

    override fun updateConfig(configJson: String): String {
        return invoke("UpdateConfig", arrayOf(String::class.java), configJson)
    }

    override fun stop(): String {
        return invoke("Stop", emptyArray())
    }

    override fun sendText(text: String): String {
        return invoke("SendText", arrayOf(String::class.java), text)
    }

    override fun readClipboard(): String {
        return invoke("ReadClipboard", emptyArray())
    }

    override fun applyEvent(eventId: String): String {
        return invoke("ApplyEvent", arrayOf(String::class.java), eventId)
    }

    override fun setClipboardText(text: String): String {
        return invoke("SetClipboardText", arrayOf(String::class.java), text)
    }

    override fun status(): String {
        return invoke("Status", emptyArray())
    }

    private fun invoke(name: String, parameterTypes: Array<Class<*>>, vararg args: Any): String {
        val method = cls.getMethod(name, *parameterTypes)
        return method.invoke(null, *args) as? String ?: ""
    }

    private fun resolveClass(): Class<*> {
        val names = listOf(
            "com.myflowhub.gomobile.nodemobile.Nodemobile",
            "com.myflowhub.gomobile.Nodemobile",
            "nodemobile.Nodemobile",
            "go.nodemobile.Nodemobile",
        )
        var last: ClassNotFoundException? = null
        for (name in names) {
            try {
                return Class.forName(name)
            } catch (e: ClassNotFoundException) {
                last = e
            }
        }
        throw last ?: ClassNotFoundException("nodemobile binding not found")
    }
}
