<script>
  import { onMount } from 'svelte'
  import SearchBar from './components/SearchBar.svelte'
  import MovieGrid from './components/MovieGrid.svelte'
  import MovieDetail from './components/MovieDetail.svelte'
  import Player from './components/Player.svelte'
  import { Search, PlayMagnet, StopStream, GetPopularMovies, SearchMovies } from '../wailsjs/go/main/App.js'

  // Estado de la app
  let view = 'home'   // 'home' | 'search' | 'detail' | 'player'
  let movies = []
  let selectedMovie = null
  let torrentResults = []
  let detailLoading = false
  let loading = false
  let error = ''
  let currentName = ''
  let currentSize = ''

  // Cargar peliculas populares al inicio
  onMount(async () => {
    loading = true
    try {
      const res = await GetPopularMovies()
      movies = res || []
    } catch (e) {
      error = typeof e === 'string' ? e : (e.message || JSON.stringify(e))
    } finally {
      loading = false
    }
  })

  async function handleSearch(event) {
    const query = event.detail
    loading = true
    error = ''
    movies = []

    try {
      const res = await SearchMovies(query)
      movies = res || []
      view = 'search'
      if (movies.length === 0) {
        error = 'No se encontraron resultados'
      }
    } catch (e) {
      error = typeof e === 'string' ? e : (e.message || JSON.stringify(e))
    } finally {
      loading = false
    }
  }

  async function handleHome() {
    view = 'home'
    error = ''
    selectedMovie = null
    torrentResults = []
    loading = true
    movies = []

    try {
      const res = await GetPopularMovies()
      movies = res || []
    } catch (e) {
      error = typeof e === 'string' ? e : (e.message || JSON.stringify(e))
    } finally {
      loading = false
    }
  }

  async function handleSelectMovie(event) {
    selectedMovie = event.detail
    torrentResults = []
    detailLoading = true
    view = 'detail'
    error = ''

    try {
      // Buscar torrents en rargb usando el titulo de la pelicula
      const searchQuery = `${selectedMovie.title} ${selectedMovie.year}`
      const res = await Search(searchQuery)
      torrentResults = res || []
    } catch (e) {
      error = typeof e === 'string' ? e : (e.message || JSON.stringify(e))
    } finally {
      detailLoading = false
    }
  }

  async function handlePlayTorrent(event) {
    const result = event.detail
    currentName = result.name
    currentSize = result.size
    view = 'player'
    error = ''

    try {
      await PlayMagnet(result.magnetLink)
    } catch (e) {
      error = typeof e === 'string' ? e : (e.message || JSON.stringify(e))
      view = 'detail'
    }
  }

  function handleBackFromDetail() {
    view = movies.length > 0 ? 'search' : 'home'
    selectedMovie = null
    torrentResults = []
    error = ''
  }

  async function handleClosePlayer() {
    try {
      await StopStream()
    } catch (e) {
      // ignorar
    }
    currentName = ''
    currentSize = ''
    view = selectedMovie ? 'detail' : 'home'
  }
</script>

{#if view === 'home' || view === 'search'}
  <SearchBar on:search={handleSearch} on:home={handleHome} {loading} />

  {#if error}
    <div class="message error">{error}</div>
  {/if}

  <MovieGrid
    {movies}
    {loading}
    title={view === 'home' ? 'Peliculas Populares' : `${movies.length} resultados`}
    on:select={handleSelectMovie}
  />

{:else if view === 'detail'}
  <SearchBar on:search={handleSearch} on:home={handleHome} loading={false} />

  {#if error}
    <div class="message error">{error}</div>
  {/if}

  <MovieDetail
    movie={selectedMovie}
    results={torrentResults}
    loading={detailLoading}
    on:play={handlePlayTorrent}
    on:back={handleBackFromDetail}
  />

{:else if view === 'player'}
  <Player
    torrentName={currentName}
    torrentSize={currentSize}
    on:close={handleClosePlayer}
  />
{/if}

<style>
  .message {
    padding: 12px 24px;
    font-size: 0.9rem;
  }

  .error {
    color: #f44336;
    background: #1a0000;
    border-bottom: 1px solid #2a1a1a;
  }
</style>
