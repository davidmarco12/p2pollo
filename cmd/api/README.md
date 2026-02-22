# API HTTP de p2pollo

Servidor HTTP REST que expone el backend Go (torrent, catálogo YTS, streaming, mpv) para que el frontend Qt/QML pueda consumirlo.

## Ejecución

```bash
# Desde la raíz del proyecto
./api.bat

# O directamente
./build/bin/api.exe
```

El servidor escucha en `http://127.0.0.1:9876`

## Endpoints

### Health Check

```http
GET /api/health
```

Respuesta:
```json
{
  "status": "ok"
}
```

---

### Catálogo de Películas (YTS)

#### Obtener películas populares

```http
GET /api/popular
```

Respuesta:
```json
[
  {
    "id": "12345",
    "imdbId": "tt1234567",
    "title": "Movie Title",
    "year": 2024,
    "rating": 8.5,
    "posterUrl": "https://...",
    "genres": "Action, Adventure"
  },
  ...
]
```

#### Buscar películas

```http
GET /api/search?q=batman
```

Parámetros:
- `q` (string): query de búsqueda

Respuesta: igual que `/api/popular`

#### Obtener detalles de película (incluyendo torrents)

```http
POST /api/movie/details
Content-Type: application/json

{
  "id": "12345",
  "imdbId": "tt1234567",
  "title": "Movie Title",
  "year": 2024,
  "rating": 8.5,
  "posterUrl": "https://...",
  "genres": "Action"
}
```

Respuesta:
```json
{
  "id": "12345",
  "imdbId": "tt1234567",
  "title": "Movie Title",
  "year": 2024,
  "rating": 8.5,
  "posterUrl": "https://...",
  "genres": "Action",
  "description": "Movie plot...",
  "runtime": 120,
  "torrents": [
    {
      "hash": "abc123...",
      "quality": "1080p",
      "type": "web",
      "size": "2.3 GB",
      "seeds": 123,
      "peers": 45,
      "magnetLink": "magnet:?xt=urn:btih:..."
    },
    ...
  ]
}
```

---

### Streaming

#### Iniciar streaming

```http
POST /api/play
Content-Type: application/json

{
  "magnetLink": "magnet:?xt=urn:btih:...",
  "fileIndex": -1
}
```

Parámetros:
- `magnetLink` (string): enlace magnet del torrent
- `fileIndex` (int): índice del archivo a reproducir, -1 para auto-selección

Respuesta:
```json
{
  "status": "streaming started"
}
```

#### Obtener path del archivo temporal

**IMPORTANTE para Qt/QML con MpvQt**: Este endpoint retorna la ruta local del archivo temporal que se está descargando. Qt debe usar esta ruta para cargar el archivo en mpv.

```http
GET /api/stream-path
```

Respuesta:
```json
{
  "path": "C:\\Users\\...\\p2pollo-stream-123.mp4",
  "ready": true
}
```

- `ready`: `true` cuando el buffer inicial está listo y mpv puede empezar a reproducir
- `path`: ruta completa al archivo temporal

**Uso en Qt/QML:**
```qml
// Polling cada 1 segundo hasta que ready == true
Timer {
    interval: 1000
    repeat: true
    running: true
    onTriggered: {
        fetch("/api/stream-path", function(data) {
            if (data.ready && data.path) {
                mpvPlayer.loadFile(data.path)
                stop()
            }
        })
    }
}
```

#### Obtener progreso de descarga

```http
GET /api/progress
```

Respuesta:
```json
{
  "preparing": false,
  "error": "",
  "headWritten": 12345678,
  "totalSize": 100000000,
  "speedMBps": 5.2,
  "percent": 12,
  "peers": 45,
  "videoDuration": 7200.5
}
```

#### Detener streaming

```http
POST /api/stop
```

Respuesta:
```json
{
  "status": "stopped"
}
```

---

### Control de mpv

#### Obtener estado de reproducción

```http
GET /api/mpv/state
```

Respuesta:
```json
{
  "timePos": 125.5,
  "duration": 7200.0,
  "paused": false,
  "volume": 80
}
```

#### Obtener tracks (audio, video, subtítulos)

```http
GET /api/mpv/tracks
```

Respuesta:
```json
[
  {
    "id": 1,
    "type": "video",
    "language": "eng",
    "title": "H.264 1080p",
    "selected": true
  },
  {
    "id": 2,
    "type": "audio",
    "language": "eng",
    "title": "English",
    "selected": true
  },
  {
    "id": 3,
    "type": "sub",
    "language": "spa",
    "title": "Spanish",
    "selected": false
  },
  ...
]
```

#### Enviar comando a mpv

```http
POST /api/mpv/command
Content-Type: application/json

{
  "args": ["cycle", "pause"]
}
```

Parámetros:
- `args` (array): array de argumentos, el primer elemento es el comando

Ejemplos de comandos:
- `["cycle", "pause"]` - toggle play/pause
- `["seek", 10, "relative"]` - saltar 10 segundos adelante
- `["seek", 100, "absolute"]` - ir a la posición 100 segundos
- `["cycle", "fullscreen"]` - toggle pantalla completa
- `["add", "sub-delay", 0.5]` - ajustar delay de subtítulos

Respuesta:
```json
{
  "status": "ok"
}
```

#### Establecer propiedad de mpv

```http
POST /api/mpv/property
Content-Type: application/json

{
  "name": "volume",
  "value": 80
}
```

Propiedades comunes:
- `volume` (int): volumen 0-100
- `pause` (bool): pausar/reproducir
- `sid` (int|"no"): seleccionar track de subtítulos (ID o "no" para desactivar)
- `sub-delay` (float): delay de subtítulos en segundos

Respuesta:
```json
{
  "status": "ok"
}
```

#### Obtener propiedad de mpv

```http
GET /api/mpv/property/:name
```

Ejemplo:
```http
GET /api/mpv/property/volume
```

Respuesta:
```json
{
  "value": 80
}
```

---

## Flujo típico de uso (Qt/QML)

1. **Home/Inicio**: `GET /api/popular` → mostrar grilla de posters
2. **Búsqueda**: `GET /api/search?q=batman` → mostrar resultados
3. **Detalles**: `POST /api/movie/details` → mostrar info + torrents disponibles
4. **Play**:
   - `POST /api/play` con magnetLink
   - Polling `GET /api/stream-path` cada 1 segundo hasta que `ready == true`
   - Cuando `ready`, cargar `path` en mpv con `mpvPlayer.loadFile(path)`
   - Polling `GET /api/progress` para mostrar progreso de descarga
   - Polling `GET /api/mpv/state` para sincronizar UI con mpv
5. **Control**:
   - `POST /api/mpv/command` para play/pause, seek, etc.
   - `POST /api/mpv/property` para volumen, subtítulos, etc.
   - `GET /api/mpv/tracks` para lista de subtítulos
6. **Stop**: `POST /api/stop` cuando el usuario cierra el player

---

## Errores

Todos los endpoints retornan status HTTP apropiados:
- `200 OK` - éxito
- `400 Bad Request` - parámetros inválidos
- `500 Internal Server Error` - error del servidor
- `503 Service Unavailable` - mpv no está ejecutándose

Formato de error:
```json
{
  "error": "descripción del error"
}
```

---

## Desarrollo

### Compilar

```bash
go build -o build/bin/api.exe ./cmd/api
```

### Ejecutar en modo desarrollo

```bash
# Con logs debug
export GIN_MODE=debug
go run ./cmd/api
```

### Testing con curl

```bash
# Health check
curl http://127.0.0.1:9876/api/health

# Películas populares
curl http://127.0.0.1:9876/api/popular

# Buscar
curl "http://127.0.0.1:9876/api/search?q=batman"

# Iniciar streaming
curl -X POST http://127.0.0.1:9876/api/play \
  -H "Content-Type: application/json" \
  -d '{"magnetLink":"magnet:?xt=urn:btih:...", "fileIndex":-1}'

# Ver progreso
curl http://127.0.0.1:9876/api/progress

# Path del archivo
curl http://127.0.0.1:9876/api/stream-path

# Comando mpv (play/pause)
curl -X POST http://127.0.0.1:9876/api/mpv/command \
  -H "Content-Type: application/json" \
  -d '{"args":["cycle","pause"]}'
```

---

## Notas técnicas

- El servidor usa CORS permisivo (`Allow-Origin: *`) para desarrollo
- mpv se ejecuta como proceso separado (ventana externa)
- Los archivos temporales se guardan en el directorio del sistema (temp)
- El streaming usa descarga secuencial con priorización de piezas del inicio
- El buffer inicial es de 50MB (configurable en `config.yaml`)
