<script>
  import { createEventDispatcher } from 'svelte'
  import logo from '../../assets/images/output-estesi.png'

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
    <img src={logo} class="logo" on:click={handleHome} alt="p2pollo" />
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

<style>
  .search-bar {
    padding: 20px 24px 12px;
    background: #161616;
    border-bottom: 1px solid #2a2a2a;
  }

  .search-container {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .logo {
    height: 40px;
    width: auto;
    flex-shrink: 0;
    cursor: pointer;
    user-select: none;
  }

  input {
    flex: 1;
    min-width: 0;
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
