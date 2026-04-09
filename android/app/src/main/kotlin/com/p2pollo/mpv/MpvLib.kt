package com.p2pollo.mpv

import android.content.Context
import android.util.Log
import android.view.Surface
import java.util.concurrent.ArrayBlockingQueue

/**
 * JNI wrapper para mpv-android.
 * Basado en mpv-android/app/src/main/java/is/xyz/mpv/MPVLib.kt (Apache 2.0).
 *
 * Requiere libmpv.so en jniLibs/arm64-v8a/.
 * Build: https://github.com/mpv-android/mpv-android
 */
object MpvLib {

    private const val TAG = "MpvLib"

    external fun create(ctx: Context)
    external fun init()
    external fun destroy()
    external fun attachSurface(surface: Surface)
    external fun detachSurface()
    external fun command(args: Array<String?>)
    external fun setOptionString(name: String, value: String)
    external fun setPropertyString(name: String, value: String)
    external fun getPropertyString(name: String): String?
    external fun getPropertyBoolean(name: String): Boolean
    external fun getPropertyDouble(name: String): Double
    external fun getPropertyInt(name: String): Int
    external fun setPropertyBoolean(name: String, value: Boolean)
    external fun setPropertyDouble(name: String, value: Double)
    external fun setPropertyInt(name: String, value: Int)
    external fun observeProperty(name: String, format: Int)
    external fun processPendingEvents()

    // Event callback types
    const val EVENT_NONE = 0
    const val EVENT_SHUTDOWN = 1
    const val EVENT_LOG_MESSAGE = 2
    const val EVENT_GET_PROPERTY_REPLY = 3
    const val EVENT_SET_PROPERTY_REPLY = 4
    const val EVENT_COMMAND_REPLY = 5
    const val EVENT_START_FILE = 6
    const val EVENT_END_FILE = 7
    const val EVENT_FILE_LOADED = 8
    const val EVENT_IDLE = 11
    const val EVENT_TICK = 14
    const val EVENT_CLIENT_MESSAGE = 16
    const val EVENT_VIDEO_RECONFIG = 17
    const val EVENT_AUDIO_RECONFIG = 18
    const val EVENT_SEEK = 20
    const val EVENT_PLAYBACK_RESTART = 21
    const val EVENT_PROPERTY_CHANGE = 22
    const val EVENT_QUEUE_OVERFLOW = 24
    const val EVENT_HOOK = 25

    // Property format codes
    const val FORMAT_NONE = 0
    const val FORMAT_STRING = 1
    const val FORMAT_OSD_STRING = 2
    const val FORMAT_FLAG = 3
    const val FORMAT_INT64 = 4
    const val FORMAT_DOUBLE = 5
    const val FORMAT_NODE = 6

    private val observers = mutableListOf<EventObserver>()

    // Cola de wakeups. El hilo de eventos espera aquí y llama processPendingEvents().
    private val wakeupQueue = ArrayBlockingQueue<Unit>(1)
    @Volatile private var eventThreadRunning = false
    private var eventThread: Thread? = null

    fun startEventThread() {
        if (eventThreadRunning) return
        eventThreadRunning = true
        eventThread = Thread({
            while (eventThreadRunning) {
                wakeupQueue.poll(500, java.util.concurrent.TimeUnit.MILLISECONDS)
                if (eventThreadRunning) processPendingEvents()
            }
        }, "mpv-events").also { it.isDaemon = true; it.start() }
    }

    fun stopEventThread() {
        eventThreadRunning = false
        wakeupQueue.offer(Unit)
        eventThread?.join(1000)
        eventThread = null
    }

    interface EventObserver {
        fun eventProperty(property: String)
        fun eventProperty(property: String, value: Long)
        fun eventProperty(property: String, value: Boolean)
        fun eventProperty(property: String, value: Double)
        fun eventProperty(property: String, value: String)
        fun event(eventId: Int)
    }

    fun addObserver(o: EventObserver) = observers.add(o)
    fun removeObserver(o: EventObserver) = observers.remove(o)

    // Called from JNI
    @JvmStatic
    fun eventProperty(property: String, value: Long) {
        observers.forEach { it.eventProperty(property, value) }
    }

    @JvmStatic
    fun eventProperty(property: String, value: Boolean) {
        observers.forEach { it.eventProperty(property, value) }
    }

    @JvmStatic
    fun eventProperty(property: String, value: Double) {
        observers.forEach { it.eventProperty(property, value) }
    }

    @JvmStatic
    fun eventProperty(property: String, value: String) {
        observers.forEach { it.eventProperty(property, value) }
    }

    @JvmStatic
    fun eventProperty(property: String) {
        observers.forEach { it.eventProperty(property) }
    }

    @JvmStatic
    fun event(eventId: Int) {
        observers.forEach { it.event(eventId) }
    }

    @JvmStatic
    fun logMessage(prefix: String, level: Int, text: String) {
        Log.d(TAG, "[$prefix] $text")
    }

    // Llamado desde el wakeup callback nativo (solo notifica, no llama API de mpv)
    @JvmStatic
    fun wakeup() {
        wakeupQueue.offer(Unit)
    }

    init {
        // mpv primero (dependencia de mpvjni), luego el bridge JNI
        System.loadLibrary("mpv")
        System.loadLibrary("mpvjni")
    }
}
