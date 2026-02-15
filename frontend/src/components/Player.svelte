<script>
  import { createEventDispatcher, onMount, onDestroy } from 'svelte'
  import { GetStreamProgress } from '../../wailsjs/go/main/App.js'
  import { WindowFullscreen, WindowUnfullscreen } from '../../wailsjs/runtime/runtime.js'

  const dispatch = createEventDispatcher()

  export let torrentName = ''
  export let torrentSize = ''

  let progress = null
  let streamURL = ''
  let streamError = ''
  let videoError = ''
  let videoDebug = ''
  let subtitleBlobURL = ''
  let progressInterval = null
  let showDownloadInfo = false

  // Seek state
  let canSeekNatively = false
  let videoDuration = 0
  let seekOffset = 0
  let currentTime = 0
  let videoElement = null

  $: videoSrc = streamURL
    ? (seekOffset > 0 ? `${streamURL}?t=${Math.floor(seekOffset)}` : streamURL)
    : ''
  $: actualTime = seekOffset + currentTime
  $: seekPercent = videoDuration > 0 ? Math.min((actualTime / videoDuration) * 100, 100) : 0

  function startProgressPolling() {
    if (progressInterval) clearInterval(progressInterval)
    progressInterval = setInterval(async () => {
      try {
        const p = await GetStreamProgress()
        progress = p
        if (p.streamURL && !streamURL) {
          streamURL = p.streamURL
          loadSubtitles(p.streamURL)
        }
        if (p.canSeekNatively !== undefined) {
          canSeekNatively = p.canSeekNatively
        }
        if (p.videoDuration > 0) {
          videoDuration = p.videoDuration
        }
        if (p.error && !streamError) {
          streamError = p.error
        }
      } catch (e) {
        // ignorar errores de polling
      }
    }, 1000)
  }

  // Cargar subtitulos via fetch + blob URL para evitar problemas de CORS
  async function loadSubtitles(url) {
    try {
      const subURL = url.replace('/stream', '/subtitles')
      const res = await fetch(subURL)
      if (!res.ok) return
      const text = await res.text()
      if (text && text.includes('WEBVTT')) {
        const blob = new Blob([text], { type: 'text/vtt' })
        subtitleBlobURL = URL.createObjectURL(blob)
      }
    } catch (e) {
      // no hay subtitulos disponibles
    }
  }

  function stopProgressPolling() {
    if (progressInterval) {
      clearInterval(progressInterval)
      progressInterval = null
    }
  }

  function handleClose() {
    stopProgressPolling()
    if (subtitleBlobURL) URL.revokeObjectURL(subtitleBlobURL)
    dispatch('close')
  }

  function handleVideoError(event) {
    const video = event.target
    const err = video.error
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

  function formatBytes(bytes) {
    if (!bytes) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB']
    let i = 0
    let val = bytes
    while (val >= 1024 && i < units.length - 1) {
      val /= 1024
      i++
    }
    return `${val.toFixed(1)} ${units[i]}`
  }

  function handleTimeUpdate(event) {
    currentTime = event.target.currentTime
  }

  function seekTo(targetSeconds) {
    if (targetSeconds < 0) targetSeconds = 0
    if (videoDuration > 0 && targetSeconds > videoDuration) targetSeconds = videoDuration

    if (canSeekNatively && videoElement) {
      videoElement.currentTime = targetSeconds
      return
    }

    // ffmpeg mode: recargar con ?t=<seconds>
    seekOffset = Math.floor(targetSeconds)
    currentTime = 0
  }

  function skipBack() {
    seekTo(actualTime - 10)
  }

  function skipForward() {
    seekTo(actualTime + 30)
  }

  function handleSeekBarClick(event) {
    if (!videoDuration) return
    const bar = event.currentTarget
    const rect = bar.getBoundingClientRect()
    const x = event.clientX - rect.left
    const percent = x / rect.width
    seekTo(percent * videoDuration)
  }

  function formatTime(seconds) {
    if (!seconds || !isFinite(seconds)) return '0:00'
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = Math.floor(seconds % 60)
    if (h > 0) return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
    return `${m}:${s.toString().padStart(2, '0')}`
  }

  // Sincronizar fullscreen del video con fullscreen de la ventana Wails
  function handleFullscreenChange() {
    if (document.fullscreenElement) {
      WindowFullscreen()
    } else {
      WindowUnfullscreen()
    }
  }

  // Atajos de teclado para seek (funcionan en fullscreen nativo)
  function handleKeydown(event) {
    if (!streamURL || videoError) return

    if (event.key === 'ArrowLeft') {
      event.preventDefault()
      skipBack()
    } else if (event.key === 'ArrowRight') {
      event.preventDefault()
      skipForward()
    }
  }

  onMount(() => {
    startProgressPolling()
    document.addEventListener('keydown', handleKeydown)
    document.addEventListener('fullscreenchange', handleFullscreenChange)
  })
  onDestroy(() => {
    stopProgressPolling()
    if (subtitleBlobURL) URL.revokeObjectURL(subtitleBlobURL)
    document.removeEventListener('keydown', handleKeydown)
    document.removeEventListener('fullscreenchange', handleFullscreenChange)
  })
</script>

<div class="player-container">
  <div class="player-header">
    <div class="player-title">
      <span class="now-playing">
        {#if streamURL && !videoError}
          Reproduciendo
        {:else if videoError || streamError}
          Error
        {:else}
          Preparando...
        {/if}
      </span>
      <span class="torrent-name">{torrentName}</span>
      {#if torrentSize}
        <span class="torrent-size">{torrentSize}</span>
      {/if}
      {#if progress && progress.fileExt}
        <span class="torrent-size">Formato: {progress.fileExt}</span>
      {/if}
      {#if videoDebug}
        <span class="torrent-size">{videoDebug}</span>
      {/if}
    </div>
    <button class="close-btn" on:click={handleClose}>Cerrar</button>
  </div>

  <div class="video-wrapper">
    {#if streamURL && !videoError}
      <video
        bind:this={videoElement}
        controls
        autoplay
        src={videoSrc}
        class="video-player {!canSeekNatively ? 'no-native-seek' : ''}"
        on:error={handleVideoError}
        on:loadeddata={handleVideoLoaded}
        on:timeupdate={handleTimeUpdate}
      >
        {#if subtitleBlobURL}
          <track kind="subtitles" src={subtitleBlobURL} label="Subtitulos" default />
        {/if}
      </video>

      {#if !canSeekNatively && videoDuration > 0}
        <div class="seek-overlay">
          <div class="seek-controls">
            <button class="seek-btn" on:click={skipBack} title="Retroceder 10s">-10s</button>
            <div class="seek-bar" on:click={handleSeekBarClick}>
              <div class="seek-fill" style="width: {seekPercent}%"></div>
            </div>
            <button class="seek-btn" on:click={skipForward} title="Avanzar 30s">+30s</button>
            <span class="seek-time">{formatTime(actualTime)} / {formatTime(videoDuration)}</span>
          </div>
        </div>
      {/if}
    {:else if videoError}
      <div class="loading-video">
        <span class="error-text">{videoError}</span>
        <span class="error-hint">Asegurate de tener ffmpeg instalado para reproducir archivos MKV</span>
      </div>
    {:else if streamError}
      <div class="loading-video">
        <span class="error-text">{streamError}</span>
      </div>
    {:else}
      <div class="loading-video">
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

  {#if progress && !streamError}
    <div class="download-footer">
      <button class="toggle-stats-btn" on:click={() => showDownloadInfo = !showDownloadInfo}>
        {#if showDownloadInfo}
          Ocultar descarga
        {:else}
          {progress.percent}% | {formatBytes(progress.headWritten)} | {progress.peers} peers | {progress.speedMBps.toFixed(1)} MB/s
        {/if}
      </button>
      {#if showDownloadInfo}
        <div class="progress-bar-container">
          <div class="progress-bar">
            <div class="progress-fill" style="width: {progress.percent}%"></div>
          </div>
          <div class="progress-stats">
            <span>{formatBytes(progress.headWritten)} / {formatBytes(progress.totalSize)}</span>
            <span>{progress.speedMBps.toFixed(1)} MB/s</span>
            <span>{progress.peers} peers</span>
            <span>{progress.percent}%</span>
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .player-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #000;
  }

  .player-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 20px;
    background: #161616;
    border-bottom: 1px solid #2a2a2a;
  }

  .player-title {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .now-playing {
    font-size: 0.75rem;
    color: #ff6b35;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .torrent-name {
    font-size: 0.9rem;
    color: #e0e0e0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .torrent-size {
    font-size: 0.75rem;
    color: #888;
    margin-top: 2px;
  }

  .close-btn {
    padding: 6px 14px;
    border-radius: 4px;
    border: 1px solid #333;
    background: transparent;
    color: #e0e0e0;
    font-size: 0.85rem;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.2s;
    flex-shrink: 0;
  }

  .close-btn:hover {
    background: #2a2a2a;
  }

  .video-wrapper {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #000;
    min-height: 0;
    position: relative;
  }

  .video-player {
    width: 100%;
    height: 100%;
    object-fit: contain;
  }

  /* En modo ffmpeg, esconder la barra de timeline nativa (muestra duracion incorrecta) */
  .video-player.no-native-seek::-webkit-media-controls-timeline {
    display: none;
  }
  .video-player.no-native-seek::-webkit-media-controls-current-time-display,
  .video-player.no-native-seek::-webkit-media-controls-time-remaining-display {
    display: none;
  }

  .loading-video {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: #888;
    font-size: 1rem;
  }

  .error-text {
    color: #f44336;
  }

  .error-hint {
    font-size: 0.8rem;
    color: #666;
  }

  .peers-info {
    font-size: 0.8rem;
    color: #666;
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid #333;
    border-top-color: #ff6b35;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .seek-overlay {
    position: absolute;
    bottom: 40px;
    left: 0;
    right: 0;
    background: linear-gradient(transparent, rgba(0, 0, 0, 0.85));
    padding-top: 20px;
    opacity: 0;
    transition: opacity 0.3s;
    pointer-events: none;
  }

  .video-wrapper:hover .seek-overlay {
    opacity: 1;
    pointer-events: auto;
  }

  .seek-controls {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 20px 12px;
  }

  .seek-btn {
    padding: 4px 10px;
    border-radius: 4px;
    border: 1px solid rgba(255, 255, 255, 0.2);
    background: rgba(255, 255, 255, 0.1);
    color: #e0e0e0;
    font-size: 0.75rem;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.2s;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .seek-btn:hover {
    background: rgba(255, 255, 255, 0.2);
  }

  .seek-bar {
    flex: 1;
    height: 6px;
    background: #2a2a2a;
    border-radius: 3px;
    cursor: pointer;
    position: relative;
    overflow: hidden;
  }

  .seek-bar:hover {
    height: 8px;
  }

  .seek-fill {
    height: 100%;
    background: #ff6b35;
    border-radius: 3px;
    transition: width 0.3s;
  }

  .seek-time {
    font-size: 0.75rem;
    color: #888;
    white-space: nowrap;
    flex-shrink: 0;
    min-width: 90px;
    text-align: right;
  }

  .download-footer {
    background: #161616;
    border-top: 1px solid #2a2a2a;
  }

  .toggle-stats-btn {
    width: 100%;
    padding: 6px 20px;
    border: none;
    background: transparent;
    color: #666;
    font-size: 0.75rem;
    font-family: inherit;
    cursor: pointer;
    text-align: center;
    transition: color 0.2s;
  }

  .toggle-stats-btn:hover {
    color: #999;
  }

  .progress-bar-container {
    padding: 4px 20px 10px;
  }

  .progress-bar {
    width: 100%;
    height: 3px;
    background: #2a2a2a;
    border-radius: 2px;
    overflow: hidden;
    margin-bottom: 6px;
  }

  .progress-fill {
    height: 100%;
    background: #ff6b35;
    border-radius: 2px;
    transition: width 0.5s;
  }

  .progress-stats {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
    color: #666;
  }
</style>
