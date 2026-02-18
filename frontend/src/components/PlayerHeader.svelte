<script>
  import { createEventDispatcher } from 'svelte'

  const dispatch = createEventDispatcher()

  export let torrentName = ''
  export let torrentSize = ''
  export let streamURL = ''
  export let videoError = ''
  export let streamError = ''
  export let progress = null
  export let videoDebug = ''
</script>

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
      <span class="torrent-meta">{torrentSize}</span>
    {/if}
    {#if progress && progress.fileExt}
      <span class="torrent-meta">Formato: {progress.fileExt}</span>
    {/if}
    {#if videoDebug}
      <span class="torrent-meta">{videoDebug}</span>
    {/if}
  </div>
  <button class="close-btn" on:click={() => dispatch('close')}>Cerrar</button>
</div>

<style src="./PlayerHeader.css"></style>
