# Plan de Migración: HTML5 Video → mpv IPC

## Contexto

**Problema actual:** Los subtítulos embedded en videos (MKV, MP4) tardan ~40 segundos en extraerse con ffmpeg porque HTML5 `<video>` no puede leer subtítulos directamente de containers.

**Solución:** Migrar a mpv controlado via IPC desde Wails. mpv puede renderizar subtítulos embedded instantáneamente sin extracción.

## Arquitectura Nueva vs Antigua

### Antigua (HTML5 `<video>`)
```
Usuario hace click en subtitulo
  → Go extrae subtítulo con ffmpeg (~40s)
  → Go sirve WebVTT via HTTP
  → <video> carga track
```

### Nueva (mpv IPC)
```
Usuario hace click en subtitulo
  → Svelte llama MPVCommand("cycle", "sub")
  → Go envia JSON IPC a mpv
  → mpv cambia subtítulo instantáneamente (0.1s)
```

## Fases de Implementación

### Fase 1: Nuevo mpv IPC Controller
**Objetivo:** Crear el controlador de mpv con IPC (reemplaza internal/player legacy).

**Archivos a crear:**
- `internal/player/mpv.go` — Controlador mpv con IPC
- `internal/player/ipc.go` — Comunicación IPC (pipe/socket)
- `internal/player/commands.go` — Comandos y propiedades de mpv

**Funcionalidad:**
- Spawn mpv con `--input-ipc-server` y `--idle`
- Comunicación bidireccional via named pipe (Windows) o Unix socket
- Comandos: `loadfile`, `set_property`, `cycle`, `seek`, etc
- Propiedades: `time-pos`, `duration`, `pause`, `track-list`, etc
- Manejo de eventos de mpv (EOF, error, property changes)

**Test:**
```bash
go test -v ./internal/player
```

### Fase 2: Integración con Streaming Service
**Objetivo:** Conectar mpv controller con el servicio de streaming.

**Archivos a modificar:**
- `internal/streaming/streaming.go`:
  - Agregar campo `player *player.MPV`
  - Inicializar mpv en `StartStreamAsync`
  - Enviar `loadfile` cuando el stream esté listo
  - Cleanup de mpv en `Stop()`

**Cambios específicos:**
- Eliminar servidor HTTP (mpv lee directamente del archivo temporal)
- Mantener descarga torrent con buffer
- mpv renderiza video + subtítulos embedded directamente
- Eliminar extracción de subtítulos (ffmpeg)

**Test:**
```bash
wails dev
# Probar playback de video con subtítulos embedded
```

### Fase 3: Frontend - Bindings Go
**Objetivo:** Exponer comandos de mpv al frontend via Wails.

**Archivos a modificar:**
- `app.go`:
  - Agregar método `MPVCommand(cmd string, args []interface{}) error`
  - Agregar método `GetMPVProperty(name string) (interface{}, error)`
  - Agregar método `GetMPVTracks() []TrackInfo` — lista de audio/video/sub tracks
  - Actualizar `StreamProgress` con estado de mpv (playing, paused, time-pos)

**Tipos nuevos:**
```go
type TrackInfo struct {
    ID       int    `json:"id"`
    Type     string `json:"type"` // "video", "audio", "sub"
    Language string `json:"language"`
    Title    string `json:"title"`
    Selected bool   `json:"selected"`
}
```

**Test:**
```bash
wails dev
# Verificar que bindings se generan en frontend/wailsjs/go/main/
```

### Fase 4: Frontend - UI Controls
**Objetivo:** Actualizar componentes Svelte para controlar mpv.

**Archivos a modificar:**
- `frontend/src/components/Player.svelte`:
  - Eliminar `<video>` element
  - Reemplazar con info de reproducción (título, tiempo, duración)
  - Controles: play/pause → `MPVCommand("cycle", "pause")`
  - Seek → `MPVCommand("seek", [seconds])`
  - Polling de progreso con `GetStreamProgress()` (incluye time-pos de mpv)

- `frontend/src/components/player/SubtitleControl.svelte`:
  - Cargar tracks con `GetMPVTracks()`
  - Click subtítulo → `MPVCommand("set_property", ["sid", trackID])`
  - Mostrar track activo desde mpv
  - Delay de subtítulos → `MPVCommand("add", ["sub-delay", 0.5])`

- `frontend/src/components/player/PlaybackControls.svelte`:
  - Play/Pause → `MPVCommand("cycle", "pause")`
  - Seek → `MPVCommand("seek", [seconds, "absolute"])`
  - Fullscreen → `MPVCommand("cycle", "fullscreen")`
  - Volume → `MPVCommand("set_property", ["volume", value])`

**Test:**
```bash
wails dev
# Probar todos los controles:
# - Play/Pause
# - Seek
# - Cambio de subtítulos (debe ser instantáneo)
# - Delay de subtítulos
# - Volume
# - Fullscreen
```

### Fase 5: Cleanup - Eliminar Legacy
**Objetivo:** Eliminar todo código obsoleto.

**Archivos/directorios a eliminar:**
```bash
rm -rf cmd/                          # CLI legacy (Cobra)
rm -rf internal/search/              # Búsqueda legacy por trackers
rm internal/streaming/streaming.go   # Revisar y eliminar código de:
                                     # - handleStream (HTTP server)
                                     # - handleSubtitles (HTTP server)
                                     # - extractSubtitles (ffmpeg)
                                     # - startHTTPServer
                                     # - subtitleCache, prewarmSubtitleCache
```

**Archivos a modificar:**
- `app.go`:
  - Eliminar métodos: `GetSubtitleTracks()` (ahora es `GetMPVTracks()`)
  - Revisar si hay referencias a código eliminado

**Verificar imports:**
```bash
go mod tidy
```

**Test:**
```bash
go vet ./...
go test -v ./...
wails build
```

### Fase 6: Documentación
**Objetivo:** Actualizar docs con la nueva arquitectura.

**Archivos a actualizar:**
- ✅ `CLAUDE.md` — Ya actualizado con arquitectura mpv IPC
- `README.md` — Agregar requisitos de mpv, instrucciones de instalación
- `config/config.yaml` — Agregar sección `player.mpv_path` si no existe

**Crear nuevo archivo:**
- `docs/MPV_INTEGRATION.md` — Guía técnica de la integración mpv IPC

## Orden de Ejecución

1. ✅ **Actualizar CLAUDE.md** (completado)
2. **Fase 1:** Implementar mpv IPC controller
3. **Fase 2:** Integrar mpv con streaming service
4. **Fase 3:** Exponer bindings Go a frontend
5. **Fase 4:** Actualizar UI Svelte
6. **Fase 5:** Eliminar código legacy
7. **Fase 6:** Actualizar documentación final
8. **Verificación final:** Build completo y test manual de todas las features

## Criterios de Éxito

- ✅ Cambio de subtítulos es instantáneo (<1 segundo)
- ✅ Reproducción funciona igual que antes (buffer torrent + playback)
- ✅ Todos los controles funcionan (play/pause, seek, volume, fullscreen)
- ✅ No quedan referencias a código legacy (cmd/, internal/search/)
- ✅ `wails build` compila sin errores
- ✅ Tests pasan: `go test -v ./...`
- ✅ Lint pasa: `make lint`

## Riesgos y Mitigaciones

**Riesgo 1:** mpv no está instalado en la máquina del usuario
- **Mitigación:** Detectar en startup, mostrar error con link de descarga

**Riesgo 2:** IPC pipe/socket no funciona en todas las plataformas
- **Mitigación:** Probar en Windows, Linux, macOS. Fallback a archivo temporal con polling.

**Riesgo 3:** Pérdida de features de HTML5 `<video>` (PiP, media keys)
- **Mitigación:** mpv soporta media keys nativamente. PiP se puede implementar después si se necesita.

## Rollback Plan

Si la migración falla, el código anterior está en git:
```bash
git checkout HEAD~1  # Volver al commit anterior
```

Mantener rama separada durante desarrollo:
```bash
git checkout -b feature/mpv-integration
# Desarrollar aquí
# Merge a main solo cuando todo funcione
```

## Timeline Estimado

- Fase 1: 2-3 horas (mpv IPC controller)
- Fase 2: 1-2 horas (integración streaming)
- Fase 3: 1 hora (bindings Go)
- Fase 4: 2-3 horas (UI Svelte)
- Fase 5: 30 min (cleanup)
- Fase 6: 30 min (docs)
- Testing: 1 hora

**Total: ~8-10 horas de desarrollo**
