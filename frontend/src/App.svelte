<script>
  import SearchBar from './components/SearchBar.svelte'
  import ResultsTable from './components/ResultsTable.svelte'
  import Player from './components/Player.svelte'
  import { Search, PlayMagnet, StopStream } from '../wailsjs/go/main/App.js'

  // Estado de la app
  let view = 'search'   // 'search' | 'player'
  let results = []
  let loading = false
  let error = ''
  let currentName = ''
  let currentSize = ''

  async function handleSearch(event) {
    const query = event.detail
    loading = true
    error = ''
    results = []

    try {
      const res = await Search(query)
      results = res || []
      if (results.length === 0) {
        error = 'No se encontraron resultados'
      }
    } catch (e) {
      error = typeof e === 'string' ? e : (e.message || JSON.stringify(e))
    } finally {
      loading = false
    }
  }

  async function handlePlay(event) {
    const result = event.detail
    currentName = result.name
    currentSize = result.size
    view = 'player'
    error = ''

    try {
      await PlayMagnet(result.magnetLink)
    } catch (e) {
      error = typeof e === 'string' ? e : (e.message || JSON.stringify(e))
      view = 'search'
    }
  }

  async function handleClosePlayer() {
    try {
      await StopStream()
    } catch (e) {
      // ignorar
    }
    currentName = ''
    currentSize = ''
    view = 'search'
  }
</script>

{#if view === 'search'}
  <SearchBar on:search={handleSearch} {loading} />

  {#if error}
    <div class="message error">{error}</div>
  {/if}

  {#if !loading && results.length === 0 && !error}
    <div class="empty-state">
      <div class="empty-icon">&#127871;</div>
      <p>Busca una pelicula o serie para empezar</p>
    </div>
  {/if}

  <ResultsTable {results} on:play={handlePlay} />

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

  .empty-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: #444;
    gap: 8px;
    padding-top: 120px;
  }

  .empty-icon {
    font-size: 3rem;
  }

  .empty-state p {
    font-size: 1rem;
  }
</style>
