package com.p2pollo

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.os.IBinder
import android.util.Log
import golib.Golib
import kotlin.concurrent.thread

class BackendService : Service() {

    companion object {
        private const val TAG = "BackendService"
        private const val NOTIF_ID = 1
        private const val CHANNEL_ID = "p2pollo_backend"
    }

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        Log.i(TAG, "Iniciando backend Go...")
        startForeground(NOTIF_ID, buildNotification())

        thread(name = "go-backend") {
            val storageDir = filesDir.absolutePath
            Log.i(TAG, "storageDir = $storageDir")
            val result = Golib.start(storageDir)
            if (result.isNotEmpty()) {
                Log.e(TAG, "Error al iniciar backend: $result")
            } else {
                Log.i(TAG, "Backend Go corriendo en localhost:9876")
            }
        }

        return START_STICKY
    }

    override fun onDestroy() {
        super.onDestroy()
        Log.i(TAG, "Deteniendo backend Go...")
        thread(name = "go-backend-stop") {
            Golib.stop()
            Log.i(TAG, "Backend Go detenido")
        }
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun createNotificationChannel() {
        val channel = NotificationChannel(
            CHANNEL_ID,
            "p2pollo backend",
            NotificationManager.IMPORTANCE_LOW
        ).apply {
            description = "Servicio de streaming P2P"
            setShowBadge(false)
        }
        val manager = getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(channel)
    }

    private fun buildNotification(): Notification =
        Notification.Builder(this, CHANNEL_ID)
            .setContentTitle("p2pollo")
            .setContentText("Streaming P2P activo")
            .setSmallIcon(android.R.drawable.ic_media_play)
            .setOngoing(true)
            .build()
}
