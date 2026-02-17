<script>
  import { createEventDispatcher } from 'svelte'

  const dispatch = createEventDispatcher()

  let query = ''
  let loading = false

  export { loading }

  function handleSearch() {
    const q = query.trim()
    if (q && !loading) {
      dispatch('search', q)
    }
  }

  function handleHome() {
    dispatch('home')
  }

  function handleKeydown(e) {
    if (e.key === 'Enter') handleSearch()
  }
</script>

<div class="search-bar">
  <div class="search-container">
    <h1 class="logo" on:click={handleHome}>p2pollo</h1>
    <div class="input-row">
      <input
        type="text"
        bind:value={query}
        on:keydown={handleKeydown}
        placeholder="Buscar pelicula, serie, anime..."
        disabled={loading}
      />
      <button on:click={handleSearch} disabled={loading || !query.trim()}>
        {#if loading}
          Buscando...
        {:else}
          Buscar
        {/if}
      </button>
    </div>
  </div>
</div>

<style>
  .search-bar {
    padding: 20px 24px 12px;
    background: #161616;
    border-bottom: 1px solid #2a2a2a;
  }

  .logo {
    font-size: 1.4rem;
    font-weight: 700;
    color: #ff6b35;
    margin-bottom: 12px;
    letter-spacing: -0.5px;
    cursor: pointer;
    user-select: none;
  }

  .input-row {
    display: flex;
    gap: 8px;
  }

  input {
    flex: 1;
    padding: 10px 14px;
    border-radius: 6px;
    border: 1px solid #333;
    background: #1e1e1e;
    color: #e0e0e0;
    font-size: 0.95rem;
    font-family: inherit;
    outline: none;
    transition: border-color 0.2s;
  }

  input:focus {
    border-color: #ff6b35;
  }

  input:disabled {
    opacity: 0.5;
  }

  input::placeholder {
    color: #666;
  }

  button {
    padding: 10px 20px;
    border-radius: 6px;
    border: none;
    background: #ff6b35;
    color: white;
    font-size: 0.95rem;
    font-family: inherit;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.2s;
    white-space: nowrap;
  }

  button:hover:not(:disabled) {
    background: #e55a2b;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
