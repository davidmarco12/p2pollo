<script>
  import { createEventDispatcher } from 'svelte'

  const dispatch = createEventDispatcher()

  export let movie = null
  export let results = []
  export let loading = false

  function handleBack() {
    dispatch('back')
  }

  function handlePlay(result) {
    dispatch('play', result)
  }

  function seedColor(seeds) {
    if (seeds >= 50) return '#4caf50'
    if (seeds >= 10) return '#ff9800'
    return '#f44336'
  }
</script>

<div class="detail-container">
  <button class="back-btn" on:click={handleBack}>
    &#8592; Volver
  </button>

  {#if movie}
    <div class="detail-header">
      <div class="detail-poster">
        {#if movie.posterUrl}
          <img src={movie.posterUrl} alt={movie.title} />
        {:else}
          <div class="no-poster">Sin imagen</div>
        {/if}
      </div>
      <div class="detail-info">
        <h2 class="detail-title">{movie.title}</h2>
        <div class="detail-meta">
          <span class="meta-year">{movie.year}</span>
          {#if movie.rating > 0}
            <span class="meta-rating">&#9733; {movie.rating}</span>
          {/if}
        </div>
        {#if movie.genres}
          <div class="detail-genres">{movie.genres}</div>
        {/if}
      </div>
    </div>

    <div class="torrents-section">
      <h3 class="torrents-title">Archivos disponibles</h3>

      {#if loading}
        <div class="torrents-loading">
          <div class="spinner"></div>
          <span>Buscando torrents...</span>
        </div>
      {:else if results.length > 0}
        <div class="torrent-list">
          {#each results as result}
            <div class="torrent-row" on:click={() => handlePlay(result)}>
              <div class="torrent-name-col">
                <span class="torrent-name">{result.name}</span>
                <span class="torrent-source">{result.source}</span>
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
        <div class="no-torrents">No se encontraron torrents para esta pelicula</div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .detail-container {
    padding: 16px 24px;
    flex: 1;
    overflow-y: auto;
  }

  .back-btn {
    padding: 6px 14px;
    border-radius: 4px;
    border: 1px solid #333;
    background: transparent;
    color: #e0e0e0;
    font-size: 0.85rem;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.2s;
    margin-bottom: 16px;
  }

  .back-btn:hover {
    background: #2a2a2a;
  }

  .detail-header {
    display: flex;
    gap: 24px;
    margin-bottom: 24px;
  }

  .detail-poster {
    flex-shrink: 0;
    width: 200px;
    border-radius: 8px;
    overflow: hidden;
    background: #222;
  }

  .detail-poster img {
    width: 100%;
    display: block;
  }

  .no-poster {
    width: 100%;
    aspect-ratio: 2 / 3;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #555;
    font-size: 0.85rem;
  }

  .detail-info {
    flex: 1;
    min-width: 0;
  }

  .detail-title {
    font-size: 1.5rem;
    font-weight: 700;
    color: #e0e0e0;
    margin-bottom: 8px;
    line-height: 1.3;
  }

  .detail-meta {
    display: flex;
    gap: 12px;
    margin-bottom: 8px;
    flex-wrap: wrap;
  }

  .meta-year {
    color: #aaa;
    font-size: 0.9rem;
  }

  .meta-rating {
    color: #f5c518;
    font-size: 0.9rem;
    font-weight: 600;
  }

  .detail-genres {
    font-size: 0.8rem;
    color: #888;
    margin-bottom: 12px;
  }

  .torrents-section {
    margin-top: 8px;
  }

  .torrents-title {
    font-size: 1rem;
    font-weight: 600;
    color: #e0e0e0;
    margin-bottom: 12px;
  }

  .torrents-loading {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 20px 0;
    color: #888;
    font-size: 0.9rem;
  }

  .spinner {
    width: 24px;
    height: 24px;
    border: 3px solid #333;
    border-top-color: #ff6b35;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .torrent-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .torrent-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    background: #1a1a1a;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .torrent-row:hover {
    background: #252525;
  }

  .torrent-name-col {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .torrent-name {
    font-size: 0.85rem;
    color: #e0e0e0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .torrent-source {
    font-size: 0.7rem;
    color: #555;
  }

  .torrent-size {
    font-size: 0.85rem;
    color: #aaa;
    min-width: 80px;
    text-align: right;
    flex-shrink: 0;
  }

  .torrent-seeds {
    font-size: 0.8rem;
    font-weight: 600;
    min-width: 70px;
    flex-shrink: 0;
  }

  .torrent-leechers {
    font-size: 0.8rem;
    color: #666;
    min-width: 80px;
    flex-shrink: 0;
  }

  .play-icon {
    color: #ff6b35;
    font-size: 0.9rem;
    flex-shrink: 0;
  }

  .no-torrents {
    color: #666;
    font-size: 0.9rem;
    padding: 20px 0;
  }
</style>
