package com.p2pollo.ui.player

import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.p2pollo.data.Repository
import com.p2pollo.mpv.MpvLib
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch

data class SubTrack(val id: Int, val lang: String, val label: String)

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
    val subTracks: List<SubTrack> = emptyList(),
    val showSubMenu: Boolean = false,
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
            // Usamos file:// directo al archivo temporal en lugar de HTTP para evitar
            // el loop de reconnects de mpv al buscar el índice MKV al final del archivo,
            // que causaba que MediaCodec HEVC no pudiera inicializarse (surface NULL).
            Repository.streamPathFlow().collect { resp ->
                if (resp.ready && resp.path.isNotBlank()) {
                    val url = "file://${resp.path}"
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
        viewModelScope.launch(Dispatchers.IO) {
            Repository.stopStream()
            // No llamar MpvLib.command("stop") aquí — mpv_terminate_destroy() en
            // onDispose ya maneja el shutdown completo. Llamarlo antes causa
            // "assertion !queue->lock_requests failed" (SIGABRT) en dispatch.c
            // cuando destroy() llega mientras el stop command aún tiene la queue bloqueada.
        }
    }

    fun togglePlayPause() {
        val paused = !_state.value.isPaused
        _state.value = _state.value.copy(isPaused = paused)
        viewModelScope.launch(Dispatchers.IO) {
            MpvLib.setPropertyBoolean("pause", paused)
        }
    }

    fun seek(seconds: Double) {
        viewModelScope.launch(Dispatchers.IO) {
            MpvLib.command(arrayOf("seek", seconds.toString(), "absolute"))
        }
    }

    fun setVolume(volume: Int) {
        _state.value = _state.value.copy(volume = volume)
        viewModelScope.launch(Dispatchers.IO) {
            MpvLib.setPropertyInt("volume", volume)
        }
    }

    fun toggleSubMenu() {
        _state.value = _state.value.copy(showSubMenu = !_state.value.showSubMenu)
    }

    fun dismissSubMenu() {
        _state.value = _state.value.copy(showSubMenu = false)
    }

    fun selectSubtitle(id: Int) {
        _state.value = _state.value.copy(subTrack = id, showSubMenu = false)
        viewModelScope.launch(Dispatchers.IO) {
            if (id == 0) {
                // "no" desactiva subtítulos; setPropertyString porque sid acepta "no" como string
                MpvLib.command(arrayOf("set", "sid", "no"))
                MpvLib.setPropertyBoolean("sub-visibility", false)
            } else {
                // Usar el comando "set" en lugar de setPropertyInt para que mpv maneje
                // correctamente la conversión del tipo de la propiedad sid.
                MpvLib.command(arrayOf("set", "sid", id.toString()))
                // sub-visibility debe estar en true explícitamente — setPropertyInt en sid
                // no garantiza que la visibilidad se active si estaba en false previamente.
                MpvLib.setPropertyBoolean("sub-visibility", true)
            }
        }
    }

    private fun loadSubtitleTracks() {
        viewModelScope.launch(Dispatchers.IO) {
            val json = MpvLib.getPropertyString("track-list") ?: return@launch
            val tracks = parseSubtitleTracks(json)
            _state.value = _state.value.copy(subTracks = tracks)
        }
    }

    private fun parseSubtitleTracks(json: String): List<SubTrack> {
        return try {
            val arr = org.json.JSONArray(json)
            val subs = mutableListOf<SubTrack>()
            for (i in 0 until arr.length()) {
                val obj = arr.getJSONObject(i)
                if (obj.optString("type") != "sub") continue
                val id = obj.getInt("id")
                val lang = obj.optString("lang", "")
                val title = obj.optString("title", "")
                val label = when {
                    title.isNotBlank() -> title
                    lang.isNotBlank() -> lang.uppercase()
                    else -> "Track $id"
                }
                subs.add(SubTrack(id, lang, label))
            }
            subs
        } catch (_: Exception) {
            emptyList()
        }
    }

    /** Pausa o reanuda el polling de posición según visibilidad de controles. */
    fun setPositionPollingEnabled(enabled: Boolean) {
        if (enabled) {
            if (positionJob?.isActive != true) startPositionPolling()
        } else {
            positionJob?.cancel()
            positionJob = null
        }
    }

    private fun startProgressPolling() {
        progressJob = viewModelScope.launch(Dispatchers.IO) {
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
        positionJob = viewModelScope.launch(Dispatchers.IO) {
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
                loadSubtitleTracks()
            }
            MpvLib.EVENT_END_FILE -> {
                Log.i(TAG, "Reproducción terminada")
            }
        }
    }
}
