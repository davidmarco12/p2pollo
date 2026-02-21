<script>
  import { createEventDispatcher } from 'svelte'

  const dispatch = createEventDispatcher()

  export let tracks = []
  export let activeTrackID = null
  export let delay = 0

  let showMenu = false

  function selectTrack(trackID) {
    showMenu = false
    dispatch('select', trackID)
  }

  function changeDelay(delta) {
    dispatch('delayChange', delta)
  }

  function resetDelay() {
    dispatch('delayReset')
  }

  $: activeTrack = tracks.find(t => t.id === activeTrackID)
  $: hasSubtitles = activeTrackID !== null
</script>

{#if tracks.length > 0}
  <div class="subtitle-control">
    <button
      class="subtitle-btn"
      class:active={hasSubtitles}
      on:click={() => showMenu = !showMenu}
      title="Subtítulos"
    >
      CC
    </button>

    {#if showMenu}
      <div class="subtitle-menu">
        <button
          class="subtitle-option"
          class:selected={!hasSubtitles}
          on:click={() => selectTrack(null)}
        >
          Desactivados
        </button>

        {#each tracks as track}
          <button
            class="subtitle-option"
            class:selected={activeTrackID === track.id}
            on:click={() => selectTrack(track.id)}
          >
            {track.title || track.language || `Track ${track.id}`}
            {#if track.selected}
              <span class="track-badge">✓</span>
            {/if}
          </button>
        {/each}

        {#if hasSubtitles}
          <div class="subtitle-delay-row">
            <span>Retardo:</span>
            <button class="delay-btn" on:click={() => changeDelay(-0.5)}>-0.5s</button>
            <span class="delay-value">{delay >= 0 ? '+' : ''}{delay.toFixed(1)}s</span>
            <button class="delay-btn" on:click={() => changeDelay(0.5)}>+0.5s</button>
            {#if delay !== 0}
              <button class="delay-btn" on:click={resetDelay}>↺</button>
            {/if}
          </div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style src="./SubtitleControl.css"></style>
