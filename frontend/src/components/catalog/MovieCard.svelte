<script>
  export let movie = {}
</script>

<div class="movie-card" on:click>
  <div class="poster-wrapper">
    {#if movie.posterUrl}
      <img src={movie.posterUrl} alt={movie.title} loading="lazy" />
    {:else}
      <div class="no-poster">
        <span>Sin imagen</span>
      </div>
    {/if}
    {#if movie.rating > 0}
      <span class="rating-badge">
        <span class="star">&#9733;</span> {movie.rating.toFixed(1)}
      </span>
    {/if}
  </div>
  <div class="card-info">
    <span class="card-title" title={movie.title}>{movie.title}</span>
    <span class="card-year">{movie.year}</span>
  </div>
</div>

<style>
  .movie-card {
    cursor: pointer;
    transition: transform 0.2s, box-shadow 0.2s;
    border-radius: 8px;
    overflow: hidden;
    background: #1a1a1a;
  }

  .movie-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  }

  .poster-wrapper {
    position: relative;
    aspect-ratio: 2 / 3;
    background: #222;
    overflow: hidden;
  }

  .poster-wrapper img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .no-poster {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #555;
    font-size: 0.8rem;
  }

  .rating-badge {
    position: absolute;
    top: 6px;
    right: 6px;
    background: rgba(0, 0, 0, 0.75);
    color: #f5c518;
    font-size: 0.75rem;
    font-weight: 700;
    padding: 3px 7px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    gap: 3px;
  }

  .star {
    font-size: 0.7rem;
  }

  .card-info {
    padding: 8px 10px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .card-title {
    font-size: 0.85rem;
    color: #e0e0e0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.3;
  }

  .card-year {
    font-size: 0.75rem;
    color: #777;
  }
</style>
