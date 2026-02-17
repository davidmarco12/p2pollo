<script>
  import { createEventDispatcher } from 'svelte'
  import MovieCard from './MovieCard.svelte'

  const dispatch = createEventDispatcher()

  export let movies = []
  export let loading = false
  export let title = ''

  function handleSelect(movie) {
    dispatch('select', movie)
  }
</script>

<div class="movie-grid-container">
  {#if title}
    <h2 class="grid-title">{title}</h2>
  {/if}

  {#if loading}
    <div class="loading-state">
      <div class="spinner"></div>
      <span>Cargando peliculas...</span>
    </div>
  {:else if movies.length > 0}
    <div class="movie-grid">
      {#each movies as movie}
        <MovieCard {movie} on:click={() => handleSelect(movie)} />
      {/each}
    </div>
  {/if}
</div>

<style>
  .movie-grid-container {
    padding: 16px 24px;
    flex: 1;
    overflow-y: auto;
  }

  .grid-title {
    font-size: 1.1rem;
    font-weight: 600;
    color: #e0e0e0;
    margin-bottom: 16px;
  }

  .movie-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 16px;
  }

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    padding: 60px 0;
    color: #888;
    font-size: 0.95rem;
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
</style>
