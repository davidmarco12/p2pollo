# 🔄 Diagramas Detallados de Flujo - p2pollo

## Índice
1. [Diagrama de Secuencia Completo](#diagrama-de-secuencia-completo)
2. [Flujo de Descarga](#flujo-de-descarga)
3. [Flujo de Reproducción](#flujo-de-reproducción)
4. [Estados de Goroutines](#estados-de-goroutines)
5. [Comunicación por Canales](#comunicación-por-canales)

---

## Diagrama de Secuencia Completo

```
┏━━━━━━━━━━━━━┓  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃   USUARIO   ┃  ┃                      PROGRAMA P2POLLO                              ┃
┗━━━━━━━━━━━━━┛  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
     │                                      │
     │ p2pollo play "magnet:..."           │
     ├─────────────────────────────────────→│ START: runPlay()
     │                                      │
     │                                      ├─→ config.Load()
     │                                      │
     │                                      ├─→ client.New()
     │                                      │    ├─→ anacrolix.NewClient()
     │                                      │    └─→ Connect to peers
     │                                      │
     │                                      ├─→ AddMagnet(link)
     │                                      │    ├─→ Fetch metadata
     │                                      │    └─→ info.Files = [11 archivos]
     │                                      │
     │                                      ├─→ Buffer initial (50MB)
     │ [esperando...........................│ BLOQUEADA EN WHILE LOOP]
     │  3s              6s             9s  │
     │  [|||           ||||||||||      ||]│ Progreso: 0% → 100%
     │                                      │
     │                                      ├─→ Abre tmpFile
     │                                      │
     │                    GO 1              │
     │                    │                 │
     │    ┌────────────────┴─────────────────┬─ go CopiarDatos()
     │    │                                  │
     │    │  Loop: Read anacrolix → Write tmp
     │    │  [Lee 64KB → Escribe → Sync]
     │    │  [Lee 64KB → Escribe → Sync]
     │    │
     │    │                                  │
     │    │                    GO 2          │
     │    │                    │             │
     │    │   ┌────────────────┴─────────────┬─ go Monitorear()
     │    │   │                              │
     │    │   │ Tick cada 2s:
     │    │   │ [📊 56% | 69.8MB | 12 peers]
     │    │   │ [📊 78% | 96.7MB | 15 peers]
     │    │   │ [📊 100% | 123.3MB | 0 peers]
     │    │   │
     │                                      │
     │                                      ├─→ Buffer ready → Lanza MPV
     │                                      │    ├─→ exec.LookPath("mpv")
     │                                      │    ├─→ os.StartProcess()
     │                                      │    └─→ mpv.exe abre
     │                                      │
┌────┴──────────────────────────────────────│────────────────────────────┐
│    USUARIO MIRANDO VIDEO                  │    PROGRAMA DESCARGANDO    │
│    ├─ Abierto: 0:00:00                    │    │                       │
│    │ (video se reproduce)                │    ├─ Descargando: 87%      │
│    │ 0:00:30 (30 segundos después)       │    │ [||||||||||||    |||||]│
│    │ 0:01:15                              │    │                       │
│    │ 0:02:00                              │    ├─ Descargando: 100%    │
│    │ ...                                  │    │ [||||||||||||||||||||]│
│    │ 0:12:14 (final, cierra video)       │    │                       │
│    │ [Cierra MPV]                         │    │ Descarga completa     │
│                                            │    │                       │
└────┴──────────────────────────────────────│────────────────────────────┘
     │                                      │
     │                                      ├─→ p.Wait() RETORNA
     │                                      │    (MPV se cerró)
     │                                      │
     │                                      ├─→ close(done)
     │                                      │    close(stopCopy)
     │                                      │
     │                                      ├─→ os.Remove(tmpPath)
     │                                      │
     │                                      ├─ EXIT
     │←─────────────────────────────────────┤
     │
     ✓ Terminado
```

---

## Flujo de Descarga

### Visión Arquitectónica

```
┌──────────────────────────────────────────────────────────────┐
│                   ANACROLIX/TORRENT                          │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │          CACHE (~/.cache/p2pollo/...)              │    │
│  │  ┌─────────────────────────────────────────────┐   │    │
│  │  │ Sintel.mp4.part (123.25 MB)                │   │    │
│  │  │ [████████████████░░░░░░░░░░░░] 87%         │   │    │
│  │  │                                             │   │    │
│  │  │ Bytes leídos:                              │   │    │
│  │  │ - Pieza 0:   [✓] 16KB                      │   │    │
│  │  │ - Pieza 1:   [✓] 16KB                      │   │    │
│  │  │ - Pieza 2:   [║] 50% descargado            │   │    │
│  │  │ - Pieza 3:   [ ] por descargar            │   │    │
│  │  └─────────────────────────────────────────────┘   │    │
│  └─────────────────────────────────────────────────────┘    │
│              ↑                        ↑                      │
│              │                        │                      │
│     Read desde reader             Progress tracking         │
│     (secuencial)                  (para monitoreo)           │
└──────────────────────────────────────────────────────────────┘
     │                                 │
     ↓                                 ↓
    
┌─────────────────────────────────────────────────────────────┐
│           ARCHIVO TEMPORAL (Temp folder)                    │
│                                                             │
│  p2pollo-stream-{timestamp}.mp4 (123.25 MB)                │
│  [████████████████████████░░░░░░░░░░░░░░░░] 69%            │
│                                                             │
│  Se actualiza en TIEMPO REAL conforme se lee              │
│  de anacrolix cache                                        │
└─────────────────────────────────────────────────────────────┘
     │
     ↓
┌─────────────────────────────────────────────────────────────┐
│                    MPV PLAYER                               │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Sintel.mp4 - 1024x436 - 14:48 total              │   │
│  │ [████████████████░░░░░░░░░░░░░░░░] 00:30:15       │   │
│  │                                                   │   │
│  │ Viendo frame 765 de 21,312 frames               │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Loop de Lectura (GOROUTINE 1)

```
START: go CopiarDatos()
   │
   ├─ fileReader := t.NewFileReader(playFileIndex)
   │  (crea reader sobre anacrolix)
   │
   ├─ tmpFile := os.OpenFile(tmpPath)
   │
   └─ for {
        │
        ├─ select {
        │   │
        │   ├─ case <-stopCopy:
        │   │   └─ return  ← Parada ordenada
        │   │
        │   └─ default:
        │       │
        │       ├─ buf := make([]byte, 64*1024)  (64KB buffer)
        │       │
        │       ├─ n, err := fileReader.Read(buf)
        │       │   │
        │       │   ├─ Si n > 0:
        │       │   │  ├─ tmpFile.Write(buf[:n])
        │       │   │  ├─ tmpFile.Sync()  ← Fuerza escritura a disco
        │       │   │  └─ bytesWritten += int64(n)
        │       │   │
        │       │   └─ Si err == io.EOF:
        │       │      └─ LISTO: todos los bytes copiados
        │       │
        │       └─ repeat ↻
        │   }
        │}
        │
        └─ END: Goroutine termina
```

---

## Flujo de Reproducción

### Pre-Reproducción

```
FASE 1: Buffer Inicial (0-30 segundos)
═════════════════════════════════════

Esperar 50 MB descargados:
┌────────────────────────────────────────┐
│ for !bufferReady {                     │
│     select {                           │
│     case <-timeout (2 min):            │
│         → Error, salir                 │
│                                        │
│     case <-ticker.C (cada 500ms):      │
│         → Ver progreso                 │
│         → Si >= 50MB → bufferReady     │
│     }                                  │
│ }                                      │
└────────────────────────────────────────┘
Resultado: ✓ Buffer listo (51.5 MB)


FASE 2: Copia al Temporal (30-45 segundos)
═══════════════════════════════════════════

Monitorear mientras copia:
┌────────────────────────────────────────┐
│ for !bufferReady {                     │
│     select {                           │
│     case <-timeout (10 min):           │
│         → Error, salir                 │
│                                        │
│     case <-ticker.C (cada 500ms):      │
│         stat, _ := os.Stat(tmpPath)    │
│         → Mostrar progreso             │
│         → Si >= minBuffer → OK         │
│     }                                  │
│ }                                      │
└────────────────────────────────────────┘

Progreso visual:
⏳ Buffer: 16.8/123.3 MB (13%) [necesarios 62 MB]
⏳ Buffer: 35.2/123.3 MB (28%) [necesarios 62 MB]
⏳ Buffer: 69.8/123.3 MB (56%) [necesarios 62 MB]
✓ Buffer listo (69.8 MB) - Iniciando reproducción


FASE 3: Lanzar MPV (45 segundos)
═════════════════════════════════

player.PlayFile(tmpPath)
   │
   ├─ fullPath := exec.LookPath("mpv")
   │  → "C:\ProgramData\...\mpv.exe"
   │
   ├─ procAttr := os.ProcAttr{
   │    Files: [stdin, stdout, stderr]
   │  }
   │
   ├─ proc := os.StartProcess(fullPath, args, procAttr)
   │  → CREAR PROCESO
   │
   ├─ m.cmd = &exec.Cmd{ ... }
   │  → GUARDAR REFERENCIA
   │
   └─ return nil
      → ✓ MPV ABIERTO
```

### Durante-Reproducción

```
GOROUTINE 3: Monitoreo de Descarga
═════════════════════════════════

go func() {
    ticker := time.NewTicker(2 * time.Second)
    │
    └─ for {
        │
        ├─ select {
        │   │
        │   ├─ case <-done:
        │   │   └─ return  ← Parada
        │   │
        │   └─ case <-ticker.C:  ← Cada 2 segundos
        │       │
        │       ├─ bytesCompleted := t.BytesCompleted()
        │       ├─ progress := (completed / total) * 100
        │       ├─ peers := t.Peers()
        │       │
        │       └─ fmt.Println("📊 78% | 96.7 MB | 15 peers")
        │   }
        │}
}()


MAIN THREAD: Espera Cierre
═════════════════════════

waitErr := p.Wait()
   │
   └─ Bloquea hasta:
      ├─ Usuario cierra ventana MPV
      ├─ MPV se crashea
      ├─ Presiona ESC
      └─ O termina video

      [Aquí el usuario MIRA el video]


CRONOGRAMA TEMPORAL:
═══════════════════

T=0s     : Inicia descarga
T=5s     : Buffer 50MB completo
T=10s    : Copia comienza
T=20s    : Buffer 200MB listo
T=20s    : MPV ABRE
T=20-22s : 📊 0% | 123.0 MB | 24 peers
T=22-24s : 📊 23% | 28.4 MB | 20 peers
T=24-26s : 📊 45% | 55.7 MB | 18 peers
...
T=120s   : Usuario cierra MPV
T=121s   : p.Wait() retorna
T=122s   : Programa termina
```

---

## Estados de Goroutines

### Estado Machine

```
GOROUTINE 1: CopiarDatos
════════════════════════

    ┌─────────────────────┐
    │   CREADA (blocked)  │
    │  (esperando canal)  │
    └──────────┬──────────┘
               │
        (no recibe stopCopy)
               │
               ↓
    ┌─────────────────────┐
    │  EJECUTANDO (ready) │
    │  loop: read-write   │
    └──────────┬──────────┘
               │
        (lee piezas del torrent)
        (escribe a tmpFile)
        (se bloquea en I/O)
               │
        ┌──────┴──────────────────┐
        │                         │
   (I/O bloqueado)         (main envía close)
        │                         │
        ↓                         ↓
   ┌─────────────┐    ┌─────────────────────┐
   │   BLOCKED   │    │  RECIBE SEÑAL (done)│
   │ (I/O wait)  │    │   -> case <-stopCopy│
   └─────────────┘    └──────────┬──────────┘
        │ (I/O listo)             │
        └─────┬────────────────────┘
              │
              ↓
    ┌─────────────────────┐
    │  TERMINA (exited)   │
    │   return            │
    └─────────────────────┘
```

### Timeline Visual

```
MAIN         GOROUTINE 1      GOROUTINE 2      GOROUTINE 3      MPV
════════════════════════════════════════════════════════════════════

Start ─────────────────────────────────────────────────────────────→
│ 
├─→ go CopiarDatos ─→ [RUNNING]
│                     (read-write loop)
│
├─→ wait buffer ─┐
│   (blocked)    │
│                ├─→ go Monitorear ─→ [WAITING]
├─────────────────→ [READY]           (ticker)
│                  50MB ok
│
├─→ lanza MPV ──→ os.StartProcess() ─→ [NEW PROCESS]
│                                     (opens window)
│
├─→ p.Wait() ──┐ 
│   (blocked)  │ ← Aquí se bloquea esperando...
│              │
│ [USUARIO MIRANDO VIDEO]
│ ←─────────────────────────────────────→
│              │   (sigue copying)        │
│              │ + (monitoring)           │
│              │ (MPV reproduciendo)      │
│              │
│ (User closes MPV)
│              │
├──────────────┴──→ [p.Wait() returns]
│
├─→ close(done) ──→ GOROUTINE 2 [DONE]
│
├─→ close(stopCopy) ──→ GOROUTINE 1 [DONE]
│
├─→ cleanup
│
End ────────────────────────────────────────────────────────────→
```

---

## Comunicación por Canales

### Arquitectura de Canales

```
┌─────────────────────────────────────────────────────────────────┐
│                   MAIN GOROUTINE                                │
└─────────────────────────────────────────────────────────────────┘
     │                      │                        │
     │ crea canal           │ crea canal             │ usa time
     ↓                      ↓                        ↓
   
┌──────────┐            ┌─────┐              ┌──────────────┐
│stopCopy  │            │done │              │time.After()  │
│chan bool │            │chan │              │chan Time     │
└──────────┘            │bool │              └──────────────┘
     │                  └─────┘                     │
     │                    │                         │
     ├─ Enviador ─────────┼─ Enviador  ───────────┤
     │ (main)             │ (main)                 │ (runtime)
     │                    │                        │
     └────────┬───────────┴────────┬───────────────┘
              │                    │
        ┌─────┴─────┐         ┌────┴────────┐
        │ Receptores│         │  Receptores │
        ├─→ G1      │         ├─→ MAIN      │
        │  (copy)   │         │  (timeout)  │
        └───────────┘         └─────┬───────┘
                                    │
                              ┌─────┴─────┐
                              │ Receptores│
                              ├─→ MAIN    │
                              │  (ticker) │
                              └───────────┘
```

### Flujo Detallado

```
CANAL: stopCopy
═══════════════

Paso 1: Creación
────────────────
main() {
    stopCopy := make(chan bool)
}

Paso 2: Se envía por el canal
────────────────────────────
En main(), al final:
    close(stopCopy)  ← Cierra el canal
    
    Esto significa: "No voy a enviar más por este canal"
    
    Resultado: Todas las goroutines esperando en 
               <-stopCopy se despiertan


Paso 3: Se recibe en la goroutine
─────────────────────────────────
go func() {
    for {
        select {
        case <-stopCopy:  ← AQUÍ ESPERA
            fmt.Println("Recibí señal de parada")
            return
        default:
            // hacer trabajo
        }
    }
}()


CANAL: done (similar)
═════════════════════

Paso 1: Creación
────────────────
done := make(chan bool)

Paso 2: Se lanza goroutine que espera
───────────────────────────────────────
go func() {
    ticker := time.NewTicker(2 * time.Second)
    for {
        select {
        case <-done:      ← ESPERA AQUÍ
            return
        case <-ticker.C:
            // imprime progreso
        }
    }
}()

Paso 3: Se envía señal de parada
────────────────────────────────
En main(), cuando MPV se cierra:
    close(done)  ← Goroutine se despierta
    
Resultado: Goroutine de monitoreo termina
```

### Patrón Select

```
SELECT: Esperar múltiples eventos
═════════════════════════════════

ANTES (sin select):
───────────────────
reader.Read(buf)  ← Si tarda, programa se bloquea
// Más código después
writer.Write(buf)


CON SELECT:
───────────
select {
    case msg := <-channel1:
        // Ejecuta si channel1 envía algo
        
    case <-channel2:
        // Ejecuta si channel2 envía algo
        
    case <-time.After(5 * time.Second):
        // Ejecuta si pasan 5 segundos
        
    default:
        // Ejecuta si ninguno está listo
        // (opción no-bloqueante)
}

IMPORTANTE: Select BLOQUEA hasta que UNO de los casos esté listo
            Luego EJECUTA ESE CASO Y SALE


EJEMPLO EN P2POLLO:
──────────────────
select {
case <-downloadTimeout:
    // Usuario tardó mucho, salir con error
    fmt.Println("❌ Timeout!")
    break
    
case <-ticker.C:
    // Cada 500ms: chequear progreso
    fmt.Println("✓ Descargando... 50%")
    // Vuelve a esperar al select
}

Cronograma:
T=0ms   : Entra select
T=500ms : Se gatilla <-ticker.C → imprime → vuelve select
T=1000ms: Se gatilla <-ticker.C → imprime → vuelve select
T=1500ms: Se gatilla <-ticker.C → imprime → vuelve select
...
T=60s   : Archivo descargado → sale del loop
```

---

## Sincronización Entre Goroutines

### Problema: Race Condition

```
SIN PROTECCIÓN (❌ MALO):
═════════════════════════

var counter int

go func() {
    counter = counter + 1  // Goroutine A
}()

go func() {
    counter = counter + 1  // Goroutine B
}()

Posible ejecución intercalada:
Goroutine A: Lee counter (0) → suma 1 → Escribe 1
Goroutine B: Lee counter (0) → suma 1 → Escribe 1
Resultado: counter = 1 (INCORRECTO! Debería ser 2)

Esto es RACE CONDITION (condición de carrera)


CON MUTEX (✓ CORRECTO):
══════════════════════

var mu sync.Mutex
var counter int

go func() {
    mu.Lock()           // Solo esta goroutine
    counter++           // opera aquí
    mu.Unlock()         // Otra goroutine puede entrar
}()

go func() {
    mu.Lock()           // Espera a que la anterior termine
    counter++
    mu.Unlock()
}()

Ejecución:
Goroutine A: Entra en lock → counter = 1 → Unlock
Goroutine B: Entra en lock → counter = 2 → Unlock
Resultado: counter = 2 (CORRECTO)
```

### En p2pollo

```
MUTEX: internal/client/client.go
═══════════════════════════════

type Client struct {
    c   *torrent.Client
    mu  sync.RWMutex  ← Mutex para proteger 'c'
}

type Torrent struct {
    t   *torrent.Torrent
    mu  sync.RWMutex  ← Mutex para proteger 't'
}


LECTURA:
────────
func (t *Torrent) BytesCompleted() int64 {
    t.mu.RLock()        ← Lock para lectura
    defer t.mu.RUnlock()
    
    return t.t.BytesCompleted()
}

RLock = Read Lock (múltiples goroutines pueden leer)


ESCRITURA:
──────────
func (t *Torrent) Download() {
    t.mu.Lock()         ← Lock exclusivo
    defer t.mu.Unlock()
    
    t.t.DownloadAll()
}

Lock = Lock exclusivo (solo una goroutine)


SINCRONIZACIÓN:
───────────────
Goroutine A (lectura)    Goroutine B (lectura)    Goroutine C (escritura)
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ RLock()         │    │ RLock()          │    │ Lock()          │
│ Read data ✓     │    │ Read data ✓      │    │ [ESPERA...]     │
│ RUnlock()       │    │ RUnlock()        │    │ [ESPERA...]     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
  (ambas leen                (ambas leen        (espera a que
   al mismo tiempo)           al mismo tiempo)   otros terminen)
```

---

## Resumen Visual: Estados y Transiciones

```
┌─────────────────────────────────────────────────────────┐
│           CICLO DE VIDA DE UNA GOROUTINE                │
└─────────────────────────────────────────────────────────┘

1. CREACIÓN
──────────
go func() {
    // El scheduler agrega a la cola
}

    ↓

2. EN COLA (RUNNABLE)
─────────────────────
[G1][G2][G3][G4] ← Esperan CPU


    ↓

3. EJECUTANDO
─────────────
Goroutine en un thread real

    ├─ Opción A: Termina (return)
    │  ↓
    │  MUERTA (exit)
    │
    └─ Opción B: Se bloquea en:
       ├─ <-channel: Espera data
       ├─ mutex.Lock(): Espera lock
       ├─ I/O: Lee/escribe archivo
       ├─ time.Sleep(): Espera N segundos
       │  ↓
       │  BLOQUEADA (waiting)
       │  ↓
       │  (evento ocurre)
       │  ↓
       │  Vuelve a RUNNABLE
       │  ↓
       │  Vuelve a EJECUTANDO
```

---

## Debugging: Ver qué está pasando

```
OPCIÓN 1: Agregar prints/logs
═════════════════════════════

fmt.Println("[G1] Iniciando copia")
go func() {
    fmt.Println("[G1] Entrando en loop")
    for {
        fmt.Println("[G1] Leyendo...")
        n, _ := reader.Read(buf)
        fmt.Println("[G1] Escribiendo", n, "bytes")
        writer.Write(buf[:n])
    }
}()

Salida:
[G1] Iniciando copia
[G1] Entrando en loop
[G1] Leyendo...
[G1] Escribiendo 65536 bytes
[G1] Leyendo...
[G1] Escribiendo 65536 bytes
...


OPCIÓN 2: Ver goroutines activas
════════════════════════════════

import "runtime"

fmt.Println("Goroutines:", runtime.NumGoroutine())

// En el programa:
// Goroutines: 1 (solo main)
// Goroutines: 4 (main + 3 goroutines)
// Goroutines: 1 (volvemos a 1 al final)


OPCIÓN 3: Usar Profiler
═══════════════════════

import _ "net/http/pprof"

func main() {
    go http.ListenAndServe(":6060", nil)
    
    // main code
}

Luego abrir: http://localhost:6060/debug/pprof/goroutine

Ver todas las goroutines en vivo
```

---

## Conclusión: Mapa Mental

```
GO CONCURRENCY
├─ GOROUTINES (ejecución)
│  ├─ son ligeras (2KB vs 1MB threads)
│  ├─ scheduler automático
│  └─ creo con "go func() { ... }()"
│
├─ CHANNELS (comunicación)
│  ├─ enviar: ch <- value
│  ├─ recibir: value := <-ch
│  ├─ cerrar: close(ch)
│  └─ select para múltiples
│
├─ MUTEX (sincronización)
│  ├─ Lock/Unlock (exclusión mutua)
│  ├─ RLock/RUnlock (múltiples lectores)
│  └─ defer mu.Unlock() (siempre desbloquear)
│
├─ TIME (timeouts)
│  ├─ time.After() → canal de timeout
│  ├─ time.NewTicker() → canal periódico
│  └─ select para esperar eventos
│
└─ PATRONES
   ├─ producer-consumer (canales)
   ├─ fan-out (múltiples goroutines)
   ├─ fan-in (recolectar resultados)
   ├─ timeout (time.After)
   └─ worker pool (N goroutines)
```

Este es el corazón de Go y lo que hace que p2pollo funcione de manera eficiente! 🚀
