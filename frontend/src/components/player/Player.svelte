<script>
  import { createEventDispatcher, onMount, onDestroy } from 'svelte'
  import { GetStreamProgress, GetSubtitleTracks } from '../../../wailsjs/go/main/App.js'
  import { WindowFullscreen, WindowUnfullscreen } from '../../../wailsjs/runtime/runtime.js'
  import { parseVTTCues, formatBytes } from '../../utils/playerUtils.js'
  import PlayerHeader from './PlayerHeader.svelte'
  import SeekOverlay from './SeekOverlay.svelte'
  import SubtitleControl from './SubtitleControl.svelte'
  import DownloadFooter from './DownloadFooter.svelte'

  const dispatch = createEventDispatcher()

  export let torrentName = ''
  export let torrentSize = ''

  // Stream state
  let progress = null
  let streamURL = ''
  let streamError = ''
  let videoError = ''
  let videoDebug = ''
  let progressInterval = null

  // Subtitle state
  let subtitleTracks = []
  let subtitleCues = []
  let currentSubtitle = ''
  let subtitleDelay = 0
  let activeSubtitleLabel = ''
  let activeTrack = null

  // Seek state
  let canSeekNatively = false
  let videoDuration = 0
  let seekOffset = 0
  let currentTime = 0

  // DOM refs
  let videoElement = null
  let videoWrapper = null
  let playerContainer = null

  // Fullscreen
  let isFullscreen = false
  let ignoringFullscreenChange = false

  // Reactivos
  $: videoSrc = streamURL
    ? (seekOffset > 0 ? `${streamURL}?t=${Math.floor(seekOffset)}` : streamURL)
    : ''
  $: actualTime = seekOffset + currentTime
  $: seekPercent = videoDuration > 0 ? Math.min((actualTime / videoDuration) * 100, 100) : 0
  $: currentSubtitle = (() => {
    if (!subtitleCues.length) return ''
    const t = actualTime + subtitleDelay
    const cue = subtitleCues.find(c => t >= c.start && t < c.end)
    return cue ? cue.text : ''
  })()

  // --- Progress polling ---

  function startProgressPolling() {
    if (progressInterval) clearInterval(progressInterval)
    progressInterval = setInterval(async () => {
      try {
        const p = await GetStreamProgress()
        progress = p
        if (p.streamURL && !streamURL) {
          streamURL = p.streamURL
          loadSubtitleTracks()
        }
        if (p.canSeekNatively !== undefined) canSeekNatively = p.canSeekNatively
        if (p.videoDuration > 0) videoDuration = p.videoDuration
        if (p.error && !streamError) streamError = p.error
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
      subtitleTracks = (await GetSubtitleTracks()) || []
    } catch (e) {
      subtitleTracks = []
    }
  }

  async function selectSubtitle(track) {
    if (!track) {
      subtitleCues = []
      currentSubtitle = ''
      activeSubtitleLabel = ''
      activeTrack = null
      return
    }
    try {
      const baseURL = streamURL.replace('/stream', '/subtitles')
      const subURL = `${baseURL}?type=${track.type}&index=${track.index}`
      const res = await fetch(subURL)
      if (!res.ok) return
      const text = await res.text()
      if (text && text.includes('WEBVTT')) {
        subtitleCues = parseVTTCues(text)
        activeSubtitleLabel = track.title || track.language
        activeTrack = track
      }
    } catch (e) { /* error cargando subtitulo */ }
  }

  // --- Seek ---

  function seekTo(targetSeconds) {
    if (targetSeconds < 0) targetSeconds = 0
    if (videoDuration > 0 && targetSeconds > videoDuration) targetSeconds = videoDuration
    if (canSeekNatively && videoElement) {
      videoElement.currentTime = targetSeconds
      return
    }
    seekOffset = Math.floor(targetSeconds)
    currentTime = 0
  }

  // --- Video event handlers ---

  function handleVideoError(event) {
    const err = event.target.error
    if (!err) return
    const codes = {
      1: 'Reproduccion cancelada',
      2: 'Error de red al cargar el video',
      3: 'Error decodificando el video (formato no soportado)',
      4: 'Formato de video no soportado por el navegador',
    }
    videoError = codes[err.code] || `Error de video (codigo ${err.code})`
  }

  function handleVideoLoaded(event) {
    const v = event.target
    videoDebug = `${v.videoWidth}x${v.videoHeight} | duracion: ${Math.round(v.duration)}s | readyState: ${v.readyState}`
  }

  // --- Fullscreen ---

  function toggleFullscreen() {
    if (document.fullscreenElement) {
      document.exitFullscreen().catch(() => {})
    } else {
      playerContainer?.requestFullscreen().catch(() => {})
    }
  }

  function handleFullscreenChange() {
    if (ignoringFullscreenChange) return
    const fsEl = document.fullscreenElement
    if (!fsEl) {
      isFullscreen = false
      WindowUnfullscreen()
      return
    }
    if (fsEl === videoElement && playerContainer) {
      ignoringFullscreenChange = true
      document.exitFullscreen()
        .then(() => playerContainer.requestFullscreen())
        .then(() => { ignoringFullscreenChange = false })
        .catch(() => { ignoringFullscreenChange = false })
      return
    }
    isFullscreen = true
    WindowFullscreen()
  }

  // --- Keyboard ---

  function handleKeydown(event) {
    if (!streamURL || videoError) return
    if (event.key === 'ArrowLeft') { event.preventDefault(); seekTo(actualTime - 10) }
    else if (event.key === 'ArrowRight') { event.preventDefault(); seekTo(actualTime + 30) }
  }

  function handleClose() {
    stopProgressPolling()
    dispatch('close')
  }

  onMount(() => {
    startProgressPolling()
    document.addEventListener('keydown', handleKeydown)
    document.addEventListener('fullscreenchange', handleFullscreenChange)
  })
  onDestroy(() => {
    stopProgressPolling()
    document.removeEventListener('keydown', handleKeydown)
    document.removeEventListener('fullscreenchange', handleFullscreenChange)
  })
</script>

<div class="player-container" bind:this={playerContainer}>

  <PlayerHeader
    {torrentName} {torrentSize} {streamURL} {videoError} {streamError} {progress} {videoDebug}
    on:close={handleClose}
  />

  <div class="video-wrapper" bind:this={videoWrapper}>
    {#if streamURL && !videoError}
      <video
        bind:this={videoElement}
        controls
        autoplay
        src={videoSrc}
        class="video-player {!canSeekNatively ? 'no-native-seek' : ''}"
        on:error={handleVideoError}
        on:loadeddata={handleVideoLoaded}
        on:timeupdate={(e) => currentTime = e.target.currentTime}
      >
        <track kind="captions" />
      </video>

      {#if currentSubtitle}
        <div class="subtitle-overlay">
          {@html currentSubtitle.replace(/\n/g, '<br>')}
        </div>
      {/if}

      {#if !canSeekNatively && videoDuration > 0}
        <SeekOverlay
          {seekPercent} {actualTime} {videoDuration}
          on:back={() => seekTo(actualTime - 10)}
          on:forward={() => seekTo(actualTime + 30)}
          on:seek={(e) => seekTo(e.detail * videoDuration)}
        />
      {/if}

      <div class="video-controls-overlay">
        <button class="overlay-btn" on:click={toggleFullscreen} title={isFullscreen ? 'Salir de pantalla completa' : 'Pantalla completa'}>
          {#if isFullscreen}⊡{:else}⊞{/if}
        </button>
        <SubtitleControl
          tracks={subtitleTracks}
          {activeTrack}
          activeLabel={activeSubtitleLabel}
          hasCues={subtitleCues.length > 0}
          delay={subtitleDelay}
          on:select={(e) => selectSubtitle(e.detail)}
          on:delayChange={(e) => subtitleDelay = e.detail}
        />
      </div>

    {:else if videoError}
      <div class="loading-state">
        <span class="error-text">{videoError}</span>
        <span class="error-hint">Asegurate de tener ffmpeg instalado para reproducir archivos MKV</span>
      </div>
    {:else if streamError}
      <div class="loading-state">
        <span class="error-text">{streamError}</span>
      </div>
    {:else}
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
    {/if}
  </div>

  <DownloadFooter {progress} />

</div>

<style src="./Player.css"></style>
