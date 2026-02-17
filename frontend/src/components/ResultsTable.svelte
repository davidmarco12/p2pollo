<script>
  import { createEventDispatcher } from 'svelte'

  const dispatch = createEventDispatcher()

  export let results = []

  function handlePlay(result) {
    dispatch('play', result)
  }

  function healthColor(health) {
    if (health >= 70) return '#4caf50'
    if (health >= 40) return '#ff9800'
    return '#f44336'
  }

  function seedColor(seeds) {
    if (seeds >= 50) return '#4caf50'
    if (seeds >= 10) return '#ff9800'
    return '#f44336'
  }
</script>

{#if results.length > 0}
  <div class="results-container">
    <div class="results-header">
      <span class="count">{results.length} resultados</span>
    </div>
    <div class="results-list">
      {#each results as result, i}
        <div class="result-row" on:click={() => handlePlay(result)}>
          <div class="result-info">
            <span class="result-name">{result.name}</span>
            <span class="result-source">{result.source}</span>
          </div>
          <div class="result-stats">
            <span class="stat size">{result.size}</span>
            <span class="stat seeds" style="color: {seedColor(result.seeds)}">
              {result.seeds} seeds
            </span>
            <span class="stat leechers">
              {result.leechers} leechers
            </span>
            <div class="health-bar">
              <div
                class="health-fill"
                style="width: {result.health}%; background: {healthColor(result.health)}"
              ></div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  </div>
{/if}

<style>
  .results-container {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .results-header {
    padding: 10px 24px;
    border-bottom: 1px solid #2a2a2a;
    background: #131313;
  }

  .count {
    font-size: 0.85rem;
    color: #888;
  }

  .results-list {
    flex: 1;
    overflow-y: auto;
    padding: 0;
  }

  .result-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 14px 24px;
    border-bottom: 1px solid #1a1a1a;
    cursor: pointer;
    transition: background 0.15s;
  }

  .result-row:hover {
    background: #1a1a1a;
  }

  .result-info {
    flex: 1;
    min-width: 0;
    margin-right: 16px;
  }

  .result-name {
    display: block;
    font-size: 0.95rem;
    color: #e0e0e0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .result-source {
    font-size: 0.75rem;
    color: #555;
    margin-top: 2px;
  }

  .size {
    color: #aaa;
    min-width: 80px;
    text-align: right;
  }

  .result-stats {
    display: flex;
    align-items: center;
    gap: 16px;
    flex-shrink: 0;
  }

  .stat {
    font-size: 0.85rem;
    white-space: nowrap;
  }

  .leechers {
    color: #888;
  }

  .health-bar {
    width: 50px;
    height: 4px;
    background: #2a2a2a;
    border-radius: 2px;
    overflow: hidden;
  }

  .health-fill {
    height: 100%;
    border-radius: 2px;
    transition: width 0.3s;
  }
</style>
