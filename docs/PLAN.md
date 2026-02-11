# 📚 Guía de Arquitectura de p2pollo - Aprendiendo Go Concurrencia

## Tabla de Contenidos
1. [Visión General](#visión-general)
2. [Conceptos Clave de Go](#conceptos-clave-de-go)
3. [Flujo de Ejecución Principal](#flujo-de-ejecución-principal)
4. [Diagrama de Procesos](#diagrama-de-procesos)
5. [Análisis por Archivo](#análisis-por-archivo)
6. [Channels (Canales) Explicados](#channels-canales-explicados)
7. [Goroutines y Concurrencia](#goroutines-y-concurrencia)

---

## Visión General

p2pollo es una aplicación de streaming P2P de torrents en Go que permite **ver videos mientras se descargan**. La arquitectura está basada en:

- **P2P Torrent**: Descarga de piezas de torrent desde múltiples peers
- **Streaming en Vivo**: Reproducción de video mientras se descarga
- **Concurrencia**: Múltiples goroutines sincronizadas con canales

### Flujo Simple:
```
Magnet Link → Metadata → Download → Temp File → MPV Player → Video
   ↑                         ↓
   └─────── Buffers de 50MB → Inicia reproducción
```

---

## Conceptos Clave de Go

### 1. **Goroutines** 🚀
Son threads livianos manejados por el runtime de Go. Son mucho más eficientes que threads del SO.

```go
// Crear una goroutine (ejecución concurrente)
go func() {
    fmt.Println("Esta función ejecuta en paralelo")
}()

// La función principal continúa sin esperar
fmt.Println("Principal continúa")
```

**En p2pollo:**
- Goroutine para copiar datos del torrent al archivo temporal
- Goroutine para monitorear progreso de descarga
- Goroutine para mantener MPV abierto

### 2. **Channels (Canales)** 📡
Son tuberías para comunicación entre goroutines. Permiten sincronización segura.

```go
// Crear un canal de tipo bool
done := make(chan bool)

// Enviar un valor al canal (bloqueante)
done <- true

// Recibir del canal (bloqueante)
value := <-done

// Cerrar el canal
close(done)
```

**Tipos de canales:**
- **Sin buffer**: Bloquea hasta que alguien reciba
- **Con buffer**: Permite enviar N valores sin bloquear

```go
// Sin buffer - bloqueante
ch := make(chan int)

// Con buffer de 10
ch := make(chan int, 10)
```

### 3. **Select** 🔄
Permite esperar múltiples canales simultáneamente (como un switch para canales).

```go
select {
case msg := <-channel1:
    fmt.Println("Recibido de channel1:", msg)
case <-channel2:
    fmt.Println("Señal de channel2")
case <-timeout:
    fmt.Println("Timeout!")
}
```

**En p2pollo:** Se usa para esperar múltiples eventos (descarga, timeout, etc.)

### 4. **Mutex** 🔒
Exclusión mutua - asegura que solo una goroutine acceda a un recurso a la vez.

```go
var mu sync.Mutex
var counter int

go func() {
    mu.Lock()
    counter++
    mu.Unlock()
}()
```

---

## Flujo de Ejecución Principal

```
1. Usuario ejecuta: p2pollo play "magnet:..."
                          ↓
2. cmd/play.go → runPlay()
   - Carga configuración
   - Crea cliente torrent
                          ↓
3. internal/client/client.go → New()
   - Conecta con peers
   - Descarga metadata
                          ↓
4. Inicia descarga
   - Prioriza piezas para streaming
   - Download() para empezar descarga
                          ↓
5. Buffer inicial (50MB)
   - Espera a 50MB descargados
                          ↓
6. Estrategia adaptativa:
   - Archivo < 300MB: Espera descarga completa
   - Archivo >= 300MB: Inicia con 200MB de buffer
                          ↓
7. Goroutine de copia:
   - Lee desde anacrolix reader
   - Copia al archivo temporal
                          ↓
8. Lanza MPV
   - Abre ventana
   - Reproduce video
                          ↓
9. Monitorea:
   - Progreso de descarga
   - Estado de MPV
   - Cierre limpio
```

---

## Diagrama de Procesos

### Línea de Tiempo (Timeline)

```
TIEMPO →

main()
│
├─→ runPlay() [MAIN THREAD]
│   │
│   ├─→ client.New() ────────────────────────────────────────┐
│   │   (conecta con peers)                                  │
│   │                                                        │
│   ├─→ AddMagnet() ───────────────────────────────────────┐ │
│   │   (obtiene metadata)                                 │ │
│   │                                                      │ │
│   ├─→ Buffer inicial ──────────────────────────────────┐ │ │
│   │   Wait(50MB) [BLOQUEANTE] ◄─── (anacrolix)        │ │ │
│   │                                 descargando        │ │ │
│   │   [0s:________:50MB descargado]                   │ │ │
│   │                                                    │ │ │
│   └─→ crea tmpFile ──────────────────────────────┐    │ │ │
│       │                                          │    │ │ │
│       ├─→ [GOROUTINE] CopiarDatos()             │    │ │ │
│       │   │                                      │    │ │ │
│       │   └─→ Lee de reader anacrolix           │    │ │ │
│       │       Copia a tmpFile                   │    │ │ │
│       │       [Ejecuta en paralelo] ◄───────────┤    │ │ │
│       │       16.8MB... 69.8MB... 100%          │    │ │ │
│       │                                          │    │ │ │
│       ├─→ [GOROUTINE] Monitorear()              │    │ │ │
│       │   │                                      │    │ │ │
│       │   └─→ Cada 2s: muestra progreso        │    │ │ │
│       │       📊 56% | 69.8 MB | 12 peers      │    │ │ │
│       │       [Ejecuta en paralelo] ◄───────────┤    │ │ │
│       │                                          │    │ │ │
│       ├─→ Espera buffer mínimo [BLOQUEANTE]    │    │ │ │
│       │   ✓ Buffer listo 69.8MB                 │    │ │ │
│       │                                          │    │ │ │
│       ├─→ player.PlayFile(tmpPath)             │    │ │ │
│       │   │                                      │    │ │ │
│       │   └─→ Lanza MPV ◄─── [os.StartProcess]│    │ │ │
│       │       ✓ Reproduciendo                   │    │ │ │
│       │                                          │    │ │ │
│       ├─→ [GOROUTINE] Monitoreo Descarga      │    │ │ │
│       │   │                                      │    │ │ │
│       │   └─→ Cada 2s: muestra progreso       │    │ │ │
│       │       📊 78% | 96.7 MB | 15 peers     │    │ │ │
│       │       📊 100% | 123.3 MB | 0 peers    │    │ │ │
│       │       [Ejecuta en paralelo] ◄────────────┤  │ │
│       │                                          │  │ │
│       └─→ p.Wait() [BLOQUEANTE]                 │  │ │
│           Espera a que usuario cierre MPV      │  │ │
│           (usuario viendo video mientras)      │  │ │
│           ◄────── AQUÍ ES DONDE VES EL VIDEO! ┤  │ │
│                                                 │  │ │
└─────────────────────────────────────────────────┴──┴─┴──► tiempo
   (programa termina)
```

### Árbol de Goroutines

```
MAIN GOROUTINE (runPlay)
├── GOROUTINE 1: CopiarDatos
│   ├── Lee bucles: for { n, err := fileReader.Read(buf) }
│   ├── Escucha: case <-stopCopy (canal para parar)
│   └── Escribe: tmpFileHandle.Write(buf[:n])
│
├── GOROUTINE 2: MonitoreoBuffer (antes de reproducción)
│   ├── Tick cada 500ms
│   ├── Verifica: stat, _ := os.Stat(tmpPath)
│   └── Actualiza: progreso de buffer
│
├── GOROUTINE 3: MonitoreoDescargaMientrasReproduce
│   ├── Tick cada 2s
│   ├── Lee: t.BytesCompleted(), t.Peers()
│   └── Imprime: progreso en vivo
│
├── GOROUTINE 4: MPV (lanzado por StartProcess)
│   └── Reproduce video en ventana
│
└── [MAIN BLOQUEADA en p.Wait()]
    Espera a que MPV termine
```

---

## Análisis por Archivo

### 📄 `cmd/play.go` - ORQUESTADOR PRINCIPAL

**Responsabilidad:** Coordinar todo el flujo de reproducción

**Flujo:**
```go
func runPlay(cmd *cobra.Command, args []string) {
    magnetLink := args[0]
    
    // 1. Configuración
    cfg, err := config.Load(cfgFile)
    
    // 2. Cliente torrent
    c, err := client.New(cfg)
    
    // 3. Agregar torrent
    t, err := c.AddMagnet(magnetLink)
    
    // 4. Esperar metadata
    err = t.WaitForInfo(30 * time.Second)
    
    // 5. Crear archivo temporal
    tmpPath := filepath.Join(os.TempDir(), "p2pollo-stream-{timestamp}.mp4")
    tmpFileHandle, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY, 0644)
    
    // 6. ★ GOROUTINE 1: Copiar datos
    stopCopy := make(chan bool)  // ← Canal para parar la copia
    go func() {
        for {
            select {
            case <-stopCopy:  // ← Espera señal de parada
                return
            default:
                n, err := fileReader.Read(buf)
                tmpFileHandle.Write(buf[:n])
            }
        }
    }()
    
    // 7. ★ GOROUTINE 2: Monitorear descarga
    downloadTimeout := time.After(10 * time.Minute)  // ← Timeout
    ticker := time.NewTicker(500 * time.Millisecond)  // ← Tick cada 500ms
    
    for !bufferReady {
        select {
        case <-downloadTimeout:  // ← Se ejecuta si pasa 10 min
            fmt.Println("❌ Timeout")
            return
        case <-ticker.C:  // ← Se ejecuta cada 500ms
            stat, _ := os.Stat(tmpPath)
            fmt.Printf("Buffer: %d%%", progress)
        }
    }
    
    // 8. Lanzar MPV
    p, err := player.New(cfg)
    err = p.PlayFile(tmpPath)
    
    // 9. ★ GOROUTINE 3: Monitorear descarga mientras reproduce
    done := make(chan bool)
    go func() {
        ticker := time.NewTicker(2 * time.Second)
        for {
            select {
            case <-done:
                return
            case <-ticker.C:
                fmt.Printf("📊 %d%% | %.1f MB | %d peers",
                    progress,
                    float64(t.BytesCompleted())/(1024*1024),
                    t.Peers())
            }
        }
    }()
    
    // 10. ★ BLOQUEANTE: Esperar a que usuario cierre MPV
    waitErr := p.Wait()
    
    // 11. Limpiar
    close(done)
    close(stopCopy)
    os.Remove(tmpPath)
}
```

**Canales usados:**
- `stopCopy`: Señaliza a la goroutine de copia que pare
- `done`: Señaliza a la goroutine de monitoreo que pare
- `time.After()`: Canal de timeout
- `ticker.C`: Canal de timer

---

### 📄 `internal/client/client.go` - CLIENTE TORRENT

**Responsabilidad:** Interactuar con anacrolix/torrent

**Conceptos:**
```go
type Client struct {
    c   *torrent.Client
    log *logrus.Entry
    // Mutex para thread-safety
    mu  sync.RWMutex
}

type Torrent struct {
    t   *torrent.Torrent
    client *Client
    // RWMutex = Read-Write Mutex
    // Permite múltiples lectores, un escritor
    mu  sync.RWMutex
}
```

**Métodos importantes:**

```go
// NewFileReader: Retorna un reader para leer archivo específico
func (t *Torrent) NewFileReader(fileIndex int) (io.ReadSeeker, error) {
    t.mu.RLock()  // ← Lock para lectura (otro puede escribir después)
    defer t.mu.RUnlock()
    
    files := t.t.Files()
    return files[fileIndex].NewReader(), nil
}

// BytesCompleted: Retorna bytes ya descargados
func (t *Torrent) BytesCompleted() int64 {
    t.mu.RLock()
    defer t.mu.RUnlock()
    
    return t.t.BytesCompleted()
}
```

**Thread-safety con Mutex:**
```
GOROUTINE 1              GOROUTINE 2
├─ Lock Read            ├─ Lock Read
├─ Lee datos            ├─ Lee datos
└─ Unlock               └─ Unlock

(ambas pueden ejecutar en paralelo porque solo leen)

GOROUTINE 1              GOROUTINE 2
├─ Lock Write           ├─ Espera...
├─ Modifica             ├─ Espera Lock...
└─ Unlock               └─ Ahora puede proceder
```

---

### 📄 `internal/player/player.go` - CONTROLADOR MPV

**Responsabilidad:** Lanzar y gestionar proceso MPV

**Flujo de PlayFile:**

```go
func (m *MPV) PlayFile(filepath string) error {
    m.mu.Lock()  // ← Mutex exclusivo
    defer m.mu.Unlock()
    
    // 1. Construir argumentos
    args := []string{
        "--force-window=yes",
        "--cache=yes",
        "--cache-secs=120",
        filepath,
        "--osd-on-seek=msg",
    }
    
    // 2. Obtener ruta completa de MPV
    mpvPath := m.config.Player.MPVPath  // "mpv"
    fullPath, err := exec.LookPath(mpvPath)  // → "C:\...\mpv.exe"
    
    // 3. Crear atributos del proceso
    procAttr := &os.ProcAttr{
        Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
    }
    
    // 4. Lanzar proceso
    proc, err := os.StartProcess(fullPath, append([]string{fullPath}, args...), procAttr)
    
    // 5. Guardar proceso en estructura
    m.cmd = &exec.Cmd{
        Path:    fullPath,
        Args:    append([]string{fullPath}, args...),
        Process: proc,
    }
    
    // 6. Flag: marcamos como ejecutándose
    m.running = true
    
    return nil
}
```

**Método Wait:**
```go
func (m *MPV) Wait() error {
    if m.cmd == nil || m.cmd.Process == nil {
        return fmt.Errorf("mpv no está ejecutándose")
    }
    
    // Bloquea hasta que proceso termine
    state, err := m.cmd.Process.Wait()
    
    m.running = false
    return err
}
```

---

## Channels (Canales) Explicados

### Concepto Visual

```
ENVIADOR          CANAL          RECEPTOR
─────────────────────────────────────────

goroutine A       chan int       goroutine B
    │                │                │
    │ value ←────→   │   ←────→ value │
    │                │                │
    └─ envía         │         recibe ─┘
       (bloqueante)  │      (bloqueante)
```

### Ejemplo en p2pollo: `stopCopy`

```go
// LÍNEA 1: Crear canal
stopCopy := make(chan bool)

// LÍNEA 2: Goroutine que recibe del canal
go func() {
    for {
        select {
        case <-stopCopy:  // ← Espera recibir del canal
            fmt.Println("Parando...")
            return  // ← Sale del loop
        default:
            // Hacer trabajo...
        }
    }
}()

// LÍNEA 3: Main goroutine envía señal
close(stopCopy)  // ← Cierra el canal
// Todas las goroutines que esperen en <-stopCopy se despiertan
```

### Canales con Buffer

```go
// Sin buffer: bloqueante si no hay receptor
ch := make(chan int)

// Con buffer: permite 10 envíos sin bloquear
ch := make(chan int, 10)

// Visualización:
go func() {
    for i := 0; i < 20; i++ {
        ch <- i  // Si buffer < 10, no bloquea
                 // Si buffer = 10, bloquea hasta que alguien reciba
    }
}()
```

### Patrón: Timeout + Work

```go
// Este es el patrón usado en play.go para monitorear descarga
timeout := time.After(10 * time.Minute)  // ← Canal que "gatilla" en 10 min
ticker := time.NewTicker(500 * time.Millisecond)  // ← Canal que gatilla cada 500ms

for !done {
    select {
    case <-timeout:  // ← Se ejecuta cuando pasan 10 minutos
        fmt.Println("❌ Timeout!")
        break
    case <-ticker.C:  // ← Se ejecuta cada 500ms
        fmt.Println("✓ Sigo descargando...")
    }
}
```

---

## Goroutines y Concurrencia

### ¿Por qué Goroutines?

```
THREADS DEL SO:
├─ Pesan ~1-2MB cada uno
├─ Tardía crear/destruir
├─ Caro cambiar contexto
└─ Máximo ~1000 por proceso

GOROUTINES:
├─ Pesan ~2KB cada uno
├─ Rápida crear/destruir
├─ Scheduler de Go es muy eficiente
└─ Puedes tener 100,000+ sin problema
```

### Modelo de Ejecución en p2pollo

```
┌─────────────────────────────────────────────────┐
│           Go Runtime (1 proceso SO)             │
│  ┌────────────────────────────────────────────┐ │
│  │  Scheduler de Go (distribuye goroutines)   │ │
│  └────────────────────────────────────────────┘ │
│       │                    │                    │
│       ↓ N threads SO (GOMAXPROCS)               │
│    ┌─────────────────────────────────┐          │
│    │ Thread 1  │ Thread 2 │ Thread 3 │          │
│    │  ┌─────┐  │  ┌─────┐ │  ┌─────┐ │          │
│    │  │ G1  │  │  │ G2  │ │  │ G3  │ │          │
│    │  ├─────┤  │  ├─────┤ │  ├─────┤ │          │
│    │  │ G4  │  │  │ G5  │ │  │ G6  │ │          │
│    │  └─────┘  │  └─────┘ │  └─────┘ │          │
│    └─────────────────────────────────┘          │
│                                                 │
│ G1-G6 = Goroutines (pueden ejecutar en paralelo)│
└─────────────────────────────────────────────────┘
```

### Ejemplo de Goroutines en p2pollo

**GOROUTINE 1: Copiar datos**
```go
go func() {
    for {
        select {
        case <-stopCopy:
            return
        default:
            n, _ := reader.Read(buf)
            writer.Write(buf[:n])
            // Bloquea si debe hacer I/O
        }
    }
}()
// Main continúa sin esperar
```

**GOROUTINE 2: Monitorear descarga**
```go
go func() {
    ticker := time.NewTicker(2 * time.Second)
    for {
        select {
        case <-done:
            return
        case <-ticker.C:
            // Imprime progreso
            // Duerme 2s, luego continúa
        }
    }
}()
// Main continúa sin esperar
```

**MAIN GOROUTINE: Orquesta todo**
```go
// Lanza goroutine 1 y 2
// Espera en p.Wait()  ← BLOQUEANTE aquí
// Las goroutines siguen ejecutando mientras main está bloqueada
```

---

## Flujo Completo de Descarga + Reproducción

### Fase 1: Inicio (0-5 segundos)

```
EVENTO                          CÓDIGO
─────────────────────────────────────────────
Usuario ejecuta comando    → runPlay() inicia
Carga config               → config.Load()
Crea cliente torrent       → client.New()
Conecta con peers          → anacrolix conecta
Solicita metadata          → WaitForInfo()
Recibe lista de archivos   → info.Files lista
```

### Fase 2: Buffer Inicial (5-30 segundos)

```
EVENTO                      GOROUTINE/THREAD
─────────────────────────────────────────────
Espera 50MB buffer          → MAIN (bloqueada en <-ticker.C)
Anacrolix descarga piezas   → P2P (anacrolix)
ticker cada 500ms           → MAIN imprime progreso
Alcanza 50MB                → MAIN se desbloquea
```

### Fase 3: Preparación Reproducción (30-45 segundos)

```
EVENTO                          GOROUTINE
─────────────────────────────────────────────
Crea archivo temporal       → MAIN
Lanza GOROUTINE 1           → GOROUTINE 1 de copia
├─ Crea reader anacrolix
├─ Abre archivo temporal
└─ Entra en loop de lectura

Monitorea buffer            → MAIN (select+ticker)
├─ Lee tamaño tmpFile
├─ Calcula progreso
└─ Espera minBufferToPlay

Lanza GOROUTINE 3           → GOROUTINE 3 de monitoreo
(no se usa aún)
```

### Fase 4: Reproducción (45s en adelante)

```
EVENTO                      RESPONSABLE
─────────────────────────────────────────────
Buffer alcanzado            → MAIN desbloquea
Lanza MPV                   → player.PlayFile()
├─ exec.LookPath("mpv")
├─ os.StartProcess()
└─ Ventana MPV abre

Lanza GOROUTINE 3           → GOROUTINE 3 monitorea
├─ Cada 2s imprime progreso
└─ t.BytesCompleted()

MAIN se bloquea en p.Wait() → MAIN espera cierre de MPV

USUARIO MIRA VIDEO          → MPV mostrador
mientras...

GOROUTINE 1 sigue copiando  → Copia datos descargados
GOROUTINE 3 sigue monitoreando → Imprime progreso
```

### Fase 5: Finalización

```
EVENTO                      GOROUTINE
─────────────────────────────────────────────
Usuario cierra MPV          → p.Wait() retorna
Envía señal a GOROUTINE 1   → close(stopCopy)
Envía señal a GOROUTINE 3   → close(done)

GOROUTINE 1 se detiene      → case <-stopCopy
GOROUTINE 3 se detiene      → case <-done

Limpia archivo temporal     → os.Remove(tmpPath)
MAIN termina                → return
```

---

## Analogía: Restaurant Concurrente

Imagina un restaurante como p2pollo:

```
CLIENTE (Usuario)
  ↓ Ordena video
  ↓
MESERO PRINCIPAL (MAIN)
  ├─ Toma orden: magnet link
  ├─ Avisa a cocina: comienza descarga
  ├─ GOROUTINE 1: COCINERO A
  │  ├─ Recibe ingredientes (anacrolix)
  │  ├─ Prepara plato (copia datos)
  │  ├─ Pone en bandeja (tmpFile)
  │  └─ Espera señal STOP (canal stopCopy)
  │
  ├─ GOROUTINE 2: MONITOR DE CALIDAD (no usado)
  │  └─ Verifica calidad cada N segundos
  │
  ├─ Mesero espera plato listo: p.Wait()
  │
  ├─ Cuando listo: llama CAMARERO
  │  └─ Lleva al cliente a mesa (MPV)
  │
  ├─ GOROUTINE 3: MESERO OBSERVADOR
  │  ├─ Cada 2s: ¿cliente feliz?
  │  ├─ ¿orden progresando?
  │  └─ Espera señal para parar (canal done)
  │
  └─ Cuando cliente termina:
     ├─ Cobro
     ├─ Limpieza (os.Remove)
     └─ Listo para próximo cliente
```

---

## Patrones de Sincronización Usados

### 1. Patrón: Producer-Consumer

```go
// PRODUCTOR (Goroutine 1: Copia)
go func() {
    for {
        data := read()
        write(data)
        // Produce bytes
    }
}()

// CONSUMIDOR (MPV)
// Lee el archivo temporal
// Los datos producidos aparecen a través del archivo

// Sincronización: El archivo sirve como "buffer"
```

### 2. Patrón: Fan-out / Fan-in

```go
// FAN-OUT: MAIN lanza múltiples goroutines
go func() { /* copia */ }()
go func() { /* monitorea */ }()

// FAN-IN: MAIN espera en p.Wait()
// (implícitamente, cuando cierre MPV, termina todo)
```

### 3. Patrón: Timeout

```go
timeout := time.After(10 * time.Minute)
ticker := time.NewTicker(500 * time.Millisecond)

select {
case <-timeout:
    // Acción si timeout
case <-ticker.C:
    // Acción repetida
}
```

---

## Ejercicios para Practicar

### Ejercicio 1: Entender Canales
```go
// Modifica play.go para usar un canal "progress" en lugar de prints
// - Goroutine 1 envía % cada 2s
// - Main recibe y imprime
// Hint: make(chan int)
```

### Ejercicio 2: Agregar Timeout de Copia
```go
// Si no hay progreso en 30s, cancela descarga
// Hint: select + time.After()
```

### Ejercicio 3: Multiple Archivos en Paralelo
```go
// Permite reproducir múltiples torrents
// Hint: use sync.WaitGroup para esperar todos
```

### Ejercicio 4: Pausa/Resume
```go
// Agrega pausa en el video
// Pausa = envía señal por canal
// Resume = envía otra señal
```

---

## Debugging de Goroutines

### Ver Goroutines Activas

```go
import "runtime"

// En cualquier parte del código:
fmt.Println("Goroutines activas:", runtime.NumGoroutine())

// Usar pprof:
import _ "net/http/pprof"
go http.ListenAndServe(":6060", nil)
// Luego: http://localhost:6060/debug/pprof/
```

### Detectar Deadlock

```go
// Si dos goroutines esperan mutuamente en canales:
ch1 := make(chan int)
ch2 := make(chan int)

go func() {
    <-ch1  // Espera ch1
    ch2 <- 1  // Envía a ch2
}()

go func() {
    <-ch2  // Espera ch2
    ch1 <- 1  // Envía a ch1
}()

// ⚠️ DEADLOCK: Ambas esperan infinitamente
```

---

## Conclusión

p2pollo demuestra:

1. **Goroutines**: Ejecución concurrente eficiente
2. **Channels**: Comunicación segura entre goroutines
3. **Select**: Espera múltiples eventos
4. **Mutex**: Protección de datos compartidos
5. **Patrones**: Producer-consumer, fan-out, timeout

Estos conceptos son fundamentales en Go y se aplican en muchas aplicaciones del mundo real.

---

## Referencias

- [Go Concurrency](https://go.dev/blog/pipelines)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)
- [Context Package](https://pkg.go.dev/context)
