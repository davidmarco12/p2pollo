# Plan de Refactor - p2pollo

## Objetivo

Separar la lógica de negocio de la capa de presentación (CLI) para que el core sea reutilizable cuando se migre a UI. Reducir `play.go` de ~486 líneas a ~80 líneas.

---

## Diagnóstico actual

### Archivos y problemas

| Archivo | Líneas | Problema |
|---------|--------|---------|
| `cmd/play.go` | 486 | God function: mezcla torrent, I/O, moov atom, buffer, progress, mpv |
| `cmd/search.go` | 121 | Presentación CLI acoplada con lógica de búsqueda |
| `cmd/config.go` | 175 | Presentación CLI acoplada con lógica de config |
| `cmd/root.go` | 99 | Profiling duplicado con `profiling.go` |
| `internal/stream/stream.go` | 327 | Existe pero NO se usa. `play.go` reimplementa todo |
| `internal/client/client.go` | 370 | OK - wrapper limpio de anacrolix/torrent |
| `internal/player/player.go` | 458 | Métodos muertos (Seek, Pause, SetVolume no envían comandos a mpv) |
| `internal/search/search.go` | 365 | Data mockeada en `searchInTrackers()` |
| `internal/config/config.go` | 289 | OK - está bien estructurado |
| `profiling.go` | 133 | Funciones exportadas que nadie llama, duplica `root.go` |

### Problema principal

`cmd/play.go:runPlay()` hace TODO en una sola función:
1. Crear cliente torrent
2. Agregar magnet y esperar metadata
3. Auto-detectar archivo multimedia
4. Priorizar piezas (inicio + moov atom)
5. Crear temp file con Truncate
6. Goroutine moov atom (WriteAt al final)
7. Goroutine copia secuencial (WriteAt desde inicio)
8. Tracking de progreso con atomic
9. Esperar condiciones (moov + 200MB)
10. Lanzar mpv
11. Monitorear descarga durante reproducción
12. Cleanup

Cuando se migre a UI, los pasos 1-9 y 11-12 son identicos. Solo cambia cómo se muestra el progreso.

---

## Estructura propuesta

```
p2pollo/
├── main.go
│
├── internal/
│   ├── torrent/                  # Renombrar de client/
│   │   ├── client.go             # Wrapper anacrolix (existente, sin cambios)
│   │   └── fileselect.go         # findMediaFile + IsVideoFile
│   │
│   ├── stream/                   # REESCRIBIR - absorbe lógica de play.go
│   │   └── manager.go            # Manager con API basada en channels
│   │
│   ├── player/                   # Limpiar métodos muertos
│   │   └── player.go
│   │
│   ├── search/                   # Preparar para scrapers reales
│   │   ├── engine.go             # Interface + filtros + cache (renombrar search.go)
│   │   └── providers/            # (futuro) cada scraper como provider
│   │       └── nyaa.go
│   │
│   └── config/                   # Sin cambios
│       └── config.go
│
├── cmd/                          # CLI thin wrappers (futuro: reemplazar por ui/)
│   ├── root.go
│   ├── play.go                   # ~80 líneas
│   ├── search.go                 # ~50 líneas
│   └── config.go                 # ~50 líneas
│
└── profiling.go                  # Limpiar: dejar solo lo que se usa
```

---

## API del nuevo stream.Manager

```go
// internal/stream/manager.go

type Options struct {
    MagnetURI   string
    FileIndex   int    // -1 = auto-detectar
    NoBuffer    bool
    LowMemory   bool
    BufferSize  int64  // bytes, default 200MB
    MoovSize    int64  // bytes, default 5MB
}

type Progress struct {
    HeadWritten int64   // bytes escritos secuencialmente
    TotalSize   int64   // tamaño total del archivo
    Speed       float64 // MB/s
    MoovReady   bool
    Peers       int
    Percent     int     // 0-100
}

type Manager struct { ... }

// New crea el manager con config y cliente torrent
func New(cfg *config.Config, torrentClient *torrent.Client) *Manager

// Prepare agrega el magnet, espera metadata, retorna info del torrent
// Acá se hace: AddMagnet + WaitForInfo + auto-detect file
func (m *Manager) Prepare(magnetURI string, fileIndex int) (*torrent.TorrentInfo, int, error)

// Start inicia la descarga + escritura a temp file
// Lanza las goroutines de moov atom y copia secuencial
// Retorna el channel de "listo para reproducir"
func (m *Manager) Start(fileIndex int) error

// Ready retorna un channel que se cierra cuando el archivo está listo para mpv
// (moov descargado + buffer mínimo alcanzado)
func (m *Manager) Ready() <-chan string  // string = tmpPath

// Progress retorna el estado actual (para que CLI/UI lo muestren)
func (m *Manager) Progress() Progress

// Stop detiene la descarga y limpia recursos
func (m *Manager) Stop()

// TmpPath retorna la ruta del archivo temporal
func (m *Manager) TmpPath() string
```

---

## Pasos de implementación

### Fase 1: Mover fileselect (bajo riesgo)

1. Crear `internal/torrent/fileselect.go`
2. Mover `findMediaFile()` de `cmd/play.go` a `torrent.FindMediaFile()`
3. Mover `IsVideoExt()` como función pública
4. Actualizar import en `cmd/play.go`
5. Correr tests

### Fase 2: Renombrar client → torrent (bajo riesgo)

1. Renombrar `internal/client/` → `internal/torrent/`
2. Cambiar `package client` → `package torrent` en todos los archivos
3. Actualizar todos los imports:
   - `cmd/play.go`
   - `cmd/search.go` (no lo usa directamente)
   - `internal/stream/stream.go`
   - `internal/player/player.go` (no lo usa directamente)
   - Tests
4. `go build` + `go test`

### Fase 3: Reescribir stream.Manager (riesgo medio)

Este es el paso más grande. Extraer de `play.go` toda la lógica de:
- Priorización de piezas (inicio + moov atom)
- Creación de temp file con Truncate
- Goroutine de moov atom
- Goroutine de copia secuencial
- Buffer tracking con atomic
- Condición de "listo" (moov + buffer)

El nuevo `stream/manager.go` NO debe tener ningún `fmt.Print`, `color.Red`, ni nada de presentación. Solo expone `Progress()` y channels.

**Archivos a crear/modificar:**
- Reescribir `internal/stream/stream.go` → `internal/stream/manager.go`
- Eliminar el `stream.go` viejo (no se usa)

### Fase 4: Adelgazar cmd/play.go (riesgo medio)

Reescribir `play.go` para que use `stream.Manager`:

```go
func runPlay(cmd *cobra.Command, args []string) {
    cfg := loadConfig()

    // Crear cliente y manager
    c, _ := torrent.NewWithOptions(cfg, playLowMemory)
    defer c.Close()
    m := stream.New(cfg, c)

    // Preparar: agregar magnet, esperar metadata, auto-detect file
    info, fileIdx, _ := m.Prepare(args[0], playFileIndex)
    printTorrentInfo(info, fileIdx)

    // Iniciar descarga
    m.Start(fileIdx, stream.Options{NoBuffer: playNoBuffer})

    // Mostrar progreso hasta que esté listo
    ready := m.Ready()
    ticker := time.NewTicker(500 * time.Millisecond)
    for {
        select {
        case tmpPath := <-ready:
            // Lanzar mpv
            p, _ := player.New(cfg)
            p.PlayFile(tmpPath)
            p.Wait()
            m.Stop()
            return
        case <-ticker.C:
            prog := m.Progress()
            printProgress(prog) // función local de ~10 líneas
        }
    }
}
```

### Fase 5: Limpiar player.go (bajo riesgo)

1. Eliminar métodos muertos: `Seek()`, `SetVolume()`, `Pause()` (no envían comandos a mpv)
2. Eliminar `Play()` que crea su propio temp file (duplica lógica de stream manager)
3. Dejar solo: `New()`, `PlayFile()`, `Wait()`, `Close()`, `Stop()`

### Fase 6: Limpiar profiling (bajo riesgo)

1. Eliminar `profiling.go` del root
2. Dejar profiling solo en `cmd/root.go` (ya lo tiene)
3. Si se necesitan utilities de profiling, mover a `internal/debug/profiling.go`

### Fase 7: Limpiar search (bajo riesgo, preparar futuro)

1. Renombrar `search.go` → `engine.go`
2. Extraer interface `Provider` para futuros scrapers:
   ```go
   type Provider interface {
       Search(query string) ([]Result, error)
       Name() string
   }
   ```
3. Mover data mockeada a un `providers/mock.go`
4. Adelgazar `cmd/search.go` (separar presentación de lógica)

---

## Orden de ejecución recomendado

```
Fase 1 (fileselect)     →  30 min   → bajo riesgo
Fase 2 (rename client)  →  30 min   → bajo riesgo
Fase 3 (stream.Manager) →  2-3 hrs  → riesgo medio ← el más importante
Fase 4 (play.go thin)   →  1 hr     → riesgo medio
Fase 5 (player cleanup) →  30 min   → bajo riesgo
Fase 6 (profiling)      →  15 min   → bajo riesgo
Fase 7 (search)         →  1 hr     → bajo riesgo
```

Fases 1-2 se pueden hacer juntas. Fases 3-4 son dependientes (hacer juntas). Fases 5-7 son independientes.

---

## Criterio de éxito

- `cmd/play.go` tiene menos de 100 líneas
- `cmd/play.go` no importa `io`, `os`, `sync/atomic`, `path/filepath`
- `internal/stream/manager.go` no importa `fmt`, `color`, `cobra`
- `go test ./...` pasa
- El streaming funciona igual que antes (moov atom + 200MB buffer + mpv)
- `internal/stream` es usable desde cualquier frontend (CLI, UI, API)

---

## Notas para la migración a UI

Cuando se reemplace CLI por UI:

1. Eliminar `cmd/` completo
2. Crear `ui/` que importe `internal/stream`, `internal/torrent`, `internal/search`
3. `stream.Manager.Progress()` alimenta widgets de la UI
4. `stream.Manager.Ready()` dispara el launch de mpv
5. `search.Engine` alimenta la lista de resultados en la UI
6. `config.Load/Save` alimenta la pantalla de settings

El core (`internal/`) no cambia nada.
