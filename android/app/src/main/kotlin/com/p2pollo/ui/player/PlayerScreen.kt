package com.p2pollo.ui.player

import android.graphics.SurfaceTexture
import android.view.Surface
import android.view.TextureView
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.foundation.focusable
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.activity.compose.BackHandler
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.compose.foundation.clickable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.onFocusChanged
import androidx.tv.material3.ExperimentalTvMaterial3Api
import android.util.Log
import android.view.KeyEvent
import com.p2pollo.PlayerKeyEventBus
import com.p2pollo.mpv.MpvLib
import kotlinx.coroutines.delay
import java.io.File

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun PlayerScreen(
    magnetUri: String,
    fileIndex: Int,
    viewModel: PlayerViewModel = viewModel(),
    onBack: () -> Unit
) {
    val context = LocalContext.current
    val state by viewModel.state.collectAsState()
    var controlsVisible by remember { mutableStateOf(true) }
    // Elemento oculto que retiene el foco cuando los controles desaparecen
    val hiddenFocusRequester = remember { FocusRequester() }

    // Crear mpv al entrar. init() se llama en surfaceCreated DESPUÉS de attachSurface,
    // para que wid esté configurado antes de mpv_initialize().
    DisposableEffect(Unit) {
        MpvLib.create(context)
        // Configurar fonts para subtítulos ANTES de init() / mpv_initialize().
        // Copia un font del sistema al directorio privado de la app y apunta
        // sub-fonts-dir a él, para que libass pueda renderizar sin fontconfig.
        setupMpvFonts(context)
        MpvLib.startEventThread()
        onDispose {
            viewModel.stopStream()
            MpvLib.stopEventThread()
            MpvLib.destroy()
        }
    }

    LaunchedEffect(magnetUri) {
        viewModel.startStream(magnetUri, fileIndex)
    }

    // Escuchar teclas desde la Activity (captura incluso cuando TextureView tiene el foco).
    // Cualquier tecla DOWN hace visible los controles.
    LaunchedEffect(Unit) {
        PlayerKeyEventBus.events.collect { event ->
            if (event.action == KeyEvent.ACTION_DOWN) {
                controlsVisible = true
            }
        }
    }

    // Foco inicial en el elemento oculto
    LaunchedEffect(Unit) {
        try { hiddenFocusRequester.requestFocus() } catch (_: Exception) {}
    }

    // Botón BACK del control remoto
    BackHandler {
        viewModel.stopStream()
        onBack()
    }

    // Auto-ocultar controles: devolver foco al elemento oculto
    LaunchedEffect(controlsVisible) {
        if (controlsVisible) {
            delay(4_000)
            controlsVisible = false
            try { hiddenFocusRequester.requestFocus() } catch (_: Exception) {}
        }
        // Pausar polling de posición cuando controles están ocultos (evita
        // recomposiciones de Slider cada 500ms mientras el video se reproduce)
        viewModel.setPositionPollingEnabled(controlsVisible)
    }

    // El Box NO es focusable — así el D-pad puede navegar a los hijos (botones).
    // onPreviewKeyEvent dispara cuando cualquier descendiente tiene el foco.
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.Black)
            .onPreviewKeyEvent {
                controlsVisible = true
                false
            }
            .pointerInput(Unit) {
                detectTapGestures { controlsVisible = true }
            }
    ) {
        // Elemento invisible que retiene el foco cuando los controles están ocultos
        Box(
            modifier = Modifier
                .size(1.dp)
                .focusRequester(hiddenFocusRequester)
                .focusable()
        )

        // Superficie de video mpv
        if (state.streamUrl.isNotBlank()) {
            MpvSurface(
                url = state.streamUrl,
                modifier = Modifier.fillMaxSize()
            )
        }

        // Overlay de carga / buffer
        if (state.isPreparing) {
            BufferingOverlay(
                progress = state.downloadProgress,
                speed = state.downloadSpeed,
                peers = state.peers
            )
        }

        // Controles — sin AnimatedVisibility para evitar spike de GPU durante transición
        if (controlsVisible && !state.isPreparing) {
            PlayerControls(
                state = state,
                onBack = {
                    viewModel.stopStream()
                    onBack()
                },
                onPlayPause = { viewModel.togglePlayPause() },
                onSeek = { seconds -> viewModel.seek(seconds) },
                onVolumeChange = { v -> viewModel.setVolume(v) },
                onCycleSubtitles = { viewModel.cycleSubtitles() },
                modifier = Modifier.align(Alignment.BottomCenter)
            )
        }
    }
}

@Composable
fun MpvSurface(
    url: String,
    modifier: Modifier = Modifier
) {
    // mpvReady se activa en surfaceCreated, después de attachSurface + init().
    // Solo entonces se llama a loadfile, garantizando el orden correcto:
    // attachSurface(wid) → init(mpv_initialize) → loadfile
    var mpvReady by remember { mutableStateOf(false) }

    AndroidView(
        factory = { ctx ->
            TextureView(ctx).apply {
                isFocusable = false
                isFocusableInTouchMode = false
                var currentSurface: Surface? = null
                surfaceTextureListener = object : TextureView.SurfaceTextureListener {
                    override fun onSurfaceTextureAvailable(st: SurfaceTexture, w: Int, h: Int) {
                        Log.i("MpvSurface", "onSurfaceTextureAvailable ${w}x${h}")
                        currentSurface = Surface(st)
                        MpvLib.attachSurface(currentSurface!!)
                        if (!mpvReady) {
                            MpvLib.init()
                            mpvReady = true
                        }
                    }
                    override fun onSurfaceTextureSizeChanged(st: SurfaceTexture, w: Int, h: Int) {
                        Log.i("MpvSurface", "onSurfaceTextureSizeChanged ${w}x${h}")
                    }
                    override fun onSurfaceTextureDestroyed(st: SurfaceTexture): Boolean {
                        Log.i("MpvSurface", "onSurfaceTextureDestroyed")
                        MpvLib.detachSurface()
                        currentSurface?.release()
                        currentSurface = null
                        return true
                    }
                    override fun onSurfaceTextureUpdated(st: SurfaceTexture) {}
                }
            }
        },
        modifier = modifier
    )

    // loadfile solo cuando mpv está inicializado con superficie válida
    LaunchedEffect(mpvReady, url) {
        Log.i("MpvSurface", "LaunchedEffect mpvReady=$mpvReady url=${url.take(40)}")
        if (mpvReady && url.isNotBlank()) {
            MpvLib.command(arrayOf("loadfile", url))
        }
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun BufferingOverlay(
    progress: Double,
    speed: Long,
    peers: Int
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(Color(0xCC000000)),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        CircularProgressIndicator(
            color = Color(0xFFFF6B35),
            modifier = Modifier.size(64.dp)
        )
        Spacer(Modifier.height(24.dp))
        Text(
            text = "Descargando buffer...",
            color = Color.White,
            fontSize = 18.sp
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = "${progress.toInt()}% • ${formatSpeed(speed)} • $peers peers",
            color = Color(0xFFBBB5AF),
            fontSize = 14.sp
        )
        Spacer(Modifier.height(16.dp))
        LinearProgressIndicator(
            progress = { (progress / 100.0).toFloat() },
            modifier = Modifier.width(320.dp),
            color = Color(0xFFFF6B35),
            trackColor = Color(0xFF3A3330)
        )
    }
}

/** Botón simple sin animaciones de TV — más ligero para SoC Amlogic. */
@Composable
private fun PlayerButton(
    onClick: () -> Unit,
    containerColor: Color = Color(0xFF2A2320),
    modifier: Modifier = Modifier,
    content: @Composable () -> Unit
) {
    var focused by remember { mutableStateOf(false) }
    Box(
        modifier = modifier
            .onFocusChanged { focused = it.isFocused }
            .clickable(onClick = onClick)
            .clip(RoundedCornerShape(4.dp))
            .background(if (focused) Color(0xFF5A5350) else containerColor)
            .padding(horizontal = 16.dp, vertical = 8.dp),
        contentAlignment = Alignment.Center
    ) {
        content()
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun PlayerControls(
    state: PlayerState,
    onBack: () -> Unit,
    onPlayPause: () -> Unit,
    onSeek: (Double) -> Unit,
    onVolumeChange: (Int) -> Unit,
    onCycleSubtitles: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .background(
                androidx.compose.ui.graphics.Brush.verticalGradient(
                    colors = listOf(Color.Transparent, Color(0xCC000000))
                )
            )
            .padding(horizontal = 48.dp, vertical = 32.dp)
    ) {
        // Barra de progreso — Box simple, sin Slider (demasiado pesado para Amlogic)
        if (state.duration > 0) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(4.dp)
                    .clip(RoundedCornerShape(2.dp))
                    .background(Color(0xFF3A3330))
            ) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth((state.position / state.duration).toFloat().coerceIn(0f, 1f))
                        .fillMaxHeight()
                        .background(Color(0xFFFF6B35))
                )
            }

            Spacer(Modifier.height(8.dp))

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                Text(
                    text = formatTime(state.position),
                    color = Color.White,
                    fontSize = 12.sp
                )
                Text(
                    text = formatTime(state.duration),
                    color = Color(0xFF8A837C),
                    fontSize = 12.sp
                )
            }
        }

        Spacer(Modifier.height(16.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            PlayerButton(onClick = onBack) {
                Text("← Salir", color = Color.White)
            }

            Row(horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                PlayerButton(onClick = { onSeek(state.position - 10.0) }) {
                    Text("−10s", color = Color.White)
                }

                PlayerButton(
                    onClick = onPlayPause,
                    containerColor = Color(0xFFFF6B35)
                ) {
                    Text(if (state.isPaused) "▶" else "⏸", color = Color.White, fontSize = 20.sp)
                }

                PlayerButton(onClick = { onSeek(state.position + 10.0) }) {
                    Text("+10s", color = Color.White)
                }

                PlayerButton(
                    onClick = onCycleSubtitles,
                    containerColor = if (state.subTrack > 0) Color(0xFFFF6B35) else Color(0xFF2A2320)
                ) {
                    Text("CC", color = Color.White)
                }
            }

            // Volumen como texto — el slider es demasiado pesado para este SoC
            Text(
                text = "Vol ${state.volume}%",
                color = Color(0xFF8A837C),
                fontSize = 12.sp
            )
        }
    }
}

/** Copia un font del sistema al directorio privado y configura sub-fonts-dir en mpv. */
private fun setupMpvFonts(context: android.content.Context) {
    val fontsDir = File(context.filesDir, "mpv-fonts")
    fontsDir.mkdirs()
    // Intentar copiar el primer font disponible del sistema
    val candidates = listOf("Roboto-Regular.ttf", "NotoSans-Regular.ttf", "DroidSans.ttf")
    for (name in candidates) {
        val src = File("/system/fonts/$name")
        if (src.exists()) {
            try { src.copyTo(File(fontsDir, name), overwrite = false) } catch (_: Exception) {}
            break
        }
    }
    // Apuntar mpv al directorio (aunque esté vacío, /system/fonts como fallback)
    val dir = if (fontsDir.listFiles()?.isNotEmpty() == true) fontsDir.absolutePath
              else "/system/fonts"
    MpvLib.setOptionString("sub-fonts-dir", dir)
}

private fun formatTime(seconds: Double): String {
    val total = seconds.toLong()
    val h = total / 3600
    val m = (total % 3600) / 60
    val s = total % 60
    return if (h > 0) "%d:%02d:%02d".format(h, m, s) else "%d:%02d".format(m, s)
}

private fun formatSpeed(bytesPerSec: Long): String = when {
    bytesPerSec >= 1_048_576 -> "%.1f MB/s".format(bytesPerSec / 1_048_576.0)
    bytesPerSec >= 1024 -> "%.0f KB/s".format(bytesPerSec / 1024.0)
    else -> "$bytesPerSec B/s"
}
