<script>
  import { formatBytes } from '../utils/playerUtils.js'

  export let progress = null

  let showDetails = false
</script>

{#if progress && progress.totalSize > 0}
  <div class="download-footer">
    <button class="toggle-btn" on:click={() => showDetails = !showDetails}>
      {#if showDetails}
        Ocultar descarga
      {:else}
        {progress.percent}% | {formatBytes(progress.headWritten)} | {progress.peers} peers | {progress.speedMBps.toFixed(1)} MB/s
      {/if}
    </button>

    {#if showDetails}
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

<style src="./DownloadFooter.css"></style>
