<script>
  import { createEventDispatcher } from 'svelte'
  import TorrentList from './TorrentList.svelte'

  const dispatch = createEventDispatcher()

  export let movie = null
  export let results = []
  export let loading = false
</script>

<div class="detail-container">
  <button class="back-btn" on:click={() => dispatch('back')}>
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
            <span class="meta-rating">&#9733; {movie.rating.toFixed(1)}</span>
          {/if}
          {#if movie.runtime > 0}
            <span class="meta-runtime">{movie.runtime} min</span>
          {/if}
        </div>
        {#if movie.genres}
          <div class="detail-genres">{movie.genres}</div>
        {/if}
        {#if movie.description}
          <div class="detail-description">{movie.description}</div>
        {/if}
      </div>
    </div>

    <div class="torrents-section">
      <h3 class="torrents-title">Archivos disponibles</h3>
      <TorrentList {results} {loading} on:play={(e) => dispatch('play', e.detail)} />
    </div>
  {/if}
</div>

<style src="./MovieDetail.css"></style>
