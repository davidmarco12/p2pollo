<script>
  import { createEventDispatcher } from 'svelte'
  import { formatTime } from '../utils/playerUtils.js'

  const dispatch = createEventDispatcher()

  export let seekPercent = 0
  export let actualTime = 0
  export let videoDuration = 0

  function handleSeekBarClick(event) {
    const bar = event.currentTarget
    const rect = bar.getBoundingClientRect()
    const percent = (event.clientX - rect.left) / rect.width
    dispatch('seek', percent)
  }
</script>

<div class="seek-overlay">
  <div class="seek-controls">
    <button class="seek-btn" on:click={() => dispatch('back')} title="Retroceder 10s">-10s</button>
    <div class="seek-bar" on:click={handleSeekBarClick}>
      <div class="seek-fill" style="width: {seekPercent}%"></div>
    </div>
    <button class="seek-btn" on:click={() => dispatch('forward')} title="Avanzar 30s">+30s</button>
    <span class="seek-time">{formatTime(actualTime)} / {formatTime(videoDuration)}</span>
  </div>
</div>

<style src="./SeekOverlay.css"></style>
