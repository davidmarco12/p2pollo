// Parsear timestamp VTT "HH:MM:SS.mmm" a segundos
export function parseVTTTime(ts) {
  ts = ts.trim().split(' ')[0]
  const parts = ts.split(':')
  let secs = 0
  for (const p of parts) secs = secs * 60 + parseFloat(p)
  return secs
}

// Parsear archivo WebVTT a array de cues { start, end, text }
export function parseVTTCues(vttText) {
  const cues = []
  const lines = vttText.split('\n')
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    if (!line.includes('-->')) continue
    const arrowIdx = line.indexOf('-->')
    const start = parseVTTTime(line.substring(0, arrowIdx))
    const end = parseVTTTime(line.substring(arrowIdx + 3))
    const textLines = []
    i++
    while (i < lines.length && lines[i].trim() !== '') {
      textLines.push(lines[i])
      i++
    }
    if (textLines.length > 0) {
      cues.push({ start, end, text: textLines.join('\n') })
    }
  }
  return cues
}

export function formatTime(seconds) {
  if (!seconds || !isFinite(seconds)) return '0:00'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  if (h > 0) return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  return `${m}:${s.toString().padStart(2, '0')}`
}

export function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let val = bytes
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024
    i++
  }
  return `${val.toFixed(1)} ${units[i]}`
}
