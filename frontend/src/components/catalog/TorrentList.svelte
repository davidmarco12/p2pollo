<script>
  import { createEventDispatcher } from 'svelte'

  const dispatch = createEventDispatcher()

  export let results = []
  export let loading = false

  function seedColor(seeds) {
    if (seeds >= 50) return '#4caf50'
    if (seeds >= 10) return '#ff9800'
    return '#f44336'
  }
</script>

{#if loading}
  <div class="list-loading">
    <div class="spinner"></div>
    <span>Buscando torrents...</span>
  </div>
{:else if results.length > 0}
  <div class="torrent-list">
    {#each results as result}
      <div class="torrent-row" on:click={() => dispatch('play', result)}>
        <div class="torrent-name-col">
          <span class="torrent-name">{result.name}</span>
          <div class="torrent-meta">
            <span class="source-chip">{result.source}</span>
            {#if result.subtitles && result.subtitles.length > 0}
              <span class="sub-tags">
                SUB:
                {#each result.subtitles as lang}
                  <span class="sub-tag">{lang}</span>
                {/each}
              </span>
            {:else}
              <span class="sub-none">SUB: None</span>
            {/if}
          </div>
        </div>
        <span class="torrent-size">{result.size}</span>
        <span class="torrent-seeds" style="color: {seedColor(result.seeds)}">
          {result.seeds} seeds
        </span>
        <span class="torrent-leechers">{result.leechers} leechers</span>
        <span class="play-icon">&#9654;</span>
      </div>
    {/each}
  </div>
{:else}
  <div class="no-results">No se encontraron torrents para esta pelicula</div>
{/if}

<style src="./TorrentList.css"></style>
