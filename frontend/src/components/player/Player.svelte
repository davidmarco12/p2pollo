<script>
  import { createEventDispatcher, onMount, onDestroy } from 'svelte'
  import { GetStreamProgress, GetMPVPlaybackState, GetMPVTracks, MPVCommand, SetMPVProperty } from '../../../wailsjs/go/main/App.js'
  import { formatBytes } from '../../utils/playerUtils.js'
  import PlayerHeader from './PlayerHeader.svelte'
  import SubtitleControl from './SubtitleControl.svelte'
  import DownloadFooter from './DownloadFooter.svelte'

  const dispatch = createEventDispatcher()

  export let torrentName = ''
  export let torrentSize = ''

  // Stream state (torrent download)
  let progress = null
  let streamError = ''
  let progressInterval = null

  // Playback state (mpv)
  let playbackState = null
  let isPlaying = false
  let currentTime = 0
  let duration = 0
  let volume = 100

  // Subtitle state
  let subtitleTracks = []
  let activeSubtitleID = null
  let subtitleDelay = 0

  // Reactivos
  $: isPlaying = playbackState && !playbackState.paused
  $: currentTime = playbackState?.timePos || 0
  $: duration = playbackState?.videoDuration || 0
  $: volume = playbackState?.volume || 100
  $: seekPercent = duration > 0 ? Math.min((currentTime / duration) * 100, 100) : 0

  // --- Progress polling ---

  function startProgressPolling() {
    if (progressInterval) clearInterval(progressInterval)
    progressInterval = setInterval(async () => {
      try {
        // Poll torrent download progress
        const p = await GetStreamProgress()
        progress = p
        if (p.error && !streamError) streamError = p.error

        // Poll mpv playback state
        try {
          playbackState = await GetMPVPlaybackState()
          // Cargar tracks solo una vez cuando mpv está listo
          if (playbackState && subtitleTracks.length === 0) {
            loadSubtitleTracks()
          }
        } catch (e) {
          // mpv aún no está listo
        }
      } catch (e) { /* ignorar errores de polling */ }
    }, 1000)
  }

  function stopProgressPolling() {
    if (progressInterval) {
      clearInterval(progressInterval)
      progressInterval = null
    }
  }

  // --- Subtitles ---

  async function loadSubtitleTracks() {
    try {
      const tracks = await GetMPVTracks()
      subtitleTracks = tracks.filter(t => t.type === 'sub')
      // Detectar cual está activo
      const active = subtitleTracks.find(t => t.selected)
      if (active) activeSubtitleID = active.id
    } catch (e) {
      subtitleTracks = []
    }
  }

  async function selectSubtitle(trackID) {
    try {
      if (trackID === null) {
        // Desactivar subtítulos
        await SetMPVProperty('sid', 'no')
        activeSubtitleID = null
      } else {
        await SetMPVProperty('sid', trackID)
        activeSubtitleID = trackID
      }
      // Recargar tracks para actualizar estado
      await loadSubtitleTracks()
    } catch (e) {
      console.error('Error seleccionando subtítulo:', e)
    }
  }

  async function changeSubtitleDelay(delta) {
    try {
      await MPVCommand('add', 'sub-delay', delta)
      subtitleDelay += delta
    } catch (e) {
      console.error('Error ajustando delay de subtítulos:', e)
    }
  }

  async function resetSubtitleDelay() {
    try {
      await SetMPVProperty('sub-delay', 0)
      subtitleDelay = 0
    } catch (e) {
      console.error('Error reseteando delay de subtítulos:', e)
    }
  }

  // --- Playback controls ---

  async function togglePlayPause() {
    try {
      await MPVCommand('cycle', 'pause')
    } catch (e) {
      console.error('Error toggle play/pause:', e)
    }
  }

  async function seekRelative(seconds) {
    try {
      await MPVCommand('seek', seconds, 'relative')
    } catch (e) {
      console.error('Error seeking:', e)
    }
  }

  async function seekAbsolute(seconds) {
    try {
      await MPVCommand('seek', seconds, 'absolute')
    } catch (e) {
      console.error('Error seeking:', e)
    }
  }

  async function toggleFullscreen() {
    try {
      await MPVCommand('cycle', 'fullscreen')
    } catch (e) {
      console.error('Error toggle fullscreen:', e)
    }
  }

  async function setVolume(vol) {
    try {
      await SetMPVProperty('volume', vol)
    } catch (e) {
      console.error('Error setting volume:', e)
    }
  }

  // --- Keyboard ---

  function handleKeydown(event) {
    if (streamError) return
    if (event.key === ' ') { event.preventDefault(); togglePlayPause() }
    else if (event.key === 'ArrowLeft') { event.preventDefault(); seekRelative(-10) }
    else if (event.key === 'ArrowRight') { event.preventDefault(); seekRelative(30) }
    else if (event.key === 'f') { event.preventDefault(); toggleFullscreen() }
  }

  function handleClose() {
    stopProgressPolling()
    dispatch('close')
  }

  function formatTime(seconds) {
    if (!seconds || seconds < 0) return '0:00'
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = Math.floor(seconds % 60)
    if (h > 0) return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
    return `${m}:${s.toString().padStart(2, '0')}`
  }

  onMount(() => {
    startProgressPolling()
    document.addEventListener('keydown', handleKeydown)
  })

  onDestroy(() => {
    stopProgressPolling()
    document.removeEventListener('keydown', handleKeydown)
  })
</script>

<div class="player-container">

  <PlayerHeader
    {torrentName} {torrentSize} streamURL="" videoError="" {streamError} {progress} videoDebug=""
    on:close={handleClose}
  />

  <div class="video-wrapper">
    {#if streamError}
      <div class="loading-state">
        <span class="error-text">{streamError}</span>
      </div>
    {:else if !playbackState}
      <div class="loading-state">
        <div class="spinner"></div>
        <span>
          {#if progress && progress.totalSize > 0}
            Descargando buffer... {formatBytes(progress.headWritten)} / {formatBytes(progress.totalSize)}
          {:else if progress && progress.preparing}
            Conectando al torrent...
          {:else}
            Preparando stream...
          {/if}
        </span>
        {#if progress && progress.peers > 0}
          <span class="peers-info">{progress.peers} peers conectados</span>
        {/if}
      </div>
    {:else}
      <!-- mpv está reproduciendo en su propia ventana -->
      <div class="playback-info">
        <div class="mpv-notice">
          <div class="mpv-icon">▶</div>
          <div>
            <div class="mpv-title">Reproduciendo en mpv</div>
            <div class="mpv-hint">Usa los controles de abajo o las teclas: Espacio (play/pause), ← → (seek), F (fullscreen)</div>
          </div>
        </div>

        <!-- Progress bar -->
        <div class="playback-progress">
          <div class="progress-bar" on:click={(e) => {
            const rect = e.currentTarget.getBoundingClientRect()
            const percent = (e.clientX - rect.left) / rect.width
            seekAbsolute(percent * duration)
          }}>
            <div class="progress-fill" style="width: {seekPercent}%"></div>
          </div>
          <div class="progress-time">
            {formatTime(currentTime)} / {formatTime(duration)}
          </div>
        </div>

        <!-- Playback controls -->
        <div class="playback-controls">
          <button class="control-btn" on:click={() => seekRelative(-10)} title="Retroceder 10s">
            ⏪
          </button>
          <button class="control-btn play-btn" on:click={togglePlayPause} title={isPlaying ? 'Pausar' : 'Reproducir'}>
            {isPlaying ? '⏸' : '▶'}
          </button>
          <button class="control-btn" on:click={() => seekRelative(30)} title="Adelantar 30s">
            ⏩
          </button>
          <button class="control-btn" on:click={toggleFullscreen} title="Pantalla completa">
            ⊞
          </button>

          <div class="volume-control">
            <span>🔊</span>
            <input
              type="range"
              min="0"
              max="100"
              bind:value={volume}
              on:input={(e) => setVolume(parseInt(e.target.value))}
              class="volume-slider"
            />
            <span class="volume-value">{volume}%</span>
          </div>

          <SubtitleControl
            tracks={subtitleTracks}
            activeTrackID={activeSubtitleID}
            delay={subtitleDelay}
            on:select={(e) => selectSubtitle(e.detail)}
            on:delayChange={(e) => changeSubtitleDelay(e.detail)}
            on:delayReset={resetSubtitleDelay}
          />
        </div>
      </div>
    {/if}
  </div>

  <DownloadFooter {progress} />

</div>

<style src="./Player.css"></style>
