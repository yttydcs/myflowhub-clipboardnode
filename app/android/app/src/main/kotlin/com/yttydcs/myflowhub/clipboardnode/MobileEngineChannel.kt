package com.yttydcs.myflowhub.clipboardnode

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.os.Handler
import android.os.Looper
import android.util.Log
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import org.json.JSONObject
import java.lang.reflect.InvocationTargetException

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
    private val clipboardManager: ClipboardManager by lazy {
        context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
    }
    private val clipboardListener = ClipboardManager.OnPrimaryClipChangedListener {
        onPrimaryClipChanged()
    }
    private val mainHandler = Handler(Looper.getMainLooper())
    private val appliedTextPoller = object : Runnable {
        override fun run() {
            pollAppliedText()
            if (pollingAppliedText) {
                mainHandler.postDelayed(this, 500)
            }
        }
    }
    private var listening = false
    private var pollingAppliedText = false
    private var suppressNextClipboardChange = false

    init {
        channel.setMethodCallHandler(this)
    }

    fun handleIntent(intent: Intent?) {
        val sharedText = extractSharedText(intent) ?: return
        try {
            bridge.setClipboardText(sharedText)
        } catch (t: Throwable) {
            Log.w(TAG, "shared text handoff failed", t)
            // The Flutter UI can still read the system clipboard manually if gomobile is unavailable.
        }
    }

    override fun onMethodCall(call: MethodCall, result: MethodChannel.Result) {
        try {
            when (call.method) {
                "start" -> {
                    val config = JSONObject(call.arguments as? Map<*, *> ?: emptyMap<Any, Any>()).toString()
                    val status = bridge.start(config, context.filesDir.absolutePath)
                    syncClipboardPolicy(config, status)
                    result.success(status)
                }
                "updateConfig" -> {
                    val config = JSONObject(call.arguments as? Map<*, *> ?: emptyMap<Any, Any>()).toString()
                    val status = bridge.updateConfig(config)
                    syncClipboardPolicy(config, status)
                    result.success(status)
                }
                "stop" -> {
                    stopClipboardPolicy()
                    result.success(bridge.stop())
                }
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
                    val decision = bridge.applyEvent(args["event_id"] as? String ?: "")
                    applyDecisionToSystemClipboard(decision)
                    result.success(decision)
                }
                "status" -> result.success(bridge.status())
                else -> result.notImplemented()
            }
        } catch (t: Throwable) {
            val error = t.unwrapInvocationTarget()
            result.error("clipboardnode_engine", error.message ?: error.toString(), null)
        }
    }

    private fun readSystemClipboard(): String {
        val item = clipboardManager.primaryClip?.takeIf { it.itemCount > 0 }?.getItemAt(0)
            ?: throw IllegalStateException("clipboard does not contain text")
        return item.coerceToText(context)?.toString()
            ?: throw IllegalStateException("clipboard does not contain text")
    }

    private fun syncClipboardPolicy(configJson: String, statusJson: String) {
        val policy = try {
            val config = JSONObject(configJson)
            val status = JSONObject(statusJson)
            val unavailable = status.optBoolean("running", true) == false &&
                status.optString("last_error").isNotBlank()
            ClipboardPolicy(
                enabled = config.optBoolean("enabled", false) && !unavailable,
                autoWatch = config.optBoolean("auto_watch", false),
                autoApply = config.optBoolean("auto_apply", false),
            )
        } catch (_: Throwable) {
            ClipboardPolicy(enabled = false, autoWatch = false, autoApply = false)
        }
        if (policy.enabled && policy.autoWatch) {
            startClipboardListener()
        } else {
            stopClipboardListener()
        }
        if (policy.enabled && policy.autoApply) {
            startAppliedTextPolling()
        } else {
            stopAppliedTextPolling()
        }
    }

    private fun stopClipboardPolicy() {
        stopClipboardListener()
        stopAppliedTextPolling()
    }

    private fun startClipboardListener() {
        if (listening) {
            return
        }
        clipboardManager.addPrimaryClipChangedListener(clipboardListener)
        listening = true
    }

    private fun stopClipboardListener() {
        if (!listening) {
            return
        }
        clipboardManager.removePrimaryClipChangedListener(clipboardListener)
        listening = false
    }

    private fun onPrimaryClipChanged() {
        if (!listening) {
            return
        }
        if (suppressNextClipboardChange) {
            suppressNextClipboardChange = false
            return
        }
        val text = try {
            readSystemClipboard()
        } catch (_: Throwable) {
            return
        }
        if (text.isBlank()) {
            return
        }
        try {
            bridge.setClipboardText(text)
            bridge.readClipboard()
        } catch (t: Throwable) {
            Log.w(TAG, "foreground clipboard watcher publish failed", t)
        }
    }

    private fun startAppliedTextPolling() {
        if (pollingAppliedText) {
            return
        }
        pollingAppliedText = true
        mainHandler.post(appliedTextPoller)
    }

    private fun stopAppliedTextPolling() {
        if (!pollingAppliedText) {
            return
        }
        pollingAppliedText = false
        mainHandler.removeCallbacks(appliedTextPoller)
    }

    private fun pollAppliedText() {
        val text = try {
            bridge.takeLastAppliedText()
        } catch (t: Throwable) {
            Log.w(TAG, "applied text polling failed", t.unwrapInvocationTarget())
            stopAppliedTextPolling()
            return
        }
        if (text.isNotEmpty()) {
            setSystemClipboard(text)
        }
    }

    private fun applyDecisionToSystemClipboard(decisionJson: String) {
        val action = try {
            JSONObject(decisionJson).optString("Action")
        } catch (_: Throwable) {
            ""
        }
        if (action != "remote_applied") {
            return
        }
        val text = bridge.takeLastAppliedText()
        if (text.isEmpty()) {
            throw IllegalStateException("remote applied text is unavailable")
        }
        setSystemClipboard(text)
    }

    private fun setSystemClipboard(text: String) {
        suppressNextClipboardChange = true
        clipboardManager.setPrimaryClip(ClipData.newPlainText("ClipboardNode", text))
        mainHandler.postDelayed({ suppressNextClipboardChange = false }, 1000)
    }

    private data class ClipboardPolicy(
        val enabled: Boolean,
        val autoWatch: Boolean,
        val autoApply: Boolean,
    )

    private companion object {
        private const val TAG = "ClipboardNodeMobile"
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
    fun takeLastAppliedText(): String
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
    override fun takeLastAppliedText(): String = ""
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
        return invoke("start", arrayOf(String::class.java, String::class.java), configJson, workDir)
    }

    override fun updateConfig(configJson: String): String {
        return invoke("updateConfig", arrayOf(String::class.java), configJson)
    }

    override fun stop(): String {
        return invoke("stop", emptyArray())
    }

    override fun sendText(text: String): String {
        return invoke("sendText", arrayOf(String::class.java), text)
    }

    override fun readClipboard(): String {
        return invoke("readClipboard", emptyArray())
    }

    override fun applyEvent(eventId: String): String {
        return invoke("applyEvent", arrayOf(String::class.java), eventId)
    }

    override fun setClipboardText(text: String): String {
        return invoke("setClipboardText", arrayOf(String::class.java), text)
    }

    override fun takeLastAppliedText(): String {
        return invoke("takeLastAppliedText", emptyArray())
    }

    override fun status(): String {
        return invoke("status", emptyArray())
    }

    private fun invoke(name: String, parameterTypes: Array<Class<*>>, vararg args: Any): String {
        val method = cls.getMethod(name, *parameterTypes)
        return try {
            method.invoke(null, *args) as? String ?: ""
        } catch (e: InvocationTargetException) {
            throw e.targetException ?: e
        }
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

private fun Throwable.unwrapInvocationTarget(): Throwable {
    return if (this is InvocationTargetException) {
        targetException ?: this
    } else {
        this
    }
}
