package com.p2pollo.ui.player

import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.p2pollo.data.Repository
import com.p2pollo.mpv.MpvLib
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch

data class PlayerState(
    val isPreparing: Boolean = true,
    val streamUrl: String = "",
    val isPaused: Boolean = false,
    val position: Double = 0.0,
    val duration: Double = 0.0,
    val volume: Int = 80,
    val downloadProgress: Double = 0.0,
    val downloadSpeed: Long = 0,
    val peers: Int = 0,
    val subTrack: Int = 0,  // 0 = sin subtítulos, >0 = track activo
    val error: String? = null
)

class PlayerViewModel : ViewModel(), MpvLib.EventObserver {

    private val TAG = "PlayerViewModel"

    private val _state = MutableStateFlow(PlayerState())
    val state: StateFlow<PlayerState> = _state

    private var progressJob: Job? = null
    private var positionJob: Job? = null

    init {
        MpvLib.addObserver(this)
    }

    override fun onCleared() {
        MpvLib.removeObserver(this)
        progressJob?.cancel()
        positionJob?.cancel()
        super.onCleared()
    }

    fun startStream(magnetUri: String, fileIndex: Int) {
        viewModelScope.launch {
            _state.value = PlayerState(isPreparing = true)

            // Iniciar stream en Go backend
            Repository.play(magnetUri, fileIndex)
                .onFailure { e ->
                    _state.value = _state.value.copy(error = e.message)
                    return@launch
                }

            // Iniciar polling de progreso
            startProgressPolling()

            // Esperar a que stream esté listo, polling cada segundo.
            // resp.path es un path local del filesystem; el servidor Go lo sirve via /api/stream.
            Repository.streamPathFlow().collect { resp ->
                if (resp.ready) {
                    val url = "http://127.0.0.1:9876/api/stream"
                    Log.i(TAG, "Stream listo: $url")
                    _state.value = _state.value.copy(
                        streamUrl = url,
                        isPreparing = false
                    )
                    startPositionPolling()
                }
            }
        }
    }

    fun stopStream() {
        progressJob?.cancel()
        positionJob?.cancel()
        viewModelScope.launch {
            Repository.stopStream()
        }
        MpvLib.command(arrayOf("stop"))
    }

    fun togglePlayPause() {
        val paused = !_state.value.isPaused
        MpvLib.setPropertyBoolean("pause", paused)
        _state.value = _state.value.copy(isPaused = paused)
    }

    fun seek(seconds: Double) {
        MpvLib.command(arrayOf("seek", seconds.toString(), "absolute"))
    }

    fun setVolume(volume: Int) {
        MpvLib.setPropertyInt("volume", volume)
        _state.value = _state.value.copy(volume = volume)
    }

    fun cycleSubtitles() {
        MpvLib.command(arrayOf("cycle", "sub"))
    }

    private fun startProgressPolling() {
        progressJob = viewModelScope.launch {
            while (true) {
                Repository.getProgress().onSuccess { p ->
                    _state.value = _state.value.copy(
                        downloadProgress = p.percent,
                        downloadSpeed = (p.speedMBps * 1_048_576).toLong(),
                        peers = p.peers
                    )
                }
                delay(1_000)
            }
        }
    }

    private fun startPositionPolling() {
        positionJob = viewModelScope.launch {
            while (true) {
                val pos = MpvLib.getPropertyDouble("time-pos")
                val dur = MpvLib.getPropertyDouble("duration")
                val sid = MpvLib.getPropertyInt("sid")
                _state.value = _state.value.copy(
                    position = if (pos.isNaN() || pos < 0) 0.0 else pos,
                    duration = if (dur.isNaN() || dur < 0) 0.0 else dur,
                    subTrack = sid
                )
                delay(500)
            }
        }
    }

    // MpvLib.EventObserver callbacks

    override fun eventProperty(property: String) {}

    override fun eventProperty(property: String, value: Long) {
        if (property == "time-pos") {
            _state.value = _state.value.copy(position = value.toDouble())
        }
    }

    override fun eventProperty(property: String, value: Boolean) {
        if (property == "pause") {
            _state.value = _state.value.copy(isPaused = value)
        }
    }

    override fun eventProperty(property: String, value: Double) {
        when (property) {
            "time-pos" -> _state.value = _state.value.copy(position = value)
            "duration" -> _state.value = _state.value.copy(duration = value)
        }
    }

    override fun eventProperty(property: String, value: String) {}

    override fun event(eventId: Int) {
        when (eventId) {
            MpvLib.EVENT_FILE_LOADED -> {
                Log.i(TAG, "Archivo cargado en mpv")
                _state.value = _state.value.copy(isPreparing = false)
            }
            MpvLib.EVENT_END_FILE -> {
                Log.i(TAG, "Reproducción terminada")
            }
        }
    }
}
