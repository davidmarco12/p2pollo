<script>
  import { createEventDispatcher } from 'svelte'

  const dispatch = createEventDispatcher()

  export let tracks = []
  export let activeTrack = null
  export let activeLabel = ''
  export let hasCues = false
  export let delay = 0

  let showMenu = false

  function selectTrack(track) {
    showMenu = false
    dispatch('select', track)
  }
</script>

{#if tracks.length > 0}
  <div class="subtitle-control">
    <button
      class="subtitle-btn"
      class:active={hasCues}
      on:click={() => showMenu = !showMenu}
      title="Subtitulos"
    >
      CC
    </button>

    {#if showMenu}
      <div class="subtitle-menu">
        <button
          class="subtitle-option"
          class:selected={!activeTrack}
          on:click={() => selectTrack(null)}
        >
          Desactivados
        </button>

        {#each tracks as track}
          <button
            class="subtitle-option"
            class:selected={activeLabel === (track.title || track.language)}
            on:click={() => selectTrack(track)}
          >
            {track.title || track.language}
            {#if track.type === 'external'}
              <span class="track-badge">SRT</span>
            {/if}
          </button>
        {/each}

        {#if activeTrack}
          <div class="subtitle-delay-row">
            <span>Retardo:</span>
            <button class="delay-btn" on:click={() => dispatch('delayChange', delay - 0.5)}>-0.5s</button>
            <span class="delay-value">{delay >= 0 ? '+' : ''}{delay.toFixed(1)}s</span>
            <button class="delay-btn" on:click={() => dispatch('delayChange', delay + 0.5)}>+0.5s</button>
            {#if delay !== 0}
              <button class="delay-btn" on:click={() => dispatch('delayChange', 0)}>↺</button>
            {/if}
          </div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style src="./SubtitleControl.css"></style>
