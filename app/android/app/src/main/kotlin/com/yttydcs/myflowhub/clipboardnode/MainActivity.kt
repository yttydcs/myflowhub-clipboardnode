package com.yttydcs.myflowhub.clipboardnode

import android.content.Intent
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine

class MainActivity : FlutterActivity() {
    private var mobileEngineChannel: MobileEngineChannel? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        mobileEngineChannel = MobileEngineChannel(flutterEngine, applicationContext)
        mobileEngineChannel?.handleIntent(intent)
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        mobileEngineChannel?.handleIntent(intent)
    }
}
